package app

import (
	"context"
	"fmt"
	"time"

	"simpeg/backend/internal/audit"
	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/cache"
	"simpeg/backend/internal/directory/repository"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
)

// CategoryCacheTTL adalah umur cache daftar kategori. Data referensi dengan
// kardinalitas rendah dan jarang berubah, sehingga TTL pendek sudah memadai.
const CategoryCacheTTL = 5 * time.Minute

// cachedCategories adalah bentuk yang disimpan di cache.
type cachedCategories struct {
	Items []models.Kategori `json:"items"`
	Total int64             `json:"total"`
}

// CategoryService membaca daftar kategori pegawai.
//
// Cache bersifat opsional: tanpa cache, pembacaan tetap berjalan langsung ke
// PostgreSQL. Cache tidak pernah menjadi sumber kebenaran karena kegagalan
// Redis membuat pembacaan jatuh kembali ke store.
type CategoryService struct {
	repo  repository.CategoryRepository
	cache *cache.Cache
}

// NewCategoryService membuat service kategori dengan cache opsional.
func NewCategoryService(repo repository.CategoryRepository, cacheClient *cache.Cache) *CategoryService {
	return &CategoryService{repo: repo, cache: cacheClient}
}

// List mengembalikan daftar kategori sesuai filter dan pagination.
func (s *CategoryService) List(ctx context.Context, options listing.Options) (listing.Page[models.Kategori], error) {
	items, total, err := s.read(ctx, options)
	if err != nil {
		return listing.Page[models.Kategori]{}, err
	}
	return listing.Page[models.Kategori]{
		Items:      items,
		Page:       options.Page,
		PageSize:   options.PageSize,
		TotalItems: total,
		TotalPages: listing.PageCount(total, options.PageSize),
	}, nil
}

// read membaca daftar kategori, memakai cache bila tersedia.
func (s *CategoryService) read(ctx context.Context, options listing.Options) ([]models.Kategori, int64, error) {
	if s.cache == nil {
		return s.repo.ListCategories(ctx, options)
	}

	key := categoryCacheKey(options)
	cached := cachedCategories{}
	err := s.cache.ReadThrough(ctx, key, &cached, func(ctx context.Context) error {
		items, total, err := s.repo.ListCategories(ctx, options)
		if err != nil {
			return err
		}
		cached = cachedCategories{Items: items, Total: total}
		return nil
	})
	if err != nil {
		return nil, 0, err
	}
	return cached.Items, cached.Total, nil
}

// categoryCacheKey menyusun kunci cache yang memuat seluruh parameter daftar
// supaya setiap kombinasi filter memiliki entri sendiri.
func categoryCacheKey(options listing.Options) string {
	return fmt.Sprintf("kategori:page=%d:size=%d:sort=%s:order=%s:q=%s",
		options.Page, options.PageSize, options.Sort, options.Order, options.Search)
}

// AuditService menjalankan use case pembacaan jejak audit.
type AuditService struct {
	repo   repository.AuditRepository
	policy auth.Policy
	scope  scopeResolver
}

// NewAuditService membuat service jejak audit.
func NewAuditService(repo repository.AuditRepository, policy auth.Policy, scopeRepo repository.ScopeRepository) *AuditService {
	return &AuditService{repo: repo, policy: policy, scope: newScopeResolver(scopeRepo, policy)}
}

// List mengembalikan jejak audit sesuai filter dan cakupan principal.
//
// Jejak audit untuk role berscope unit dibatasi pada unit penugasan beserta
// turunannya. Role global membaca seluruh jejak.
func (s *AuditService) List(ctx context.Context, principal auth.Principal, options listing.Options) (listing.Page[models.AuditLog], error) {
	if !allows(s.policy, principal, "audit:read") {
		scope, err := s.scope.resolve(ctx, principal)
		if err != nil {
			return listing.Page[models.AuditLog]{}, err
		}
		if scope.Empty() {
			return listing.EmptyPage[models.AuditLog](options.Page, options.PageSize), nil
		}
		options.ScopeHomebaseIDs = scope.List()
	}

	entries, total, err := s.repo.ListAuditLogs(ctx, options)
	if err != nil {
		return listing.Page[models.AuditLog]{}, err
	}
	return listing.Page[models.AuditLog]{
		Items:      entries,
		Page:       options.Page,
		PageSize:   options.PageSize,
		TotalItems: total,
		TotalPages: listing.PageCount(total, options.PageSize),
	}, nil
}

// StreamAuditLogs mengirimkan seluruh jejak audit dalam cakupan principal ke
// fn, satu batch per batch, tanpa pagination. Urutan tetap kronologis naik.
// Izin dan cakupan sama persis dengan List.
func (s *AuditService) StreamAuditLogs(ctx context.Context, principal auth.Principal, options listing.Options, fn func(models.AuditLog) error) error {
	if !allows(s.policy, principal, "audit:read") {
		scope, err := s.scope.resolve(ctx, principal)
		if err != nil {
			return err
		}
		if scope.Empty() {
			return nil
		}
		options.ScopeHomebaseIDs = scope.List()
	}
	return s.repo.StreamAuditLogs(ctx, options, fn)
}

// Record mencatat satu entri jejak audit di luar operasi CRUD.
func (s *AuditService) Record(ctx context.Context, entry audit.Entry) error {
	return s.repo.RecordAudit(ctx, entry)
}
