// GormKepesertaan menyimpan adapter GORM untuk kepesertaan DPLK per pegawai.
package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"simpeg/backend/internal/audit"
	"simpeg/backend/internal/models"
)

// ListKepesertaan mengembalikan kepesertaan beserta nama pegawai. Bila
// scopeIDs non-nil, hanya pegawai pada unit tersebut yang disertakan;
// slice kosong menghasilkan daftar kosong.
func (s GormStore) ListKepesertaan(ctx context.Context, scopeIDs []string) ([]KepesertaanRow, error) {
	if scopeIDs != nil && len(scopeIDs) == 0 {
		return []KepesertaanRow{}, nil
	}
	var rows []KepesertaanRow
	query := s.DB.WithContext(ctx).
		Table(`"pensiunKepesertaan" AS k`).
		Select(`k.*, u.nama AS nama`).
		Joins(`JOIN "user" u ON u.id = k."userId"`)
	if scopeIDs != nil {
		query = query.Where(`u."homebaseId" IN ?`, scopeIDs)
	}
	err := query.Order(`k."aktif" DESC, u.nama ASC`).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list pensiunKepesertaan: %w", err)
	}
	return rows, nil
}

// ListUnitScopeIDs mengembalikan unit penugasan beserta seluruh turunannya.
func (s GormStore) ListUnitScopeIDs(ctx context.Context, rootHomebaseID string) ([]string, error) {
	if rootHomebaseID == "" {
		return nil, nil
	}
	var unitIDs []string
	err := s.DB.WithContext(ctx).Raw(`
		WITH RECURSIVE subtree AS (
			SELECT id FROM "Homebase" WHERE id = ?
			UNION ALL
			SELECT child.id FROM "Homebase" child
			JOIN subtree ON child."parentId" = subtree.id
		)
		SELECT id FROM subtree`, rootHomebaseID).Scan(&unitIDs).Error
	if err != nil {
		return nil, fmt.Errorf("resolve unit scope: %w", err)
	}
	return unitIDs, nil
}

// UpsertKepesertaan menulis atau mengganti kepesertaan berdasarkan userId.
func (s GormStore) UpsertKepesertaan(ctx context.Context, actorID string, item models.PensiunKepesertaan) (models.PensiunKepesertaan, error) {
	var result models.PensiunKepesertaan
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing models.PensiunKepesertaan
		err := tx.Where(`"userId" = ?`, item.UserId).First(&existing).Error
		switch {
		case err == nil:
			applyKepesertaan(&existing, item)
			if err := tx.Save(&existing).Error; err != nil {
				return fmt.Errorf("update pensiunKepesertaan: %w", err)
			}
			result = existing
			note := fmt.Sprintf("kepesertaan DPLK diperbarui: user=%s", item.UserId)
			return auditWithNote(ctx, tx, actorID, audit.ActionPensiunKepesertaanUpdated, note)
		case errors.Is(err, gorm.ErrRecordNotFound):
			if err := tx.Create(&item).Error; err != nil {
				return fmt.Errorf("create pensiunKepesertaan: %w", err)
			}
			result = item
			note := fmt.Sprintf("kepesertaan DPLK dibuat: user=%s penyedia=%s", item.UserId, item.PenyediaDplk)
			return auditWithNote(ctx, tx, actorID, audit.ActionPensiunKepesertaanCreated, note)
		default:
			return fmt.Errorf("baca pensiunKepesertaan: %w", err)
		}
	})
	return result, err
}

// UpdateKepesertaan mengganti seluruh isian mutable kepesertaan.
func (s GormStore) UpdateKepesertaan(ctx context.Context, actorID string, id string, item models.PensiunKepesertaan) (models.PensiunKepesertaan, error) {
	var existing models.PensiunKepesertaan
	if err := s.DB.WithContext(ctx).Where("id = ?", id).First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.PensiunKepesertaan{}, ErrNotFoundKepesertaan
		}
		return models.PensiunKepesertaan{}, fmt.Errorf("baca pensiunKepesertaan: %w", err)
	}
	applyKepesertaan(&existing, item)
	if err := s.DB.WithContext(ctx).Save(&existing).Error; err != nil {
		return models.PensiunKepesertaan{}, fmt.Errorf("update pensiunKepesertaan: %w", err)
	}
	note := fmt.Sprintf("kepesertaan DPLK diperbarui: user=%s", existing.UserId)
	if err := auditWithNote(ctx, s.DB, actorID, audit.ActionPensiunKepesertaanUpdated, note); err != nil {
		return models.PensiunKepesertaan{}, err
	}
	return existing, nil
}

// DeactivateKepesertaan menonaktifkan kepesertaan (riwayat tetap tersimpan).
func (s GormStore) DeactivateKepesertaan(ctx context.Context, actorID string, id string) error {
	var existing models.PensiunKepesertaan
	if err := s.DB.WithContext(ctx).Where("id = ?", id).First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFoundKepesertaan
		}
		return fmt.Errorf("baca pensiunKepesertaan: %w", err)
	}
	if err := s.DB.WithContext(ctx).Model(&models.PensiunKepesertaan{}).Where("id = ?", id).Update("aktif", false).Error; err != nil {
		return fmt.Errorf("deactivate pensiunKepesertaan: %w", err)
	}
	note := fmt.Sprintf("kepesertaan DPLK dinonaktifkan: user=%s", existing.UserId)
	return auditWithNote(ctx, s.DB, actorID, audit.ActionPensiunKepesertaanDeactivated, note)
}

func applyKepesertaan(dst *models.PensiunKepesertaan, src models.PensiunKepesertaan) {
	dst.PenyediaDplk = src.PenyediaDplk
	dst.NomorKepesertaan = src.NomorKepesertaan
	dst.TanggalMulaiPensiun = src.TanggalMulaiPensiun
	dst.Mulai = src.Mulai
	dst.Aktif = src.Aktif
}
