// Package repository memuat port persistensi dan adapter GORM untuk area BPJS.
package repository

import (
	"context"

	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
)

// Repository adalah port persistensi data BPJS.
type Repository interface {
	// ListKonfigurasi membaca seluruh tarif iuran BPJS.
	ListKonfigurasi(context.Context) ([]models.BpjsKonfigurasi, error)
	// GetKonfigurasi membaca satu tarif iuran BPJS.
	GetKonfigurasi(context.Context, string) (models.BpjsKonfigurasi, error)
	// CreateKonfigurasi menyimpan tarif iuran BPJS baru.
	CreateKonfigurasi(context.Context, string, models.BpjsKonfigurasi) (models.BpjsKonfigurasi, error)
	// UpdateKonfigurasi menyimpan perubahan tarif iuran BPJS.
	UpdateKonfigurasi(context.Context, string, string, map[string]any) (models.BpjsKonfigurasi, error)

	// ListPeserta membaca daftar kepesertaan BPJS sesuai filter dan pagination.
	ListPeserta(context.Context, listing.Options) ([]models.BpjsPeserta, int64, error)
	// GetPeserta membaca satu baris kepesertaan BPJS.
	GetPeserta(context.Context, string) (models.BpjsPeserta, error)
	// CreatePeserta menyimpan kepesertaan BPJS baru.
	CreatePeserta(context.Context, string, models.BpjsPeserta) (models.BpjsPeserta, error)
	// UpdatePeserta menyimpan perubahan kepesertaan BPJS.
	UpdatePeserta(context.Context, string, string, map[string]any) (models.BpjsPeserta, error)
}
