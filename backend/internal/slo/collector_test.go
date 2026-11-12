package slo

import (
	"context"
	"sync"
	"testing"
	"time"

	"simpeg/backend/internal/slo/domain"
)

// fakeStore mencatat batch metrik yang ditulis.
type fakeStore struct {
	mu      sync.Mutex
	batches [][]domain.Metric
	windows []domain.Window
	err     error
}

// RecordBatch menyimpan batch metrik yang diterima.
func (f *fakeStore) RecordBatch(_ context.Context, window domain.Window, metrics []domain.Metric) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	f.batches = append(f.batches, metrics)
	f.windows = append(f.windows, window)
	return nil
}

// total mengembalikan jumlah metrik yang tercatat.
func (f *fakeStore) total() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	count := 0
	for _, batch := range f.batches {
		count += len(batch)
	}
	return count
}

func TestCollectorAggregatesPerRouteAndMethod(t *testing.T) {
	store := &fakeStore{}
	collector := NewCollector(CollectorOptions{Store: store, WindowSize: time.Hour})

	collector.Record("GET", "/api/v1/employees", 200, 10*time.Millisecond)
	collector.Record("GET", "/api/v1/employees", 500, 20*time.Millisecond)
	collector.Record("POST", "/api/v1/employees", 201, 30*time.Millisecond)

	collector.mu.Lock()
	buckets := len(collector.windows)
	collector.mu.Unlock()
	if buckets != 1 {
		t.Fatalf("jumlah jendela di memori = %d, ingin 1", buckets)
	}

	if err := collector.flushAll(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.total() != 2 {
		t.Fatalf("metrik tercatat = %d, ingin 2 (satu per rute dan metode)", store.total())
	}
}

func TestCollectorCountsOnlyServerErrors(t *testing.T) {
	store := &fakeStore{}
	collector := NewCollector(CollectorOptions{Store: store, WindowSize: time.Hour})

	collector.Record("GET", "/api/v1/me", 200, time.Millisecond)
	collector.Record("GET", "/api/v1/me", 403, time.Millisecond)
	collector.Record("GET", "/api/v1/me", 404, time.Millisecond)
	collector.Record("GET", "/api/v1/me", 500, time.Millisecond)
	collector.Record("GET", "/api/v1/me", 503, time.Millisecond)

	if err := collector.flushAll(context.Background()); err != nil {
		t.Fatal(err)
	}
	batches := store.batches
	if len(batches) != 1 {
		t.Fatalf("jumlah batch = %d, ingin 1", len(batches))
	}
	metric := batches[0][0]
	if metric.Requests != 5 {
		t.Fatalf("permintaan = %d, ingin 5", metric.Requests)
	}
	if metric.Errors != 2 {
		t.Fatalf("galat = %d, ingin 2 (hanya 5xx)", metric.Errors)
	}
}

func TestFlushRemovesCompletedWindowsFromMemory(t *testing.T) {
	store := &fakeStore{}
	collector := NewCollector(CollectorOptions{Store: store, WindowSize: time.Hour})

	collector.Record("GET", "/api/v1/me", 200, time.Millisecond)
	collector.mu.Lock()
	before := len(collector.windows)
	collector.mu.Unlock()
	if before != 1 {
		t.Fatalf("jendela sebelum flush = %d, ingin 1", before)
	}

	// Jendela satu jam belum berakhir, jadi tidak ditulis dan tidak dibuang.
	if err := collector.Flush(context.Background()); err != nil {
		t.Fatal(err)
	}
	collector.mu.Lock()
	during := len(collector.windows)
	collector.mu.Unlock()
	if during != 1 {
		t.Fatalf("jendela selama flush = %d, ingin 1 (jendela berjalan belum dibuang)", during)
	}
	if store.total() != 0 {
		t.Fatalf("metrik tertulis = %d, ingin 0", store.total())
	}

	// Flush penuh menulis jendela berjalan dan mengosongkan memori.
	if err := collector.flushAll(context.Background()); err != nil {
		t.Fatal(err)
	}
	collector.mu.Lock()
	after := len(collector.windows)
	collector.mu.Unlock()
	if after != 0 {
		t.Fatalf("jendela setelah flush penuh = %d, ingin 0", after)
	}
	if store.total() != 1 {
		t.Fatalf("metrik tertulis = %d, ingin 1", store.total())
	}
}

func TestCollectorWithoutStoreDoesNothing(t *testing.T) {
	collector := NewCollector(CollectorOptions{})
	collector.Record("GET", "/api/v1/me", 200, time.Millisecond)
	if err := collector.Flush(context.Background()); err != nil {
		t.Fatal(err)
	}
}
