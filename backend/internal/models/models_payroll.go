package models

import (
	"time"
)

type KomponenGaji struct {
	BaseModel
	Nama    string `gorm:"uniqueIndex"`
	Arah    ArahKomponenGaji
	Nominal float64
	Aktif   bool
}

type Gaji struct {
	BaseModel
	UserId         string `gorm:"type:uuid;uniqueIndex:gaji_user_periode_key"`
	NomorSlip      string `gorm:"uniqueIndex"`
	KodeVerifikasi *string
	// Pegawai adalah pemilik slip. Relasi dipakai untuk membaca nama dan
	// alamat email saat notifikasi gaji disusun.
	Pegawai       *User     `gorm:"foreignKey:UserId"`
	Periode       time.Time `gorm:"type:date;uniqueIndex:gaji_user_periode_key;index:gaji_periode_idx"`
	TanggalGaji   time.Time `gorm:"type:date"`
	GajiPokok     float64
	TotalGaji     float64
	Status        StatusSlipGaji `gorm:"index:gaji_status_idx"`
	PkpSetahun    *float64
	PotonganPph21 float64

	// Konfirmasi pembayaran
	TanggalDibayar         *time.Time `gorm:"type:timestamp(3);index:gaji_tanggalDibayar_idx"`
	KonfirmasiPembayaranAt *time.Time `gorm:"type:timestamp(3)"`
	KonfirmasiOlehId       *string    `gorm:"type:uuid;index:gaji_konfirmasiOlehId_idx"`
	KonfirmasiOleh         *User      `gorm:"foreignKey:KonfirmasiOlehId"`

	GajiTunjangans []GajiTunjangan
	GajiPotongans  []GajiPotongan
	Pembayaran     []Pembayaran
}

type GajiTunjangan struct {
	BaseModel
	GajiId  string `gorm:"type:uuid;index:gaji_tunjangan_gaji_idx"`
	Jenis   string
	Nominal float64
}

type GajiPotongan struct {
	BaseModel
	GajiId  string `gorm:"type:uuid;index:gaji_potongan_gaji_idx"`
	Jenis   string
	Nominal float64
}

// Pembayaran adalah konfirmasi pembayaran gaji. Setiap slip gaji yang
// berstatus TERBAYAR harus memiliki konfirmasi pembayaran yang valid sebelum
// notifikasi email "gaji sudah masuk" boleh dikirim.
type Pembayaran struct {
	BaseModel
	GajiId             string `gorm:"type:uuid;index:pembayaran_gaji_idx"`
	Gaji               *Gaji  `gorm:"foreignKey:GajiId"`
	ReferensiTransaksi *string
	WaktuKonfirmasi    time.Time `gorm:"type:timestamp(3)"`
	PelakuPengesahan   string
	Keterangan         *string
}

type HariLibur struct {
	BaseModel
	Tanggal time.Time `gorm:"type:date;uniqueIndex;index:hari_libur_tanggal_idx"`
	Nama    string
}

// JadwalGaji adalah tanggal gajian tetap tiap bulan untuk satu unit
// organisasi. Nilai 1-28 menghindari pergeseran bulan pendek. Pemberitahuan
// slip gaji dikirim H+1 pada hari kalender berikutnya tanpa penyesuaian hari
// libur (keputusan 11 November 2026).
type JadwalGaji struct {
	BaseModel
	UnitId string `gorm:"type:uuid;uniqueIndex"`
	// Unit adalah unit pemilik jadwal; relasi dipakai untuk menampilkan
	// nama unit pada daftar jadwal.
	Unit          *Homebase `gorm:"foreignKey:UnitId"`
	TanggalGajian int
}

type SasaranKinerja struct {
	BaseModel
	UserId      string `gorm:"type:uuid;index:sasaran_kinerja_user_periode_idx"`
	Tahun       int    `gorm:"index:sasaran_kinerja_user_periode_idx"`
	Semester    int    `gorm:"index:sasaran_kinerja_user_periode_idx"`
	Uraian      string
	Target      float64
	Realisasi   float64
	Bobot       float64
	DinilaiOleh *string `gorm:"type:uuid"`
}

type Pesangon struct {
	BaseModel
	UserId         string    `gorm:"type:uuid;index:pesangon_user_idx"`
	TanggalPensiun time.Time `gorm:"type:date"`
	MasaKerjaBulan int
	GajiPokok      float64
	Kelipatan      int
	Nominal        float64
	NomorSk        *string
	DiterbitkanBy  *string `gorm:"type:uuid"`
}

type UjiKapabilitas struct {
	BaseModel
	UserId         string  `gorm:"type:uuid;index:uji_kapabilitas_user_idx"`
	PeriodeSakitId *string `gorm:"type:uuid"`
	BulanKe        int
	Hasil          *HasilUjiKapabilitas `gorm:"index:uji_kapabilitas_hasil_idx"`
	TanggalUji     *time.Time           `gorm:"type:timestamp(3)"`
	DiputusBy      *string              `gorm:"type:uuid"`
	Catatan        *string
}
