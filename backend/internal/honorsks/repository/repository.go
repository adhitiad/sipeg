// Package repository memuat port persistensi dan adapter GORM untuk honor
// mengajar per SKS (dosen LB).
package repository

import (
	"context"

	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
)

// Repository adalah port persistensi honor mengajar per SKS.
type Repository interface {
	// ListRates membaca seluruh tarif honor per SKS.
	ListRates(context.Context) ([]models.HonorSksRate, error)
	// GetRate membaca satu tarif honor per SKS.
	GetRate(context.Context, string) (models.HonorSksRate, error)
	// LatestRateOn mengembalikan tarif berlaku terakhir pada tanggal tertentu.
	// Mengembalikan domain.ErrNotFound bila belum ada tarif berlaku.
	LatestRateOn(context.Context, string) (models.HonorSksRate, error)
	// CreateRate menyimpan tarif honor per SKS baru.
	CreateRate(context.Context, string, models.HonorSksRate) (models.HonorSksRate, error)
	// UpdateRate menyimpan perubahan tarif honor per SKS.
	UpdateRate(context.Context, string, string, map[string]any) (models.HonorSksRate, error)

	// ListPeriode membaca daftar beban mengajar sesuai filter dan pagination.
	ListPeriode(context.Context, listing.Options) ([]models.HonorSksPeriode, int64, error)
	// GetPeriode membaca satu baris beban mengajar.
	GetPeriode(context.Context, string) (models.HonorSksPeriode, error)
	// CreatePeriode menyimpan baris beban mengajar baru.
	CreatePeriode(context.Context, string, models.HonorSksPeriode) (models.HonorSksPeriode, error)
	// UpdatePeriode menyimpan perubahan beban mengajar.
	UpdatePeriode(context.Context, string, string, map[string]any) (models.HonorSksPeriode, error)
	// DeletePeriode menghapus baris beban mengajar.
	DeletePeriode(context.Context, string, string) error
}
