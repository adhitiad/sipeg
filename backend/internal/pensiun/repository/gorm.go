// GormStore menyimpan adapter GORM untuk port pensiun.
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"simpeg/backend/internal/audit"
	"simpeg/backend/internal/models"
)

// GormStore mengimplementasikan Store dengan PostgreSQL.
type GormStore struct {
	DB *gorm.DB
}

// ListKonfigurasi mengembalikan konfigurasi per kategori status (tanggal
// berlaku terbaru lebih dulu).
func (s GormStore) ListKonfigurasi(ctx context.Context, kategoriStatusID string) ([]models.PensiunKonfigurasi, error) {
	query := s.DB.WithContext(ctx).Model(&models.PensiunKonfigurasi{})
	if kategoriStatusID != "" {
		query = query.Where(`"kategoriStatusId" = ?`, kategoriStatusID)
	}
	var items []models.PensiunKonfigurasi
	if err := query.Order(`"berlakuMulai" DESC, "createdAt" DESC`).Find(&items).Error; err != nil {
		return nil, fmt.Errorf("list pensiunKonfigurasi: %w", err)
	}
	return items, nil
}

// UpsertKonfigurasi menulis atau mengganti baris (kategoriStatusId, berlakuMulai).
func (s GormStore) UpsertKonfigurasi(ctx context.Context, actorID string, item models.PensiunKonfigurasi) (models.PensiunKonfigurasi, error) {
	var result models.PensiunKonfigurasi
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing models.PensiunKonfigurasi
		err := tx.Where(`"kategoriStatusId" = ? AND "berlakuMulai" = ?`, item.KategoriStatusId, item.BerlakuMulai).First(&existing).Error
		switch {
		case err == nil:
			applyKonfigurasi(&existing, item)
			if err := tx.Save(&existing).Error; err != nil {
				return fmt.Errorf("update pensiunKonfigurasi: %w", err)
			}
			result = existing
			note := fmt.Sprintf("konfigurasi pensiun kategori=%s berlaku=%s", item.KategoriStatusId, item.BerlakuMulai.Format("2006-01-02"))
			return auditWithNote(ctx, tx, actorID, audit.ActionPensiunKonfigurasiUpdated, note)
		case errors.Is(err, gorm.ErrRecordNotFound):
			if err := tx.Create(&item).Error; err != nil {
				return fmt.Errorf("create pensiunKonfigurasi: %w", err)
			}
			result = item
			note := fmt.Sprintf("konfigurasi pensiun kategori=%s berlaku=%s BUP=%d", item.KategoriStatusId, item.BerlakuMulai.Format("2006-01-02"), item.BupUmur)
			return auditWithNote(ctx, tx, actorID, audit.ActionPensiunKonfigurasiCreated, note)
		default:
			return fmt.Errorf("baca pensiunKonfigurasi: %w", err)
		}
	})
	return result, err
}

// UpdateKonfigurasi mengganti seluruh isian mutable satu baris konfigurasi.
func (s GormStore) UpdateKonfigurasi(ctx context.Context, actorID string, id string, item models.PensiunKonfigurasi) (models.PensiunKonfigurasi, error) {
	var existing models.PensiunKonfigurasi
	if err := s.DB.WithContext(ctx).Where("id = ?", id).First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.PensiunKonfigurasi{}, ErrNotFoundKonfigurasi
		}
		return models.PensiunKonfigurasi{}, fmt.Errorf("baca pensiunKonfigurasi: %w", err)
	}
	applyKonfigurasi(&existing, item)
	if err := s.DB.WithContext(ctx).Save(&existing).Error; err != nil {
		return models.PensiunKonfigurasi{}, fmt.Errorf("update pensiunKonfigurasi: %w", err)
	}
	note := fmt.Sprintf("konfigurasi pensiun kategori=%s berlaku=%s BUP=%d", existing.KategoriStatusId, existing.BerlakuMulai.Format("2006-01-02"), existing.BupUmur)
	if err := auditWithNote(ctx, s.DB, actorID, audit.ActionPensiunKonfigurasiUpdated, note); err != nil {
		return models.PensiunKonfigurasi{}, err
	}
	return existing, nil
}

// DeleteKonfigurasi menghapus satu baris konfigurasi.
func (s GormStore) DeleteKonfigurasi(ctx context.Context, actorID string, id string) error {
	var existing models.PensiunKonfigurasi
	if err := s.DB.WithContext(ctx).Where("id = ?", id).First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFoundKonfigurasi
		}
		return fmt.Errorf("baca pensiunKonfigurasi: %w", err)
	}
	if err := s.DB.WithContext(ctx).Where("id = ?", id).Delete(&models.PensiunKonfigurasi{}).Error; err != nil {
		return fmt.Errorf("delete pensiunKonfigurasi: %w", err)
	}
	note := fmt.Sprintf("konfigurasi pensiun kategori=%s berlaku=%s", existing.KategoriStatusId, existing.BerlakuMulai.Format("2006-01-02"))
	return auditWithNote(ctx, s.DB, actorID, audit.ActionPensiunKonfigurasiDeleted, note)
}

// UserBirthInfo membaca tanggal lahir dan kategori status pegawai.
func (s GormStore) UserBirthInfo(ctx context.Context, userID string) (*time.Time, *string, error) {
	var row struct {
		TanggalLahir   *time.Time `gorm:"column:tanggal_lahir"`
		KategoriStatus *string    `gorm:"column:kategoriStatusId"`
	}
	err := s.DB.WithContext(ctx).Table("user").Select(`"tanggal_lahir", "kategoriStatusId"`).Where(`id = ?`, userID).Scan(&row).Error
	if err != nil {
		return nil, nil, fmt.Errorf("baca data pegawai: %w", err)
	}
	return row.TanggalLahir, row.KategoriStatus, nil
}

// ExistsUser melaporkan apakah id menunjuk ke baris user yang ada.
func (s GormStore) ExistsUser(ctx context.Context, userID string) (bool, error) {
	var count int64
	if err := s.DB.WithContext(ctx).Table("user").Where(`id = ?`, userID).Count(&count).Error; err != nil {
		return false, fmt.Errorf("cek pegawai: %w", err)
	}
	return count > 0, nil
}

// ActiveKonfigurasi membaca konfigurasi aktif yang berlaku pada tanggal
// acuan (periode berlaku terbaru yang sudah mulai). Bila belum ada yang
// berlaku, konfigurasi aktif dengan periode berlaku terdekat setelah tanggal
// acuan dipakai — tanggal mulai pensiun harus tetap dapat dihitung sebelum
// konfigurasi resmi berlaku.
func (s GormStore) ActiveKonfigurasi(ctx context.Context, kategoriStatusID string, ref time.Time) (models.PensiunKonfigurasi, error) {
	var item models.PensiunKonfigurasi
	err := s.DB.WithContext(ctx).
		Where(`"kategoriStatusId" = ? AND "berlakuMulai" <= ? AND "aktif" = true`, kategoriStatusID, ref).
		Order(`"berlakuMulai" DESC`).First(&item).Error
	if err == nil {
		return item, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return models.PensiunKonfigurasi{}, fmt.Errorf("baca konfigurasi aktif: %w", err)
	}
	err = s.DB.WithContext(ctx).
		Where(`"kategoriStatusId" = ? AND "berlakuMulai" > ? AND "aktif" = true`, kategoriStatusID, ref).
		Order(`"berlakuMulai" ASC`).First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.PensiunKonfigurasi{}, ErrNotFoundKonfigurasi
		}
		return models.PensiunKonfigurasi{}, fmt.Errorf("baca konfigurasi aktif berikutnya: %w", err)
	}
	return item, nil
}

func applyKonfigurasi(dst *models.PensiunKonfigurasi, src models.PensiunKonfigurasi) {
	dst.BupUmur = src.BupUmur
	dst.DplkWajib = src.DplkWajib
	dst.IuranPegawaiPct = src.IuranPegawaiPct
	dst.IuranPemberiKerjaPct = src.IuranPemberiKerjaPct
	dst.PenyediaDplk = src.PenyediaDplk
	dst.BerlakuMulai = src.BerlakuMulai
	dst.Aktif = src.Aktif
}

func auditWithNote(ctx context.Context, tx *gorm.DB, actorID, action, note string) error {
	return audit.Write(ctx, tx, audit.Entry{
		ActorID: actorID,
		Action:  action,
		Note:    &note,
	})
}
