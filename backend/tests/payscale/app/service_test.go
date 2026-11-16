package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
	"simpeg/backend/internal/payscale/app"
	"simpeg/backend/internal/payscale/domain"
	"simpeg/backend/internal/payscale/repository"
)

// fakePolicy adalah auth.Policy sintetis untuk pengujian.
type fakePolicy struct {
	grants map[string]map[string]bool
}

func (p fakePolicy) Allows(role string, permission auth.Permission) bool {
	return p.grants[role][string(permission)]
}

func adminPolicy() fakePolicy {
	return fakePolicy{grants: map[string]map[string]bool{
		"admin":    {"payroll:manage": true, "payroll:read:any": true},
		"operator": {"payroll:read:homebase": true},
		"pegawai":  {},
	}}
}

// fakeRepository adalah repository skala gaji sintetis.
type fakeRepository struct {
	skala         []models.SkalaGaji
	details       []models.SkalaGajiDetail
	kategoris     []models.KategoriStatusPegawai
	createErr     error
	createDetails int
}

func (r *fakeRepository) ListSkala(_ context.Context, options listing.Options) ([]models.SkalaGaji, int64, error) {
	return r.skala, int64(len(r.skala)), nil
}

func (r *fakeRepository) GetSkala(_ context.Context, id string) (models.SkalaGaji, error) {
	for _, skala := range r.skala {
		if skala.ID == id {
			return skala, nil
		}
	}
	return models.SkalaGaji{}, domain.ErrNotFound
}

func (r *fakeRepository) GetSkalaWithDetails(_ context.Context, id string) (models.SkalaGaji, []models.SkalaGajiDetail, error) {
	skala, err := r.GetSkala(context.Background(), id)
	if err != nil {
		return models.SkalaGaji{}, nil, err
	}
	items := []models.SkalaGajiDetail{}
	for _, detail := range r.details {
		if detail.SkalaGajiId == id {
			items = append(items, detail)
		}
	}
	return skala, items, nil
}

func (r *fakeRepository) ListKategoriStatus(_ context.Context) ([]models.KategoriStatusPegawai, error) {
	return r.kategoris, nil
}

func (r *fakeRepository) CreateSkala(_ context.Context, actorID string, skala models.SkalaGaji) (models.SkalaGaji, error) {
	if r.createErr != nil {
		return models.SkalaGaji{}, r.createErr
	}
	skala.ID = "skala-1"
	skala.CreatedAt = time.Now()
	skala.UpdatedAt = skala.CreatedAt
	r.skala = append(r.skala, skala)
	return skala, nil
}

func (r *fakeRepository) UpdateSkala(_ context.Context, actorID, id string, updates map[string]any) (models.SkalaGaji, error) {
	for index := range r.skala {
		if r.skala[index].ID == id {
			if value, ok := updates["nama"]; ok {
				r.skala[index].Nama = value.(string)
			}
			if value, ok := updates["kode"]; ok {
				r.skala[index].Kode = value.(string)
			}
			if value, ok := updates["aktif"]; ok {
				r.skala[index].Aktif = value.(bool)
			}
			return r.skala[index], nil
		}
	}
	return models.SkalaGaji{}, domain.ErrNotFound
}

func (r *fakeRepository) GetDetail(_ context.Context, id string) (models.SkalaGajiDetail, error) {
	for _, detail := range r.details {
		if detail.ID == id {
			return detail, nil
		}
	}
	return models.SkalaGajiDetail{}, domain.ErrNotFound
}

func (r *fakeRepository) ListDetails(_ context.Context, skalaID string) ([]models.SkalaGajiDetail, error) {
	items := []models.SkalaGajiDetail{}
	for _, detail := range r.details {
		if detail.SkalaGajiId == skalaID {
			items = append(items, detail)
		}
	}
	return items, nil
}

func (r *fakeRepository) CreateDetail(_ context.Context, actorID string, detail models.SkalaGajiDetail) (models.SkalaGajiDetail, error) {
	detail.ID = "detail-1"
	detail.CreatedAt = time.Now()
	detail.UpdatedAt = detail.CreatedAt
	r.details = append(r.details, detail)
	r.createDetails++
	return detail, nil
}

func (r *fakeRepository) UpdateDetail(_ context.Context, actorID, id string, updates map[string]any) (models.SkalaGajiDetail, error) {
	for index := range r.details {
		if r.details[index].ID == id {
			if value, ok := updates["tingkat"]; ok {
				switch level := value.(type) {
				case int:
					r.details[index].Tingkat = level
				case float64:
					r.details[index].Tingkat = int(level)
				}
			}
			if value, ok := updates["nominal"]; ok {
				switch nominal := value.(type) {
				case float64:
					r.details[index].Nominal = nominal
				case int:
					r.details[index].Nominal = float64(nominal)
				}
			}
			return r.details[index], nil
		}
	}
	return models.SkalaGajiDetail{}, domain.ErrNotFound
}

func (r *fakeRepository) DeleteDetail(_ context.Context, actorID, id string) error {
	for index := range r.details {
		if r.details[index].ID == id {
			r.details = append(r.details[:index], r.details[index+1:]...)
			return nil
		}
	}
	return domain.ErrNotFound
}

var _ repository.Repository = (*fakeRepository)(nil)

func principal(role string) auth.Principal {
	return auth.Principal{UserID: "user-1", Email: "uji@example.test", RoleName: role}
}

func TestListMenerimaAdminDanOperator(t *testing.T) {
	for _, role := range []string{"admin", "operator"} {
		repo := &fakeRepository{}
		service := app.NewService(repo, adminPolicy())
		if _, err := service.List(context.Background(), principal(role), listing.Options{Page: 1, PageSize: 20}); err != nil {
			t.Fatalf("role %s harusnya boleh membaca: %v", role, err)
		}
	}
}

func TestListMenolakPegawai(t *testing.T) {
	repo := &fakeRepository{}
	service := app.NewService(repo, adminPolicy())
	if _, err := service.List(context.Background(), principal("pegawai"), listing.Options{Page: 1, PageSize: 20}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("pegawai harusnya ditolak, err=%v", err)
	}
}

func TestCreateHanyaAdmin(t *testing.T) {
	repo := &fakeRepository{}
	service := app.NewService(repo, adminPolicy())

	if _, err := service.Create(context.Background(), principal("operator"), validCreate()); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("operator harusnya ditolak, err=%v", err)
	}
	created, err := service.Create(context.Background(), principal("admin"), validCreate())
	if err != nil {
		t.Fatalf("admin harusnya berhasil: %v", err)
	}
	if created.Nama != "Skala Gaji Yayasan 2027" {
		t.Fatalf("nama = %q", created.Nama)
	}
}

func TestAddDetailMemeriksaValidasi(t *testing.T) {
	repo := &fakeRepository{}
	service := app.NewService(repo, adminPolicy())

	if _, err := service.AddDetail(context.Background(), principal("admin"), domain.CreateSkalaGajiDetail{
		SkalaGajiId: "bukan-uuid", KategoriStatusId: "d0000000-0000-0000-0000-000000000001", Tingkat: 1, Nominal: 1,
	}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("harusnya ditolak validasi, err=%v", err)
	}
	if _, err := service.AddDetail(context.Background(), principal("admin"), domain.CreateSkalaGajiDetail{
		SkalaGajiId: "00000000-0000-0000-0000-000000000001", KategoriStatusId: "d0000000-0000-0000-0000-000000000001", Tingkat: 4, Nominal: 9750000,
	}); err != nil {
		t.Fatalf("harusnya berhasil: %v", err)
	}
	if len(repo.details) != 1 {
		t.Fatalf("jumlah detail = %d", len(repo.details))
	}
}

func TestGetWithDetailsMenolakPegawai(t *testing.T) {
	repo := &fakeRepository{skala: []models.SkalaGaji{{Kode: "SKJ-2027", Nama: "Skala", BerlakuMulai: time.Now(), Aktif: true}}}
	repo.skala[0].ID = "skala-1"
	service := app.NewService(repo, adminPolicy())
	if _, _, err := service.GetWithDetails(context.Background(), principal("pegawai"), "skala-1"); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("pegawai harusnya ditolak, err=%v", err)
	}
}

func validCreate() domain.CreateSkalaGaji {
	start := time.Date(2027, 4, 1, 0, 0, 0, 0, time.UTC)
	return domain.CreateSkalaGaji{Kode: "SKJ-2027", Nama: "Skala Gaji Yayasan 2027", BerlakuMulai: start}
}
