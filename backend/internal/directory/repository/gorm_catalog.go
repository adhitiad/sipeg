package repository

import (
	"context"
	"fmt"

	"simpeg/backend/internal/audit"
	"simpeg/backend/internal/directory/domain"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
)

// ListAuditLogs membaca jejak perubahan sesuai filter dan pagination.
func (s GormStore) ListAuditLogs(ctx context.Context, options listing.Options) ([]models.AuditLog, int64, error) {
	query := s.DB.WithContext(ctx).Model(&models.AuditLog{})
	if options.Aksi != "" {
		query = query.Where("aksi = ?", options.Aksi)
	}
	if options.Search != "" {
		pattern := "%" + escapeLike(options.Search) + "%"
		query = query.Where("(aksi ILIKE ? ESCAPE '\\' OR keterangan ILIKE ? ESCAPE '\\')", pattern, pattern)
	}
	if len(options.ScopeHomebaseIDs) > 0 {
		query = query.Where(`"unitId" IN ?`, options.ScopeHomebaseIDs)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count audit logs: %w", err)
	}
	var entries []models.AuditLog
	order := listing.OrderClause(domain.AuditLogSortColumns, options.Sort, options.Order, "createdAt")
	err := query.Order(order).Offset((options.Page - 1) * options.PageSize).Limit(options.PageSize).Find(&entries).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list audit logs: %w", err)
	}
	return entries, total, nil
}

// RecordAudit menulis satu entri jejak audit di luar transaksi store.
// Dipakai untuk aksi yang tercatat di luar operasi CRUD, misalnya pembacaan
// jejak audit itu sendiri.
func (s GormStore) RecordAudit(ctx context.Context, entry audit.Entry) error {
	return createAudit(ctx, s.DB.WithContext(ctx), entry)
}

// ListCategories membaca daftar kategori pegawai aktif sesuai pagination.
func (s GormStore) ListCategories(ctx context.Context, options listing.Options) ([]models.Kategori, int64, error) {
	query := s.DB.WithContext(ctx).Model(&models.Kategori{}).Where("aktif = ?", true)
	if options.Search != "" {
		pattern := "%" + escapeLike(options.Search) + "%"
		query = query.Where("(kode ILIKE ? ESCAPE '\\' OR nama ILIKE ? ESCAPE '\\')", pattern, pattern)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count categories: %w", err)
	}
	var categories []models.Kategori
	order := listing.OrderClause(domain.CategorySortColumns, options.Sort, options.Order, "kode")
	err := query.Order(order).Offset((options.Page - 1) * options.PageSize).Limit(options.PageSize).Find(&categories).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list categories: %w", err)
	}
	return categories, total, nil
}

// ListUnitScopeIDs mengembalikan unit yang berada dalam cakupan sebuah unit:
// unit itu sendiri beserta seluruh turunannya pada pohon Homebase.
// Cakupan ini dipakai untuk membatasi operator dan atasan ke unit yang ditugaskan.
//
// Query memakai recursive CTE agar turunan menggambar foreign key parentId
// pada skema acuan (kolom "parentId" perlu kutip ganda di PostgreSQL).
func (s GormStore) ListUnitScopeIDs(ctx context.Context, rootID string) ([]string, error) {
	if rootID == "" {
		return nil, nil
	}
	var unitIDs []string
	err := s.DB.WithContext(ctx).Raw(`
		WITH RECURSIVE subtree AS (
			SELECT id FROM "Homebase" WHERE id = ?
			UNION ALL
			SELECT child.id FROM "Homebase" child
			JOIN subtree ON child."parentId" = subtree.id
		)
		SELECT id FROM subtree`, rootID).Scan(&unitIDs).Error
	if err != nil {
		return nil, fmt.Errorf("resolve unit scope: %w", err)
	}
	return unitIDs, nil
}
