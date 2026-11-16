// Adapter GORM untuk baca Surat Keputusan.
package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"simpeg/backend/internal/listing"

	"simpeg/backend/internal/sk/domain"
)

// GormStore mengimplementasikan Store dengan PostgreSQL.
type GormStore struct {
	DB *gorm.DB
}

// List mengembalikan SK yang cocok dengan filter. Cakupan unit (homebase +
// turunannya) diterapkan lewat `user."homebaseId"`.
func (s GormStore) List(ctx context.Context, filter domain.Filter) (listing.Page[domain.Row], error) {
	filter = filter.Normalize()
	query := s.baseQuery(ctx).Where(`1 = 1`)

	if filter.Search != "" {
		like := "%" + filter.Search + "%"
		query = query.Where(`(sk."nomorSk" ILIKE ? OR u.nama ILIKE ?)`, like, like)
	}
	if filter.Jenis != "" {
		query = query.Where(`sk."jenis" = ?`, filter.Jenis)
	}
	if filter.Penerbit != "" {
		query = query.Where(`sk."penerbit" = ?`, filter.Penerbit)
	}
	if filter.UserID != "" {
		query = query.Where(`sk."userId" = ?`, filter.UserID)
	}
	if filter.HomebaseID != "" {
		query = query.Where(`u."homebaseId" = ?`, filter.HomebaseID)
	}
	if len(filter.ScopeHomebaseIDs) > 0 {
		query = query.Where(`u."homebaseId" IN ?`, filter.ScopeHomebaseIDs)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return listing.Page[domain.Row]{}, fmt.Errorf("count surat keputusan: %w", err)
	}

	order := listing.OrderClause(domain.SortColumns, filter.Sort, filter.Order, domain.FallbackSort)
	items := make([]domain.Row, 0, filter.PageSize)
	err := query.
		Select(`sk.*, u.nama AS nama, u."homebaseId" AS "homebaseId"`).
		Order(order).
		Limit(filter.PageSize).
		Offset((filter.Page - 1) * filter.PageSize).
		Scan(&items).Error
	if err != nil {
		return listing.Page[domain.Row]{}, fmt.Errorf("list surat keputusan: %w", err)
	}

	return listing.Page[domain.Row]{
		Items:      items,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalItems: total,
		TotalPages: listing.PageCount(total, filter.PageSize),
	}, nil
}

// Get membaca satu SK beserta nama pegawai.
func (s GormStore) Get(ctx context.Context, id string) (domain.Row, error) {
	var row domain.Row
	err := s.baseQuery(ctx).
		Select(`sk.*, u.nama AS nama, u."homebaseId" AS "homebaseId"`).
		Where(`sk.id = ?`, id).
		Scan(&row).Error
	if err != nil {
		return domain.Row{}, fmt.Errorf("read surat keputusan: %w", err)
	}
	if row.ID == "" {
		return domain.Row{}, ErrNotFound
	}
	return row, nil
}

// baseQuery menyiapkan join SK dengan data pegawai.
func (s GormStore) baseQuery(ctx context.Context) *gorm.DB {
	return s.DB.WithContext(ctx).
		Table(`"suratKeputusan" AS sk`).
		Joins(`JOIN "user" u ON u.id = sk."userId"`)
}
