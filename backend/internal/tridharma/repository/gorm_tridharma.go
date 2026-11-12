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

// ═══════════════════════════ Tridharma Pendidikan ═══════════════════════════

func (s GormStore) ListTridharmaPendidikan(ctx context.Context, options listing.Options) ([]models.TridharmaPendidikan, int64, error) {
	query := s.DB.WithContext(ctx).Model(&models.TridharmaPendidikan{})
	if options.UserID != "" {
		query = query.Where(`"userId" = ?`, options.UserID)
	}
	if options.Status != "" {
		query = query.Where("status = ?", options.Status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count tridharma pendidikan: %w", err)
	}
	var items []models.TridharmaPendidikan
	order := listing.OrderClause(sortColumnsTridharmaPendidikan, options.Sort, options.Order, "createdAt")
	err := query.Order(order).
		Offset((options.Page - 1) * options.PageSize).
		Limit(options.PageSize).
		Find(&items).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list tridharma pendidikan: %w", err)
	}
	return items, total, nil
}

func (s GormStore) GetTridharmaPendidikan(ctx context.Context, id string) (models.TridharmaPendidikan, error) {
	var item models.TridharmaPendidikan
	err := s.DB.WithContext(ctx).Where("id = ?", id).First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.TridharmaPendidikan{}, tridharmadomain.ErrNotFound
		}
		return models.TridharmaPendidikan{}, fmt.Errorf("read tridharma pendidikan: %w", err)
	}
	return item, nil
}

func (s GormStore) CreateTridharmaPendidikan(ctx context.Context, actorID string, item models.TridharmaPendidikan) (models.TridharmaPendidikan, error) {
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&models.TridharmaPendidikan{}).
			Where(`"userId" = ? AND "periodeAkademikId" = ?`, item.UserId, item.PeriodeAkademikId).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return tridharmadomain.ErrConflict
		}
		if err := tx.Create(&item).Error; err != nil {
			return fmt.Errorf("create tridharma pendidikan: %w", err)
		}
		return nil
	})
	return item, err
}

func (s GormStore) UpdateTridharmaPendidikan(ctx context.Context, actorID, id string, updates map[string]any) (models.TridharmaPendidikan, error) {
	var item models.TridharmaPendidikan
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&item).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return tridharmadomain.ErrNotFound
			}
			return fmt.Errorf("read tridharma pendidikan: %w", err)
		}
		if err := tx.Model(&item).Updates(updates).Error; err != nil {
			return fmt.Errorf("update tridharma pendidikan: %w", err)
		}
		if err := tx.First(&item, "id = ?", id).Error; err != nil {
			return fmt.Errorf("reload tridharma pendidikan: %w", err)
		}
		return nil
	})
	return item, err
}

func (s GormStore) DeleteTridharmaPendidikan(ctx context.Context, actorID, id string) error {
	result := s.DB.WithContext(ctx).Where("id = ?", id).Delete(&models.TridharmaPendidikan{})
	if result.Error != nil {
		return fmt.Errorf("delete tridharma pendidikan: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return tridharmadomain.ErrNotFound
	}
	return nil
}

// ═══════════════════════════ Tridharma Penelitian ═══════════════════════════

func (s GormStore) ListTridharmaPenelitian(ctx context.Context, options listing.Options) ([]models.TridharmaPenelitian, int64, error) {
	query := s.DB.WithContext(ctx).Model(&models.TridharmaPenelitian{})
	if options.UserID != "" {
		query = query.Where(`"userId" = ?`, options.UserID)
	}
	if options.Status != "" {
		query = query.Where("status = ?", options.Status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count tridharma penelitian: %w", err)
	}
	var items []models.TridharmaPenelitian
	order := listing.OrderClause(sortColumnsTridharmaPenelitian, options.Sort, options.Order, "createdAt")
	err := query.Order(order).
		Offset((options.Page - 1) * options.PageSize).
		Limit(options.PageSize).
		Find(&items).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list tridharma penelitian: %w", err)
	}
	return items, total, nil
}

func (s GormStore) GetTridharmaPenelitian(ctx context.Context, id string) (models.TridharmaPenelitian, error) {
	var item models.TridharmaPenelitian
	err := s.DB.WithContext(ctx).Where("id = ?", id).First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.TridharmaPenelitian{}, tridharmadomain.ErrNotFound
		}
		return models.TridharmaPenelitian{}, fmt.Errorf("read tridharma penelitian: %w", err)
	}
	return item, nil
}

func (s GormStore) CreateTridharmaPenelitian(ctx context.Context, actorID string, item models.TridharmaPenelitian) (models.TridharmaPenelitian, error) {
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&models.TridharmaPenelitian{}).
			Where(`"userId" = ? AND "periodeAkademikId" = ?`, item.UserId, item.PeriodeAkademikId).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return tridharmadomain.ErrConflict
		}
		if err := tx.Create(&item).Error; err != nil {
			return fmt.Errorf("create tridharma penelitian: %w", err)
		}
		return nil
	})
	return item, err
}

func (s GormStore) UpdateTridharmaPenelitian(ctx context.Context, actorID, id string, updates map[string]any) (models.TridharmaPenelitian, error) {
	var item models.TridharmaPenelitian
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&item).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return tridharmadomain.ErrNotFound
			}
			return fmt.Errorf("read tridharma penelitian: %w", err)
		}
		if err := tx.Model(&item).Updates(updates).Error; err != nil {
			return fmt.Errorf("update tridharma penelitian: %w", err)
		}
		if err := tx.First(&item, "id = ?", id).Error; err != nil {
			return fmt.Errorf("reload tridharma penelitian: %w", err)
		}
		return nil
	})
	return item, err
}

func (s GormStore) DeleteTridharmaPenelitian(ctx context.Context, actorID, id string) error {
	result := s.DB.WithContext(ctx).Where("id = ?", id).Delete(&models.TridharmaPenelitian{})
	if result.Error != nil {
		return fmt.Errorf("delete tridharma penelitian: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return tridharmadomain.ErrNotFound
	}
	return nil
}

// ═══════════════════════════ Tridharma Pengabdian ═══════════════════════════

func (s GormStore) ListTridharmaPengabdian(ctx context.Context, options listing.Options) ([]models.TridharmaPengabdian, int64, error) {
	query := s.DB.WithContext(ctx).Model(&models.TridharmaPengabdian{})
	if options.UserID != "" {
		query = query.Where(`"userId" = ?`, options.UserID)
	}
	if options.Status != "" {
		query = query.Where("status = ?", options.Status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count tridharma pengabdian: %w", err)
	}
	var items []models.TridharmaPengabdian
	order := listing.OrderClause(sortColumnsTridharmaPengabdian, options.Sort, options.Order, "createdAt")
	err := query.Order(order).
		Offset((options.Page - 1) * options.PageSize).
		Limit(options.PageSize).
		Find(&items).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list tridharma pengabdian: %w", err)
	}
	return items, total, nil
}

func (s GormStore) GetTridharmaPengabdian(ctx context.Context, id string) (models.TridharmaPengabdian, error) {
	var item models.TridharmaPengabdian
	err := s.DB.WithContext(ctx).Where("id = ?", id).First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.TridharmaPengabdian{}, tridharmadomain.ErrNotFound
		}
		return models.TridharmaPengabdian{}, fmt.Errorf("read tridharma pengabdian: %w", err)
	}
	return item, nil
}

func (s GormStore) CreateTridharmaPengabdian(ctx context.Context, actorID string, item models.TridharmaPengabdian) (models.TridharmaPengabdian, error) {
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&models.TridharmaPengabdian{}).
			Where(`"userId" = ? AND "periodeAkademikId" = ?`, item.UserId, item.PeriodeAkademikId).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return tridharmadomain.ErrConflict
		}
		if err := tx.Create(&item).Error; err != nil {
			return fmt.Errorf("create tridharma pengabdian: %w", err)
		}
		return nil
	})
	return item, err
}

func (s GormStore) UpdateTridharmaPengabdian(ctx context.Context, actorID, id string, updates map[string]any) (models.TridharmaPengabdian, error) {
	var item models.TridharmaPengabdian
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&item).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return tridharmadomain.ErrNotFound
			}
			return fmt.Errorf("read tridharma pengabdian: %w", err)
		}
		if err := tx.Model(&item).Updates(updates).Error; err != nil {
			return fmt.Errorf("update tridharma pengabdian: %w", err)
		}
		if err := tx.First(&item, "id = ?", id).Error; err != nil {
			return fmt.Errorf("reload tridharma pengabdian: %w", err)
		}
		return nil
	})
	return item, err
}

func (s GormStore) DeleteTridharmaPengabdian(ctx context.Context, actorID, id string) error {
	result := s.DB.WithContext(ctx).Where("id = ?", id).Delete(&models.TridharmaPengabdian{})
	if result.Error != nil {
		return fmt.Errorf("delete tridharma pengabdian: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return tridharmadomain.ErrNotFound
	}
	return nil
}
