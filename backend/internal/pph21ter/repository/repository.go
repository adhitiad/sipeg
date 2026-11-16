// Package repository memuat port persistensi dan adapter GORM untuk PPh 21 TER.
package repository

import (
	"context"

	"simpeg/backend/internal/models"
)

// Repository adalah port persistensi tabel tarif efektif PPh 21 TER.
type Repository interface {
	// ListTer membaca seluruh baris tarif TER.
	ListTer(context.Context) ([]models.Pph21TerTarif, error)
	// ListTerOn mengembalikan baris tarif TER yang berlaku pada tanggal tertentu.
	ListTerOn(context.Context, string) ([]models.Pph21TerTarif, error)
	// CreateTer menyimpan baris tarif TER baru.
	CreateTer(context.Context, string, models.Pph21TerTarif) (models.Pph21TerTarif, error)
	// UpdateTer menyimpan perubahan baris tarif TER.
	UpdateTer(context.Context, string, string, map[string]any) (models.Pph21TerTarif, error)
	// DeleteTer menghapus satu baris tarif TER.
	DeleteTer(context.Context, string, string) error
}
