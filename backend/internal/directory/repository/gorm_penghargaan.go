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

func (s GormStore) ListPenghargaan(ctx context.Context, options listing.Options) ([]models.RiwayatPenghargaan, int64, error) {
	query := s.DB.WithContext(ctx).Model(&models.RiwayatPenghargaan{})
	if options.UserID != "" {
		query = query.Where("userId = ?", options.UserID)
	}
	if len(options.ScopeHomebaseIDs) > 0 {
		query = query.Where("userId IN (SELECT id FROM user WHERE homebaseId IN ?)", options.ScopeHomebaseIDs)
	}
	if options.Search != "" {
		pattern := "%" + escapeLike(options.Search) + "%"
		query = query.Where("(namaPenghargaan ILIKE ? ESCAPE '\\' OR penyelenggara ILIKE ? ESCAPE '\\')", pattern, pattern)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count penghargaan: %w", err)
	}
	var items []models.RiwayatPenghargaan
	order := listing.OrderClause(domain.PenghargaanSortColumns, options.Sort, options.Order, "tanggalTerima")
	err := query.Order(order).Offset((options.Page - 1) * options.PageSize).Limit(options.PageSize).Find(&items).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list penghargaan: %w", err)
	}
	return items, total, nil
}

func (s GormStore) GetPenghargaan(ctx context.Context, id string) (models.RiwayatPenghargaan, error) {
	var item models.RiwayatPenghargaan
	err := s.DB.WithContext(ctx).Where("id = ?", id).First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.RiwayatPenghargaan{}, domain.ErrNotFound
		}
		return models.RiwayatPenghargaan{}, fmt.Errorf("read penghargaan: %w", err)
	}
	return item, nil
}

func (s GormStore) CreatePenghargaan(ctx context.Context, actorID string, item models.RiwayatPenghargaan) (models.RiwayatPenghargaan, error) {
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&item).Error; err != nil {
			return classifyRecordError(err)
		}
		return createAudit(ctx, tx, audit.Entry{ActorID: actorID, Action: audit.ActionPenghargaanCreated, TargetID: &item.ID, UnitID: &item.UserId})
	})
	return item, err
}

func (s GormStore) UpdatePenghargaan(ctx context.Context, actorID, id string, patch domain.RiwayatPenghargaanPatch) (models.RiwayatPenghargaan, error) {
	updates := penghargaanColumns(patch)
	var item models.RiwayatPenghargaan
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&item).Error; err != nil {
			return classifyRecordError(err)
		}
		if err := tx.Model(&item).Updates(updates).Error; err != nil {
			return classifyRecordError(err)
		}
		if err := tx.First(&item, "id = ?", id).Error; err != nil {
			return fmt.Errorf("reload penghargaan: %w", err)
		}
		return createAudit(ctx, tx, audit.Entry{ActorID: actorID, Action: audit.ActionPenghargaanUpdated, TargetID: optionalRef(id), UnitID: &item.UserId})
	})
	return item, err
}

func (s GormStore) ArchivePenghargaan(ctx context.Context, actorID, id string) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var item models.RiwayatPenghargaan
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&item).Error; err != nil {
			return classifyRecordError(err)
		}
		result := tx.Model(&models.RiwayatPenghargaan{}).Where("id = ?").Update("aktif", false)
		if result.Error != nil {
			return fmt.Errorf("archive penghargaan: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return domain.ErrNotFound
		}
		return createAudit(ctx, tx, audit.Entry{ActorID: actorID, Action: audit.ActionPenghargaanArchived, TargetID: optionalRef(id), UnitID: &item.UserId})
	})
}

func penghargaanColumns(patch domain.RiwayatPenghargaanPatch) map[string]any {
	updates := make(map[string]any)
	if patch.NamaPenghargaan.Present {
		if patch.NamaPenghargaan.Null {
			updates["namaPenghargaan"] = nil
		} else {
			updates["namaPenghargaan"] = patch.NamaPenghargaan.Value
		}
	}
	if patch.JenisPenghargaan.Present {
		if patch.JenisPenghargaan.Null {
			updates["jenisPenghargaan"] = nil
		} else {
			updates["jenisPenghargaan"] = patch.JenisPenghargaan.Value
		}
	}
	if patch.Penyelenggara.Present {
		if patch.Penyelenggara.Null {
			updates["penyelenggara"] = nil
		} else {
			updates["penyelenggara"] = patch.Penyelenggara.Value
		}
	}
	if patch.TanggalTerima.Present {
		if patch.TanggalTerima.Null {
			updates["tanggalTerima"] = nil
		} else {
			updates["tanggalTerima"] = patch.TanggalTerima.Value
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
