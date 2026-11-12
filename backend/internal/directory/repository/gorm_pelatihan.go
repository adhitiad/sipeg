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

func (s GormStore) ListPelatihan(ctx context.Context, options listing.Options) ([]models.RiwayatPelatihan, int64, error) {
	query := s.DB.WithContext(ctx).Model(&models.RiwayatPelatihan{})
	if options.UserID != "" {
		query = query.Where("userId = ?", options.UserID)
	}
	if len(options.ScopeHomebaseIDs) > 0 {
		query = query.Where("userId IN (SELECT id FROM user WHERE homebaseId IN ?)", options.ScopeHomebaseIDs)
	}
	if options.Search != "" {
		pattern := "%" + escapeLike(options.Search) + "%"
		query = query.Where("(namaPelatihan ILIKE ? ESCAPE '\\' OR penyelenggara ILIKE ? ESCAPE '\\')", pattern, pattern)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count pelatihan: %w", err)
	}
	var items []models.RiwayatPelatihan
	order := listing.OrderClause(domain.PelatihanSortColumns, options.Sort, options.Order, "tanggalMulai")
	err := query.Order(order).Offset((options.Page - 1) * options.PageSize).Limit(options.PageSize).Find(&items).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list pelatihan: %w", err)
	}
	return items, total, nil
}

func (s GormStore) GetPelatihan(ctx context.Context, id string) (models.RiwayatPelatihan, error) {
	var item models.RiwayatPelatihan
	err := s.DB.WithContext(ctx).Where("id = ?", id).First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.RiwayatPelatihan{}, domain.ErrNotFound
		}
		return models.RiwayatPelatihan{}, fmt.Errorf("read pelatihan: %w", err)
	}
	return item, nil
}

func (s GormStore) CreatePelatihan(ctx context.Context, actorID string, item models.RiwayatPelatihan) (models.RiwayatPelatihan, error) {
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&item).Error; err != nil {
			return classifyRecordError(err)
		}
		return createAudit(ctx, tx, audit.Entry{ActorID: actorID, Action: audit.ActionPelatihanCreated, TargetID: &item.ID, UnitID: &item.UserId})
	})
	return item, err
}

func (s GormStore) UpdatePelatihan(ctx context.Context, actorID, id string, patch domain.RiwayatPelatihanPatch) (models.RiwayatPelatihan, error) {
	updates := pelatihanColumns(patch)
	var item models.RiwayatPelatihan
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&item).Error; err != nil {
			return classifyRecordError(err)
		}
		if err := tx.Model(&item).Updates(updates).Error; err != nil {
			return classifyRecordError(err)
		}
		if err := tx.First(&item, "id = ?", id).Error; err != nil {
			return fmt.Errorf("reload pelatihan: %w", err)
		}
		return createAudit(ctx, tx, audit.Entry{ActorID: actorID, Action: audit.ActionPelatihanUpdated, TargetID: optionalRef(id), UnitID: &item.UserId})
	})
	return item, err
}

func (s GormStore) ArchivePelatihan(ctx context.Context, actorID, id string) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var item models.RiwayatPelatihan
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&item).Error; err != nil {
			return classifyRecordError(err)
		}
		result := tx.Model(&models.RiwayatPelatihan{}).Where("id = ?").Update("aktif", false)
		if result.Error != nil {
			return fmt.Errorf("archive pelatihan: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return domain.ErrNotFound
		}
		return createAudit(ctx, tx, audit.Entry{ActorID: actorID, Action: audit.ActionPelatihanArchived, TargetID: optionalRef(id), UnitID: &item.UserId})
	})
}

func pelatihanColumns(patch domain.RiwayatPelatihanPatch) map[string]any {
	updates := make(map[string]any)
	if patch.NamaPelatihan.Present {
		if patch.NamaPelatihan.Null {
			updates["namaPelatihan"] = nil
		} else {
			updates["namaPelatihan"] = patch.NamaPelatihan.Value
		}
	}
	if patch.JenisPelatihan.Present {
		if patch.JenisPelatihan.Null {
			updates["jenisPelatihan"] = nil
		} else {
			updates["jenisPelatihan"] = patch.JenisPelatihan.Value
		}
	}
	if patch.Penyelenggara.Present {
		if patch.Penyelenggara.Null {
			updates["penyelenggara"] = nil
		} else {
			updates["penyelenggara"] = patch.Penyelenggara.Value
		}
	}
	if patch.TanggalMulai.Present {
		if patch.TanggalMulai.Null {
			updates["tanggalMulai"] = nil
		} else {
			updates["tanggalMulai"] = patch.TanggalMulai.Value
		}
	}
	if patch.TanggalSelesai.Present {
		if patch.TanggalSelesai.Null {
			updates["tanggalSelesai"] = nil
		} else {
			updates["tanggalSelesai"] = patch.TanggalSelesai.Value
		}
	}
	if patch.DurasiJam.Present {
		if patch.DurasiJam.Null {
			updates["durasiJam"] = nil
		} else {
			updates["durasiJam"] = patch.DurasiJam.Value
		}
	}
	if patch.MemilikiSertifikat.Present {
		if patch.MemilikiSertifikat.Null {
			updates["memilikiSertifikat"] = nil
		} else {
			updates["memilikiSertifikat"] = patch.MemilikiSertifikat.Value
		}
	}
	return updates
}
