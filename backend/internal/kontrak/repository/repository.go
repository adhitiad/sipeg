// Package repository memuat port persistensi kontrak kerja dan perpanjangan
// masa kerja (DEC-083).
package repository

import (
	"context"
	"time"

	"simpeg/backend/internal/models"
)

// KontrakInput memuat seluruh isian yang dibutuhkan use case untuk
// membuat perpanjangan.
type KontrakInput struct {
	UserID string
	Mulai  time.Time
	Durasi int // bulan; bawaan domain.DurasiBulanPerjanjian
	// AllowOverlap memaksa perjanjian dibuat walau tumpang tindih dengan
	// kontrak AKTIF; tercatat di jejak audit (override admin, `DEC-083`).
	AllowOverlap bool
}

// KontrakFilter membatasi daftar kontrak.
type KontrakFilter struct {
	UserID         string
	Jenis          string
	MulaiSesudah   *time.Time
	SelesaiSebelum *time.Time
	HomebaseIDs    []string
	Limit          int
	Offset         int
}

// Store adalah port persistensi kontrak.
type Store interface {
	// EmployeeHomebaseID mengembalikan unit penugasan pegawai; nil bila
	// belum diisi.
	EmployeeHomebaseID(ctx context.Context, userID string) (*string, error)
	// EmployeeExists melaporkan apakah pegawai dikenal.
	EmployeeExists(ctx context.Context, userID string) (bool, error)
	// EmployeeKategoriStatusID mengembalikan kategori status pegawai; nil
	// bila belum diisi.
	EmployeeKategoriStatusID(ctx context.Context, userID string) (*string, error)
	// Create menulis kontrak PKWT perjanjian kerja tahunan, memindahkan
	// kategoriStatusId pegawai ke DOSEN_TIDAK_TETAP, dan mencatat audit.
	Create(ctx context.Context, actor string, input KontrakInput, nomorSk string, newKategoriStatusID string, unitID *string) (models.KontrakKerja, error)
	// CountByTypeYear menghitung kontrak per jenis dan tahun (dipakai untuk
	// nomor urut SK kontrak).
	CountByTypeYear(ctx context.Context, jenis models.JenisKontrak, year int) (int64, error)
	// ActiveOverlapping melaporkan apakah ada kontrak AKTIF pegawai yang
	// tumpang tindih dengan rentang [mulai, selesai); kontrak PKWTT dengan
	// selesai NULL dianggap tumpang tindih dengan seluruh periode.
	ActiveOverlapping(ctx context.Context, userID string, mulai, selesai time.Time) (bool, error)
	// List membaca kontrak sesuai filter dan cakupan.
	List(ctx context.Context, filter KontrakFilter) ([]models.KontrakKerja, int64, error)
}
