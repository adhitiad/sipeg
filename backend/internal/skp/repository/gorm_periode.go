package repository

import (
	"context"
	"errors"
	"fmt"

	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
	skpdomain "simpeg/backend/internal/skp/domain"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s GormStore) ListSKPPeriode(ctx context.Context, options listing.Options) ([]models.SKPPeriode, int64, error) {
	query := s.DB.WithContext(ctx).Model(&models.SKPPeriode{})
	if options.Status != "" {
		query = query.Where("status = ?", options.Status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count skp periode: %w", err)
	}
	var items []models.SKPPeriode
	order := listing.OrderClause(sortColumnsSKPPeriode, options.Sort, options.Order, "tahun")
	err := query.Order(order).
		Offset((options.Page - 1) * options.PageSize).
		Limit(options.PageSize).
		Find(&items).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list skp periode: %w", err)
	}
	return items, total, nil
}

func (s GormStore) GetSKPPeriode(ctx context.Context, id string) (models.SKPPeriode, error) {
	var periode models.SKPPeriode
	err := s.DB.WithContext(ctx).Where("id = ?", id).First(&periode).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.SKPPeriode{}, skpdomain.ErrNotFound
		}
		return models.SKPPeriode{}, fmt.Errorf("read skp periode: %w", err)
	}
	return periode, nil
}

func (s GormStore) CreateSKPPeriode(ctx context.Context, actorID string, periode models.SKPPeriode) (models.SKPPeriode, error) {
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&periode).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return skpdomain.ErrConflict
			}
			return fmt.Errorf("create skp periode: %w", err)
		}
		return nil
	})
	return periode, err
}

func (s GormStore) UpdateSKPPeriode(ctx context.Context, actorID, id string, updates map[string]any) (models.SKPPeriode, error) {
	var periode models.SKPPeriode
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&periode).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return skpdomain.ErrNotFound
			}
			return fmt.Errorf("read skp periode: %w", err)
		}
		if err := tx.Model(&periode).Updates(updates).Error; err != nil {
			return fmt.Errorf("update skp periode: %w", err)
		}
		if err := tx.First(&periode, "id = ?", id).Error; err != nil {
			return fmt.Errorf("reload skp periode: %w", err)
		}
		return nil
	})
	return periode, err
}

func (s GormStore) DeleteSKPPeriode(ctx context.Context, actorID, id string) error {
	result := s.DB.WithContext(ctx).Where("id = ?", id).Delete(&models.SKPPeriode{})
	if result.Error != nil {
		return fmt.Errorf("delete skp periode: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return skpdomain.ErrNotFound
	}
	return nil
}

func (s GormStore) ListSKPTarget(ctx context.Context, options listing.Options) ([]models.SKPTargetKinerja, int64, error) {
	query := s.DB.WithContext(ctx).Model(&models.SKPTargetKinerja{})
	if options.UserID != "" {
		query = query.Where(`"userId" = ?`, options.UserID)
	}
	if options.Status != "" {
		query = query.Where("status = ?", options.Status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count skp target: %w", err)
	}
	var items []models.SKPTargetKinerja
	order := listing.OrderClause(sortColumnsSKPTarget, options.Sort, options.Order, "createdAt")
	err := query.Order(order).
		Offset((options.Page - 1) * options.PageSize).
		Limit(options.PageSize).
		Find(&items).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list skp target: %w", err)
	}
	return items, total, nil
}

func (s GormStore) GetSKPTarget(ctx context.Context, id string) (models.SKPTargetKinerja, error) {
	var target models.SKPTargetKinerja
	err := s.DB.WithContext(ctx).Where("id = ?", id).First(&target).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.SKPTargetKinerja{}, skpdomain.ErrNotFound
		}
		return models.SKPTargetKinerja{}, fmt.Errorf("read skp target: %w", err)
	}
	return target, nil
}

func (s GormStore) CreateSKPTarget(ctx context.Context, actorID string, target models.SKPTargetKinerja) (models.SKPTargetKinerja, error) {
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&models.SKPTargetKinerja{}).
			Where(`"userId" = ? AND "skpPeriodeId" = ?`, target.UserId, target.SKPPeriodeId).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return skpdomain.ErrConflict
		}
		if err := tx.Create(&target).Error; err != nil {
			return fmt.Errorf("create skp target: %w", err)
		}
		return nil
	})
	return target, err
}

func (s GormStore) UpdateSKPTarget(ctx context.Context, actorID, id string, updates map[string]any) (models.SKPTargetKinerja, error) {
	var target models.SKPTargetKinerja
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&target).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return skpdomain.ErrNotFound
			}
			return fmt.Errorf("read skp target: %w", err)
		}
		if err := tx.Model(&target).Updates(updates).Error; err != nil {
			return fmt.Errorf("update skp target: %w", err)
		}
		if err := tx.First(&target, "id = ?", id).Error; err != nil {
			return fmt.Errorf("reload skp target: %w", err)
		}
		return nil
	})
	return target, err
}

func (s GormStore) DeleteSKPTarget(ctx context.Context, actorID, id string) error {
	result := s.DB.WithContext(ctx).Where("id = ?", id).Delete(&models.SKPTargetKinerja{})
	if result.Error != nil {
		return fmt.Errorf("delete skp target: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return skpdomain.ErrNotFound
	}
	return nil
}
