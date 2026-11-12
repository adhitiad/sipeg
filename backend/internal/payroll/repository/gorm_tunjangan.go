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

// ListTunjangan membaca daftar komponen tunjangan sesuai filter dan pagination.
func (s GormStore) ListTunjangan(ctx context.Context, options listing.Options) ([]models.KomponenTunjangan, int64, error) {
	query := s.DB.WithContext(ctx).Model(&models.KomponenTunjangan{})
	if options.UserID != "" {
		query = query.Where("userId = ?", options.UserID)
	}
	if len(options.ScopeHomebaseIDs) > 0 {
		query = query.Where("userId IN (SELECT id FROM user WHERE homebaseId IN ?)", options.ScopeHomebaseIDs)
	}
	if options.Search != "" {
		pattern := "%" + escapeLike(options.Search) + "%"
		query = query.Where("jenisTunjangan ILIKE ? ESCAPE '\\'", pattern)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count tunjangan: %w", err)
	}
	var items []models.KomponenTunjangan
	order := listing.OrderClause(domain.TunjanganSortColumns, options.Sort, options.Order, "createdAt")
	err := query.Order(order).Offset((options.Page - 1) * options.PageSize).Limit(options.PageSize).Find(&items).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list tunjangan: %w", err)
	}
	return items, total, nil
}

func (s GormStore) GetTunjangan(ctx context.Context, id string) (models.KomponenTunjangan, error) {
	var item models.KomponenTunjangan
	err := s.DB.WithContext(ctx).Where("id = ?", id).First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.KomponenTunjangan{}, domain.ErrNotFound
		}
		return models.KomponenTunjangan{}, fmt.Errorf("read tunjangan: %w", err)
	}
	return item, nil
}

func (s GormStore) CreateTunjangan(ctx context.Context, actorID string, item models.KomponenTunjangan) (models.KomponenTunjangan, error) {
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&item).Error; err != nil {
			return classifyRecordError(err)
		}
		return createPayrollAudit(ctx, tx, actorID, audit.ActionTunjanganCreated, &item.ID, &item.UserId)
	})
	return item, err
}

func (s GormStore) UpdateTunjangan(ctx context.Context, actorID, id string, patch domain.KomponenTunjanganPatch) (models.KomponenTunjangan, error) {
	updates := tunjanganColumns(patch)
	var item models.KomponenTunjangan
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&item).Error; err != nil {
			return classifyRecordError(err)
		}
		if err := tx.Model(&item).Updates(updates).Error; err != nil {
			return classifyRecordError(err)
		}
		if err := tx.First(&item, "id = ?", id).Error; err != nil {
			return fmt.Errorf("reload tunjangan: %w", err)
		}
		return createPayrollAudit(ctx, tx, actorID, audit.ActionTunjanganUpdated, optionalRef(id), &item.UserId)
	})
	return item, err
}

func (s GormStore) ArchiveTunjangan(ctx context.Context, actorID, id string) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var item models.KomponenTunjangan
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&item).Error; err != nil {
			return classifyRecordError(err)
		}
		result := tx.Model(&models.KomponenTunjangan{}).Where("id = ?").Update("aktif", false)
		if result.Error != nil {
			return fmt.Errorf("archive tunjangan: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return domain.ErrNotFound
		}
		return createPayrollAudit(ctx, tx, actorID, audit.ActionTunjanganArchived, optionalRef(id), &item.UserId)
	})
}

func tunjanganColumns(patch domain.KomponenTunjanganPatch) map[string]any {
	updates := make(map[string]any)
	if patch.JenisTunjanging.Present {
		if patch.JenisTunjanging.Null {
			updates["jenisTunjangan"] = nil
		} else {
			updates["jenisTunjangan"] = patch.JenisTunjanging.Value
		}
	}
	if patch.Nilai.Present {
		if patch.Nilai.Null {
			updates["nilai"] = nil
		} else {
			updates["nilai"] = patch.Nilai.Value
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
