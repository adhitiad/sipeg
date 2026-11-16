package app

import (
	"context"
	"errors"
	"testing"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"

	"simpeg/backend/internal/sk/app"
	"simpeg/backend/internal/sk/domain"
	"simpeg/backend/internal/sk/repository"
)

// fakeStore adalah Store sintetis untuk uji use case SK.
type fakeStore struct {
	rows       []domain.Row
	lastScope  []string
	lastFilter domain.Filter
}

func (f *fakeStore) List(ctx context.Context, filter domain.Filter) (listing.Page[domain.Row], error) {
	f.lastFilter = filter
	f.lastScope = filter.ScopeHomebaseIDs
	return listing.Page[domain.Row]{Items: f.rows, Page: filter.Page, PageSize: filter.PageSize}, nil
}

func (f *fakeStore) Get(ctx context.Context, id string) (domain.Row, error) {
	for _, row := range f.rows {
		if row.ID == id {
			return row, nil
		}
	}
	return domain.Row{}, repository.ErrNotFound
}

// fakeScope mengembalikan unit penugasan beserta turunannya.
type fakeScope struct {
	ids []string
}

func (s fakeScope) ListUnitScopeIDs(ctx context.Context, rootID string) ([]string, error) {
	return s.ids, nil
}

func policy() auth.StaticPolicy {
	return auth.StaticPolicy{
		"admin":    {"sk:read:any": true},
		"operator": {"sk:read:homebase": true},
		"atasan":   {"sk:read:homebase": true},
		"pegawai":  {},
	}
}

func sampleRows() []domain.Row {
	unitA := "unit-a"
	unitB := "unit-b"
	return []domain.Row{
		{
			SuratKeputusan: models.SuratKeputusan{ID: "sk-1", NomorSk: "SK-PHK-2026-0001", Jenis: models.JenisSkPhk, Penerbit: models.TingkatSkYayasan},
			Nama:           "Pegawai Unit A",
			HomebaseID:     &unitA,
		},
		{
			SuratKeputusan: models.SuratKeputusan{ID: "sk-2", NomorSk: "SK-PHK-2026-0002", Jenis: models.JenisSkPhk, Penerbit: models.TingkatSkYayasan},
			Nama:           "Pegawai Unit B",
			HomebaseID:     &unitB,
		},
	}
}

func TestListSKAdminTanpaPembatasanCakupan(t *testing.T) {
	store := &fakeStore{rows: sampleRows()}
	service := app.NewService(store, policy(), fakeScope{ids: []string{"unit-a"}})
	page, err := service.List(context.Background(), auth.Principal{UserID: "admin", RoleName: "admin"}, domain.Filter{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(page.Items) != 2 {
		t.Fatalf("admin harus melihat seluruh SK, got %d", len(page.Items))
	}
	if len(store.lastScope) != 0 {
		t.Fatalf("admin tidak boleh dibatasi cakupan, scope = %v", store.lastScope)
	}
}

func TestListSKOperatorDibatasiCakupanUnit(t *testing.T) {
	store := &fakeStore{rows: sampleRows()}
	service := app.NewService(store, policy(), fakeScope{ids: []string{"unit-a", "unit-a-1"}})
	_, err := service.List(context.Background(), auth.Principal{UserID: "op", RoleName: "operator", HomebaseID: "unit-a"}, domain.Filter{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(store.lastScope) != 2 {
		t.Fatalf("operator harus dibatasi pada subtree unitnya, scope = %v", store.lastScope)
	}
}

func TestListSKOperatorTanpaHomebaseKosong(t *testing.T) {
	store := &fakeStore{rows: sampleRows()}
	service := app.NewService(store, policy(), fakeScope{ids: []string{"unit-a"}})
	page, err := service.List(context.Background(), auth.Principal{UserID: "op", RoleName: "operator"}, domain.Filter{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(page.Items) != 0 {
		t.Fatal("operator tanpa unit penugasan tidak boleh melihat SK")
	}
}

func TestListSKPegawaiDitolak(t *testing.T) {
	service := app.NewService(&fakeStore{}, policy(), fakeScope{})
	_, err := service.List(context.Background(), auth.Principal{UserID: "peg", RoleName: "pegawai"}, domain.Filter{})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("pegawai harus ditolak, got %v", err)
	}
}

func TestListSKFilterTidakSahDitolak(t *testing.T) {
	service := app.NewService(&fakeStore{}, policy(), fakeScope{})
	_, err := service.List(context.Background(), auth.Principal{UserID: "admin", RoleName: "admin"}, domain.Filter{Jenis: "NGGAWUR"})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("jenis SK tak dikenal harus ditolak, got %v", err)
	}
	_, err = service.List(context.Background(), auth.Principal{UserID: "admin", RoleName: "admin"}, domain.Filter{Penerbit: "SEKOLAH"})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("penerbit SK tak dikenal harus ditolak, got %v", err)
	}
}

func TestGetSKDiLuarCakupanDisembunyikan(t *testing.T) {
	store := &fakeStore{rows: sampleRows()}
	service := app.NewService(store, policy(), fakeScope{ids: []string{"unit-a"}})
	if _, err := service.Get(context.Background(), auth.Principal{UserID: "op", RoleName: "operator", HomebaseID: "unit-a"}, "sk-1"); err != nil {
		t.Fatalf("SK dalam cakupan harus terbaca: %v", err)
	}
	_, err := service.Get(context.Background(), auth.Principal{UserID: "op", RoleName: "operator", HomebaseID: "unit-a"}, "sk-2")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("SK di luar cakupan harus terlihat tidak ditemukan, got %v", err)
	}
}
