package models

import (
	"time"
)

// PeriodeAkademik adalah periode akademik untuk kegiatan tridharma dosen.
type PeriodeAkademik struct {
	BaseModel
	Tahun          int                   `gorm:"uniqueIndex:periode_akademik_tahun_semester_idx"`
	Semester       int                   `gorm:"uniqueIndex:periode_akademik_tahun_semester_idx"`
	TanggalMulai   time.Time             `gorm:"type:date"`
	TanggalSelesai time.Time             `gorm:"type:date"`
	Status         StatusPeriodeAkademik `gorm:"type:StatusPeriodeAkademik;default:'DRAFT'"`
	Aktif          bool                  `gorm:"default:true"`
}

// EvaluasiBKD adalah evaluasi BKD dosen untuk satu periode akademik.
type EvaluasiBKD struct {
	BaseModel
	UserId            string `gorm:"type:uuid;index:evaluasi_bkd_user_periode_idx"`
	PeriodeAkademikId string `gorm:"type:uuid;index:evaluasi_bkd_user_periode_idx"`
	Nilai             float64
	Predikat          string
	Catatan           *string
	DinilaiOleh       *string   `gorm:"type:uuid"`
	TanggalEvaluasi   time.Time `gorm:"type:date"`
}

// TridharmaPendidikan adalah kegiatan tridharma pendidikan dosen.
type TridharmaPendidikan struct {
	BaseModel
	UserId            string `gorm:"type:uuid;index:tridharma_pendidikan_user_periode_idx"`
	PeriodeAkademikId string `gorm:"type:uuid;index:tridharma_pendidikan_user_periode_idx"`
	Uraian            string
	Target            float64
	Realisasi         float64             `gorm:"default:0"`
	Bobot             float64             `gorm:"default:1"`
	Status            StatusTridharmaItem `gorm:"type:StatusTridharmaItem;default:'DRAFT'"`
	DinilaiOleh       *string             `gorm:"type:uuid"`
}

// TridharmaPenelitian adalah kegiatan tridharma penelitian dosen.
type TridharmaPenelitian struct {
	BaseModel
	UserId            string `gorm:"type:uuid;index:tridharma_penelitian_user_periode_idx"`
	PeriodeAkademikId string `gorm:"type:uuid;index:tridharma_penelitian_user_periode_idx"`
	Judul             string
	JenisPenelitian   string
	Target            float64
	Realisasi         float64             `gorm:"default:0"`
	Bobot             float64             `gorm:"default:1"`
	Status            StatusTridharmaItem `gorm:"type:StatusTridharmaItem;default:'DRAFT'"`
	DinilaiOleh       *string             `gorm:"type:uuid"`
}

// TridharmaPengabdian adalah kegiatan tridharma pengabdian dosen.
type TridharmaPengabdian struct {
	BaseModel
	UserId            string `gorm:"type:uuid;index:tridharma_pengabdian_user_periode_idx"`
	PeriodeAkademikId string `gorm:"type:uuid;index:tridharma_pengabdian_user_periode_idx"`
	Judul             string
	JenisPengabdian   string
	Target            float64
	Realisasi         float64             `gorm:"default:0"`
	Bobot             float64             `gorm:"default:1"`
	Status            StatusTridharmaItem `gorm:"type:StatusTridharmaItem;default:'DRAFT'"`
	DinilaiOleh       *string             `gorm:"type:uuid"`
}

// TridharmaPenunjang adalah kegiatan tridharma penunjang dosen.
type TridharmaPenunjang struct {
	BaseModel
	UserId            string `gorm:"type:uuid;index:tridharma_penunjang_user_periode_idx"`
	PeriodeAkademikId string `gorm:"type:uuid;index:tridharma_penunjang_user_periode_idx"`
	Uraian            string
	Target            float64
	Realisasi         float64             `gorm:"default:0"`
	Bobot             float64             `gorm:"default:1"`
	Status            StatusTridharmaItem `gorm:"type:StatusTridharmaItem;default:'DRAFT'"`
	DinilaiOleh       *string             `gorm:"type:uuid"`
}

// StatusPeriodeAkademik adalah status periode akademik.
type StatusPeriodeAkademik string

const (
	StatusPeriodeAkademikDraft  StatusPeriodeAkademik = "DRAFT"
	StatusPeriodeAkademikActive StatusPeriodeAkademik = "AKTIF"
	StatusPeriodeAkademikClosed StatusPeriodeAkademik = "TUTUP"
)

// StatusTridharmaItem adalah status item tridharma.
type StatusTridharmaItem string

const (
	StatusTridharmaItemDraft     StatusTridharmaItem = "DRAFT"
	StatusTridharmaItemSubmitted StatusTridharmaItem = "DIAJUKAN"
	StatusTridharmaItemApproved  StatusTridharmaItem = "DISETUJUI"
	StatusTridharmaItemRejected  StatusTridharmaItem = "DITOLAK"
)
