// Package domain memuat aturan THR (`DEC-070`/`DEC-083`/`DEC-085`/`DEC-086`):
// validasi konfigurasi per tahun dan perhitungan nominal. Sumber nominal
// `SATU_GAJI_POKOK` dan `GAJI_KOTOR` hanya memakai slip TERBIT/TERBAYAR terakhir
// pegawai (mengikuti komentar pada migrasi 000025_payroll_core.sql, §THR);
// `PERSEN` dikalikan dari gaji pokok slip TERBIT/TERBAYAR terakhir;
// `PERSEN_TAKE_HOME` (`DEC-086`) dikalikan dari take-home pay (gaji bersih)
// slip TERBIT/TERBAYAR terakhir.
package domain

import (
	"errors"
	"time"
)

// Besaran THR yang dikenal.
const (
	BesaranSatuGajiPokok  = "SATU_GAJI_POKOK"
	BesaranGajiKotor      = "GAJI_KOTOR"
	BesaranPersen         = "PERSEN"
	BesaranPersenTakeHome = "PERSEN_TAKE_HOME"
)

// Basis berisi angka-angka slip sumber yang dipakai untuk menghitung
// nominal THR. `GajiBersih` = total gaji bruto dikurangi seluruh potongan
// (take-home pay).
type Basis struct {
	GajiPokok  float64
	TotalGaji  float64
	GajiBersih float64
}

// ErrValidation menandai isian yang melanggar aturan THR.
var ErrValidation = errors.New("isian THR tidak sah")

// ErrForbidden menandai penolakan karena permission.
var ErrForbidden = errors.New("tidak berwenang mengelola THR")

// ErrConfigNotFound menandai konfigurasi THR tahun yang tidak ada.
var ErrConfigNotFound = errors.New("konfigurasi THR untuk tahun tersebut belum ada")

// ErrSlipConflict menandai slip THR yang tidak lagi dapat dibuat (sudah
// TERBIT/TERBAYAR/BATAL, atau duplikat).
var ErrSlipConflict = errors.New("slip THR untuk (pegawai, tanggal) ini sudah ada")

// BesaranList mengembalikan seluruh besaran yang dikenal.
func BesaranList() []string {
	return []string{BesaranSatuGajiPokok, BesaranGajiKotor, BesaranPersen, BesaranPersenTakeHome}
}

// ValidBesaran melaporkan apakah besaran dikenal.
func ValidBesaran(besaran string) bool {
	for _, candidate := range BesaranList() {
		if candidate == besaran {
			return true
		}
	}
	return false
}

// ValidateConfig memeriksa konfigurasi THR.
func ValidateConfig(tahun int, besaran string, persen *float64, syaratMasaKerjaBulan int, berlakuTanggal time.Time) error {
	if tahun < 2026 {
		return ErrValidation
	}
	if syaratMasaKerjaBulan < 0 || syaratMasaKerjaBulan > 60 {
		return ErrValidation
	}
	switch besaran {
	case BesaranSatuGajiPokok, BesaranGajiKotor:
		if persen != nil {
			return ErrValidation
		}
	case BesaranPersen:
		if persen == nil || *persen < 0 || *persen > 400 {
			return ErrValidation
		}
	case BesaranPersenTakeHome:
		if persen == nil || *persen < 0 || *persen > 400 {
			return ErrValidation
		}
	default:
		return ErrValidation
	}
	if berlakuTanggal.IsZero() {
		return ErrValidation
	}
	return nil
}

// Nominal menghitung nominal THR dari konfigurasi dan slip TERBIT/TERBAYAR terakhir.
func Nominal(besaran string, persen *float64, basis Basis) (float64, error) {
	switch besaran {
	case BesaranSatuGajiPokok:
		return basis.GajiPokok, nil
	case BesaranGajiKotor:
		return basis.TotalGaji, nil
	case BesaranPersen:
		if persen == nil {
			return 0, ErrValidation
		}
		return basis.GajiPokok * (*persen) / 100, nil
	case BesaranPersenTakeHome:
		if persen == nil {
			return 0, ErrValidation
		}
		return basis.GajiBersih * (*persen) / 100, nil
	default:
		return 0, ErrValidation
	}
}

// CutoffFor mengembalikan batas awal masa kerja agar pegawai berhak THR,
// dihitung dari tanggal berlaku dikurangi syarat masa kerja (bulan).
func CutoffFor(berlakuTanggal time.Time, syaratMasaKerjaBulan int) time.Time {
	return berlakuTanggal.AddDate(0, -syaratMasaKerjaBulan, 0)
}
