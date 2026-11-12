package models

import (
	"time"
)

// KeluargaPegawai mencatat data keluarga dan tanggungan pegawai
// (Suami/Istri dan Anak) untuk administrasi BPJS dan tunjangan keluarga.
type KeluargaPegawai struct {
	BaseModel
	UserId           string                `gorm:"type:uuid;index:keluarga_pegawai_user_idx"`
	Hubungan         JenisHubunganKeluarga `gorm:"type:JenisHubunganKeluarga;index:keluarga_pegawai_hubungan_idx"`
	Nama             string
	Nik              string    `gorm:"index:keluarga_pegawai_nik_idx"`
	TanggalLahir     time.Time `gorm:"type:date"`
	Pekerjaan        *string
	StatusTanggungan StatusTanggungan `gorm:"type:StatusTanggungan;default:'TANGGUNGAN'"`
}

// RiwayatGolongan mencatat riwayat golongan/pangkat pegawai (misal III/b,
// IV/a) dengan TMT dan Angka Kredit sebagai penentu standar gaji pokok.
type RiwayatGolongan struct {
	BaseModel
	UserId      string `gorm:"type:uuid;index:riwayat_golongan_user_idx"`
	Golongan    string `gorm:"index:riwayat_golongan_golongan_idx"`
	Ruang       string
	TMT         time.Time `gorm:"type:date;index:riwayat_golongan_tmt_idx"`
	AngkaKredit int
	NomorSk     *string
}

// KomponenTunjangan menyimpan nilai tunjangan per pegawai per jenis
// (struktural, fungsional, beras, makan, transportasi).
type KomponenTunjangan struct {
	BaseModel
	UserId         string         `gorm:"type:uuid;index:komponen_tunjangan_user_idx"`
	JenisTunjangan JenisTunjangan `gorm:"type:JenisTunjangan;index:komponen_tunjangan_jenis_idx"`
	Nilai          float64        `gorm:"default:0"`
	Aktif          bool           `gorm:"default:true"`
}

// KomponenPotongan menyimpan variabel potongan per pegawai (PPh 21, BPJS,
// kasbon, koperasi) dengan persentase dan/atau nominal.
type KomponenPotongan struct {
	BaseModel
	UserId         string        `gorm:"type:uuid;index:komponen_potongan_user_idx"`
	JenisPotongan  JenisPotongan `gorm:"type:JenisPotongan;index:komponen_potongan_jenis_idx"`
	Persentase     *float64
	Nominal        float64 `gorm:"default:0"`
	PorsiInstitusi *float64
	PorsiPegawai   *float64
	Aktif          bool `gorm:"default:true"`
}

// ArsipDokumenPegawai menyimpan metadata dan URL cloud storage dari dokumen
// legal pegawai (KTP, NPWP, KK, rekening, SK, pakta integritas, dst.).
type ArsipDokumenPegawai struct {
	BaseModel
	UserId             string            `gorm:"type:uuid;index:arsip_dokumen_pegawai_user_idx"`
	JenisDokumen       JenisDokumenArsip `gorm:"type:JenisDokumenArsip;index:arsip_dokumen_pegawai_jenis_idx"`
	Judul              string
	FileUrl            string
	FileSize           int64  `gorm:"column:fileSize"` // ukuran berkas dalam byte
	MimeType           string `gorm:"column:mimeType"` // tipe MIME berkas (image/png, application/pdf, dst.)
	Keterangan         *string
	BerlakuSampai      *time.Time `gorm:"type:date"`
	CloudinaryPublicId *string    `gorm:"column:cloudinaryPublicId"` // ID publik berkas pada Cloudinary
}

// RiwayatSertifikasi mencatat status kelulusan sertifikasi (Serdos, profesi,
// kompetensi) dengan nomor registrasi dan masa berlaku.
type RiwayatSertifikasi struct {
	BaseModel
	UserId           string           `gorm:"type:uuid;index:riwayat_sertifikasi_user_idx"`
	JenisSertifikasi JenisSertifikasi `gorm:"type:JenisSertifikasi;index:riwayat_sertifikasi_jenis_idx"`
	NamaSertifikat   string
	NomorRegistrasi  *string
	Penyelenggara    string
	TanggalTerbit    time.Time         `gorm:"type:date"`
	BerlakuSampai    *time.Time        `gorm:"type:date"`
	Status           StatusSertifikasi `gorm:"type:StatusSertifikasi;default:'AKTIF';index:riwayat_sertifikasi_status_idx"`
}

// RiwayatPelatihan mencatat log keikutsertaan pegawai dalam diklat,
// lokakarya, atau peningkatan soft-skills.
type RiwayatPelatihan struct {
	BaseModel
	UserId             string `gorm:"type:uuid;index:riwayat_pelatihan_user_idx"`
	NamaPelatihan      string
	JenisPelatihan     JenisPelatihan `gorm:"type:JenisPelatihan;index:riwayat_pelatihan_jenis_idx"`
	Penyelenggara      string
	TanggalMulai       time.Time `gorm:"type:date"`
	TanggalSelesai     time.Time `gorm:"type:date"`
	DurasiJam          int       `gorm:"default:0"`
	MemilikiSertifikat bool      `gorm:"default:false"`
}

// SisterSyncMap menyimpan referensi ID eksternal (SISTER, PDDIKTI) yang
// dipasangkan dengan id_pegawai internal untuk mencegah duplikasi data.
type SisterSyncMap struct {
	BaseModel
	UserId               string             `gorm:"type:uuid;index:sister_sync_map_user_idx;uniqueIndex:sister_sync_map_user_sistem_key"`
	SistemEksternal      SistemEksternal    `gorm:"type:SistemEksternal;uniqueIndex:sister_sync_map_user_sistem_key"`
	IDEksternal          string             `gorm:"index:sister_sync_map_id_eksternal_idx"`
	SinkronisasiTerakhir *time.Time         `gorm:"type:timestamp(3)"`
	Status               StatusSinkronisasi `gorm:"type:StatusSinkronisasi;default:'AKTIF'"`
}

// RiwayatPenghargaan mencatat riwayat penghargaan pegawai (dosen teladan,
// satyalancana, dst.).
type RiwayatPenghargaan struct {
	BaseModel
	UserId           string `gorm:"type:uuid;index:riwayat_penghargaan_user_idx"`
	NamaPenghargaan  string
	JenisPenghargaan JenisPenghargaan `gorm:"type:JenisPenghargaan;index:riwayat_penghargaan_jenis_idx"`
	Penyelenggara    string
	TanggalTerima    time.Time `gorm:"type:date"`
	NomorSk          *string
}
