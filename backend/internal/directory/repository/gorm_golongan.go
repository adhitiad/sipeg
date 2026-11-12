package repository

import (
	"context"
	"errors"
	"fmt"

	"simpeg/backend/internal/audit"
	"simpeg/backend/internal/directory/domain"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s GormStore) ListGolongan(ctx context.Context, options listing.Options) ([]models.RiwayatGolongan, int64, error) {
	query := s.DB.WithContext(ctx).Model(&models.RiwayatGolongan{})
	if options.UserID != "" {
		query = query.Where("userId = ?", options.UserID)
	}
	if len(options.ScopeHomebaseIDs) > 0 {
		query = query.Where("userId IN (SELECT id FROM user WHERE homebaseId IN ?)", options.ScopeHomebaseIDs)
	}
	if options.Search != "" {
		pattern := "%" + escapeLike(options.Search) + "%"
		query = query.Where("(golongan ILIKE ? ESCAPE '\\' OR ruang ILIKE ? ESCAPE '\\')", pattern, pattern)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count golongan: %w", err)
	}
	var items []models.RiwayatGolongan
	order := listing.OrderClause(domain.GolonganSortColumns, options.Sort, options.Order, "tmt")
	err := query.Order(order).Offset((options.Page - 1) * options.PageSize).Limit(options.PageSize).Find(&items).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list golongan: %w", err)
	}
	return items, total, nil
}

func (s GormStore) GetGolongan(ctx context.Context, id string) (models.RiwayatGolongan, error) {
	var item models.RiwayatGolongan
	err := s.DB.WithContext(ctx).Where("id = ?", id).First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.RiwayatGolongan{}, domain.ErrNotFound
		}
		return models.RiwayatGolongan{}, fmt.Errorf("read golongan: %w", err)
	}
	return item, nil
}

func (s GormStore) CreateGolongan(ctx context.Context, actorID string, item models.RiwayatGolongan) (models.RiwayatGolongan, error) {
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&item).Error; err != nil {
			return classifyRecordError(err)
		}
		return createAudit(ctx, tx, audit.Entry{ActorID: actorID, Action: audit.ActionGolonganCreated, TargetID: &item.ID, UnitID: &item.UserId})
	})
	return item, err
}

func (s GormStore) UpdateGolongan(ctx context.Context, actorID, id string, patch domain.RiwayatGolonganPatch) (models.RiwayatGolongan, error) {
	updates := golonganColumns(patch)
	var item models.RiwayatGolongan
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&item).Error; err != nil {
			return classifyRecordError(err)
		}
		if err := tx.Model(&item).Updates(updates).Error; err != nil {
			return classifyRecordError(err)
		}
		if err := tx.First(&item, "id = ?", id).Error; err != nil {
			return fmt.Errorf("reload golongan: %w", err)
		}
		return createAudit(ctx, tx, audit.Entry{ActorID: actorID, Action: audit.ActionGolonganUpdated, TargetID: optionalRef(id), UnitID: &item.UserId})
	})
	return item, err
}

func (s GormStore) ArchiveGolongan(ctx context.Context, actorID, id string) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var item models.RiwayatGolongan
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&item).Error; err != nil {
			return classifyRecordError(err)
		}
		result := tx.Model(&models.RiwayatGolongan{}).Where("id = ?").Update("aktif", false)
		if result.Error != nil {
			return fmt.Errorf("archive golongan: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return domain.ErrNotFound
		}
		return createAudit(ctx, tx, audit.Entry{ActorID: actorID, Action: audit.ActionGolonganArchived, TargetID: optionalRef(id), UnitID: &item.UserId})
	})
}

func golonganColumns(patch domain.RiwayatGolonganPatch) map[string]any {
	updates := make(map[string]any)
	if patch.Golongan.Present {
		if patch.Golongan.Null {
			updates["golongan"] = nil
		} else {
			updates["golongan"] = patch.Golongan.Value
		}
	}
	if patch.Ruang.Present {
		if patch.Ruang.Null {
			updates["ruang"] = nil
		} else {
			updates["ruang"] = patch.Ruang.Value
		}
	}
	if patch.TMT.Present {
		if patch.TMT.Null {
			updates["tmt"] = nil
		} else {
			updates["tmt"] = patch.TMT.Value
		}
	}
	if patch.AngkaKredit.Present {
		if patch.AngkaKredit.Null {
			updates["angkaKredit"] = nil
		} else {
			updates["angkaKredit"] = patch.AngkaKredit.Value
		}
	}
	if patch.NomorSk.Present {
		if patch.NomorSk.Null {
			updates["nomorSk"] = nil
		} else {
			updates["nomorSk"] = patch.NomorSk.Value
		}
	}
	return updates
}
