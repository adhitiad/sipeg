package repository

import (
	"context"
	"errors"
	"fmt"

	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
	tridharmadomain "simpeg/backend/internal/tridharma/domain"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ═══════════════════════════ Tridharma Penunjang ═══════════════════════════

func (s GormStore) ListTridharmaPenunjang(ctx context.Context, options listing.Options) ([]models.TridharmaPenunjang, int64, error) {
	query := s.DB.WithContext(ctx).Model(&models.TridharmaPenunjang{})
	if options.UserID != "" {
		query = query.Where(`"userId" = ?`, options.UserID)
	}
	if options.Status != "" {
		query = query.Where("status = ?", options.Status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count tridharma penunjang: %w", err)
	}
	var items []models.TridharmaPenunjang
	order := listing.OrderClause(sortColumnsTridharmaPenunjang, options.Sort, options.Order, "createdAt")
	err := query.Order(order).
		Offset((options.Page - 1) * options.PageSize).
		Limit(options.PageSize).
		Find(&items).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list tridharma penunjang: %w", err)
	}
	return items, total, nil
}

func (s GormStore) GetTridharmaPenunjang(ctx context.Context, id string) (models.TridharmaPenunjang, error) {
	var item models.TridharmaPenunjang
	err := s.DB.WithContext(ctx).Where("id = ?", id).First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.TridharmaPenunjang{}, tridharmadomain.ErrNotFound
		}
		return models.TridharmaPenunjang{}, fmt.Errorf("read tridharma penunjang: %w", err)
	}
	return item, nil
}

func (s GormStore) CreateTridharmaPenunjang(ctx context.Context, actorID string, item models.TridharmaPenunjang) (models.TridharmaPenunjang, error) {
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&models.TridharmaPenunjang{}).
			Where(`"userId" = ? AND "periodeAkademikId" = ?`, item.UserId, item.PeriodeAkademikId).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return tridharmadomain.ErrConflict
		}
		if err := tx.Create(&item).Error; err != nil {
			return fmt.Errorf("create tridharma penunjang: %w", err)
		}
		return nil
	})
	return item, err
}

func (s GormStore) UpdateTridharmaPenunjang(ctx context.Context, actorID, id string, updates map[string]any) (models.TridharmaPenunjang, error) {
	var item models.TridharmaPenunjang
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&item).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return tridharmadomain.ErrNotFound
			}
			return fmt.Errorf("read tridharma penunjang: %w", err)
		}
		if err := tx.Model(&item).Updates(updates).Error; err != nil {
			return fmt.Errorf("update tridharma penunjang: %w", err)
		}
		if err := tx.First(&item, "id = ?", id).Error; err != nil {
			return fmt.Errorf("reload tridharma penunjang: %w", err)
		}
		return nil
	})
	return item, err
}

func (s GormStore) DeleteTridharmaPenunjang(ctx context.Context, actorID, id string) error {
	result := s.DB.WithContext(ctx).Where("id = ?", id).Delete(&models.TridharmaPenunjang{})
	if result.Error != nil {
		return fmt.Errorf("delete tridharma penunjang: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return tridharmadomain.ErrNotFound
	}
	return nil
}
