package repository

import (
	"context"
	"errors"
	"fmt"

	"simpeg/backend/internal/audit"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
	"simpeg/backend/internal/payroll/domain"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ListPotongan membaca daftar komponen potongan sesuai filter dan pagination.
func (s GormStore) ListPotongan(ctx context.Context, options listing.Options) ([]models.KomponenPotongan, int64, error) {
	query := s.DB.WithContext(ctx).Model(&models.KomponenPotongan{})
	if options.UserID != "" {
		query = query.Where("userId = ?", options.UserID)
	}
	if len(options.ScopeHomebaseIDs) > 0 {
		query = query.Where("userId IN (SELECT id FROM user WHERE homebaseId IN ?)", options.ScopeHomebaseIDs)
	}
	if options.Search != "" {
		pattern := "%" + escapeLike(options.Search) + "%"
		query = query.Where("jenisPotongan ILIKE ? ESCAPE '\\'", pattern)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count potongan: %w", err)
	}
	var items []models.KomponenPotongan
	order := listing.OrderClause(domain.PotonganSortColumns, options.Sort, options.Order, "createdAt")
	err := query.Order(order).Offset((options.Page - 1) * options.PageSize).Limit(options.PageSize).Find(&items).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list potongan: %w", err)
	}
	return items, total, nil
}

func (s GormStore) GetPotongan(ctx context.Context, id string) (models.KomponenPotongan, error) {
	var item models.KomponenPotongan
	err := s.DB.WithContext(ctx).Where("id = ?", id).First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.KomponenPotongan{}, domain.ErrNotFound
		}
		return models.KomponenPotongan{}, fmt.Errorf("read potongan: %w", err)
	}
	return item, nil
}

func (s GormStore) CreatePotongan(ctx context.Context, actorID string, item models.KomponenPotongan) (models.KomponenPotongan, error) {
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&item).Error; err != nil {
			return classifyRecordError(err)
		}
		return createPayrollAudit(ctx, tx, actorID, audit.ActionPotonganCreated, &item.ID, &item.UserId)
	})
	return item, err
}

func (s GormStore) UpdatePotongan(ctx context.Context, actorID, id string, patch domain.KomponenPotonganPatch) (models.KomponenPotongan, error) {
	updates := potonganColumns(patch)
	var item models.KomponenPotongan
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&item).Error; err != nil {
			return classifyRecordError(err)
		}
		if err := tx.Model(&item).Updates(updates).Error; err != nil {
			return classifyRecordError(err)
		}
		if err := tx.First(&item, "id = ?", id).Error; err != nil {
			return fmt.Errorf("reload potongan: %w", err)
		}
		return createPayrollAudit(ctx, tx, actorID, audit.ActionPotonganUpdated, optionalRef(id), &item.UserId)
	})
	return item, err
}

func (s GormStore) ArchivePotongan(ctx context.Context, actorID, id string) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var item models.KomponenPotongan
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&item).Error; err != nil {
			return classifyRecordError(err)
		}
		result := tx.Model(&models.KomponenPotongan{}).Where("id = ?").Update("aktif", false)
		if result.Error != nil {
			return fmt.Errorf("archive potongan: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return domain.ErrNotFound
		}
		return createPayrollAudit(ctx, tx, actorID, audit.ActionPotonganArchived, optionalRef(id), &item.UserId)
	})
}

func potonganColumns(patch domain.KomponenPotonganPatch) map[string]any {
	updates := make(map[string]any)
	if patch.JenisPotongan.Present {
		if patch.JenisPotongan.Null {
			updates["jenisPotongan"] = nil
		} else {
			updates["jenisPotongan"] = patch.JenisPotongan.Value
		}
	}
	if patch.Persentase.Present {
		if patch.Persentase.Null {
			updates["persentase"] = nil
		} else {
			updates["persentase"] = patch.Persentase.Value
		}
	}
	if patch.Nominal.Present {
		if patch.Nominal.Null {
			updates["nominal"] = nil
		} else {
			updates["nominal"] = patch.Nominal.Value
		}
	}
	if patch.PorsiInstitusi.Present {
		if patch.PorsiInstitusi.Null {
			updates["porsiInstitusi"] = nil
		} else {
			updates["porsiInstitusi"] = patch.PorsiInstitusi.Value
		}
	}
	if patch.PorsiPegawai.Present {
		if patch.PorsiPegawai.Null {
			updates["porsiPegawai"] = nil
		} else {
			updates["porsiPegawai"] = patch.PorsiPegawai.Value
		}
	}
	if patch.Aktif.Present {
		if patch.Aktif.Null {
			updates["aktif"] = nil
		} else {
			updates["aktif"] = patch.Aktif.Value
		}
	}
	return updates
}
