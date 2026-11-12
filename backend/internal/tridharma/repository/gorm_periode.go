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

// ═══════════════════════════ Periode Akademik ═══════════════════════════

func (s GormStore) ListPeriodeAkademik(ctx context.Context, options listing.Options) ([]models.PeriodeAkademik, int64, error) {
	query := s.DB.WithContext(ctx).Model(&models.PeriodeAkademik{})
	if options.Status != "" {
		query = query.Where("status = ?", options.Status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count periode akademik: %w", err)
	}
	var items []models.PeriodeAkademik
	order := listing.OrderClause(sortColumnsPeriodeAkademik, options.Sort, options.Order, "tahun")
	err := query.Order(order).
		Offset((options.Page - 1) * options.PageSize).
		Limit(options.PageSize).
		Find(&items).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list periode akademik: %w", err)
	}
	return items, total, nil
}

func (s GormStore) GetPeriodeAkademik(ctx context.Context, id string) (models.PeriodeAkademik, error) {
	var periode models.PeriodeAkademik
	err := s.DB.WithContext(ctx).Where("id = ?", id).First(&periode).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.PeriodeAkademik{}, tridharmadomain.ErrNotFound
		}
		return models.PeriodeAkademik{}, fmt.Errorf("read periode akademik: %w", err)
	}
	return periode, nil
}

func (s GormStore) CreatePeriodeAkademik(ctx context.Context, actorID string, periode models.PeriodeAkademik) (models.PeriodeAkademik, error) {
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&periode).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return tridharmadomain.ErrConflict
			}
			return fmt.Errorf("create periode akademik: %w", err)
		}
		return nil
	})
	return periode, err
}

func (s GormStore) UpdatePeriodeAkademik(ctx context.Context, actorID, id string, updates map[string]any) (models.PeriodeAkademik, error) {
	var periode models.PeriodeAkademik
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&periode).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return tridharmadomain.ErrNotFound
			}
			return fmt.Errorf("read periode akademik: %w", err)
		}
		if err := tx.Model(&periode).Updates(updates).Error; err != nil {
			return fmt.Errorf("update periode akademik: %w", err)
		}
		if err := tx.First(&periode, "id = ?", id).Error; err != nil {
			return fmt.Errorf("reload periode akademik: %w", err)
		}
		return nil
	})
	return periode, err
}

func (s GormStore) DeletePeriodeAkademik(ctx context.Context, actorID, id string) error {
	result := s.DB.WithContext(ctx).Where("id = ?", id).Delete(&models.PeriodeAkademik{})
	if result.Error != nil {
		return fmt.Errorf("delete periode akademik: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return tridharmadomain.ErrNotFound
	}
	return nil
}

// ═══════════════════════════ Evaluasi BKD ═══════════════════════════

func (s GormStore) ListEvaluasiBKD(ctx context.Context, options listing.Options) ([]models.EvaluasiBKD, int64, error) {
	query := s.DB.WithContext(ctx).Model(&models.EvaluasiBKD{})
	if options.UserID != "" {
		query = query.Where(`"userId" = ?`, options.UserID)
	}
	if options.Status != "" {
		query = query.Where("status = ?", options.Status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count evaluasi BKD: %w", err)
	}
	var items []models.EvaluasiBKD
	order := listing.OrderClause(sortColumnsEvaluasiBKD, options.Sort, options.Order, "tanggalEvaluasi")
	err := query.Order(order).
		Offset((options.Page - 1) * options.PageSize).
		Limit(options.PageSize).
		Find(&items).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list evaluasi BKD: %w", err)
	}
	return items, total, nil
}

func (s GormStore) GetEvaluasiBKD(ctx context.Context, id string) (models.EvaluasiBKD, error) {
	var evaluasi models.EvaluasiBKD
	err := s.DB.WithContext(ctx).Where("id = ?", id).First(&evaluasi).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.EvaluasiBKD{}, tridharmadomain.ErrNotFound
		}
		return models.EvaluasiBKD{}, fmt.Errorf("read evaluasi BKD: %w", err)
	}
	return evaluasi, nil
}

func (s GormStore) CreateEvaluasiBKD(ctx context.Context, actorID string, evaluasi models.EvaluasiBKD) (models.EvaluasiBKD, error) {
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&models.EvaluasiBKD{}).
			Where(`"userId" = ? AND "periodeAkademikId" = ?`, evaluasi.UserId, evaluasi.PeriodeAkademikId).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return tridharmadomain.ErrConflict
		}
		if err := tx.Create(&evaluasi).Error; err != nil {
			return fmt.Errorf("create evaluasi BKD: %w", err)
		}
		return nil
	})
	return evaluasi, err
}

func (s GormStore) UpdateEvaluasiBKD(ctx context.Context, actorID, id string, updates map[string]any) (models.EvaluasiBKD, error) {
	var evaluasi models.EvaluasiBKD
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&evaluasi).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return tridharmadomain.ErrNotFound
			}
			return fmt.Errorf("read evaluasi BKD: %w", err)
		}
		if err := tx.Model(&evaluasi).Updates(updates).Error; err != nil {
			return fmt.Errorf("update evaluasi BKD: %w", err)
		}
		if err := tx.First(&evaluasi, "id = ?", id).Error; err != nil {
			return fmt.Errorf("reload evaluasi BKD: %w", err)
		}
		return nil
	})
	return evaluasi, err
}

func (s GormStore) DeleteEvaluasiBKD(ctx context.Context, actorID, id string) error {
	result := s.DB.WithContext(ctx).Where("id = ?", id).Delete(&models.EvaluasiBKD{})
	if result.Error != nil {
		return fmt.Errorf("delete evaluasi BKD: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return tridharmadomain.ErrNotFound
	}
	return nil
}
