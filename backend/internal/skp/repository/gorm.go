package repository

import (
	"gorm.io/gorm"
)

// GormStore adalah adapter persistensi SKP berbasis GORM.
type GormStore struct {
	DB *gorm.DB
}

var (
	sortColumnsSKPPeriode = map[string]string{
		"tahun":     `"tahun"`,
		"semester":  `"semester"`,
		"status":    `"status"`,
		"createdAt": `"createdAt"`,
	}
	sortColumnsSKPTarget = map[string]string{
		"uraian":    `"uraian"`,
		"target":    `"target"`,
		"realisasi": `"realisasi"`,
		"bobot":     `"bobot"`,
		"status":    `"status"`,
		"createdAt": `"createdAt"`,
	}
	sortColumnsLogAktivitas = map[string]string{
		"tanggal":     `"tanggal"`,
		"aktivitas":   `"aktivitas"`,
		"durasiMenit": `"durasiMenit"`,
		"createdAt":   `"createdAt"`,
	}
	sortColumnsPenilaian = map[string]string{
		"nilaiAkhir":       `"nilaiAkhir"`,
		"predikat":         `"predikat"`,
		"tanggalPenilaian": `"tanggalPenilaian"`,
		"createdAt":        `"createdAt"`,
	}
)
