// Package domain memuat aturan pemberhentian pegawai (DEC-083): sub-alasan
// pengajuan pemberhentian, pemetaan ke jenis pemberhentian (HORMAT/TIDAK_HORMAT)
// beserta penghitung kompensasi.
package domain

import (
	"errors"
	"strings"

	"simpeg/backend/internal/models"
)

// Alasan pengajuan pemberhentian (uppercase, sesuai DB enum).
const (
	AlasanMeninggalDunia     = "MENINGGAL_DUNIA"
	AlasanSakitBerkelanjutan = "SAKIT_BERKEPANJANGAN"
	AlasanBup                = "BUP"
	AlasanMengundurDiri      = "MENGUNDUR_DIRI"
	AlasanPenyederhanaan     = "PENYEDERHANAAN_ORGANISASI"
	AlasanKesalahanBerat     = "KESALAHAN_BERAT"
)

// AlasanList mengembalikan seluruh sub-alasan yang dikenal.
func AlasanList() []string {
	return []string{
		AlasanMeninggalDunia,
		AlasanSakitBerkelanjutan,
		AlasanBup,
		AlasanMengundurDiri,
		AlasanPenyederhanaan,
		AlasanKesalahanBerat,
	}
}

// ErrInvalidAlasan menandai alasan yang tidak dikenal.
var ErrInvalidAlasan = errors.New("alasan pemberhentian tidak dikenal")

// ErrInvalidDate menandai tanggal efektif yang kosong/tidak sah.
var ErrInvalidDate = errors.New("tanggal efektif wajib diisi")

// ErrInvalidUserID menandai userId yang kosong.
var ErrInvalidUserID = errors.New("userId wajib diisi")

// ErrForbidden menandai penolakan karena permission/cakupan.
var ErrForbidden = errors.New("tidak berwenang memproses pemberhentian")

// ErrNotPending menandai keputusan pada pengajuan yang bukan MENUNGGU.
var ErrNotPending = errors.New("pemberhentian tidak menunggu persetujuan")

// ErrPending menandai pengajuan ganda untuk pegawai yang sama.
var ErrPending = errors.New("pengajuan pemberhentian untuk pegawai ini masih menunggu persetujuan")

// ErrNotFound menandai pegawai yang tidak dikenal.
var ErrNotFound = errors.New("pegawai tidak ditemukan")

// ValidAlasan melaporkan apakah alasan dikenal.
func ValidAlasan(alasan string) bool {
	normalized := NormalizeAlasan(alasan)
	for _, candidate := range AlasanList() {
		if candidate == normalized {
			return true
		}
	}
	return false
}

// NormalizeAlasan menormalkan alasan pengajuan (trim + uppercase).
func NormalizeAlasan(alasan string) string {
	return strings.ToUpper(strings.TrimSpace(alasan))
}

// Honoured melaporkan apakah alasan berhak kompensasi (HORMAT). Kesalahan
// berat satu-satunya alasan TIDAK_HORMAT.
func Honoured(alasan string) bool {
	switch NormalizeAlasan(alasan) {
	case AlasanMeninggalDunia,
		AlasanSakitBerkelanjutan,
		AlasanBup,
		AlasanMengundurDiri,
		AlasanPenyederhanaan:
		return true
	case AlasanKesalahanBerat:
		return false
	default:
		return false
	}
}

// JenisFromAlasan memetakan alasan pengajuan ke jenis pemberhentian yang
// disimpan pada tabel Pemberhentian.
func JenisFromAlasan(alasan string) (models.JenisPemberhentian, error) {
	switch NormalizeAlasan(alasan) {
	case AlasanBup:
		return models.PemberhentianBup, nil
	case AlasanMengundurDiri:
		return models.PemberhentianAtasPermintaan, nil
	case AlasanKesalahanBerat:
		return models.PemberhentianTidakHormat, nil
	case AlasanMeninggalDunia,
		AlasanSakitBerkelanjutan,
		AlasanPenyederhanaan:
		return models.PemberhentianHormat, nil
	default:
		return "", ErrInvalidAlasan
	}
}

// Compensation adalah rincian kompensasi pemberhentian: pesangon dari tabel
// admin (admin input manual) plus tambahan yayasan 1 bulan gaji pokok per
// 5 tahun masa kerja penuh (DEC-070).
type Compensation struct {
	PesangonBulan float64
	TambahanBulan float64
	TotalBulan    float64
	Tersedia      bool
}

// HitungCompensation menghitung kompensasi berdasarkan tabel aturan admin
// dan tambahan yayasan. `rules` memetakan tahun masa kerja ke pesangonBulan;
// nilai untuk masa kerja tertentu memakai baris terbesar yang tidak melebihi
// masa kerja aktual (floor). Bila tabel kosong, `Tersedia=false` dan
// totalBulan 0.
func HitungCompensation(rules map[int]float64, masaKerjaTahun int, honoured bool) Compensation {
	if !honoured {
		return Compensation{Tersedia: true}
	}
	if len(rules) == 0 {
		return Compensation{Tersedia: false}
	}
	bestKey := -1
	base := 0.0
	for key := range rules {
		if key <= masaKerjaTahun && key > bestKey {
			bestKey = key
			base = rules[key]
		}
	}
	tambahan := float64(masaKerjaTahun / 5) // floor
	return Compensation{
		PesangonBulan: base,
		TambahanBulan: tambahan,
		TotalBulan:    base + tambahan,
		Tersedia:      true,
	}
}
