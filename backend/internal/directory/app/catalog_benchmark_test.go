package app

import (
	"context"
	"testing"
	"time"

	"simpeg/backend/internal/cache"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
)

// categoryPageSink menahan hasil pembacaan agar compiler tidak menghilangkan
// pekerjaan yang sedang diukur.
var categoryPageSink listing.Page[models.Kategori]

// BenchmarkCategoryListCacheHit mengukur biaya pembacaan daftar kategori
// ketika cache sudah terisi.
//
// Angka ini memperkirakan berapa banyak pekerjaan database yang dapat
// dihindari pada pembacaan berulang.
func BenchmarkCategoryListCacheHit(b *testing.B) {
	client := newMemoryCacheClient()
	repo := &countingCategoryRepository{categories: sampleCategories()}
	service := NewCategoryService(repo, cache.New(client, cache.Config{Namespace: "simpeg", TTL: 5 * time.Minute}))
	options := listing.Options{Page: 1, PageSize: 20, Sort: "kode", Order: "asc"}

	// Pemanasan cache supaya benchmark mengukur jalur hit.
	if _, err := service.List(context.Background(), options); err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		page, err := service.List(context.Background(), options)
		if err != nil {
			b.Fatal(err)
		}
		categoryPageSink = page
	}
	// Setiap pembacaan pada jalur hit tidak menyentuh repository lagi.
	b.ReportMetric(float64(repo.calls)/float64(b.N), "repo-calls/op")
}

// BenchmarkCategoryListWithoutCache mengukur biaya pembacaan daftar kategori
// ketika setiap pembacaan turun ke repository.
//
// Angka ini menjadi pembanding jalur hit cache pada benchmark di atas.
func BenchmarkCategoryListWithoutCache(b *testing.B) {
	repo := &countingCategoryRepository{categories: sampleCategories()}
	service := NewCategoryService(repo, nil)
	options := listing.Options{Page: 1, PageSize: 20, Sort: "kode", Order: "asc"}

	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		page, err := service.List(context.Background(), options)
		if err != nil {
			b.Fatal(err)
		}
		categoryPageSink = page
	}
	// Setiap pembacaan tanpa cache selalu menyentuh repository.
	b.ReportMetric(float64(repo.calls)/float64(b.N), "repo-calls/op")
}
