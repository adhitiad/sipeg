package repository

import (
	"gorm.io/gorm"
)

// GormStore adalah adapter persistensi Tridharma & BKD berbasis GORM.
type GormStore struct {
	DB *gorm.DB
}

// Kolom sortir yang diizinkan untuk tiap resource. Nilai diambil dari
// allowlist, bukan dari input mentah pengguna.
var (
	sortColumnsPeriodeAkademik = map[string]string{
		"tahun":     `"tahun"`,
		"semester":  `"semester"`,
		"status":    `"status"`,
		"createdAt": `"createdAt"`,
	}
	sortColumnsEvaluasiBKD = map[string]string{
		"tahun":     `"tahun"`,
		"semester":  `"semester"`,
		"status":    `"status"`,
		"createdAt": `"createdAt"`,
	}
	sortColumnsTridharmaPendidikan = map[string]string{
		"uraian":    `"uraian"`,
		"target":    `"target"`,
		"realisasi": `"realisasi"`,
		"bobot":     `"bobot"`,
		"status":    `"status"`,
		"createdAt": `"createdAt"`,
	}
	sortColumnsTridharmaPenelitian = map[string]string{
		"uraian":    `"uraian"`,
		"target":    `"target"`,
		"realisasi": `"realisasi"`,
		"bobot":     `"bobot"`,
		"status":    `"status"`,
		"createdAt": `"createdAt"`,
	}
	sortColumnsTridharmaPengabdian = map[string]string{
		"uraian":    `"uraian"`,
		"target":    `"target"`,
		"realisasi": `"realisasi"`,
		"bobot":     `"bobot"`,
		"status":    `"status"`,
		"createdAt": `"createdAt"`,
	}
	sortColumnsTridharmaPenunjang = map[string]string{
		"uraian":    `"uraian"`,
		"target":    `"target"`,
		"realisasi": `"realisasi"`,
		"bobot":     `"bobot"`,
		"status":    `"status"`,
		"createdAt": `"createdAt"`,
	}
)
