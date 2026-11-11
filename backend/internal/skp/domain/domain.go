// Package domain memuat aturan bisnis SKP & Log Aktivitas: error, allowlist
// kolom sort, dan tipe nilai yang dipakai seluruh modul SKP.
//
// Lapisan ini tidak melakukan akses database maupun HTTP.
package domain

import "errors"

var (
	// ErrNotFound menandai record SKP yang diminta tidak ada.
	ErrNotFound = errors.New("skp record not found")
	// ErrValidation menandai nilai yang melanggar aturan SKP.
	ErrValidation = errors.New("skp value is not valid")
	// ErrConflict menandai record yang bentrok dengan data yang sudah ada.
	ErrConflict = errors.New("skp record conflicts with existing data")
	// ErrForbidden menandai aksi yang di luar cakupan atau permission principal.
	ErrForbidden = errors.New("action is not permitted for this principal")
)

// Sort allowlist untuk daftar periode SKP.
var SKPPeriodeSortColumns = map[string]string{
	"tahun":       `"tahun"`,
	"semester":    `"semester"`,
	"status":      `"status"`,
	"createdAt":   `"createdAt"`,
}

// SKPPeriodeSortFields mengembalikan kunci sort yang valid untuk daftar periode SKP.
func SKPPeriodeSortFields() []string {
	return []string{"tahun", "semester", "status", "createdAt"}
}

// Sort allowlist untuk daftar target kinerja SKP.
var SKPTargetSortColumns = map[string]string{
	"uraian":     `"uraian"`,
	"target":     `"target"`,
	"realisasi":  `"realisasi"`,
	"bobot":      `"bobot"`,
	"status":     `"status"`,
	"createdAt":  `"createdAt"`,
}

// SKPTargetSortFields mengembalikan kunci sort yang valid untuk daftar target kinerja.
func SKPTargetSortFields() []string {
	return []string{"createdAt", "uraian", "status"}
}

// Sort allowlist untuk daftar log aktivitas harian.
var LogAktivitasSortColumns = map[string]string{
	"tanggal":     `"tanggal"`,
	"aktivitas":   `"aktivitas"`,
	"durasiMenit": `"durasiMenit"`,
	"createdAt":   `"createdAt"`,
}

// LogAktivitasSortFields mengembalikan kunci sort yang valid untuk daftar log aktivitas.
func LogAktivitasSortFields() []string {
	return []string{"tanggal", "aktivitas", "createdAt"}
}

// Sort allowlist untuk daftar penilaian kinerja akhir.
var PenilaianSortColumns = map[string]string{
	"nilaiAkhir":     `"nilaiAkhir"`,
	"predikat":       `"predikat"`,
	"tanggalPenilaian": `"tanggalPenilaian"`,
	"createdAt":      `"createdAt"`,
}

// PenilaianSortFields mengembalikan kunci sort yang valid untuk daftar penilaian akhir.
func PenilaianSortFields() []string {
	return []string{"tanggalPenilaian", "nilaiAkhir", "createdAt"}
}
