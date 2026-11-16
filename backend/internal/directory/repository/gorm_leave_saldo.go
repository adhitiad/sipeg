// Helper saldo cuti tahunan (lazy carry, hangus 30 Juni), pemeriksaan masa
// kerja, dan pencatatan periode sakit berkepanjangan untuk persetujuan cuti.
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"simpeg/backend/internal/directory/domain"
	"simpeg/backend/internal/models"
)

// ensureSaldoCutiTx memastikan baris SaldoCuti (pegawai, tahun) tersedia,
// menghitung carry dari tahun sebelumnya (maksimal 6 hari), dan menggantung
// sisa carry setelah 30 Juni tahun berjalan. Sebaiknya dipanggil di dalam
// transaksi supaya pembaruan carry/hangus atomik dengan aksi yang memakai.
func ensureSaldoCutiTx(ctx context.Context, tx *gorm.DB, userID string, tahun int) (models.SaldoCuti, error) {
	var row models.SaldoCuti
	err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where(`"userId" = ? AND "tahun" = ?`, userID, tahun).
		First(&row).Error
	switch {
	case err == nil:
	case errors.Is(err, gorm.ErrRecordNotFound):
		row = models.SaldoCuti{
			UserId:       userID,
			Tahun:        tahun,
			KuotaTahunan: 12,
			Terpakai:     0,
			SisaCarry:    0,
		}
		var prev models.SaldoCuti
		prevErr := tx.WithContext(ctx).
			Where(`"userId" = ? AND "tahun" = ?`, userID, tahun-1).
			First(&prev).Error
		if prevErr == nil {
			carry := prev.KuotaTahunan - prev.Terpakai
			if carry > domain.CarryCutiTahunanMaks {
				carry = domain.CarryCutiTahunanMaks
			}
			if carry < 0 {
				carry = 0
			}
			row.SisaCarry = carry
		} else if !errors.Is(prevErr, gorm.ErrRecordNotFound) {
			return models.SaldoCuti{}, fmt.Errorf("read saldo cuti tahun sebelumnya: %w", prevErr)
		}
		if mkErr := tx.WithContext(ctx).Create(&row).Error; mkErr != nil {
			return models.SaldoCuti{}, fmt.Errorf("create saldo cuti: %w", mkErr)
		}
	default:
		return models.SaldoCuti{}, fmt.Errorf("read saldo cuti: %w", err)
	}

	now := time.Now()
	hangusAt := time.Date(tahun, domain.SisaCarryHangusBulan, domain.SisaCarryHangusHari+1, 0, 0, 0, 0, time.UTC)
	if !row.Hangus && !now.Before(hangusAt) {
		if upd := tx.WithContext(ctx).Model(&row).Updates(map[string]any{"hangus": true}).Error; upd != nil {
			return models.SaldoCuti{}, fmt.Errorf("hangus saldo cuti: %w", upd)
		}
		row.Hangus = true
	}
	return row, nil
}

// consumeSaldoCutiTx memastikan ketersediaan dan mengurangi saldo cuti
// tahunan sebanyak hari yang dipotong. Gagal (domain.ErrValidation) bila
// sisa tidak mencukupi.
func consumeSaldoCutiTx(ctx context.Context, tx *gorm.DB, userID string, tahun, hari int) error {
	if hari <= 0 {
		return domain.ErrValidation
	}
	row, err := ensureSaldoCutiTx(ctx, tx, userID, tahun)
	if err != nil {
		return err
	}
	if domain.SisaCuti(row) < hari {
		return domain.ErrValidation
	}
	if upd := tx.WithContext(ctx).Model(&row).Updates(map[string]any{"terpakai": row.Terpakai + hari}).Error; upd != nil {
		return fmt.Errorf("consume saldo cuti: %w", upd)
	}
	return nil
}

// SaldoCuti memastikan baris saldo pegawai untuk tahun tertentu tersedia
// (carry & hangus diperlakukan lazy) dan mengembalikan keadaan terakhirnya.
func (s GormStore) SaldoCuti(ctx context.Context, userID string, tahun int) (models.SaldoCuti, error) {
	var result models.SaldoCuti
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		result, err = ensureSaldoCutiTx(ctx, tx, userID, tahun)
		return err
	})
	return result, err
}

// EmployeeStartDate mengembalikan tanggal mulai kerja pegawai; nil bila belum
// terisi (syarat masa kerja dilewati, dicatat sebagai asumsi kerja).
func (s GormStore) EmployeeStartDate(ctx context.Context, userID string) (*time.Time, error) {
	var row struct {
		TanggalMasuk *time.Time `gorm:"column:tanggalMasuk"`
	}
	if err := s.DB.WithContext(ctx).Table(`"user"`).
		Select(`"tanggalMasuk"`).
		Where(`"id" = ?`, userID).
		Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("read tanggal masuk pegawai: %w", err)
	}
	return row.TanggalMasuk, nil
}

// EmployeeKategoriStatusKode mengembalikan kode kategori status pegawai
// (tabel kategoriStatusPegawai, migrasi 000025) bila tersedia. Mengembalikan
// string kosong bila pegawai belum ditetapkan kategorinya atau migrasi belum
// diterapkan (best-effort, tidak memblokir pengajuan cuti) — DEC-092.
func (s GormStore) EmployeeKategoriStatusKode(ctx context.Context, userID string) (string, error) {
	var kode string
	if err := s.DB.WithContext(ctx).Raw(
		`SELECT COALESCE(k."kode", '') FROM "user" u LEFT JOIN "kategoriStatusPegawai" k ON k."id" = u."kategoriStatusId" WHERE u."id" = ?`,
		userID,
	).Scan(&kode).Error; err != nil {
		return "", nil
	}
	return kode, nil
}

// overlapCutiCount menghitung pengajuan cuti MENUNGGU/DISETUJUI yang
// tumpang tindih dengan [mulai, selesai] untuk seorang pegawai.
// Pengajuan yang ditolak tidak dihitung; excludeID dipakai saat memeriksa
// pengajuan terhadap dirinya sendiri.
func overlapCutiCount(ctx context.Context, dao *gorm.DB, userID string, mulai, selesai time.Time, excludeID string) (int64, error) {
	query := dao.WithContext(ctx).Model(&models.Cuti{}).
		Where(`"userId" = ? AND "status" IN (?, ?) AND "tanggalMulai" <= ? AND "tanggalSelesai" >= ?`,
			userID, models.StatusMenunggu, models.StatusDisetujui, selesai, mulai)
	if excludeID != "" {
		query = query.Where(`"id" <> ?`, excludeID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count overlapping cuti: %w", err)
	}
	return count, nil
}

// OverlappingCuti melaporkan jumlah pengajuan cuti yang tumpang tindih
// dengan periode pegawai (tanpa transaksi). Dipakai lapisan aplikasi
// untuk penolakan awal; penegakan atomik tetap dilakukan saat approve.
func (s GormStore) OverlappingCuti(ctx context.Context, userID string, mulai, selesai time.Time, excludeID string) (int64, error) {
	return overlapCutiCount(ctx, s.DB, userID, mulai, selesai, excludeID)
}

// releaseSaldoCutiTx mengembalikan saldo cuti tahunan sebanyak hari yang
// terpotong saat persetujuan; tidak pernah negatif. Dipanggil di dalam
// transaksi pembatalan.
func releaseSaldoCutiTx(ctx context.Context, tx *gorm.DB, userID string, tahun, hari int) error {
	if hari <= 0 {
		return nil
	}
	row, err := ensureSaldoCutiTx(ctx, tx, userID, tahun)
	if err != nil {
		return err
	}
	terpakai := row.Terpakai - hari
	if terpakai < 0 {
		terpakai = 0
	}
	return tx.WithContext(ctx).Model(&row).Updates(map[string]any{"terpakai": terpakai}).Error
}

// AdvancePeriodeSakitTx mencatat/melanjutkan PeriodeSakit saat cuti sakit
// berkepanjangan disetujui. Bulan ke-n dihitung sejak bulan mulai cuti
// pertama;
// persen mengikuti tier 100/75/50/25 (bulan 1–4/5–8/9–12/13–16), sesudahnya 0.
// Baris untuk bulan yang sudah ada tidak dibuat ulang (idempoten).
func advancePeriodeSakitTx(ctx context.Context, tx *gorm.DB, userID string, mulai time.Time) (string, error) {
	bullet := time.Date(mulai.Year(), mulai.Month(), 1, 0, 0, 0, 0, time.UTC)

	var earliest models.PeriodeSakit
	err := tx.WithContext(ctx).
		Where(`"userId" = ?`, userID).
		Order(`"tanggalMulai" ASC`).
		Take(&earliest).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", fmt.Errorf("read periode sakit awal: %w", err)
	}
	bulanKe := 1
	if err == nil {
		first := time.Date(earliest.TanggalMulai.Year(), earliest.TanggalMulai.Month(), 1, 0, 0, 0, 0, time.UTC)
		monthsBetween := (bullet.Year()-first.Year())*12 + int(bullet.Month()) - int(first.Month())
		if monthsBetween < 0 {
			monthsBetween = 0
		}
		bulanKe = earliest.BulanKe + monthsBetween
	}
	persen := domain.PersenSakitBerkepanjangan(bulanKe)

	var count int64
	if err := tx.WithContext(ctx).Model(&models.PeriodeSakit{}).
		Where(`"userId" = ? AND date_trunc('month', "tanggalMulai") = ?`, userID, bullet).
		Count(&count).Error; err != nil {
		return "", fmt.Errorf("count periode sakit bulan ini: %w", err)
	}
	if count == 0 {
		row := models.PeriodeSakit{
			UserId:         userID,
			TanggalMulai:   bullet,
			BulanKe:        bulanKe,
			PersenBerjalan: persen,
		}
		if err := tx.WithContext(ctx).Create(&row).Error; err != nil {
			return "", fmt.Errorf("create periode sakit: %w", err)
		}
	}
	return fmt.Sprintf("sakit berkepanjangan bulan ke-%d (%.0f%% gaji pokok)", bulanKe, persen), nil
}
