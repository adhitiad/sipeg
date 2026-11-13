package repository

import (
	"context"
	"fmt"

	"simpeg/backend/internal/audit"
	"simpeg/backend/internal/directory/domain"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ListJabatans membaca daftar master jabatan aktif sesuai filter dan pagination.
func (s GormStore) ListJabatans(ctx context.Context, options listing.Options) ([]models.Jabatan, int64, error) {
	query := s.DB.WithContext(ctx).Model(&models.Jabatan{}).Where("aktif = ?", true)
	if options.Jenis != "" {
		query = query.Where("jenis = ?", options.Jenis)
	}
	if options.Search != "" {
		pattern := "%" + escapeLike(options.Search) + "%"
		query = query.Where("nama ILIKE ? ESCAPE '\\'", pattern)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count jabatans: %w", err)
	}
	var jabatans []models.Jabatan
	order := listing.OrderClause(domain.JabatanSortColumns, options.Sort, options.Order, "nama")
	err := query.Order(order).Offset((options.Page - 1) * options.PageSize).Limit(options.PageSize).Find(&jabatans).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list jabatans: %w", err)
	}
	return jabatans, total, nil
}

// GetJabatan membaca satu master jabatan aktif.
func (s GormStore) GetJabatan(ctx context.Context, id string) (models.Jabatan, error) {
	var jabatan models.Jabatan
	err := s.DB.WithContext(ctx).Where("id = ? AND aktif = ?", id, true).First(&jabatan).Error
	return jabatan, classifyRecordError(err)
}

// CreateJabatan membuat satu master jabatan dan mencatat jejak audit.
func (s GormStore) CreateJabatan(ctx context.Context, actorID string, jabatan models.Jabatan) (models.Jabatan, error) {
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&jabatan).Error; err != nil {
			return classifyRecordError(err)
		}
		return createAudit(ctx, tx, audit.Entry{ActorID: actorID, Action: audit.ActionJabatanCreated, TargetID: &jabatan.ID})
	})
	return jabatan, err
}

// UpdateJabatan memperbarui master jabatan dan mencatat jejak audit.
func (s GormStore) UpdateJabatan(ctx context.Context, actorID, id string, patch domain.JabatanPatch) (models.Jabatan, error) {
	updates := jabatanColumns(patch)
	var jabatan models.Jabatan
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND aktif = ?", id, true).First(&jabatan).Error; err != nil {
			return classifyRecordError(err)
		}
		if err := tx.Model(&jabatan).Updates(updates).Error; err != nil {
			return classifyRecordError(err)
		}
		if err := tx.First(&jabatan, "id = ?", id).Error; err != nil {
			return fmt.Errorf("reload jabatan: %w", err)
		}
		return createAudit(ctx, tx, audit.Entry{ActorID: actorID, Action: audit.ActionJabatanUpdated, TargetID: optionalRef(id)})
	})
	return jabatan, err
}

// ArchiveJabatan menonaktifkan master jabatan yang tidak lagi dipakai.
func (s GormStore) ArchiveJabatan(ctx context.Context, actorID, id string) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var jabatan models.Jabatan
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND aktif = ?", id, true).First(&jabatan).Error; err != nil {
			return classifyRecordError(err)
		}
		result := tx.Model(&models.Jabatan{}).Where("id = ? AND aktif = ?", id, true).Update("aktif", false)
		if result.Error != nil {
			return fmt.Errorf("archive jabatan: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return domain.ErrNotFound
		}
		return createAudit(ctx, tx, audit.Entry{ActorID: actorID, Action: audit.ActionJabatanArchived, TargetID: optionalRef(id)})
	})
}

func jabatanColumns(patch domain.JabatanPatch) map[string]any {
	updates := make(map[string]any)
	if patch.Nama.Present {
		updates["nama"] = patch.Nama.Value
	}
	if patch.Jenis.Present {
		updates["jenis"] = domain.NormalizeJabatanType(patch.Jenis.Value)
	}
	return updates
}
