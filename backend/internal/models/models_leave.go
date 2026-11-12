package models

import (
	"time"
)

type Cuti struct {
	BaseModel
	UserId                string `gorm:"type:uuid;index:cuti_user_tanggal_idx"`
	Jenis                 JenisCuti
	TanggalMulai          time.Time `gorm:"type:timestamp(3);index:cuti_user_tanggal_idx"`
	TanggalSelesai        time.Time `gorm:"type:timestamp(3)"`
	AlamatCuti            *string
	Alasan                *string
	Status                StatusPersetujuan `gorm:"index:cuti_status_idx"`
	DisetujuiBy           *string           `gorm:"type:uuid"`
	Catatan               *string
	SubJenis              *SubJenisIzinPenting
	NomorSurat            *string
	TanggalSuratDokter    *time.Time `gorm:"type:timestamp(3)"`
	JenisEviden           *JenisEviden
	FileEvidenUrl         *string
	WaktuEviden           *time.Time `gorm:"type:timestamp(3)"`
	CatatanEviden         *string
	PersenGaji            float64
	UntukHaji             bool
	TahapSaatIni          *TingkatTerusan
	TanggalPerkiraanLahir *time.Time `gorm:"type:timestamp(3)"`
	TanggalLahirAktual    *time.Time `gorm:"type:timestamp(3)"`
	Terpilih              bool
	MasaJabatanMulai      *time.Time `gorm:"type:timestamp(3)"`
	MasaJabatanSelesai    *time.Time `gorm:"type:timestamp(3)"`
	CreatedBy             *string    `gorm:"type:uuid"`
	UpdatedBy             *string    `gorm:"type:uuid"`
}

type TerusanCuti struct {
	// Skema acuan menyimpan createdAt tanpa updatedAt untuk model ini.
	IDModel
	CutiId     string `gorm:"type:uuid;index:terusan_cuti_idx"`
	DariUserId string `gorm:"type:uuid"`
	Kepada     TingkatTerusan
	Catatan    *string
	CreatedAt  time.Time `gorm:"column:createdAt;type:timestamp(3)"`
}

type SaldoCuti struct {
	BaseModel
	UserId       string `gorm:"type:uuid;uniqueIndex:saldo_cuti_user_tahun_key;index:saldo_cuti_user_idx"`
	Tahun        int    `gorm:"uniqueIndex:saldo_cuti_user_tahun_key"`
	KuotaTahunan int
	Terpakai     int
	SisaCarry    int
	Hangus       bool
}

type PeriodeSakit struct {
	BaseModel
	UserId         string    `gorm:"type:uuid;index:periode_sakit_user_idx"`
	TanggalMulai   time.Time `gorm:"type:timestamp(3);index:periode_sakit_user_idx"`
	BulanKe        int
	PersenBerjalan float64
}
