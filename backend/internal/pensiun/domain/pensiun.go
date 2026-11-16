// Aturan master pensiun: konfigurasi pensiun per kategori status
// (effective-dated) dan kepesertaan DPLK per pegawai.
package domain

import (
	"errors"
	"strings"
	"time"
)

// KonfigurasiInput adalah isian untuk membuat/mengganti baris konfigurasi
// pensiun per kategori status dan tanggal berlaku.
type KonfigurasiInput struct {
	KategoriStatusID     string
	BupUmur              int
	DplkWajib            bool
	IuranPegawaiPct      float64
	IuranPemberiKerjaPct float64
	PenyediaDplk         string
	BerlakuMulai         time.Time
	Aktif                bool
}

// KepesertaanInput adalah isian untuk membuat/mengganti kepesertaan DPLK
// seorang pegawai. TanggalMulaiPensiun opsional; bila kosong, use case
// menghitung dari tanggal lahir pegawai ditambah BUP pada konfigurasi
// yang berlaku.
type KepesertaanInput struct {
	UserID              string
	PenyediaDplk        string
	NomorKepesertaan    string
	TanggalMulaiPensiun *time.Time
	Mulai               time.Time
	Aktif               bool
}

var (
	// ErrValidation menandai isian master yang tidak sah.
	ErrValidation = errors.New("isian pensiun tidak sah")
	// ErrForbidden menandai principal tanpa izin pengelolaan pensiun.
	ErrForbidden = errors.New("akses ditolak")
	// ErrConfigNotFound menandai konfigurasi yang tidak ditemukan.
	ErrConfigNotFound = errors.New("konfigurasi pensiun tidak ditemukan")
	// ErrKepesertaanNotFound menandai kepesertaan yang tidak ditemukan.
	ErrKepesertaanNotFound = errors.New("kepesertaan pensiun tidak ditemukan")
	// ErrUserNotFound menandai userId yang tidak menunjuk pegawai.
	ErrUserNotFound = errors.New("pegawai tidak ditemukan")
)

// Validate memeriksa isian konfigurasi; BUP dibatasi 45–80 tahun dan iuran
// dibatasi 0–100 persen sesuai CHECK migrasi 000025.
func (i KonfigurasiInput) Validate() error {
	if strings.TrimSpace(i.KategoriStatusID) == "" {
		return ErrValidation
	}
	if i.BupUmur < 45 || i.BupUmur > 80 {
		return ErrValidation
	}
	if i.IuranPegawaiPct < 0 || i.IuranPegawaiPct > 100 || i.IuranPemberiKerjaPct < 0 || i.IuranPemberiKerjaPct > 100 {
		return ErrValidation
	}
	if i.BerlakuMulai.IsZero() {
		return ErrValidation
	}
	return nil
}

// Validate memeriksa isian kepesertaan. Penyedia DPLK wajib diisi;
// TanggalMulaiPensiun boleh kosong untuk dihitung use case.
func (i KepesertaanInput) Validate() error {
	if strings.TrimSpace(i.UserID) == "" {
		return ErrValidation
	}
	if strings.TrimSpace(i.PenyediaDplk) == "" {
		return ErrValidation
	}
	if i.Mulai.IsZero() {
		return ErrValidation
	}
	return nil
}

// TanggalDariUsia menghitung tanggal lahir + `bup` tahun.
func TanggalDariUsia(lahir time.Time, bup int) time.Time {
	return lahir.AddDate(bup, 0, 0)
}
