package models

import (
	"time"
)

// SKPPeriode adalah periode penilaian kinerja (tahun dan semester).
type SKPPeriode struct {
	BaseModel
	Tahun       int       `gorm:"uniqueIndex:skp_periode_tahun_semester_idx"`
	Semester    int       `gorm:"uniqueIndex:skp_periode_tahun_semester_idx"`
	TanggalMulai time.Time `gorm:"type:date"`
	TanggalSelesai time.Time `gorm:"type:date"`
	Status      StatusSKP `gorm:"type:StatusSKP;default:'DRAFT'"`
	Aktif       bool      `gorm:"default:true"`
}

// SKPTargetKinerja adalah target kinerja pegawai untuk satu periode SKP.
type SKPTargetKinerja struct {
	BaseModel
	UserId        string  `gorm:"type:uuid;index:skp_target_user_periode_idx"`
	SKPPeriodeId  string  `gorm:"type:uuid;index:skp_target_user_periode_idx"`
	Uraian        string
	Target        float64
	Realisasi     float64 `gorm:"default:0"`
	Bobot         float64 `gorm:"default:1"`
	Status        StatusSKPItem `gorm:"type:StatusSKPItem;default:'DRAFT'"`
	DinilaiOleh   *string `gorm:"type:uuid"`
}

// LogAktivitasHarian adalah catatan aktivitas harian pegawai.
type LogAktivitasHarian struct {
	BaseModel
	UserId      string    `gorm:"type:uuid;index:log_aktivitas_user_tanggal_idx"`
	Tanggal     time.Time `gorm:"type:date;index:log_aktivitas_user_tanggal_idx"`
	Aktivitas   string
	DurasiMenit int
	Keterangan  *string
}

// PenilaianKinerjaAkhir adalah hasil penilaian kinerja akhir periode.
type PenilaianKinerjaAkhir struct {
	BaseModel
	UserId        string  `gorm:"type:uuid;uniqueIndex:penilaian_user_periode_idx"`
	SKPPeriodeId  string  `gorm:"type:uuid;uniqueIndex:penilaian_user_periode_idx"`
	NilaiAkhir    float64
	Predikat      string
	Catatan       *string
	DinilaiOleh   *string `gorm:"type:uuid"`
	TanggalPenilaian time.Time `gorm:"type:date"`
}

// StatusSKP adalah status periode SKP.
type StatusSKP string

const (
	StatusSKPDraft   StatusSKP = "DRAFT"
	StatusSKPActive  StatusSKP = "AKTIF"
	StatusSKPClosed  StatusSKP = "TUTUP"
)

// StatusSKPItem adalah status item target kinerja.
type StatusSKPItem string

const (
	StatusSKPItemDraft    StatusSKPItem = "DRAFT"
	StatusSKPItemSubmitted StatusSKPItem = "DIAJUKAN"
	StatusSKPItemApproved StatusSKPItem = "DISETUJUI"
	StatusSKPItemRejected StatusSKPItem = "DITOLAK"
)
