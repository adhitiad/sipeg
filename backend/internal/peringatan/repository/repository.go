// Package repository memuat port persistensi dan adapter GORM untuk peringatan
// (SP1) pegawai. Peringatan bersifat pencatatan saja dan tidak memengaruhi
// slip gaji.
package repository

import (
	"context"
	"time"

	"simpeg/backend/internal/models"
)

// ListOptions memuat filter daftar peringatan.
type ListOptions struct {
	UserID      string
	Tingkat     models.TingkatPeringatan
	HomebaseIDs []string
	Limit       int
	Offset      int
}

// Store adalah port persistensi yang dibutuhkan use case.
type Store interface {
	Create(ctx context.Context, actor string, input CreateInput, unitID *string) (models.Peringatan, error)
	Get(ctx context.Context, id string) (models.Peringatan, error)
	List(ctx context.Context, options ListOptions) ([]models.Peringatan, int64, error)
	EmployeeHomebaseID(ctx context.Context, userID string) (*string, error)
	EmployeeExists(ctx context.Context, userID string) (bool, error)
}

// CreateInput memuat data penerbitan SP1.
type CreateInput struct {
	UserID        string
	Tingkat       models.TingkatPeringatan
	Alasan        string
	NomorSurat    *string
	Tanggal       time.Time
	Penerbit      string
	Sanksi        *models.JenisSanksi
	SanksiCatatan *string
}
