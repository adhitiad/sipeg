package repository

import (
	"context"

	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
)

// Repository adalah port persistensi untuk modul SKP & Log Aktivitas.
type Repository interface {
	// SKP Periode
	ListSKPPeriode(ctx context.Context, options listing.Options) ([]models.SKPPeriode, int64, error)
	GetSKPPeriode(ctx context.Context, id string) (models.SKPPeriode, error)
	CreateSKPPeriode(ctx context.Context, actorID string, periode models.SKPPeriode) (models.SKPPeriode, error)
	UpdateSKPPeriode(ctx context.Context, actorID, id string, updates map[string]any) (models.SKPPeriode, error)
	DeleteSKPPeriode(ctx context.Context, actorID, id string) error

	// SKP Target Kinerja
	ListSKPTarget(ctx context.Context, options listing.Options) ([]models.SKPTargetKinerja, int64, error)
	GetSKPTarget(ctx context.Context, id string) (models.SKPTargetKinerja, error)
	CreateSKPTarget(ctx context.Context, actorID string, target models.SKPTargetKinerja) (models.SKPTargetKinerja, error)
	UpdateSKPTarget(ctx context.Context, actorID, id string, updates map[string]any) (models.SKPTargetKinerja, error)
	DeleteSKPTarget(ctx context.Context, actorID, id string) error

	// Log Aktivitas Harian
	ListLogAktivitas(ctx context.Context, options listing.Options) ([]models.LogAktivitasHarian, int64, error)
	GetLogAktivitas(ctx context.Context, id string) (models.LogAktivitasHarian, error)
	CreateLogAktivitas(ctx context.Context, actorID string, log models.LogAktivitasHarian) (models.LogAktivitasHarian, error)
	UpdateLogAktivitas(ctx context.Context, actorID, id string, updates map[string]any) (models.LogAktivitasHarian, error)
	DeleteLogAktivitas(ctx context.Context, actorID, id string) error

	// Penilaian Kinerja Akhir
	ListPenilaian(ctx context.Context, options listing.Options) ([]models.PenilaianKinerjaAkhir, int64, error)
	GetPenilaian(ctx context.Context, id string) (models.PenilaianKinerjaAkhir, error)
	CreatePenilaian(ctx context.Context, actorID string, penilaian models.PenilaianKinerjaAkhir) (models.PenilaianKinerjaAkhir, error)
	UpdatePenilaian(ctx context.Context, actorID, id string, updates map[string]any) (models.PenilaianKinerjaAkhir, error)
	DeletePenilaian(ctx context.Context, actorID, id string) error
}

// ScopeRepository adalah port resolusi cakupan unit pada pohon organisasi.
type ScopeRepository interface {
	ListUnitScopeIDs(context.Context, string) ([]string, error)
}
