// Package repository memuat port persistensi dan adapter GORM untuk area
// skala gaji pokok yayasan.
package repository

import (
	"context"

	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
)

// Repository adalah port persistensi data skala gaji pokok yayasan.
type Repository interface {
	// ListSkala membaca daftar skala gaji dengan filter dan pagination.
	ListSkala(context.Context, listing.Options) ([]models.SkalaGaji, int64, error)
	// GetSkala membaca satu skala gaji.
	GetSkala(context.Context, string) (models.SkalaGaji, error)
	// GetSkalaWithDetails membaca satu skala gaji beserta baris nominalnya.
	GetSkalaWithDetails(context.Context, string) (models.SkalaGaji, []models.SkalaGajiDetail, error)
	// ListKategoriStatus membaca taxonomy kategori status pegawai.
	ListKategoriStatus(context.Context) ([]models.KategoriStatusPegawai, error)
	// CreateSkala menyimpan skala gaji baru (audit ditulis transaksional).
	CreateSkala(context.Context, string, models.SkalaGaji) (models.SkalaGaji, error)
	// UpdateSkala menyimpan perubahan skala gaji (audit transaksional).
	UpdateSkala(context.Context, string, string, map[string]any) (models.SkalaGaji, error)
	// GetDetail membaca satu baris nominal.
	GetDetail(context.Context, string) (models.SkalaGajiDetail, error)
	// ListDetails membaca seluruh baris nominal pada satu skala gaji.
	ListDetails(context.Context, string) ([]models.SkalaGajiDetail, error)
	// CreateDetail menyimpan baris nominal baru (audit transaksional).
	CreateDetail(context.Context, string, models.SkalaGajiDetail) (models.SkalaGajiDetail, error)
	// UpdateDetail menyimpan perubahan baris nominal (audit transaksional).
	UpdateDetail(context.Context, string, string, map[string]any) (models.SkalaGajiDetail, error)
	// DeleteDetail menghapus baris nominal (audit transaksional).
	DeleteDetail(context.Context, string, string) error
}
