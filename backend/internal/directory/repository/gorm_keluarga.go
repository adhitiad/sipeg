package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"simpeg/backend/internal/audit"
	"simpeg/backend/internal/directory/domain"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ListKeluarga membaca daftar keluarga pegawai sesuai filter dan pagination.
func (s GormStore) ListKeluarga(ctx context.Context, options listing.Options) ([]models.KeluargaPegawai, int64, error) {
	query := s.DB.WithContext(ctx).Model(&models.KeluargaPegawai{})
	if options.UserID != "" {
		query = query.Where("userId = ?", options.UserID)
	}
	if len(options.ScopeHomebaseIDs) > 0 {
		query = query.Where("userId IN (SELECT id FROM user WHERE homebaseId IN ?)", options.ScopeHomebaseIDs)
	}
	if options.Search != "" {
		pattern := "%" + escapeLike(options.Search) + "%"
		query = query.Where("(nama ILIKE ? ESCAPE '\\' OR nik ILIKE ? ESCAPE '\\')", pattern, pattern)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count keluarga: %w", err)
	}
	var items []models.KeluargaPegawai
	order := listing.OrderClause(domain.KeluargaSortColumns, options.Sort, options.Order, "createdAt")
	err := query.Order(order).Offset((options.Page - 1) * options.PageSize).Limit(options.PageSize).Find(&items).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list keluarga: %w", err)
	}
	return items, total, nil
}

func (s GormStore) GetKeluarga(ctx context.Context, id string) (models.KeluargaPegawai, error) {
	var item models.KeluargaPegawai
	err := s.DB.WithContext(ctx).Where("id = ?", id).First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.KeluargaPegawai{}, domain.ErrNotFound
		}
		return models.KeluargaPegawai{}, fmt.Errorf("read keluarga: %w", err)
	}
	return item, nil
}

func (s GormStore) CreateKeluarga(ctx context.Context, actorID string, item models.KeluargaPegawai) (models.KeluargaPegawai, error) {
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&item).Error; err != nil {
			return classifyRecordError(err)
		}
		return createAudit(ctx, tx, audit.Entry{ActorID: actorID, Action: audit.ActionKeluargaCreated, TargetID: &item.ID, UnitID: &item.UserId})
	})
	return item, err
}

func (s GormStore) UpdateKeluarga(ctx context.Context, actorID, id string, patch domain.KeluargaPatch) (models.KeluargaPegawai, error) {
	updates := keluargaColumns(patch)
	var item models.KeluargaPegawai
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&item).Error; err != nil {
			return classifyRecordError(err)
		}
		if err := tx.Model(&item).Updates(updates).Error; err != nil {
			return classifyRecordError(err)
		}
		if err := tx.First(&item, "id = ?", id).Error; err != nil {
			return fmt.Errorf("reload keluarga: %w", err)
		}
		return createAudit(ctx, tx, audit.Entry{ActorID: actorID, Action: audit.ActionKeluargaUpdated, TargetID: optionalRef(id), UnitID: &item.UserId})
	})
	return item, err
}

func (s GormStore) ArchiveKeluarga(ctx context.Context, actorID, id string) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var item models.KeluargaPegawai
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&item).Error; err != nil {
			return classifyRecordError(err)
		}
		result := tx.Model(&models.KeluargaPegawai{}).Where("id = ?").Update("aktif", false)
		if result.Error != nil {
			return fmt.Errorf("archive keluarga: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return domain.ErrNotFound
		}
		return createAudit(ctx, tx, audit.Entry{ActorID: actorID, Action: audit.ActionKeluargaArchived, TargetID: optionalRef(id), UnitID: &item.UserId})
	})
}

func keluargaColumns(patch domain.KeluargaPatch) map[string]any {
	updates := make(map[string]any)
	if patch.Hubungan.Present {
		if patch.Hubungan.Null {
			updates["hubungan"] = nil
		} else {
			updates["hubungan"] = patch.Hubungan.Value
		}
	}
	if patch.Nama.Present {
		if patch.Nama.Null {
			updates["nama"] = nil
		} else {
			updates["nama"] = strings.TrimSpace(patch.Nama.Value)
		}
	}
	if patch.NIK.Present {
		if patch.NIK.Null {
			updates["nik"] = nil
		} else {
			updates["nik"] = strings.TrimSpace(patch.NIK.Value)
		}
	}
	if patch.TanggalLahir.Present {
		if patch.TanggalLahir.Null {
			updates["tanggalLahir"] = nil
		} else {
			updates["tanggalLahir"] = patch.TanggalLahir.Value
		}
	}
	if patch.Pekerjaan.Present {
		if patch.Pekerjaan.Null {
			updates["pekerjaan"] = nil
		} else {
			updates["pekerjaan"] = strings.TrimSpace(patch.Pekerjaan.Value)
		}
	}
	if patch.StatusTanggungan.Present {
		if patch.StatusTanggungan.Null {
			updates["statusTanggungan"] = nil
		} else {
			updates["statusTanggungan"] = patch.StatusTanggungan.Value
		}
	}
	return updates
}
