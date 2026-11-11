package models

import (
	"time"
)

type MutasiJabatan struct {
	BaseModel
	UserId              string `gorm:"type:uuid;index:mutasi_user_tanggal_idx"`
	Jenis               JenisMutasi
	JabatanLama         *string
	HomebaseLama        *string
	JabatanBaru         string
	JenisJabatanBaru    JenisJabatan
	HomebaseBaru        *string
	HomebaseBaruId      *string `gorm:"type:uuid"`
	GajiPokokBaru       *float64
	NomorSk             *string
	TanggalEfektif      time.Time `gorm:"type:timestamp(3);index:mutasi_user_tanggal_idx"`
	Alasan              *string
	Status              StatusPersetujuan `gorm:"index:mutasi_status_idx"`
	DisetujuiBy         *string           `gorm:"type:uuid"`
	Catatan             *string
	DisetujuiAtasanBy   *string    `gorm:"type:uuid"`
	DisetujuiAtasanPada *time.Time `gorm:"type:timestamp(3)"`
}

type Dokumen struct {
	BaseModel
	UserId        string `gorm:"type:uuid;index:dokumen_user_idx"`
	Judul         string
	Jenis         *string
	Nomor         *string
	Tanggal       *time.Time `gorm:"type:timestamp(3)"`
	FileUrl       string
	Keterangan    *string
	BerlakuSampai *time.Time `gorm:"type:timestamp(3)"`
	CreatedBy     *string    `gorm:"type:uuid"`
	UpdatedBy     *string    `gorm:"type:uuid"`
}

type Peringatan struct {
	BaseModel
	UserId        string `gorm:"type:uuid;index:peringatan_user_tanggal_idx"`
	Tingkat       TingkatPeringatan
	Alasan        string
	NomorSurat    *string
	Tanggal       time.Time `gorm:"type:date;index:peringatan_user_tanggal_idx"`
	Penerbit      string
	Sanksi        *JenisSanksi
	SanksiCatatan *string
	CreatedBy     *string `gorm:"type:uuid"`
}

type Pemberhentian struct {
	BaseModel
	UserId           string             `gorm:"type:uuid;index:pemberhentian_user_tanggal_idx"`
	Jenis            JenisPemberhentian `gorm:"index:pemberhentian_jenis_idx"`
	Tanggal          time.Time          `gorm:"type:date;index:pemberhentian_user_tanggal_idx"`
	NomorSk          string
	HakTerakhir      *string
	UjiKapabilitasId *string `gorm:"type:uuid;index:pemberhentian_uji_idx"`
	Keterangan       *string
	CreatedBy        *string `gorm:"type:uuid"`
}

type Paklaring struct {
	BaseModel
	UserId                 string `gorm:"type:uuid;index:paklaring_user_idx"`
	NomorSurat             string `gorm:"uniqueIndex:paklaring_nomor_key"`
	Keperluan              *string
	TanggalTerbit          time.Time `gorm:"type:date"`
	NamaSnapshot           string
	NipSnapshot            *string
	JabatanSnapshot        string
	UnitSnapshot           *string
	TanggalMasukSnapshot   *time.Time `gorm:"type:date"`
	TanggalSelesaiSnapshot *time.Time `gorm:"type:date"`
	DiterbitkanBy          *string    `gorm:"type:uuid"`
	KodeVerifikasi         string     `gorm:"uniqueIndex"`
}

type PesanKeluar struct {
	BaseModel
	Kanal          KanalPesan  `gorm:"index:pesan_keluar_kanal_status_idx"`
	Status         StatusPesan `gorm:"index:pesan_keluar_status_idx;index:pesan_keluar_kanal_status_idx"`
	PenerimaUserId *string     `gorm:"type:uuid"`
	Penerima       string
	Judul          *string
	Isi            string `gorm:"type:text"`
	Referensi      *string
	DibuatOleh     *string    `gorm:"type:uuid"`
	TerkirimPada   *time.Time `gorm:"type:timestamp(3)"`
	Galat          *string
	Percobaan      int
}

type AuditLog struct {
	IDModel
	AktorId    *string `gorm:"type:uuid;index:audit_log_aktor_idx"`
	Aksi       string  `gorm:"index:audit_log_aksi_idx"`
	Sasaran    *string
	Keterangan *string
	Ip         *string
	// UnitId menyimpan unit cakupan saat aksi terjadi agar akses audit dapat
	// dibatasi per unit, bukan hanya per aktor.
	UnitId    *string   `gorm:"type:uuid;index:audit_log_unit_idx"`
	CreatedAt time.Time `gorm:"column:createdAt;type:timestamp(3);index:audit_log_created_at_idx"`
}
