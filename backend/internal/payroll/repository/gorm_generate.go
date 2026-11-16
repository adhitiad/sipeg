// Package repository menyimpan adapter GORM untuk pembuatan slip gaji otomatis.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"simpeg/backend/internal/audit"
	"simpeg/backend/internal/models"
	"simpeg/backend/internal/payroll/domain"
)

// GetEmployeeAktif membaca apakah pegawai masih aktif; nil bila pegawai
// tidak dikenal.
func (s GormStore) GetEmployeeAktif(ctx context.Context, userID string) (*bool, error) {
	if userID == "" {
		return nil, domain.ErrValidation
	}
	var row struct {
		Aktif *bool
	}
	if err := s.DB.WithContext(ctx).
		Table(`"user"`).
		Select(`"aktif"`).
		Where(`"id" = ?`, userID).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("read pegawai aktif: %w", err)
	}
	return row.Aktif, nil
}

// ApprovedPemberhentianFor mengembalikan pemberhentian DISETUJUI terakhir
// pegawai (tanggal efektif terbaru); nil bila tidak ada.
func (s GormStore) ApprovedPemberhentianFor(ctx context.Context, userID string) (*models.Pemberhentian, error) {
	if userID == "" {
		return nil, domain.ErrValidation
	}
	var item models.Pemberhentian
	err := s.DB.WithContext(ctx).
		Model(&models.Pemberhentian{}).
		Where(`"userId" = ? AND "status" = ?`, userID, models.StatusDisetujui).
		Order(`"tanggal" DESC, "createdAt" DESC`).
		First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("read pemberhentian disetujui: %w", err)
	}
	return &item, nil
}

// GetEmployeeStatusPtkp membaca status PTKP pegawai (default TK/0).
func (s GormStore) GetEmployeeStatusPtkp(ctx context.Context, userID string) (string, error) {
	if userID == "" {
		return "", domain.ErrValidation
	}
	var row struct {
		StatusPtkp *string
	}
	if err := s.DB.WithContext(ctx).
		Table(`"user"`).
		Select(`"statusPtkp"`).
		Where(`"id" = ?`, userID).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", domain.ErrNotFound
		}
		return "", fmt.Errorf("read pegawai ptkp: %w", err)
	}
	if row.StatusPtkp == nil || *row.StatusPtkp == "" {
		return "TK/0", nil
	}
	return *row.StatusPtkp, nil
}

// GetEmployeeHomebaseID mengembalikan unit penugasan pegawai; nil bila kosong.
func (s GormStore) GetEmployeeHomebaseID(ctx context.Context, userID string) (*string, error) {
	if userID == "" {
		return nil, domain.ErrValidation
	}
	var row struct {
		HomebaseID *string `gorm:"column:homebaseId"`
	}
	if err := s.DB.WithContext(ctx).
		Table(`"user"`).
		Select(`"homebaseId"`).
		Where(`"id" = ?`, userID).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("read pegawai homebase: %w", err)
	}
	return row.HomebaseID, nil
}

// GetEmployeeKategoriStatus mengembalikan id dan kode kategori status
// pegawai (mis. DOSEN_TETAP_YAYASAN). Kode kosong bila pegawai belum
// dikaitkan ke kategori status.
func (s GormStore) GetEmployeeKategoriStatus(ctx context.Context, userID string) (string, string, error) {
	if userID == "" {
		return "", "", domain.ErrValidation
	}
	var row struct {
		KategoriID   *string `gorm:"column:kategoriStatusId"`
		KategoriKode *string `gorm:"column:kode"`
	}
	if err := s.DB.WithContext(ctx).
		Table(`"user" AS u`).
		Select(`u."kategoriStatusId", ks."kode"`).
		Joins(`LEFT JOIN "kategoriStatusPegawai" AS ks ON ks."id" = u."kategoriStatusId"`).
		Where(`u."id" = ?`, userID).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", "", domain.ErrNotFound
		}
		return "", "", fmt.Errorf("read pegawai kategori status: %w", err)
	}
	id, kode := "", ""
	if row.KategoriID != nil {
		id = *row.KategoriID
	}
	if row.KategoriKode != nil {
		kode = *row.KategoriKode
	}
	return id, kode, nil
}

// CountPresensiAlpa menghitung jumlah hari dengan status ALPA ('A')
// pada rentang [monthStart, nextMonth).
func (s GormStore) CountPresensiAlpa(ctx context.Context, userID string, monthStart, nextMonth time.Time) (int64, error) {
	if userID == "" {
		return 0, domain.ErrValidation
	}
	var count int64
	if err := s.DB.WithContext(ctx).
		Table(`"Presensi"`).
		Where(`"userId" = ? AND "status" = ? AND "tanggal" >= ? AND "tanggal" < ?`, userID, models.StatusPresensiAlpa, monthStart, nextMonth).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count presensi alpa: %w", err)
	}
	return count, nil
}

// LookupGajiPokok mengembalikan skala efektif dan baris nominal untuk
// kombinasi (kategoriStatusId, tingkat) pada tanggal periode berlaku.
func (s GormStore) LookupGajiPokok(ctx context.Context, kategoriStatusId string, tingkat int, periodEnd time.Time) (models.SkalaGaji, models.SkalaGajiDetail, error) {
	var skala models.SkalaGaji
	if err := s.DB.WithContext(ctx).
		Where(`"aktif" = ? AND "berlakuMulai" <= ? AND ("berlakuSelesai" IS NULL OR "berlakuSelesai" >= ?)`, true, periodEnd, periodEnd).
		Order(`"berlakuMulai" DESC`).
		First(&skala).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.SkalaGaji{}, models.SkalaGajiDetail{}, domain.ErrNotFound
		}
		return models.SkalaGaji{}, models.SkalaGajiDetail{}, fmt.Errorf("lookup skala: %w", err)
	}
	var detail models.SkalaGajiDetail
	if err := s.DB.WithContext(ctx).
		Where(`"skalaGajiId" = ? AND "kategoriStatusId" = ? AND "tingkat" = ?`, skala.ID, kategoriStatusId, tingkat).
		First(&detail).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return skala, models.SkalaGajiDetail{}, domain.ErrNotFound
		}
		return skala, models.SkalaGajiDetail{}, fmt.Errorf("lookup skala detail: %w", err)
	}
	return skala, detail, nil
}

// FindPeriodeAkademik mengembalikan periode akademik yang memuat periode gaji.
func (s GormStore) FindPeriodeAkademik(ctx context.Context, periodStart, periodEnd time.Time) (models.PeriodeAkademik, error) {
	var row models.PeriodeAkademik
	if err := s.DB.WithContext(ctx).
		Where(`"tanggalMulai" <= ? AND "tanggalSelesai" >= ?`, periodEnd, periodStart).
		Order(`"tahun" DESC, "semester" DESC`).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.PeriodeAkademik{}, domain.ErrNotFound
		}
		return models.PeriodeAkademik{}, fmt.Errorf("find periode akademik: %w", err)
	}
	return row, nil
}

// GetPeriodeAkademikByID membaca satu periode akademik.
func (s GormStore) GetPeriodeAkademikByID(ctx context.Context, id string) (models.PeriodeAkademik, error) {
	var row models.PeriodeAkademik
	if err := s.DB.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.PeriodeAkademik{}, domain.ErrNotFound
		}
		return models.PeriodeAkademik{}, fmt.Errorf("get periode akademik: %w", err)
	}
	return row, nil
}

// ListKomponenTunjangan membaca komponen tunjangan pegawai.
func (s GormStore) ListKomponenTunjangan(ctx context.Context, userID string) ([]models.KomponenTunjangan, error) {
	var rows []models.KomponenTunjangan
	if err := s.DB.WithContext(ctx).
		Where(`"userId" = ? AND "aktif" = ?`, userID, true).
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list komponen tunjangan: %w", err)
	}
	return rows, nil
}

// ListKomponenPotongan membaca komponen potongan pegawai.
func (s GormStore) ListKomponenPotongan(ctx context.Context, userID string) ([]models.KomponenPotongan, error) {
	var rows []models.KomponenPotongan
	if err := s.DB.WithContext(ctx).
		Where(`"userId" = ? AND "aktif" = ?`, userID, true).
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list komponen potongan: %w", err)
	}
	return rows, nil
}

// LookupHonor membaca baris honor beban mengajar dosen LB untuk periode akademik.
func (s GormStore) LookupHonor(ctx context.Context, userID, periodeAkademikId string) (models.HonorSksPeriode, error) {
	var row models.HonorSksPeriode
	if err := s.DB.WithContext(ctx).
		Where(`"userId" = ? AND "periodeAkademikId" = ?`, userID, periodeAkademikId).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.HonorSksPeriode{}, domain.ErrNotFound
		}
		return models.HonorSksPeriode{}, fmt.Errorf("lookup honor periode: %w", err)
	}
	return row, nil
}

// LatestBpjsConfig mengembalikan satu baris tarif BPJS per jenis yang berlaku.
func (s GormStore) LatestBpjsConfig(ctx context.Context, asOf time.Time) ([]models.BpjsKonfigurasi, error) {
	var all []models.BpjsKonfigurasi
	if err := s.DB.WithContext(ctx).
		Where(`"aktif" = ? AND "berlakuMulai" <= ?`, true, asOf).
		Order(`"berlakuMulai" DESC`).
		Find(&all).Error; err != nil {
		return nil, fmt.Errorf("latest bpjs config: %w", err)
	}
	seen := map[models.JenisBPJS]bool{}
	rows := []models.BpjsKonfigurasi{}
	for _, row := range all {
		if seen[row.Jenis] {
			continue
		}
		seen[row.Jenis] = true
		rows = append(rows, row)
	}
	return rows, nil
}

// TerOn mengembalikan tabel tarif efektif TER yang berlaku pada tanggal tersebut.
func (s GormStore) TerOn(ctx context.Context, asOf time.Time) ([]models.Pph21TerTarif, error) {
	var latest sql.NullTime
	if err := s.DB.WithContext(ctx).Model(&models.Pph21TerTarif{}).
		Select(`MAX("berlakuMulai")`).
		Where(`"berlakuMulai" <= ?`, asOf).
		Scan(&latest).Error; err != nil {
		return nil, fmt.Errorf("latest pph21 ter: %w", err)
	}
	if !latest.Valid {
		return []models.Pph21TerTarif{}, nil
	}
	val := latest.Time
	var rows []models.Pph21TerTarif
	if err := s.DB.WithContext(ctx).
		Where(`"berlakuMulai" = ?`, val).
		Order(`"kategori" ASC, "lapisan" ASC`).
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list ter on %s: %w", val.Format("2006-01-02"), err)
	}
	return rows, nil
}

// UpsertSlip menulis slip dan komponennya dalam satu transaksi. Slip DRAF
// yang sudah ada untuk (userId, periode) digantikan; slip TERBIT/TERBAYAR
// ditolak dengan domain.ErrConflict. unitID adalah unit cakupan aksi untuk
// jejak audit; nil berarti unit tidak diketahui. auditNote menambahkan
// catatan pada jejak audit (mis. penyesuaian sakit berkepanjangan); nil
// berarti tanpa catatan.
func (s GormStore) UpsertSlip(ctx context.Context, actorID string, slip models.Gaji, tunjangan []models.GajiTunjangan, potongan []models.GajiPotongan, unitID *string, auditNote *string) (models.Gaji, error) {
	if slip.UserId == "" || slip.Periode.IsZero() {
		return models.Gaji{}, domain.ErrValidation
	}

	var existing models.Gaji
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where(`"userId" = ? AND "periode" = ?`, slip.UserId, slip.Periode).
			First(&existing).Error; err == nil {
			switch existing.Status {
			case models.StatusSlipTerbit, models.StatusSlipTerbayar, models.StatusSlipBatal:
				return domain.ErrConflict
			}
			if err := tx.Where(`"gajiId" = ?`, existing.ID).Delete(&models.GajiTunjangan{}).Error; err != nil {
				return fmt.Errorf("delete tunjangan lama: %w", err)
			}
			if err := tx.Where(`"gajiId" = ?`, existing.ID).Delete(&models.GajiPotongan{}).Error; err != nil {
				return fmt.Errorf("delete potongan lama: %w", err)
			}
			slip.ID = existing.ID
			slip.NomorSlip = existing.NomorSlip
			slip.KodeVerifikasi = existing.KodeVerifikasi
			slip.CreatedAt = existing.CreatedAt
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("read existing slip: %w", err)
		}

		if slip.ID == "" {
			var count int64
			if err := tx.Model(&models.Gaji{}).Where(`"periode" = ?`, slip.Periode).Count(&count).Error; err != nil {
				return fmt.Errorf("count slip periode: %w", err)
			}
			slip.NomorSlip = fmt.Sprintf("SL-%s-%04d", slip.Periode.Format("2006-01"), count+1)
			token, err := domain.NewSlipVerificationToken()
			if err != nil {
				return fmt.Errorf("generate verification token: %w", err)
			}
			slip.KodeVerifikasi = &token
		}

		if err := tx.Save(&slip).Error; err != nil {
			if isUniqueViolation(err) {
				return domain.ErrConflict
			}
			return fmt.Errorf("save slip: %w", err)
		}

		for i := range tunjangan {
			tunjangan[i].GajiId = slip.ID
		}
		for i := range potongan {
			potongan[i].GajiId = slip.ID
		}
		if len(tunjangan) > 0 {
			if err := tx.Create(&tunjangan).Error; err != nil {
				return fmt.Errorf("insert tunjangan: %w", err)
			}
		}
		if len(potongan) > 0 {
			if err := tx.Create(&potongan).Error; err != nil {
				return fmt.Errorf("insert potongan: %w", err)
			}
		}

		targetID := slip.ID
		return audit.Write(ctx, tx, audit.Entry{
			ActorID:  actorID,
			Action:   audit.ActionSlipGenerated,
			TargetID: &targetID,
			UnitID:   unitID,
			Note:     auditNote,
		})
	})
	return slip, err
}

// PeriodeSakitFor mencari baris PeriodeSakit yang mencakup periode slip
// (bulan berjalan). Mengembalikan persen berjalan, bulan ke, dan true bila
// ditemukan.
func (s GormStore) PeriodeSakitFor(ctx context.Context, userID string, period time.Time) (float64, int, bool, error) {
	var row models.PeriodeSakit
	moon := time.Date(period.Year(), period.Month(), 1, 0, 0, 0, 0, time.UTC)
	err := s.DB.WithContext(ctx).
		Where(`"userId" = ? AND date_trunc('month', "tanggalMulai") = ?`, userID, moon).
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 100, 0, false, nil
		}
		return 0, 0, false, fmt.Errorf("read periode sakit: %w", err)
	}
	return row.PersenBerjalan, row.BulanKe, true, nil
}

func isUniqueViolation(err error) bool {
	var pgErr interface{ GetSQLState() string }
	if errors.As(err, &pgErr) {
		return pgErr.GetSQLState() == "23505"
	}
	msg := err.Error()
	return strings.Contains(msg, "23505") || strings.Contains(msg, "duplicate key")
}
