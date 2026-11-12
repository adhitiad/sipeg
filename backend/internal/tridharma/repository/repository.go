package repository

import (
	"context"

	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
)

// Repository adalah port persistensi untuk modul Tridharma & BKD.
type Repository interface {
	// Periode Akademik
	ListPeriodeAkademik(ctx context.Context, options listing.Options) ([]models.PeriodeAkademik, int64, error)
	GetPeriodeAkademik(ctx context.Context, id string) (models.PeriodeAkademik, error)
	CreatePeriodeAkademik(ctx context.Context, actorID string, periode models.PeriodeAkademik) (models.PeriodeAkademik, error)
	UpdatePeriodeAkademik(ctx context.Context, actorID, id string, updates map[string]any) (models.PeriodeAkademik, error)
	DeletePeriodeAkademik(ctx context.Context, actorID, id string) error

	// Evaluasi BKD
	ListEvaluasiBKD(ctx context.Context, options listing.Options) ([]models.EvaluasiBKD, int64, error)
	GetEvaluasiBKD(ctx context.Context, id string) (models.EvaluasiBKD, error)
	CreateEvaluasiBKD(ctx context.Context, actorID string, evaluasi models.EvaluasiBKD) (models.EvaluasiBKD, error)
	UpdateEvaluasiBKD(ctx context.Context, actorID, id string, updates map[string]any) (models.EvaluasiBKD, error)
	DeleteEvaluasiBKD(ctx context.Context, actorID, id string) error

	// Tridharma Pendidikan
	ListTridharmaPendidikan(ctx context.Context, options listing.Options) ([]models.TridharmaPendidikan, int64, error)
	GetTridharmaPendidikan(ctx context.Context, id string) (models.TridharmaPendidikan, error)
	CreateTridharmaPendidikan(ctx context.Context, actorID string, item models.TridharmaPendidikan) (models.TridharmaPendidikan, error)
	UpdateTridharmaPendidikan(ctx context.Context, actorID, id string, updates map[string]any) (models.TridharmaPendidikan, error)
	DeleteTridharmaPendidikan(ctx context.Context, actorID, id string) error

	// Tridharma Penelitian
	ListTridharmaPenelitian(ctx context.Context, options listing.Options) ([]models.TridharmaPenelitian, int64, error)
	GetTridharmaPenelitian(ctx context.Context, id string) (models.TridharmaPenelitian, error)
	CreateTridharmaPenelitian(ctx context.Context, actorID string, item models.TridharmaPenelitian) (models.TridharmaPenelitian, error)
	UpdateTridharmaPenelitian(ctx context.Context, actorID, id string, updates map[string]any) (models.TridharmaPenelitian, error)
	DeleteTridharmaPenelitian(ctx context.Context, actorID, id string) error

	// Tridharma Pengabdian
	ListTridharmaPengabdian(ctx context.Context, options listing.Options) ([]models.TridharmaPengabdian, int64, error)
	GetTridharmaPengabdian(ctx context.Context, id string) (models.TridharmaPengabdian, error)
	CreateTridharmaPengabdian(ctx context.Context, actorID string, item models.TridharmaPengabdian) (models.TridharmaPengabdian, error)
	UpdateTridharmaPengabdian(ctx context.Context, actorID, id string, updates map[string]any) (models.TridharmaPengabdian, error)
	DeleteTridharmaPengabdian(ctx context.Context, actorID, id string) error

	// Tridharma Penunjang
	ListTridharmaPenunjang(ctx context.Context, options listing.Options) ([]models.TridharmaPenunjang, int64, error)
	GetTridharmaPenunjang(ctx context.Context, id string) (models.TridharmaPenunjang, error)
	CreateTridharmaPenunjang(ctx context.Context, actorID string, item models.TridharmaPenunjang) (models.TridharmaPenunjang, error)
	UpdateTridharmaPenunjang(ctx context.Context, actorID, id string, updates map[string]any) (models.TridharmaPenunjang, error)
	DeleteTridharmaPenunjang(ctx context.Context, actorID, id string) error
}

// ScopeRepository adalah port resolusi cakupan unit pada pohon organisasi.
type ScopeRepository interface {
	ListUnitScopeIDs(context.Context, string) ([]string, error)
}
