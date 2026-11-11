package cache

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeClient struct {
	values     map[string]string
	getErr     error
	setErr     error
	deleteErr  error
	setCalls   int
	deleteCall int
}

func newFakeClient() *fakeClient {
	return &fakeClient{values: map[string]string{}}
}

func (c *fakeClient) Get(_ context.Context, key string) (string, error) {
	if c.getErr != nil {
		return "", c.getErr
	}
	return c.values[key], nil
}

func (c *fakeClient) Set(_ context.Context, key, value string, _ time.Duration) error {
	c.setCalls++
	if c.setErr != nil {
		return c.setErr
	}
	c.values[key] = value
	return nil
}

func (c *fakeClient) Delete(_ context.Context, key string) error {
	c.deleteCall++
	if c.deleteErr != nil {
		return c.deleteErr
	}
	delete(c.values, key)
	return nil
}

type category struct {
	Kode string `json:"kode"`
	Nama string `json:"nama"`
}

func TestCacheGetAndSet(t *testing.T) {
	client := newFakeClient()
	cache := New(client, Config{Namespace: "simpeg", TTL: time.Minute})

	if err := cache.Set(context.Background(), "kategori:DOSEN", category{Kode: "DOSEN", Nama: "Dosen"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.setCalls != 1 {
		t.Fatalf("set calls = %d, want 1", client.setCalls)
	}

	var loaded category
	if err := cache.Get(context.Background(), "kategori:DOSEN", &loaded); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loaded.Nama != "Dosen" {
		t.Fatalf("unexpected value: %+v", loaded)
	}
}

func TestCacheUsesNamespace(t *testing.T) {
	client := newFakeClient()
	cache := New(client, Config{Namespace: "simpeg"})

	if err := cache.Set(context.Background(), "kategori:DOSEN", category{Kode: "DOSEN"}); err != nil {
		t.Fatal(err)
	}
	if _, ok := client.values["simpeg:kategori:DOSEN"]; !ok {
		t.Fatalf("expected namespaced key, got %v", client.values)
	}
}

func TestCacheRejectsSensitiveKeys(t *testing.T) {
	client := newFakeClient()
	cache := New(client, Config{Namespace: "simpeg"})

	for _, key := range []string{
		"session:abc",
		"user-token:abc",
		"gaji:2026-11",
		"payroll-slip-1",
		"payment:status",
		"pembayaran:user-1",
		"credential:db",
	} {
		if err := cache.Set(context.Background(), key, category{}); !errors.Is(err, ErrForbiddenKey) {
			t.Errorf("key %q: expected ErrForbiddenKey, got %v", key, err)
		}
		if err := cache.Get(context.Background(), key, &category{}); !errors.Is(err, ErrForbiddenKey) {
			t.Errorf("key %q: expected ErrForbiddenKey on read, got %v", key, err)
		}
	}
	if client.setCalls != 0 {
		t.Fatal("sensitive keys must never be written")
	}
}

func TestCacheGetMissingKeyReturnsNotFound(t *testing.T) {
	cache := New(newFakeClient(), Config{})
	err := cache.Get(context.Background(), "kategori:TIDAK_ADA", &category{})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestReadThroughUsesCacheWhenPresent(t *testing.T) {
	client := newFakeClient()
	cache := New(client, Config{})
	if err := cache.Set(context.Background(), "kategori:DOSEN", category{Kode: "DOSEN", Nama: "Dosen"}); err != nil {
		t.Fatal(err)
	}

	loads := 0
	var loaded category
	err := cache.ReadThrough(context.Background(), "kategori:DOSEN", &loaded, func(context.Context) error {
		loads++
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loads != 0 {
		t.Fatal("source must not be read when cache holds the value")
	}
	if loaded.Nama != "Dosen" {
		t.Fatalf("unexpected value: %+v", loaded)
	}
}

func TestReadThroughFallsBackToSourceOnCacheMiss(t *testing.T) {
	client := newFakeClient()
	cache := New(client, Config{})

	loads := 0
	var loaded category
	err := cache.ReadThrough(context.Background(), "kategori:DOSEN", &loaded, func(context.Context) error {
		loads++
		loaded = category{Kode: "DOSEN", Nama: "Dosen"}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loads != 1 {
		t.Fatalf("source reads = %d, want 1", loads)
	}
	if client.setCalls != 1 {
		t.Fatalf("cache writes = %d, want 1", client.setCalls)
	}
}

func TestReadThroughSucceedsWhenCacheIsDown(t *testing.T) {
	client := newFakeClient()
	client.getErr = errors.New("redis tidak dapat dijangkau")
	cache := New(client, Config{})

	loads := 0
	var loaded category
	err := cache.ReadThrough(context.Background(), "kategori:DOSEN", &loaded, func(context.Context) error {
		loads++
		loaded = category{Kode: "DOSEN", Nama: "Dosen"}
		return nil
	})
	if err != nil {
		t.Fatalf("cache failure must not break the read: %v", err)
	}
	if loads != 1 {
		t.Fatalf("source reads = %d, want 1", loads)
	}
	if loaded.Nama != "Dosen" {
		t.Fatalf("unexpected value: %+v", loaded)
	}
}

func TestReadThroughPropagatesSourceFailure(t *testing.T) {
	cache := New(newFakeClient(), Config{})
	err := cache.ReadThrough(context.Background(), "kategori:DOSEN", &category{}, func(context.Context) error {
		return errors.New("database tidak tersedia")
	})
	if err == nil {
		t.Fatal("expected source failure to be propagated")
	}
}

func TestReadThroughRejectsSensitiveKey(t *testing.T) {
	cache := New(newFakeClient(), Config{})
	err := cache.ReadThrough(context.Background(), "session:abc", &category{}, func(context.Context) error {
		t.Fatal("source must not be read for a forbidden key")
		return nil
	})
	if !errors.Is(err, ErrForbiddenKey) {
		t.Fatalf("expected ErrForbiddenKey, got %v", err)
	}
}

func TestDefaultTTLIsApplied(t *testing.T) {
	cache := New(newFakeClient(), Config{})
	if cache.config.TTL != DefaultTTL {
		t.Fatalf("ttl = %v, want %v", cache.config.TTL, DefaultTTL)
	}
}
