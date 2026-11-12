package repository

import (
	"context"
	"errors"
	"fmt"

	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
	skpdomain "simpeg/backend/internal/skp/domain"
	"simpeg/backend/internal/timeutil"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s GormStore) ListLogAktivitas(ctx context.Context, options listing.Options) ([]models.LogAktivitasHarian, int64, error) {
	query := s.DB.WithContext(ctx).Model(&models.LogAktivitasHarian{})
	if options.UserID != "" {
		query = query.Where(`"userId" = ?`, options.UserID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count log aktivitas: %w", err)
	}
	var items []models.LogAktivitasHarian
	order := listing.OrderClause(sortColumnsLogAktivitas, options.Sort, options.Order, "tanggal")
	err := query.Order(order).
		Offset((options.Page - 1) * options.PageSize).
		Limit(options.PageSize).
		Find(&items).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list log aktivitas: %w", err)
	}
	return items, total, nil
}

func (s GormStore) GetLogAktivitas(ctx context.Context, id string) (models.LogAktivitasHarian, error) {
	var log models.LogAktivitasHarian
	err := s.DB.WithContext(ctx).Where("id = ?", id).First(&log).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.LogAktivitasHarian{}, skpdomain.ErrNotFound
		}
		return models.LogAktivitasHarian{}, fmt.Errorf("read log aktivitas: %w", err)
	}
	return log, nil
}

func (s GormStore) CreateLogAktivitas(ctx context.Context, actorID string, log models.LogAktivitasHarian) (models.LogAktivitasHarian, error) {
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&models.LogAktivitasHarian{}).
			Where(`"userId" = ? AND "tanggal" = ?`, log.UserId, timeutil.DateOnly(log.Tanggal)).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return skpdomain.ErrConflict
		}
		if err := tx.Create(&log).Error; err != nil {
			return fmt.Errorf("create log aktivitas: %w", err)
		}
		return nil
	})
	return log, err
}

func (s GormStore) UpdateLogAktivitas(ctx context.Context, actorID, id string, updates map[string]any) (models.LogAktivitasHarian, error) {
	var log models.LogAktivitasHarian
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&log).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return skpdomain.ErrNotFound
			}
			return fmt.Errorf("read log aktivitas: %w", err)
		}
		if err := tx.Model(&log).Updates(updates).Error; err != nil {
			return fmt.Errorf("update log aktivitas: %w", err)
		}
		if err := tx.First(&log, "id = ?", id).Error; err != nil {
			return fmt.Errorf("reload log aktivitas: %w", err)
		}
		return nil
	})
	return log, err
}

func (s GormStore) DeleteLogAktivitas(ctx context.Context, actorID, id string) error {
	result := s.DB.WithContext(ctx).Where("id = ?", id).Delete(&models.LogAktivitasHarian{})
	if result.Error != nil {
		return fmt.Errorf("delete log aktivitas: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return skpdomain.ErrNotFound
	}
	return nil
}

func (s GormStore) ListPenilaian(ctx context.Context, options listing.Options) ([]models.PenilaianKinerjaAkhir, int64, error) {
	query := s.DB.WithContext(ctx).Model(&models.PenilaianKinerjaAkhir{})
	if options.UserID != "" {
		query = query.Where(`"userId" = ?`, options.UserID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count penilaian: %w", err)
	}
	var items []models.PenilaianKinerjaAkhir
	order := listing.OrderClause(sortColumnsPenilaian, options.Sort, options.Order, "tanggalPenilaian")
	err := query.Order(order).
		Offset((options.Page - 1) * options.PageSize).
		Limit(options.PageSize).
		Find(&items).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list penilaian: %w", err)
	}
	return items, total, nil
}

func (s GormStore) GetPenilaian(ctx context.Context, id string) (models.PenilaianKinerjaAkhir, error) {
	var penilaian models.PenilaianKinerjaAkhir
	err := s.DB.WithContext(ctx).Where("id = ?", id).First(&penilaian).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.PenilaianKinerjaAkhir{}, skpdomain.ErrNotFound
		}
		return models.PenilaianKinerjaAkhir{}, fmt.Errorf("read penilaian: %w", err)
	}
	return penilaian, nil
}

func (s GormStore) CreatePenilaian(ctx context.Context, actorID string, penilaian models.PenilaianKinerjaAkhir) (models.PenilaianKinerjaAkhir, error) {
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&models.PenilaianKinerjaAkhir{}).
			Where(`"userId" = ? AND "skpPeriodeId" = ?`, penilaian.UserId, penilaian.SKPPeriodeId).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return skpdomain.ErrConflict
		}
		if err := tx.Create(&penilaian).Error; err != nil {
			return fmt.Errorf("create penilaian: %w", err)
		}
		return nil
	})
	return penilaian, err
}

func (s GormStore) UpdatePenilaian(ctx context.Context, actorID, id string, updates map[string]any) (models.PenilaianKinerjaAkhir, error) {
	var penilaian models.PenilaianKinerjaAkhir
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&penilaian).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return skpdomain.ErrNotFound
			}
			return fmt.Errorf("read penilaian: %w", err)
		}
		if err := tx.Model(&penilaian).Updates(updates).Error; err != nil {
			return fmt.Errorf("update penilaian: %w", err)
		}
		if err := tx.First(&penilaian, "id = ?", id).Error; err != nil {
			return fmt.Errorf("reload penilaian: %w", err)
		}
		return nil
	})
	return penilaian, err
}

func (s GormStore) DeletePenilaian(ctx context.Context, actorID, id string) error {
	result := s.DB.WithContext(ctx).Where("id = ?", id).Delete(&models.PenilaianKinerjaAkhir{})
	if result.Error != nil {
		return fmt.Errorf("delete penilaian: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return skpdomain.ErrNotFound
	}
	return nil
}
