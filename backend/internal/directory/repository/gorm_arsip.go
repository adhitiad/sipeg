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

func (s GormStore) ListArsip(ctx context.Context, options listing.Options) ([]models.ArsipDokumenPegawai, int64, error) {
	query := s.DB.WithContext(ctx).Model(&models.ArsipDokumenPegawai{})
	if options.UserID != "" {
		query = query.Where("userId = ?", options.UserID)
	}
	if len(options.ScopeHomebaseIDs) > 0 {
		query = query.Where("userId IN (SELECT id FROM user WHERE homebaseId IN ?)", options.ScopeHomebaseIDs)
	}
	if options.Search != "" {
		pattern := "%" + escapeLike(options.Search) + "%"
		query = query.Where("(judul ILIKE ? ESCAPE '\\' OR keterangan ILIKE ? ESCAPE '\\')", pattern, pattern)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count arsip: %w", err)
	}
	var items []models.ArsipDokumenPegawai
	order := listing.OrderClause(domain.ArsipSortColumns, options.Sort, options.Order, "createdAt")
	err := query.Order(order).Offset((options.Page - 1) * options.PageSize).Limit(options.PageSize).Find(&items).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list arsip: %w", err)
	}
	return items, total, nil
}

func (s GormStore) GetArsip(ctx context.Context, id string) (models.ArsipDokumenPegawai, error) {
	var item models.ArsipDokumenPegawai
	err := s.DB.WithContext(ctx).Where("id = ?", id).First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.ArsipDokumenPegawai{}, domain.ErrNotFound
		}
		return models.ArsipDokumenPegawai{}, fmt.Errorf("read arsip: %w", err)
	}
	return item, nil
}

func (s GormStore) CreateArsip(ctx context.Context, actorID string, item models.ArsipDokumenPegawai) (models.ArsipDokumenPegawai, error) {
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&item).Error; err != nil {
			return classifyRecordError(err)
		}
		return createAudit(ctx, tx, audit.Entry{ActorID: actorID, Action: audit.ActionArsipCreated, TargetID: &item.ID, UnitID: &item.UserId})
	})
	return item, err
}

func (s GormStore) UpdateArsip(ctx context.Context, actorID, id string, patch domain.ArsipDokumenPatch) (models.ArsipDokumenPegawai, error) {
	updates := arsipColumns(patch)
	var item models.ArsipDokumenPegawai
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&item).Error; err != nil {
			return classifyRecordError(err)
		}
		if err := tx.Model(&item).Updates(updates).Error; err != nil {
			return classifyRecordError(err)
		}
		if err := tx.First(&item, "id = ?", id).Error; err != nil {
			return fmt.Errorf("reload arsip: %w", err)
		}
		return createAudit(ctx, tx, audit.Entry{ActorID: actorID, Action: audit.ActionArsipUpdated, TargetID: optionalRef(id), UnitID: &item.UserId})
	})
	return item, err
}

func (s GormStore) ArchiveArsip(ctx context.Context, actorID, id string) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var item models.ArsipDokumenPegawai
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&item).Error; err != nil {
			return classifyRecordError(err)
		}
		result := tx.Model(&models.ArsipDokumenPegawai{}).Where("id = ?").Update("aktif", false)
		if result.Error != nil {
			return fmt.Errorf("archive arsip: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return domain.ErrNotFound
		}
		return createAudit(ctx, tx, audit.Entry{ActorID: actorID, Action: audit.ActionArsipArchived, TargetID: optionalRef(id), UnitID: &item.UserId})
	})
}

func arsipColumns(patch domain.ArsipDokumenPatch) map[string]any {
	updates := make(map[string]any)
	if patch.JenisDokumen.Present {
		if patch.JenisDokumen.Null {
			updates["jenisDokumen"] = nil
		} else {
			updates["jenisDokumen"] = patch.JenisDokumen.Value
		}
	}
	if patch.Judul.Present {
		if patch.Judul.Null {
			updates["judul"] = nil
		} else {
			updates["judul"] = patch.Judul.Value
		}
	}
	if patch.FileURL.Present {
		if patch.FileURL.Null {
			updates["fileUrl"] = nil
		} else {
			updates["fileUrl"] = patch.FileURL.Value
		}
	}
	if patch.FileSize.Present {
		if patch.FileSize.Null {
			updates["fileSize"] = nil
		} else {
			updates["fileSize"] = patch.FileSize.Value
		}
	}
	if patch.MimeType.Present {
		if patch.MimeType.Null {
			updates["mimeType"] = nil
		} else {
			updates["mimeType"] = patch.MimeType.Value
		}
	}
	if patch.Keterangan.Present {
		if patch.Keterangan.Null {
			updates["keterangan"] = nil
		} else {
			updates["keterangan"] = patch.Keterangan.Value
		}
	}
	if patch.BerlakuSampai.Present {
		if patch.BerlakuSampai.Null {
			updates["berlakuSampai"] = nil
		} else {
			updates["berlakuSampai"] = patch.BerlakuSampai.Value
		}
	}
	if patch.CloudinaryPublicId.Present {
		if patch.CloudinaryPublicId.Null {
			updates["cloudinaryPublicId"] = nil
		} else {
			updates["cloudinaryPublicId"] = patch.CloudinaryPublicId.Value
		}
	}
	return updates
}
