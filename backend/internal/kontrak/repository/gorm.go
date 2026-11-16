// GormStore menyimpan adapter GORM untuk port kontrak kerja dan perpanjangan
// masa kerja (DEC-083).
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

// EmployeeHomebaseID mengembalikan unit penugasan pegawai; nil bila belum diisi.
func (s GormStore) EmployeeHomebaseID(ctx context.Context, userID string) (*string, error) {
	var row struct {
		HomebaseID *string `gorm:"column:homebaseId"`
	}
	if err := s.DB.WithContext(ctx).Table(`"user"`).Select(`"homebaseId"`).Where(`"id" = ?`, userID).Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("read homebase pegawai: %w", err)
	}
	return row.HomebaseID, nil
}

// EmployeeExists melaporkan apakah pegawai dikenal.
func (s GormStore) EmployeeExists(ctx context.Context, userID string) (bool, error) {
	var count int64
	if err := s.DB.WithContext(ctx).Table(`"user"`).Where(`"id" = ?`, userID).Count(&count).Error; err != nil {
		return false, fmt.Errorf("count pegawai: %w", err)
	}
	return count > 0, nil
}

// EmployeeKategoriStatusID mengembalikan kategori status pegawai; nil bila belum diisi.
func (s GormStore) EmployeeKategoriStatusID(ctx context.Context, userID string) (*string, error) {
	var row struct {
		KategoriStatusID *string `gorm:"column:kategoriStatusId"`
	}
	if err := s.DB.WithContext(ctx).Table(`"user"`).Select(`"kategoriStatusId"`).Where(`"id" = ?`, userID).Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("read kategori status pegawai: %w", err)
	}
	return row.KategoriStatusID, nil
}

// Create menulis kontrak PKWT dalam transaksi bersama pemindahan kategori
// status pegawai ke DOSEN_TIDAK_TETAP dan jejak audit.
func (s GormStore) Create(ctx context.Context, actor string, input KontrakInput, nomorSk string, newKategoriStatusID string, unitID *string) (models.KontrakKerja, error) {
	selesai := input.Mulai.AddDate(0, input.Durasi, 0)
	item := models.KontrakKerja{
		UserId:  input.UserID,
		Jenis:   models.JenisKontrakPKWT,
		NomorSk: nomorSk,
		Mulai:   input.Mulai,
		Selesai: &selesai,
		Status:  string(models.StatusKontrakAktif),
	}
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&item).Error; err != nil {
			return fmt.Errorf("create contract: %w", err)
		}
		if err := tx.Table(`"user"`).Where(`"id" = ?`, input.UserID).Updates(map[string]any{"kategoriStatusId": newKategoriStatusID}).Error; err != nil {
			return fmt.Errorf("flip kategori status: %w", err)
		}
		targetID := item.ID
		note := fmt.Sprintf("perjanjian kerja tahunan %s — %s; kategoriStatusId dipindahkan ke DOSEN_TIDAK_TETAP", input.Mulai.Format("2006-01-02"), selesai.Format("2006-01-02"))
		if input.AllowOverlap {
			note += "; tumpang tindih kontrak AKTIF diizinkan eksplisit (override admin)"
		}
		return audit.Write(ctx, tx, audit.Entry{
			ActorID:  actor,
			Action:   audit.ActionPerpanjanganCreated,
			TargetID: &targetID,
			UnitID:   unitID,
			Note:     &note,
		})
	})
	return item, err
}

// ActiveOverlapping melaporkan apakah ada kontrak AKTIF pegawai yang
// tumpang tindih dengan rentang [mulai, selesai). Kontrak PKWTT dengan
// selesai NULL dianggap tumpang tindih dengan seluruh periode.
func (s GormStore) ActiveOverlapping(ctx context.Context, userID string, mulai, selesai time.Time) (bool, error) {
	var count int64
	err := s.DB.WithContext(ctx).Model(&models.KontrakKerja{}).
		Where(`"userId" = ? AND "status" = ? AND "mulai" < ? AND ("selesai" IS NULL OR "selesai" > ?)`,
			userID, string(models.StatusKontrakAktif), selesai, mulai).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("check overlapping contract: %w", err)
	}
	return count > 0, nil
}

// CountByTypeYear menghitung kontrak per jenis dan tahun (dipakai untuk
// nomor urut SK kontrak).
func (s GormStore) CountByTypeYear(ctx context.Context, jenis models.JenisKontrak, year int) (int64, error) {
	var count int64
	if err := s.DB.WithContext(ctx).Model(&models.KontrakKerja{}).
		Where(`"jenis" = ? AND extract(year from "mulai") = ?`, jenis, year).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count kontrak per tahun: %w", err)
	}
	return count, nil
}

// List membaca kontrak sesuai filter dan cakupan.
func (s GormStore) List(ctx context.Context, filter KontrakFilter) ([]models.KontrakKerja, int64, error) {
	query := s.DB.WithContext(ctx).Model(&models.KontrakKerja{})
	if filter.UserID != "" {
		query = query.Where(`"userId" = ?`, filter.UserID)
	}
	if filter.Jenis != "" {
		query = query.Where(`"jenis" = ?`, filter.Jenis)
	}
	if filter.MulaiSesudah != nil {
		query = query.Where(`"mulai" >= ?`, *filter.MulaiSesudah)
	}
	if filter.SelesaiSebelum != nil {
		query = query.Where(`"selesai" IS NOT NULL AND "selesai" <= ?`, *filter.SelesaiSebelum)
	}
	if len(filter.HomebaseIDs) > 0 {
		query = query.Where(`"userId" IN (SELECT "id" FROM "user" WHERE "homebaseId" IN ?)`, filter.HomebaseIDs)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count kontrak: %w", err)
	}
	if filter.Limit == 0 {
		filter.Limit = 100
	}
	var items []models.KontrakKerja
	if err := query.Order(`"mulai" DESC`).Limit(filter.Limit).Offset(filter.Offset).Find(&items).Error; err != nil {
		return nil, 0, fmt.Errorf("list kontrak: %w", err)
	}
	return items, total, nil
}
