package holiday

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeStore struct {
	entries   []Entry
	replaced  []Entry
	listErr   error
	updateErr error
}

func (s *fakeStore) ReplaceYear(_ context.Context, _ string, _ int, entries []Entry) (int, error) {
	if s.updateErr != nil {
		return 0, s.updateErr
	}
	s.replaced = append([]Entry{}, entries...)
	s.entries = append([]Entry{}, entries...)
	return len(entries), nil
}

func (s *fakeStore) ListYear(_ context.Context, _ int) ([]Entry, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return append([]Entry{}, s.entries...), nil
}

type fakeProvider struct {
	entries []Entry
	err     error
}

func (p fakeProvider) FetchYear(_ context.Context, _ int) ([]Entry, error) {
	return p.entries, p.err
}

func testEntries() []Entry {
	return []Entry{
		{Tanggal: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Nama: "Tahun Baru 2026"},
		{Tanggal: time.Date(2026, 8, 17, 0, 0, 0, 0, time.UTC), Nama: "Hari Kemerdekaan RI"},
	}
}

func TestServiceRefreshReplacesStoredCalendar(t *testing.T) {
	store := &fakeStore{}
	service := NewService(store, fakeProvider{entries: testEntries()}, nil)

	stored, err := service.Refresh(context.Background(), "admin-1", 2026)
	if err != nil {
		t.Fatal(err)
	}
	if stored != 2 || len(store.replaced) != 2 {
		t.Fatalf("stored = %d, replaced = %d", stored, len(store.replaced))
	}
}

func TestServiceRefreshRejectsInvalidYear(t *testing.T) {
	service := NewService(&fakeStore{}, fakeProvider{entries: testEntries()}, nil)
	if _, err := service.Refresh(context.Background(), "admin-1", 1990); !errors.Is(err, ErrInvalidYear) {
		t.Fatalf("err = %v, want invalid year", err)
	}
}

func TestServiceRefreshWithoutProviderFails(t *testing.T) {
	service := NewService(&fakeStore{}, nil, nil)
	if _, err := service.Refresh(context.Background(), "admin-1", 2026); !errors.Is(err, ErrProviderNotConfigured) {
		t.Fatalf("err = %v, want provider not configured", err)
	}
}

func TestServiceRefreshPropagatesProviderError(t *testing.T) {
	service := NewService(&fakeStore{}, fakeProvider{err: ErrProvider{Detail: "status HTTP 429"}}, nil)
	var providerErr ErrProvider
	if _, err := service.Refresh(context.Background(), "admin-1", 2026); !errors.As(err, &providerErr) {
		t.Fatalf("err = %v, want provider error", err)
	}
}

func TestServiceListReturnsStoredEntries(t *testing.T) {
	store := &fakeStore{entries: testEntries()}
	service := NewService(store, nil, nil)

	entries, err := service.List(context.Background(), 2026)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[0].Nama != "Tahun Baru 2026" {
		t.Fatalf("entries = %+v", entries)
	}
}

func TestServiceListRejectsInvalidYear(t *testing.T) {
	service := NewService(&fakeStore{}, nil, nil)
	if _, err := service.List(context.Background(), 2200); !errors.Is(err, ErrInvalidYear) {
		t.Fatalf("err = %v, want invalid year", err)
	}
}
