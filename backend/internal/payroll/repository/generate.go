// Package repository menyimpan port dan adapter persistensi pelengkap untuk
// pembuatan slip gaji otomatis (penggajian inti).
package repository

import (
	"context"
	"time"

	"simpeg/backend/internal/models"
)

// GenerateRepository adalah port persistensi pelengkap untuk GenerateSlip.
type GenerateRepository interface {
	// GetEmployeeStatusPtkp membaca status PTKP pegawai (default TK/0).
	GetEmployeeStatusPtkp(context.Context, string) (string, error)
	// GetEmployeeAktif membaca apakah pegawai masih aktif; nil bila
	// pegawai tidak dikenal.
	GetEmployeeAktif(context.Context, string) (*bool, error)
	// ApprovedPemberhentianFor mengembalikan pemberhentian DISETUJUI
	// terakhir pegawai (tanggal efektif terbaru); nil bila tidak ada.
	ApprovedPemberhentianFor(context.Context, string) (*models.Pemberhentian, error)
	// GetEmployeeHomebaseID mengembalikan unit penugasan pegawai; dipakai
	// untuk mengisi unitId jejak audit slip.
	GetEmployeeHomebaseID(context.Context, string) (*string, error)
	// GetEmployeeKategoriStatus mengembalikan id dan kode kategori status
	// pegawai (mis. DOSEN_TETAP_YAYASAN). Kode kosong bila pegawai belum
	// dikaitkan ke kategori status.
	GetEmployeeKategoriStatus(context.Context, string) (string, string, error)
	// CountPresensiAlpa menghitung jumlah hari dengan status ALPA
	// ('A') pada rentang [monthStart, nextMonth).
	CountPresensiAlpa(context.Context, string, time.Time, time.Time) (int64, error)
	// LookupGajiPokok mengembalikan skala efektif dan baris nominal untuk
	// kombinasi (kategoriStatusId, tingkat) pada tanggal periode berlaku.
	LookupGajiPokok(context.Context, string, int, time.Time) (models.SkalaGaji, models.SkalaGajiDetail, error)
	// FindPeriodeAkademik mengembalikan periode akademik yang memuat periode gaji.
	FindPeriodeAkademik(context.Context, time.Time, time.Time) (models.PeriodeAkademik, error)
	// GetPeriodeAkademikByID membaca satu periode akademik.
	GetPeriodeAkademikByID(context.Context, string) (models.PeriodeAkademik, error)
	// ListKomponenTunjangan membaca komponen tunjangan pegawai.
	ListKomponenTunjangan(context.Context, string) ([]models.KomponenTunjangan, error)
	// ListKomponenPotongan membaca komponen potongan pegawai.
	ListKomponenPotongan(context.Context, string) ([]models.KomponenPotongan, error)
	// LookupHonor membaca baris honor beban mengajar dosen LB untuk periode akademik.
	LookupHonor(context.Context, string, string) (models.HonorSksPeriode, error)
	// LatestBpjsConfig mengembalikan satu baris tarif BPJS per jenis yang berlaku.
	LatestBpjsConfig(context.Context, time.Time) ([]models.BpjsKonfigurasi, error)
	// TerOn mengembalikan tabel tarif efektif TER yang berlaku pada tanggal tersebut.
	TerOn(context.Context, time.Time) ([]models.Pph21TerTarif, error)
	// UpsertSlip menulis slip dan komponen dalam satu transaksi. Slip DRAF
	// yang sudah ada untuk (userId, periode) digantikan; slip TERBIT/TERBAYAR
	// ditolak dengan domain.ErrConflict. unitID adalah unit cakupan aksi untuk
	// jejak audit; nil berarti unit tidak diketahui.
	// UpsertSlip menyimpan slip gaji (upsert DRAF; slip TERBIT/TERBAYAR
	// ditolak); slip yang sudah ada untuk (userId, periode) digantikan.
	// unitID adalah unit cakupan aksi untuk jejak audit; nil berarti unit
	// tidak diketahui. auditNote menambahkan catatan pada jejak audit
	// (mis. penyesuaian sakit berkepanjangan); nil berarti tanpa catatan.
	UpsertSlip(ctx context.Context, actorID string, slip models.Gaji, tunjangan []models.GajiTunjangan, potongan []models.GajiPotongan, unitID *string, auditNote *string) (models.Gaji, error)
	// PeriodeSakitFor mencari baris PeriodeSakit yang mencakup periode slip
	// (bulan berjalan). Mengembalikan persen berjalan, bulan ke, dan true
	// bila ditemukan.
	PeriodeSakitFor(context.Context, string, time.Time) (float64, int, bool, error)
}
