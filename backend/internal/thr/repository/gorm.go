// GormStore menyimpan adapter GORM untuk port THR (`DEC-083`).
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"simpeg/backend/internal/audit"
	"simpeg/backend/internal/models"
	payrolldomain "simpeg/backend/internal/payroll/domain"

	"simpeg/backend/internal/thr/domain"
)

// GormStore mengimplementasikan Store dengan PostgreSQL.
type GormStore struct {
	DB *gorm.DB
}

// List mengembalikan seluruh konfigurasi THR (tahun terbaru lebih dulu).
func (s GormStore) List(ctx context.Context) ([]models.ThrKonfigurasi, error) {
	var items []models.ThrKonfigurasi
	if err := s.DB.WithContext(ctx).Order(`"tahun" DESC`).Find(&items).Error; err != nil {
		return nil, fmt.Errorf("list thr config: %w", err)
	}
	return items, nil
}

// GetByTahun membaca satu konfigurasi.
func (s GormStore) GetByTahun(ctx context.Context, tahun int) (models.ThrKonfigurasi, error) {
	var item models.ThrKonfigurasi
	err := s.DB.WithContext(ctx).Where(`"tahun" = ?`, tahun).First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.ThrKonfigurasi{}, domain.ErrConfigNotFound
		}
		return models.ThrKonfigurasi{}, fmt.Errorf("read thr config: %w", err)
	}
	return item, nil
}

// Create menulis konfigurasi baru. actorID tercatat pada jejak audit karena
// aturan per tahun berdampak ke seluruh unit.
func (s GormStore) Create(ctx context.Context, actorID string, item models.ThrKonfigurasi) (models.ThrKonfigurasi, error) {
	err := s.DB.WithContext(ctx).Create(&item).Error
	if err != nil {
		return models.ThrKonfigurasi{}, fmt.Errorf("create thr config: %w", err)
	}
	if err := s.auditConfig(ctx, actorID, audit.ActionThrConfigUpdated, item); err != nil {
		return models.ThrKonfigurasi{}, err
	}
	return item, nil
}

// Update mengganti isi konfigurasi tahun tersebut.
func (s GormStore) Update(ctx context.Context, actorID string, tahun int, item models.ThrKonfigurasi) (models.ThrKonfigurasi, error) {
	var existing models.ThrKonfigurasi
	if err := s.DB.WithContext(ctx).Where(`"tahun" = ?`, tahun).First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.ThrKonfigurasi{}, domain.ErrConfigNotFound
		}
		return models.ThrKonfigurasi{}, fmt.Errorf("read thr config for update: %w", err)
	}
	existing.BesaranTipe = item.BesaranTipe
	existing.Persen = item.Persen
	existing.SyaratMasaKerjaBulan = item.SyaratMasaKerjaBulan
	existing.BerlakuTanggal = item.BerlakuTanggal
	if err := s.DB.WithContext(ctx).Save(&existing).Error; err != nil {
		return models.ThrKonfigurasi{}, fmt.Errorf("update thr config: %w", err)
	}
	if err := s.auditConfig(ctx, actorID, audit.ActionThrConfigUpdated, existing); err != nil {
		return models.ThrKonfigurasi{}, err
	}
	return existing, nil
}

// Delete menghapus konfigurasi tahun tersebut.
func (s GormStore) Delete(ctx context.Context, actorID string, tahun int) error {
	var existing models.ThrKonfigurasi
	if err := s.DB.WithContext(ctx).Where(`"tahun" = ?`, tahun).First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrConfigNotFound
		}
		return fmt.Errorf("read thr config for delete: %w", err)
	}
	if err := s.DB.WithContext(ctx).Where(`"tahun" = ?`, tahun).Delete(&models.ThrKonfigurasi{}).Error; err != nil {
		return fmt.Errorf("delete thr config: %w", err)
	}
	return s.auditConfig(ctx, actorID, audit.ActionThrConfigDeleted, existing)
}

// auditConfig menulis jejak audit perubahan konfigurasi THR per tahun.
func (s GormStore) auditConfig(ctx context.Context, actorID, action string, item models.ThrKonfigurasi) error {
	persen := "-"
	if item.Persen != nil {
		persen = fmt.Sprintf("%.2f", *item.Persen)
	}
	note := fmt.Sprintf("THR %d: besaran=%s persen=%s syarat=%d bln berlaku=%s",
		item.Tahun, item.BesaranTipe, persen, item.SyaratMasaKerjaBulan, item.BerlakuTanggal.Format("2006-01-02"))
	return audit.Write(ctx, s.DB, audit.Entry{
		ActorID: actorID,
		Action:  action,
		Note:    &note,
	})
}

// Candidates mengembalikan pegawai aktif dengan slip TERBIT/TERBAYAR terakhir.
func (s GormStore) Candidates(ctx context.Context) ([]Candidate, error) {
	var rows []struct {
		UserID       string     `gorm:"column:userId"`
		Nama         string     `gorm:"column:nama"`
		TanggalMasuk *time.Time `gorm:"column:tanggalMasuk"`
		GajiPokok    *float64   `gorm:"column:gajiPokok"`
		TotalGaji    *float64   `gorm:"column:totalGaji"`
		GajiBersih   *float64   `gorm:"column:gajiBersih"`
		HomebaseID   *string    `gorm:"column:homebaseId"`
		HasSlip      bool       `gorm:"column:hasSlip"`
	}
	err := s.DB.WithContext(ctx).Raw(`
		SELECT DISTINCT ON (u."id")
			u."id" AS "userId",
			u."nama" AS "nama",
			u."tanggalMasuk" AS "tanggalMasuk",
			u."homebaseId" AS "homebaseId",
			g."gajiPokok" AS "gajiPokok",
			g."totalGaji" AS "totalGaji",
			(g."totalGaji" - COALESCE((
				SELECT SUM(p."nominal") FROM "gajiPotongan" p WHERE p."gajiId" = g.id
			), 0)) AS "gajiBersih",
			(g.id IS NOT NULL) AS "hasSlip"
		FROM "user" u
		LEFT JOIN LATERAL (
			SELECT s.id, s."gajiPokok", s."totalGaji", s."periode"
			FROM "gaji" s
			WHERE s."userId" = u."id" AND s."status" IN ('TERBIT', 'TERBAYAR')
			ORDER BY s."periode" DESC
			LIMIT 1
		) g ON true
		WHERE u."aktif" = true
		ORDER BY u."id", g."periode" DESC NULLS LAST`).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list thr candidates: %w", err)
	}
	candidates := make([]Candidate, 0, len(rows))
	for _, row := range rows {
		gajiPokok, totalGaji, gajiBersih := 0.0, 0.0, 0.0
		if row.GajiPokok != nil {
			gajiPokok = *row.GajiPokok
		}
		if row.TotalGaji != nil {
			totalGaji = *row.TotalGaji
		}
		if row.GajiBersih != nil {
			gajiBersih = *row.GajiBersih
		}
		candidates = append(candidates, Candidate{
			UserID:       row.UserID,
			Nama:         row.Nama,
			TanggalMasuk: row.TanggalMasuk,
			GajiPokok:    gajiPokok,
			TotalGaji:    totalGaji,
			GajiBersih:   gajiBersih,
			HomebaseID:   row.HomebaseID,
			HasSlip:      row.HasSlip,
		})
	}
	return candidates, nil
}

// CreateSlip menulis slip THR untuk pegawai dalam satu transaksi bersama
// komponen penerimaan `THR` dan jejak audit.
func (s GormStore) CreateSlip(ctx context.Context, actor string, unitID *string, tahun int, berlakunya time.Time, userID string, nominal float64, sumber string) (models.Gaji, error) {
	periode := time.Date(berlakunya.Year(), berlakunya.Month(), berlakunya.Day(), 0, 0, 0, 0, time.UTC)
	item := models.Gaji{
		UserId:      userID,
		Periode:     periode,
		TanggalGaji: periode,
		TotalGaji:   nominal,
		Status:      models.StatusSlipDraf,
	}
	prefix := fmt.Sprintf("THR-%d-", tahun)

	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing models.Gaji
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where(`"userId" = ? AND "periode" = ?`, userID, periode).
			First(&existing).Error
		switch {
		case err == nil:
			switch existing.Status {
			case models.StatusSlipTerbit, models.StatusSlipTerbayar, models.StatusSlipBatal:
				return domain.ErrSlipConflict
			}
			if err := tx.Where(`"gajiId" = ?`, existing.ID).Delete(&models.GajiTunjangan{}).Error; err != nil {
				return fmt.Errorf("delete tunjangan thr lama: %w", err)
			}
			if err := tx.Where(`"gajiId" = ?`, existing.ID).Delete(&models.GajiPotongan{}).Error; err != nil {
				return fmt.Errorf("delete potongan thr lama: %w", err)
			}
			item.ID = existing.ID
			item.NomorSlip = existing.NomorSlip
			item.KodeVerifikasi = existing.KodeVerifikasi
			item.CreatedAt = existing.CreatedAt
		case errors.Is(err, gorm.ErrRecordNotFound):
			var count int64
			if err := tx.Model(&models.Gaji{}).Where(`"nomorSlip" LIKE ?`, prefix+"%").Count(&count).Error; err != nil {
				return fmt.Errorf("count thr slip: %w", err)
			}
			item.NomorSlip = fmt.Sprintf("%s%04d", prefix, count+1)
			token, err := payrolldomain.NewSlipVerificationToken()
			if err != nil {
				return fmt.Errorf("generate verification token: %w", err)
			}
			item.KodeVerifikasi = &token
		default:
			return fmt.Errorf("read existing thr slip: %w", err)
		}

		if err := tx.Save(&item).Error; err != nil {
			return fmt.Errorf("save thr slip: %w", err)
		}

		tunjangan := []models.GajiTunjangan{{
			Jenis:   "THR",
			Nominal: nominal,
			GajiId:  item.ID,
		}}
		if err := tx.Create(&tunjangan).Error; err != nil {
			return fmt.Errorf("insert tunjangan thr: %w", err)
		}

		targetID := item.ID
		note := fmt.Sprintf("THR %d: nominal dihitung dari slip TERBIT/TERBAYAR terakhir (%s)", tahun, sumber)
		return audit.Write(ctx, tx, audit.Entry{
			ActorID:  actor,
			Action:   audit.ActionThrGenerated,
			TargetID: &targetID,
			UnitID:   unitID,
			Note:     &note,
		})
	})
	return item, err
}
