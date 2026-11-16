// Model untuk area kontrak kerja, SK, pesangon, dan pensiun (DEC-083).
//
// Tabel dipetakan 1:1 dari migrasi 000025_payroll_core.sql dan
// 000029_kategori_dosen_tidak_tetap.sql.
package models

import "time"

// SuratKeputusan adalah dokumen keputusan resmi yang diterbitkan yayasan
// (SK pengangkatan/mutasi/PHK) atau rektor (SK jabatan struktural/penugasan).
type SuratKeputusan struct {
	BaseModel
	NomorSk        string `gorm:"uniqueIndex"`
	Jenis          JenisSk
	Penerbit       TingkatSk
	UserId         *string    `gorm:"type:uuid"`
	HomebaseId     *string    `gorm:"type:uuid"`
	TanggalSk      time.Time  `gorm:"type:date"`
	MulaiBerlaku   *time.Time `gorm:"type:date"`
	SelesaiBerlaku *time.Time `gorm:"type:date"`
	FileUrl        *string
	// Metadata adalah payload tambahan bebas bentuk (JSON); dipakai untuk
	// alasan pemberhentian dan konteks lain yang tidak dibakukan.
	Metadata string `gorm:"type:jsonb;default:'{}'"`
}

// KontrakKerja adalah kontrak kerja PKWT/PKWTT pegawai.
type KontrakKerja struct {
	BaseModel
	UserId  string `gorm:"type:uuid;index:kontrak_kerja_user_idx"`
	Jenis   JenisKontrak
	NomorSk string     `gorm:"uniqueIndex"`
	Mulai   time.Time  `gorm:"type:date"`
	Selesai *time.Time `gorm:"type:date;index:kontrak_kerja_selesai_idx"`
	// Status siklus: AKTIF | BERAKHIR | DIPERPANJANG.
	Status string  `gorm:"type:text;default:AKTIF"`
	SkId   *string `gorm:"type:uuid"`
}

// PesangonAturan adalah faktor pesangon/uang penghargaan/ganti kerugian per
// kategori status pegawai dan per tahun masa kerja. Faktor diisi manual
// oleh admin bila yayasan menetapkan; tidak di-seed dari PP 35/2021.
type PesangonAturan struct {
	BaseModel
	KategoriStatusId   string `gorm:"type:uuid;uniqueIndex:pesangon_aturan_kunci"`
	MasaKerjaTahun     int    `gorm:"uniqueIndex:pesangon_aturan_kunci"`
	PesangonBulan      float64
	PenghargaanBulan   float64
	GantiKerugianBulan float64
	SumberRegulasi     string `gorm:"type:text;default:PP 35/2021"`
}

// PensiunKonfigurasi adalah konfigurasi pensiun per kategori status,
// effective-dated.
type PensiunKonfigurasi struct {
	BaseModel
	KategoriStatusId     string `gorm:"type:uuid;uniqueIndex:pensiun_konfigurasi_kunci"`
	BupUmur              int
	DplkWajib            bool
	IuranPegawaiPct      float64 `gorm:"type:numeric(7,4)"`
	IuranPemberiKerjaPct float64 `gorm:"type:numeric(7,4)"`
	PenyediaDplk         *string
	BerlakuMulai         time.Time `gorm:"type:date;uniqueIndex:pensiun_konfigurasi_kunci"`
	Aktif                bool
}

// PensiunKepesertaan adalah kepesertaan DPLK per pegawai (satu kepesertaan
// per pegawai; bila pernah berhenti lalu kembali, nomor kepesertaan baru
// diisi).
type PensiunKepesertaan struct {
	BaseModel
	UserId              string `gorm:"type:uuid;uniqueIndex"`
	PenyediaDplk        string
	NomorKepesertaan    *string
	TanggalMulaiPensiun *time.Time `gorm:"type:date"`
	Mulai               time.Time  `gorm:"type:date"`
	Aktif               bool
}
