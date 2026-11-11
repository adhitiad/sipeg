package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"simpeg/backend/internal/cache"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
)

type countingCategoryRepository struct {
	categories []models.Kategori
	calls      int
}

func (r *countingCategoryRepository) ListCategories(context.Context, listing.Options) ([]models.Kategori, int64, error) {
	r.calls++
	return r.categories, int64(len(r.categories)), nil
}

type memoryCacheClient struct {
	values   map[string]string
	getErr   error
	setCalls int
}

func newMemoryCacheClient() *memoryCacheClient {
	return &memoryCacheClient{values: map[string]string{}}
}

func (c *memoryCacheClient) Get(_ context.Context, key string) (string, error) {
	if c.getErr != nil {
		return "", c.getErr
	}
	return c.values[key], nil
}

func (c *memoryCacheClient) Set(_ context.Context, key, value string, _ time.Duration) error {
	c.setCalls++
	if c.getErr != nil {
		return c.getErr
	}
	c.values[key] = value
	return nil
}

func (c *memoryCacheClient) Delete(_ context.Context, key string) error {
	delete(c.values, key)
	return nil
}

func sampleCategories() []models.Kategori {
	return []models.Kategori{
		{BaseModel: models.BaseModel{IDModel: models.IDModel{ID: "kat-1"}}, Kode: "DOSEN", Nama: "Dosen", Aktif: true},
	}
}

func TestCategoryServiceWithoutCacheReadsRepository(t *testing.T) {
	repo := &countingCategoryRepository{categories: sampleCategories()}
	service := NewCategoryService(repo, nil)

	page, err := service.List(context.Background(), listing.Options{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page.Items) != 1 || page.TotalItems != 1 {
		t.Fatalf("unexpected result: %+v total=%d", page.Items, page.TotalItems)
	}
	if repo.calls != 1 {
		t.Fatalf("repository calls = %d, want 1", repo.calls)
	}
}

func TestCategoryServiceUsesCacheOnSecondRead(t *testing.T) {
	client := newMemoryCacheClient()
	repo := &countingCategoryRepository{categories: sampleCategories()}
	service := NewCategoryService(repo, cache.New(client, cache.Config{Namespace: "simpeg", TTL: 5 * time.Minute}))

	options := listing.Options{Page: 1, PageSize: 20, Sort: "kode", Order: "asc"}

	first, err := service.List(context.Background(), options)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	second, err := service.List(context.Background(), options)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(first.Items) != 1 || len(second.Items) != 1 {
		t.Fatalf("unexpected results: %+v %+v", first.Items, second.Items)
	}
	if repo.calls != 1 {
		t.Fatalf("repository calls = %d, want 1 — cache harus dipakai pada pembacaan kedua", repo.calls)
	}
	if client.setCalls != 1 {
		t.Fatalf("cache writes = %d, want 1", client.setCalls)
	}
}

func TestCategoryServiceFallsBackToRepositoryWhenCacheUnavailable(t *testing.T) {
	client := newMemoryCacheClient()
	client.getErr = errors.New("redis tidak dapat dijangkau")
	repo := &countingCategoryRepository{categories: sampleCategories()}
	service := NewCategoryService(repo, cache.New(client, cache.Config{Namespace: "simpeg", TTL: 5 * time.Minute}))

	page, err := service.List(context.Background(), listing.Options{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("cache failure must not break the read: %v", err)
	}
	if len(page.Items) != 1 || page.TotalItems != 1 {
		t.Fatalf("unexpected result: %+v total=%d", page.Items, page.TotalItems)
	}
	if repo.calls != 1 {
		t.Fatalf("repository calls = %d, want 1", repo.calls)
	}
}

func TestCategoryCacheKeyDistinguishesFilters(t *testing.T) {
	base := listing.Options{Page: 1, PageSize: 20, Sort: "kode", Order: "asc"}

	otherPage := base
	otherPage.Page = 2
	if categoryCacheKey(base) == categoryCacheKey(otherPage) {
		t.Fatal("page must be part of the cache key")
	}

	otherSearch := base
	otherSearch.Search = "DOSEN"
	if categoryCacheKey(base) == categoryCacheKey(otherSearch) {
		t.Fatal("search must be part of the cache key")
	}

	otherSize := base
	otherSize.PageSize = 50
	if categoryCacheKey(base) == categoryCacheKey(otherSize) {
		t.Fatal("page size must be part of the cache key")
	}
}
