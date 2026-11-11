package holiday

import (
	"context"
	"fmt"

	"simpeg/backend/internal/cache"
)

// Store adalah port persistensi kalender hari libur.
type Store interface {
	ReplaceYear(ctx context.Context, actorID string, year int, entries []Entry) (int, error)
	ListYear(ctx context.Context, year int) ([]Entry, error)
}

// Service menjalankan use case kalender hari libur.
type Service struct {
	store    Store
	provider Provider
	cache    *cache.Cache
}

// NewService membuat layanan kalender hari libur.
//
// provider boleh nil bila penyedia belum dikonfigurasi; pembacaan kalender
// dari database tetap bekerja, hanya penyegaran yang menolak. cache boleh
// nil; pembacaan langsung ke database.
func NewService(store Store, provider Provider, holidayCache *cache.Cache) *Service {
	return &Service{store: store, provider: provider, cache: holidayCache}
}

// Refresh mengambil kalender dari penyedia lalu menggantikan entri
// tersimpan untuk tahun yang diminta.
func (s *Service) Refresh(ctx context.Context, actorID string, year int) (int, error) {
	if err := ValidateYear(year); err != nil {
		return 0, err
	}
	if s.store == nil {
		return 0, fmt.Errorf("holiday store is required")
	}
	if s.provider == nil {
		return 0, ErrProviderNotConfigured
	}
	entries, err := s.provider.FetchYear(ctx, year)
	if err != nil {
		return 0, err
	}
	stored, err := s.store.ReplaceYear(ctx, actorID, year, entries)
	if err != nil {
		return 0, err
	}
	if s.cache != nil {
		// Kegagalan menghapus cache tidak menggagalkan penyegaran; TTL
		// membatasi umur data basi.
		_ = s.cache.Delete(ctx, cacheKey(year))
	}
	return stored, nil
}

// List membaca kalender tersimpan untuk tahun yang diminta.
func (s *Service) List(ctx context.Context, year int) ([]Entry, error) {
	if err := ValidateYear(year); err != nil {
		return nil, err
	}
	if s.store == nil {
		return nil, fmt.Errorf("holiday store is required")
	}
	var entries []Entry
	if s.cache == nil {
		if err := s.load(ctx, year, &entries); err != nil {
			return nil, err
		}
		return entries, nil
	}
	err := s.cache.ReadThrough(ctx, cacheKey(year), &entries, func(ctx context.Context) error {
		return s.load(ctx, year, &entries)
	})
	if err != nil {
		return nil, err
	}
	return entries, nil
}

func (s *Service) load(ctx context.Context, year int, target *[]Entry) error {
	rows, err := s.store.ListYear(ctx, year)
	if err != nil {
		return err
	}
	*target = rows
	return nil
}

func cacheKey(year int) string {
	return fmt.Sprintf("holiday:%d", year)
}
