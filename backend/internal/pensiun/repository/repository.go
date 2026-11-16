// Port persistensi master pensiun: konfigurasi per kategori status
// (effective-dated) dan kepesertaan DPLK per pegawai.
package repository

import (
	"context"
	"errors"
	"time"

	"simpeg/backend/internal/models"
)

// KepesertaanRow adalah satu kepesertaan DPLK beserta nama pegawainya
// (untuk tampilan admin tanpa panggilan terpisah).
type KepesertaanRow struct {
	models.PensiunKepesertaan
	Nama string `gorm:"column:nama"`
}

// Store adalah port persistensi pensiun.
type Store interface {
	ListKonfigurasi(ctx context.Context, kategoriStatusID string) ([]models.PensiunKonfigurasi, error)
	UpsertKonfigurasi(ctx context.Context, actorID string, item models.PensiunKonfigurasi) (models.PensiunKonfigurasi, error)
	UpdateKonfigurasi(ctx context.Context, actorID string, id string, item models.PensiunKonfigurasi) (models.PensiunKonfigurasi, error)
	DeleteKonfigurasi(ctx context.Context, actorID string, id string) error

	ListKepesertaan(ctx context.Context, scopeIDs []string) ([]KepesertaanRow, error)
	UpsertKepesertaan(ctx context.Context, actorID string, item models.PensiunKepesertaan) (models.PensiunKepesertaan, error)
	UpdateKepesertaan(ctx context.Context, actorID string, id string, item models.PensiunKepesertaan) (models.PensiunKepesertaan, error)
	DeactivateKepesertaan(ctx context.Context, actorID string, id string) error

	// UserBirthInfo membaca tanggal lahir dan kategori status pegawai;
	// keduanya boleh kosong bila belum diisi.
	UserBirthInfo(ctx context.Context, userID string) (lahir *time.Time, kategoriStatusID *string, err error)
	// ExistsUser melaporkan apakah id menunjuk ke baris user yang ada.
	ExistsUser(ctx context.Context, userID string) (bool, error)
	// ListUnitScopeIDs mengembalikan unit penugasan beserta seluruh turunan
	// pada pohon Homebase (cakupan homebase principal).
	ListUnitScopeIDs(ctx context.Context, rootHomebaseID string) ([]string, error)
	// ActiveKonfigurasi membaca konfigurasi aktif yang berlaku pada tanggal
	// acuan untuk kategori status tertentu (periode berlaku terbaru).
	ActiveKonfigurasi(ctx context.Context, kategoriStatusID string, ref time.Time) (models.PensiunKonfigurasi, error)
}

// ErrNotFoundKonfigurasi menandai konfigurasi yang tidak ditemukan.
var ErrNotFoundKonfigurasi = errors.New("konfigurasi pensiun tidak ditemukan")

// ErrNotFoundKepesertaan menandai kepesertaan yang tidak ditemukan.
var ErrNotFoundKepesertaan = errors.New("kepesertaan pensiun tidak ditemukan")

// ErrUserNotFound menandai user yang tidak ada (atau nonaktif untuk
// keperluan kepesertaan).
var ErrUserNotFound = errors.New("pegawai tidak ditemukan")
