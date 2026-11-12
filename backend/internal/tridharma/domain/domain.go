// Package domain memuat aturan bisnis Tridharma & BKD: error, allowlist
// kolom sort, dan tipe nilai yang dipakai seluruh modul Tridharma & BKD.
//
// Lapisan ini tidak melakukan akses database maupun HTTP.
package domain

import "errors"

var (
	// ErrNotFound menandai record Tridharma/BKD yang diminta tidak ada.
	ErrNotFound = errors.New("tridharma record not found")
	// ErrValidation menandai nilai yang melanggar aturan Tridharma/BKD.
	ErrValidation = errors.New("tridharma value is not valid")
	// ErrConflict menandai record yang bentrok dengan data yang sudah ada.
	ErrConflict = errors.New("tridharma record conflicts with existing data")
	// ErrForbidden menandai aksi yang di luar cakupan atau permission principal.
	ErrForbidden = errors.New("action is not permitted for this principal")
)

// Sort allowlist untuk daftar periode akademik.
var PeriodeAkademikSortColumns = map[string]string{
	"tahun":     `"tahun"`,
	"semester":  `"semester"`,
	"status":    `"status"`,
	"createdAt": `"createdAt"`,
}

// PeriodeAkademikSortFields mengembalikan kunci sort yang valid untuk daftar periode akademik.
func PeriodeAkademikSortFields() []string {
	return []string{"tahun", "semester", "status", "createdAt"}
}

// Sort allowlist untuk daftar evaluasi BKD.
var EvaluasiBKDSortColumns = map[string]string{
	"nilai":           `"nilai"`,
	"predikat":        `"predikat"`,
	"tanggalEvaluasi": `"tanggalEvaluasi"`,
	"createdAt":       `"createdAt"`,
}

// EvaluasiBKDSortFields mengembalikan kunci sort yang valid untuk daftar evaluasi BKD.
func EvaluasiBKDSortFields() []string {
	return []string{"tanggalEvaluasi", "nilai", "createdAt"}
}

// Sort allowlist untuk daftar tridharma pendidikan.
var TridharmaPendidikanSortColumns = map[string]string{
	"uraian":    `"uraian"`,
	"target":    `"target"`,
	"realisasi": `"realisasi"`,
	"bobot":     `"bobot"`,
	"status":    `"status"`,
	"createdAt": `"createdAt"`,
}

// TridharmaPendidikanSortFields mengembalikan kunci sort yang valid.
func TridharmaPendidikanSortFields() []string {
	return []string{"createdAt", "uraian", "status"}
}

// Sort allowlist untuk daftar tridharma penelitian.
var TridharmaPenelitianSortColumns = map[string]string{
	"judul":           `"judul"`,
	"jenisPenelitian": `"jenisPenelitian"`,
	"target":          `"target"`,
	"realisasi":       `"realisasi"`,
	"bobot":           `"bobot"`,
	"status":          `"status"`,
	"createdAt":       `"createdAt"`,
}

// TridharmaPenelitianSortFields mengembalikan kunci sort yang valid.
func TridharmaPenelitianSortFields() []string {
	return []string{"createdAt", "judul", "status"}
}

// Sort allowlist untuk daftar tridharma pengabdian.
var TridharmaPengabdianSortColumns = map[string]string{
	"judul":           `"judul"`,
	"jenisPengabdian": `"jenisPengabdian"`,
	"target":          `"target"`,
	"realisasi":       `"realisasi"`,
	"bobot":           `"bobot"`,
	"status":          `"status"`,
	"createdAt":       `"createdAt"`,
}

// TridharmaPengabdianSortFields mengembalikan kunci sort yang valid.
func TridharmaPengabdianSortFields() []string {
	return []string{"createdAt", "judul", "status"}
}

// Sort allowlist untuk daftar tridharma penunjang.
var TridharmaPenunjangSortColumns = map[string]string{
	"uraian":    `"uraian"`,
	"target":    `"target"`,
	"realisasi": `"realisasi"`,
	"bobot":     `"bobot"`,
	"status":    `"status"`,
	"createdAt": `"createdAt"`,
}

// TridharmaPenunjangSortFields mengembalikan kunci sort yang valid.
func TridharmaPenunjangSortFields() []string {
	return []string{"createdAt", "uraian", "status"}
}
