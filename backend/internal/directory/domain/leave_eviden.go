package domain

import (
	"strings"

	"simpeg/backend/internal/models"
)

// JenisCutiWajibEviden menandai jenis cuti yang wajib menyertakan bukti.
//
// Cuti melahirkan dan cuti sakit (termasuk sakit berkepanjangan) wajib
// melampirkan surat dokter. Cuti harian diwakili jenis IZIN_PENTING dan wajib
// melampirkan bukti lain, misalnya foto bertimestamp.
var JenisCutiWajibEviden = map[models.JenisCuti]bool{
	models.JenisCutiBersalin:            true,
	models.JenisCutiSakit:               true,
	models.JenisCutiSakitBerkepanjangan: true,
	models.JenisCutiIzinPenting:         true,
}

// ValidasiEviden memeriksa kecukupan bukti pengajuan cuti.
//
// Pengajuan yang tidak wajib menyertakan bukti selalu lolos. Pengajuan yang
// wajib harus menyebut jenis bukti, menunjuk berkas bukti, dan menyertakan
// keterangan khusus jenis bukti tersebut. Bukti tanpa berkas yang dapat
// ditunjuk tidak diterima karena approver tidak dapat memverifikasinya.
func ValidasiEviden(item models.Cuti) error {
	if !JenisCutiWajibEviden[item.Jenis] {
		return nil
	}
	if item.JenisEviden == nil {
		return ErrValidation
	}
	switch *item.JenisEviden {
	case models.JenisEvidenSuratDokter:
		if !jenisCutiButuhSuratDokter(item.Jenis) {
			return ErrValidation
		}
		if !teksAda(item.FileEvidenUrl) || !teksAda(item.NomorSurat) || item.TanggalSuratDokter == nil {
			return ErrValidation
		}
	case models.JenisEvidenFoto:
		if jenisCutiButuhSuratDokter(item.Jenis) {
			return ErrValidation
		}
		if !teksAda(item.FileEvidenUrl) || item.WaktuEviden == nil {
			return ErrValidation
		}
	case models.JenisEvidenLainnya:
		if jenisCutiButuhSuratDokter(item.Jenis) {
			return ErrValidation
		}
		if item.WaktuEviden == nil || !teksAda(item.CatatanEviden) {
			return ErrValidation
		}
	default:
		return ErrValidation
	}
	return nil
}

// ValidJenisEviden melaporkan apakah jenis bukti dikenal.
func ValidJenisEviden(jenis models.JenisEviden) bool {
	switch jenis {
	case models.JenisEvidenSuratDokter, models.JenisEvidenFoto, models.JenisEvidenLainnya:
		return true
	}
	return false
}

// jenisCutiButuhSuratDokter melaporkan apakah jenis cuti hanya boleh
// dilampiri surat dokter. Cuti harian memakai bukti lain, bukan surat dokter.
func jenisCutiButuhSuratDokter(jenis models.JenisCuti) bool {
	switch jenis {
	case models.JenisCutiBersalin, models.JenisCutiSakit, models.JenisCutiSakitBerkepanjangan:
		return true
	}
	return false
}

// teksAda melaporkan apakah nilai opsional memuat teks setelah dipangkas.
func teksAda(value *string) bool {
	return value != nil && strings.TrimSpace(*value) != ""
}
