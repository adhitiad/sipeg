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

func (s GormStore) ListSertifikasi(ctx context.Context, options listing.Options) ([]models.RiwayatSertifikasi, int64, error) {
	query := s.DB.WithContext(ctx).Model(&models.RiwayatSertifikasi{})
	if options.UserID != "" {
		query = query.Where("userId = ?", options.UserID)
	}
	if len(options.ScopeHomebaseIDs) > 0 {
		query = query.Where("userId IN (SELECT id FROM user WHERE homebaseId IN ?)", options.ScopeHomebaseIDs)
	}
	if options.Search != "" {
		pattern := "%" + escapeLike(options.Search) + "%"
		query = query.Where("(namaSertifikat ILIKE ? ESCAPE '\\' OR penyelenggara ILIKE ? ESCAPE '\\')", pattern, pattern)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count sertifikasi: %w", err)
	}
	var items []models.RiwayatSertifikasi
	order := listing.OrderClause(domain.SertifikasiSortColumns, options.Sort, options.Order, "tanggalTerbit")
	err := query.Order(order).Offset((options.Page - 1) * options.PageSize).Limit(options.PageSize).Find(&items).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list sertifikasi: %w", err)
	}
	return items, total, nil
}

func (s GormStore) GetSertifikasi(ctx context.Context, id string) (models.RiwayatSertifikasi, error) {
	var item models.RiwayatSertifikasi
	err := s.DB.WithContext(ctx).Where("id = ?", id).First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.RiwayatSertifikasi{}, domain.ErrNotFound
		}
		return models.RiwayatSertifikasi{}, fmt.Errorf("read sertifikasi: %w", err)
	}
	return item, nil
}

func (s GormStore) CreateSertifikasi(ctx context.Context, actorID string, item models.RiwayatSertifikasi) (models.RiwayatSertifikasi, error) {
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&item).Error; err != nil {
			return classifyRecordError(err)
		}
		return createAudit(ctx, tx, audit.Entry{ActorID: actorID, Action: audit.ActionSertifikasiCreated, TargetID: &item.ID, UnitID: &item.UserId})
	})
	return item, err
}

func (s GormStore) UpdateSertifikasi(ctx context.Context, actorID, id string, patch domain.RiwayatSertifikasiPatch) (models.RiwayatSertifikasi, error) {
	updates := sertifikasiColumns(patch)
	var item models.RiwayatSertifikasi
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&item).Error; err != nil {
			return classifyRecordError(err)
		}
		if err := tx.Model(&item).Updates(updates).Error; err != nil {
			return classifyRecordError(err)
		}
		if err := tx.First(&item, "id = ?", id).Error; err != nil {
			return fmt.Errorf("reload sertifikasi: %w", err)
		}
		return createAudit(ctx, tx, audit.Entry{ActorID: actorID, Action: audit.ActionSertifikasiUpdated, TargetID: optionalRef(id), UnitID: &item.UserId})
	})
	return item, err
}

func (s GormStore) ArchiveSertifikasi(ctx context.Context, actorID, id string) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var item models.RiwayatSertifikasi
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&item).Error; err != nil {
			return classifyRecordError(err)
		}
		result := tx.Model(&models.RiwayatSertifikasi{}).Where("id = ?").Update("aktif", false)
		if result.Error != nil {
			return fmt.Errorf("archive sertifikasi: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return domain.ErrNotFound
		}
		return createAudit(ctx, tx, audit.Entry{ActorID: actorID, Action: audit.ActionSertifikasiArchived, TargetID: optionalRef(id), UnitID: &item.UserId})
	})
}

func sertifikasiColumns(patch domain.RiwayatSertifikasiPatch) map[string]any {
	updates := make(map[string]any)
	if patch.JenisSertifikasi.Present {
		if patch.JenisSertifikasi.Null {
			updates["jenisSertifikasi"] = nil
		} else {
			updates["jenisSertifikasi"] = patch.JenisSertifikasi.Value
		}
	}
	if patch.NamaSertifikat.Present {
		if patch.NamaSertifikat.Null {
			updates["namaSertifikat"] = nil
		} else {
			updates["namaSertifikat"] = patch.NamaSertifikat.Value
		}
	}
	if patch.NomorRegistrasi.Present {
		if patch.NomorRegistrasi.Null {
			updates["nomorRegistrasi"] = nil
		} else {
			updates["nomorRegistrasi"] = patch.NomorRegistrasi.Value
		}
	}
	if patch.Penyelenggara.Present {
		if patch.Penyelenggara.Null {
			updates["penyelenggara"] = nil
		} else {
			updates["penyelenggara"] = patch.Penyelenggara.Value
		}
	}
	if patch.TanggalTerbit.Present {
		if patch.TanggalTerbit.Null {
			updates["tanggalTerbit"] = nil
		} else {
			updates["tanggalTerbit"] = patch.TanggalTerbit.Value
		}
	}
	if patch.BerlakuSampai.Present {
		if patch.BerlakuSampai.Null {
			updates["berlakuSampai"] = nil
		} else {
			updates["berlakuSampai"] = patch.BerlakuSampai.Value
		}
	}
	if patch.Status.Present {
		if patch.Status.Null {
			updates["status"] = nil
		} else {
			updates["status"] = patch.Status.Value
		}
	}
	return updates
}
