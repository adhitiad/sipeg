// Aturan baca Surat Keputusan (SK): filter,allowlist pengurutan, dan
// penanganan data SK yang sudah terbit (mis. SK-PHK otomatis saat approve
// pemberhentian).
package domain

import (
	"errors"
	"strings"

	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
)

// SortColumns adalah allowlist kolom pengurutan SK. Kolom SQL hanya boleh
// berasal dari peta ini.
var SortColumns = map[string]string{
	"tanggalSk": `"tanggalSk"`,
	"nomorSk":   `"nomorSk"`,
	"jenis":     `"jenis"`,
	"penerbit":  `"penerbit"`,
	"createdAt": `"createdAt"`,
}

// FallbackSort adalah kolom pengurutan bawaan (terbaru lebih dulu).
const FallbackSort = "tanggalSk"

// Filter adalah parameter daftar SK: parameter umum (pencarian, pagination,
// pengurutan) ditambah penyaring khusus SK.
type Filter struct {
	listing.Options
	// Jenis menyaring jenis SK (PENGANGKATAN, MUTASI, PHK,
	// JABATAN_STRUKTURAL, PENUGASAN).
	Jenis string
	// Penerbit menyaring penerbit SK (YAYASAN, REKTOR).
	Penerbit string
}

// Row adalah satu SK beserta nama pegawai dan unit penugasannya.
type Row struct {
	models.SuratKeputusan
	Nama       string  `gorm:"column:nama"`
	HomebaseID *string `gorm:"column:homebaseId"`
}

var (
	// ErrForbidden menandai principal tanpa izin baca SK.
	ErrForbidden = errors.New("akses ditolak")
	// ErrValidation menandai parameter yang tidak sah.
	ErrValidation = errors.New("parameter tidak sah")
	// ErrNotFound menandai SK yang tidak ditemukan (atau di luar cakupan).
	ErrNotFound = errors.New("surat keputusan tidak ditemukan")
)

// Normalize_membersihkan nilai filter; jenis/penerbit dinormalkan ke huruf
// besar dan pencarian dipangkas.
func (f Filter) Normalize() Filter {
	f.Search = strings.TrimSpace(f.Search)
	f.Jenis = strings.ToUpper(strings.TrimSpace(f.Jenis))
	f.Penerbit = strings.ToUpper(strings.TrimSpace(f.Penerbit))
	f.UserID = strings.TrimSpace(f.UserID)
	return f
}

// JenisTerdaftar memeriksa apakah jenis SK dikenal (enum models.JenisSk).
func JenisTerdaftar(jenis string) bool {
	switch models.JenisSk(jenis) {
	case models.JenisSkPengangkatan, models.JenisSkMutasi, models.JenisSkPhk,
		models.JenisSkJabatanStruktural, models.JenisSkPenugasan:
		return true
	default:
		return false
	}
}

// PenerbitTerdaftar memeriksa apakah penerbit SK dikenal (enum
// models.TingkatSk).
func PenerbitTerdaftar(penerbit string) bool {
	switch models.TingkatSk(penerbit) {
	case models.TingkatSkYayasan, models.TingkatSkRektor:
		return true
	default:
		return false
	}
}
