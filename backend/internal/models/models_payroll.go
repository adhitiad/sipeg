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
// slip gaji dikirim H-1 pada hari kalender sebelumnya tanpa penyesuaian
// hari libur (keputusan pemilik 2026-11-14; supersede keputusan 11 November 2026).
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

// SkalaGaji adalah tabel skala gaji pokok yayasan (effective-dated).
type SkalaGaji struct {
	BaseModel
	Kode           string `gorm:"uniqueIndex"`
	Nama           string
	BerlakuMulai   time.Time  `gorm:"type:date"`
	BerlakuSelesai *time.Time `gorm:"type:date"`
	Aktif          bool
}

// SkalaGajiDetail adalah baris nominal per kategori status dan tingkat.
type SkalaGajiDetail struct {
	BaseModel
	// SkalaGajiId menunjuk skalaGaji tempat baris nominal ini berada.
	SkalaGajiId      string `gorm:"column:skalaGajiId;type:uuid;uniqueIndex:skala_gaji_detail_kunci"`
	KategoriStatusId string `gorm:"column:kategoriStatusId;type:uuid;uniqueIndex:skala_gaji_detail_kunci;index:skala_gaji_detail_kategori_idx"`
	Tingkat          int    `gorm:"uniqueIndex:skala_gaji_detail_kunci"`
	Nominal          float64
}

// KategoriStatusPegawai adalah taxonomy kategori status pegawai.
type KategoriStatusPegawai struct {
	BaseModel
	Kode             string `gorm:"uniqueIndex"`
	Nama             string
	Deskripsi        *string
	GajiDariNegara   bool
	ButuhKontrak     bool
	WajibBpjs        bool
	BupSesuaiYayasan bool
	Aktif            bool
}

// ThrKonfigurasi adalah konfigurasi THR per tahun (`DEC-070`/`DEC-083`;
// `thrKonfigurasi` pada `000025_payroll_core.sql`). `berlakuTanggal` adalah
// tanggal pembayaran; ` syaratMasaKerjaBulan` adalah syarat masa kerja
// pegawai. `besaranTipe`: `SATU_GAJI_POKOK` | `GAJI_KOTOR` (memakai slip
// TERBIT terakhir) | `PERSEN` (dikalikan `persen` dari gaji pokok slip
// TERBIT terakhir).
type ThrKonfigurasi struct {
	BaseModel
	Tahun                int       `gorm:"uniqueIndex"`
	BesaranTipe          string    `gorm:"type:text"`
	Persen               *float64  `gorm:"type:double precision"`
	SyaratMasaKerjaBulan int       `gorm:"default:3"`
	BerlakuTanggal       time.Time `gorm:"type:date"`
}

// HonorSksRate adalah tarif honor per SKS per periode berlaku (dosen LB).
type HonorSksRate struct {
	BaseModel
	BerlakuMulai time.Time `gorm:"type:date;uniqueIndex"`
	PerSks       float64
	Aktif        bool
}

// HonorSksPeriode adalah beban mengajar dan total honor per dosen LB per
// periode akademik. Nominal disimpan sebagai bilangan bulat rupiah penuh.
type HonorSksPeriode struct {
	BaseModel
	UserId            string  `gorm:"type:uuid;uniqueIndex:honor_sks_periode_kunci;index:honor_sks_periode_user_idx"`
	PeriodeAkademikId string  `gorm:"type:uuid;uniqueIndex:honor_sks_periode_kunci"`
	Sks               float64 `gorm:"type:numeric(6,2)"`
	RatePerSks        float64
	TotalHonor        float64
	SkId              *string `gorm:"type:uuid"`
}

// BpjsKonfigurasi adalah tarif iuran BPJS per jenis dan periode berlaku.
type BpjsKonfigurasi struct {
	BaseModel
	Jenis                JenisBPJS `gorm:"type:JenisBPJS;uniqueIndex:bpjs_konfigurasi_kunci"`
	IuranPegawaiPct      float64   `gorm:"type:numeric(7,4)"`
	IuranPemberiKerjaPct float64   `gorm:"type:numeric(7,4)"`
	Dasar                string    `gorm:"type:text;default:GAJI_POKOK"`
	BatasAtas            *float64
	BerlakuMulai         time.Time `gorm:"type:date;uniqueIndex:bpjs_konfigurasi_kunci"`
	Aktif                bool
}

// BpjsPeserta adalah kepesertaan BPJS per pegawai dan jenis.
type BpjsPeserta struct {
	BaseModel
	UserId           string    `gorm:"type:uuid;uniqueIndex:bpjs_peserta_kunci;index:bpjs_peserta_user_idx"`
	Jenis            JenisBPJS `gorm:"type:JenisBPJS;uniqueIndex:bpjs_peserta_kunci"`
	NomorKepesertaan *string
	Mulai            time.Time  `gorm:"type:date"`
	Selesai          *time.Time `gorm:"type:date"`
	Aktif            bool
}

// Pph21TerTarif adalah tabel tarif efektif bulanan PPh 21 skema TER
// (PMK 168/2023). Pemetaan PTKP → kategori TER dilakukan di lapisan aplikasi.
type Pph21TerTarif struct {
	BaseModel
	Kategori     PtkpKategoriTER `gorm:"type:PtkpKategoriTER;uniqueIndex:pph21_ter_kunci"`
	Lapisan      int             `gorm:"uniqueIndex:pph21_ter_kunci"`
	BatasBawah   float64
	BatasAtas    *float64
	TarifPct     float64
	BerlakuMulai time.Time `gorm:"type:date;uniqueIndex:pph21_ter_kunci"`
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
