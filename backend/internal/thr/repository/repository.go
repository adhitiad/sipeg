// Package repository memuat port persistensi konfigurasi THR dan slip THR
// (`DEC-083`).
package repository

import (
	"context"
	"time"

	"simpeg/backend/internal/models"
)

// Candidate memuat satu pegawai aktif dan basis THR-nya: slip TERBIT
// terakhir, bila ada. Nominal THR hanya dari slip TERBIT (sesuai
// komentar migrasi 000025_payroll_core.sql §THR); pegawai tanpa slip
// TERBIT dilaporkan sebagai `no_terbit_slip`.
type Candidate struct {
	UserID       string
	Nama         string
	TanggalMasuk *time.Time
	GajiPokok    float64
	TotalGaji    float64
	// GajiBersih adalah take-home pay slip sumber: total bruto dikurangi
	// seluruh potongan. Dasar untuk besaran `PERSEN_TAKE_HOME`.
	GajiBersih float64
	HomebaseID *string
	HasSlip    bool
}

// Store adalah port persistensi THR.
type Store interface {
	// List mengembalikan seluruh konfigurasi THR (diurutkan tahun terbaru).
	List(ctx context.Context) ([]models.ThrKonfigurasi, error)
	// GetByTahun membaca satu konfigurasi; ErrConfigNotFound bila belum ada.
	GetByTahun(ctx context.Context, tahun int) (models.ThrKonfigurasi, error)
	// Create menulis konfigurasi baru; unik per tahun.
	// Create menulis konfigurasi baru; unik per tahun. actorID tercatat pada
	// jejak audit karena aturan per tahun berdampak ke seluruh unit.
	Create(ctx context.Context, actorID string, item models.ThrKonfigurasi) (models.ThrKonfigurasi, error)
	// Update mengganti isi konfigurasi tahun tersebut.
	Update(ctx context.Context, actorID string, tahun int, item models.ThrKonfigurasi) (models.ThrKonfigurasi, error)
	// Delete menghapus konfigurasi tahun tersebut; ErrConfigNotFound bila
	// belum ada.
	Delete(ctx context.Context, actorID string, tahun int) error
	// Candidates mengembalikan pegawai aktif dengan slip TERBIT/TERBAYAR terakhir
	// (DISTINCT ON) — satu baris per pegawai.
	Candidates(ctx context.Context) ([]Candidate, error)
	// CreateSlip menulis slip THR untuk pegawai dalam satu transaksi
	// bersama komponen penerimaan `THR` dan jejak audit. Nomor slip:
	// `THR-<tahun>-NNNN`. Slip yang sudah ada untuk (userId, periode) dengan
	// status TERBIT/TERBAYAR/BATAL ditolak dengan domain.ErrSlipConflict;
	// DRAF digantikan. `sumber` mencatat asal nominal (otomatis atau input
	// admin) pada jejak audit.
	CreateSlip(ctx context.Context, actor string, unitID *string, tahun int, berlakunya time.Time, userID string, nominal float64, sumber string) (models.Gaji, error)
}
