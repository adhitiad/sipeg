package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/models"

	. "simpeg/backend/internal/pensiun/app"
	"simpeg/backend/internal/pensiun/domain"
	"simpeg/backend/internal/pensiun/repository"
)

type fakeStore struct {
	configs         []models.PensiunKonfigurasi
	kep             []models.PensiunKepesertaan
	userExists      bool
	birth           *time.Time
	kategori        *string
	activeConfig    models.PensiunKonfigurasi
	hasActiveConfig bool
	scopeIDs        []string
	lastScope       *[]string
}

func (f *fakeStore) ListKonfigurasi(ctx context.Context, kategoriStatusID string) ([]models.PensiunKonfigurasi, error) {
	return f.configs, nil
}

func (f *fakeStore) UpsertKonfigurasi(ctx context.Context, actorID string, item models.PensiunKonfigurasi) (models.PensiunKonfigurasi, error) {
	item.ID = "00000000-0000-0000-0000-0000000000aa"
	return item, nil
}

func (f *fakeStore) UpdateKonfigurasi(ctx context.Context, actorID string, id string, item models.PensiunKonfigurasi) (models.PensiunKonfigurasi, error) {
	return item, nil
}

func (f *fakeStore) DeleteKonfigurasi(ctx context.Context, actorID string, id string) error {
	return nil
}

func (f *fakeStore) ListKepesertaan(ctx context.Context, scopeIDs []string) ([]repository.KepesertaanRow, error) {
	f.lastScope = &scopeIDs
	rows := make([]repository.KepesertaanRow, 0, len(f.kep))
	for _, k := range f.kep {
		rows = append(rows, repository.KepesertaanRow{PensiunKepesertaan: k, Nama: "Pegawai Sintetis"})
	}
	return rows, nil
}

func (f *fakeStore) ListUnitScopeIDs(ctx context.Context, rootHomebaseID string) ([]string, error) {
	return f.scopeIDs, nil
}

func (f *fakeStore) UpsertKepesertaan(ctx context.Context, actorID string, item models.PensiunKepesertaan) (models.PensiunKepesertaan, error) {
	item.ID = "00000000-0000-0000-0000-0000000000bb"
	return item, nil
}

func (f *fakeStore) UpdateKepesertaan(ctx context.Context, actorID string, id string, item models.PensiunKepesertaan) (models.PensiunKepesertaan, error) {
	return item, nil
}

func (f *fakeStore) DeactivateKepesertaan(ctx context.Context, actorID string, id string) error {
	return nil
}

func (f *fakeStore) UserBirthInfo(ctx context.Context, userID string) (*time.Time, *string, error) {
	return f.birth, f.kategori, nil
}

func (f *fakeStore) ExistsUser(ctx context.Context, userID string) (bool, error) {
	return f.userExists, nil
}

func (f *fakeStore) ActiveKonfigurasi(ctx context.Context, kategoriStatusID string, ref time.Time) (models.PensiunKonfigurasi, error) {
	if !f.hasActiveConfig {
		return models.PensiunKonfigurasi{}, repository.ErrNotFoundKonfigurasi
	}
	return f.activeConfig, nil
}

func adminPolicy() auth.StaticPolicy {
	return auth.StaticPolicy{
		"admin":    {"pensiun:manage": true},
		"operator": {"pensiun:read:homebase": true},
		"atasan":   {"pensiun:read:homebase": true},
		"pegawai":  {},
	}
}

func TestUpsertKonfigurasiTolakNonAdmin(t *testing.T) {
	service := NewService(&fakeStore{}, adminPolicy())
	_, err := service.UpsertKonfigurasi(context.Background(), auth.Principal{UserID: "u", RoleName: "pegawai"}, domain.KonfigurasiInput{
		KategoriStatusID: "k",
		BupUmur:          65,
		BerlakuMulai:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("pegawai harus ditolak, got %v", err)
	}
}

func TestUpsertKonfigurasiValidasi(t *testing.T) {
	service := NewService(&fakeStore{}, adminPolicy())
	_, err := service.UpsertKonfigurasi(context.Background(), auth.Principal{UserID: "u", RoleName: "admin"}, domain.KonfigurasiInput{
		KategoriStatusID: "k",
		BupUmur:          90,
		BerlakuMulai:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("BUP di luar 45-80 harus ditolak, got %v", err)
	}
}

func TestUpsertKepesertaanHitungTanggalPensiun(t *testing.T) {
	exists := true
	birth := time.Date(1970, 5, 1, 0, 0, 0, 0, time.UTC)
	kategori := "d0000000-0000-0000-0000-000000000001"
	store := &fakeStore{
		userExists:      exists,
		birth:           &birth,
		kategori:        &kategori,
		activeConfig:    models.PensiunKonfigurasi{BupUmur: 65},
		hasActiveConfig: true,
	}
	service := NewService(store, adminPolicy())
	got, err := service.UpsertKepesertaan(context.Background(), auth.Principal{UserID: "u", RoleName: "admin"}, domain.KepesertaanInput{
		UserID:       "e0000000-0000-0000-0000-000000000001",
		PenyediaDplk: "DPLK Bank Mandiri",
		Mulai:        time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC),
		Aktif:        true,
	})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if got.TanggalMulaiPensiun == nil {
		t.Fatal("tanggal mulai pensiun harus dihitung dari lahir + BUP")
	}
	want := birth.AddDate(65, 0, 0)
	if !got.TanggalMulaiPensiun.Equal(want) {
		t.Fatalf("tanggal pensiun = %v, inginginkan %v", got.TanggalMulaiPensiun, want)
	}
}

func TestListKepesertaanOperatorCakupanUnit(t *testing.T) {
	store := &fakeStore{
		kep: []models.PensiunKepesertaan{
			{ID: "1", UserId: "a0000000-4000-8000-9000-000000000011", PenyediaDplk: "DPLK Mandiri", Mulai: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
		scopeIDs: []string{"unit-a", "unit-b"},
	}
	service := NewService(store, adminPolicy())
	rows, err := service.ListKepesertaan(context.Background(), auth.Principal{UserID: "u", RoleName: "operator", HomebaseID: "unit-a"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if store.lastScope == nil || len(*store.lastScope) != 2 {
		t.Fatalf("operator harus dibatasi pada cakupan unitnya, scope = %v", store.lastScope)
	}
}

func TestListKepesertaanOperatorTanpaHomebaseKosong(t *testing.T) {
	store := &fakeStore{
		kep: []models.PensiunKepesertaan{
			{ID: "1", UserId: "a0000000-4000-8000-9000-000000000011", PenyediaDplk: "DPLK", Mulai: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
	}
	service := NewService(store, adminPolicy())
	rows, err := service.ListKepesertaan(context.Background(), auth.Principal{UserID: "u", RoleName: "operator"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rows) != 0 {
		t.Fatal("operator tanpa homebase tidak boleh melihat kepesertaan")
	}
}

func TestListKepesertaanPegawaiDitolak(t *testing.T) {
	service := NewService(&fakeStore{}, adminPolicy())
	_, err := service.ListKepesertaan(context.Background(), auth.Principal{UserID: "u", RoleName: "pegawai"})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("pegawai harus ditolak, got %v", err)
	}
}

func TestListKonfigurasiOperatorBolehBacaMaster(t *testing.T) {
	store := &fakeStore{
		configs: []models.PensiunKonfigurasi{
			{ID: "1", KategoriStatusId: "k", BupUmur: 65, BerlakuMulai: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
	}
	service := NewService(store, adminPolicy())
	rows, err := service.ListKonfigurasi(context.Background(), auth.Principal{UserID: "u", RoleName: "operator", HomebaseID: "unit-a"}, "")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("operator harus dapat membaca master konfigurasi pensiun, got %d", len(rows))
	}
}

func TestUpsertKepesertaanPegawaiTidakAda(t *testing.T) {
	service := NewService(&fakeStore{userExists: false}, adminPolicy())
	_, err := service.UpsertKepesertaan(context.Background(), auth.Principal{UserID: "u", RoleName: "admin"}, domain.KepesertaanInput{
		UserID:       "e0000000-0000-0000-0000-000000000009",
		PenyediaDplk: "DPLK Bank Mandiri",
		Mulai:        time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC),
		Aktif:        true,
	})
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("pegawai tidak ada harus ditolak, got %v", err)
	}
}

func TestUpsertKepesertaanTanpaBirthInfoTetapTersimpan(t *testing.T) {
	service := NewService(&fakeStore{userExists: true}, adminPolicy())
	got, err := service.UpsertKepesertaan(context.Background(), auth.Principal{UserID: "u", RoleName: "admin"}, domain.KepesertaanInput{
		UserID:       "e0000000-0000-0000-0000-000000000002",
		PenyediaDplk: "DPLK Bank Mandiri",
		Mulai:        time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC),
		Aktif:        true,
	})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if got.TanggalMulaiPensiun != nil {
		t.Fatalf("tanpa data lahir/kategori tanggal pensiun harus kosong, got %v", got.TanggalMulaiPensiun)
	}
}
