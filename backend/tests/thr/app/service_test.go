package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/models"

	. "simpeg/backend/internal/thr/app"
	"simpeg/backend/internal/thr/domain"
	"simpeg/backend/internal/thr/repository"
)

// fakeStore adalah Store sintetis untuk uji use case THR.
type fakeStore struct {
	configs      map[int]models.ThrKonfigurasi
	candidates   []repository.Candidate
	created      []models.Gaji
	conflictIDs  map[string]bool
	createdCount int
	sources      []string
}

func (f *fakeStore) List(ctx context.Context) ([]models.ThrKonfigurasi, error) {
	items := make([]models.ThrKonfigurasi, 0, len(f.configs))
	for _, c := range f.configs {
		items = append(items, c)
	}
	return items, nil
}

func (f *fakeStore) GetByTahun(ctx context.Context, tahun int) (models.ThrKonfigurasi, error) {
	if c, ok := f.configs[tahun]; ok {
		return c, nil
	}
	return models.ThrKonfigurasi{}, domain.ErrConfigNotFound
}

func (f *fakeStore) Create(ctx context.Context, actorID string, item models.ThrKonfigurasi) (models.ThrKonfigurasi, error) {
	f.configs[item.Tahun] = item
	return item, nil
}

func (f *fakeStore) Update(ctx context.Context, actorID string, tahun int, item models.ThrKonfigurasi) (models.ThrKonfigurasi, error) {
	f.configs[tahun] = item
	return item, nil
}

func (f *fakeStore) Delete(ctx context.Context, actorID string, tahun int) error {
	if _, ok := f.configs[tahun]; !ok {
		return domain.ErrConfigNotFound
	}
	delete(f.configs, tahun)
	return nil
}

func (f *fakeStore) Candidates(ctx context.Context) ([]repository.Candidate, error) {
	return f.candidates, nil
}

func (f *fakeStore) CreateSlip(ctx context.Context, actor string, unitID *string, tahun int, berlakunya time.Time, userID string, nominal float64, sumber string) (models.Gaji, error) {
	if f.conflictIDs[userID] {
		return models.Gaji{}, domain.ErrSlipConflict
	}
	item := models.Gaji{
		UserId:      userID,
		Periode:     berlakunya,
		TanggalGaji: berlakunya,
		TotalGaji:   nominal,
		NomorSlip:   "THR-2027-0001",
		Status:      models.StatusSlipDraf,
	}
	f.created = append(f.created, item)
	f.createdCount++
	f.sources = append(f.sources, sumber)
	return item, nil
}

func testPolicy() auth.Policy { return auth.StaticPolicy{"admin": {"thr:manage": true}} }
func TestGenerate(t *testing.T) {
	berlaku := time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC)
	masukEligible := time.Date(2020, 6, 16, 0, 0, 0, 0, time.UTC)
	masukNewbie := time.Date(2027, 1, 15, 0, 0, 0, 0, time.UTC)

	store := &fakeStore{
		configs: map[int]models.ThrKonfigurasi{
			2027: {Tahun: 2027, BesaranTipe: domain.BesaranGajiKotor, SyaratMasaKerjaBulan: 3, BerlakuTanggal: berlaku},
		},
		candidates: []repository.Candidate{
			{UserID: "eligibel-penuh", TanggalMasuk: &masukEligible, GajiPokok: 9_500_000, TotalGaji: 11_000_000, HasSlip: true},
			{UserID: "fresh-yet", TanggalMasuk: &masukNewbie, GajiPokok: 9_500_000, TotalGaji: 11_000_000, HasSlip: true},
			{UserID: "tanpa-slip-terbit", TanggalMasuk: &masukEligible, HasSlip: false},
		},
		conflictIDs: map[string]bool{},
	}

	service := NewService(store, testPolicy())
	report, err := service.Generate(context.Background(), auth.Principal{UserID: "admin-1", RoleName: "admin"}, GenerateInput{Tahun: 2027})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if report.Generated != 1 {
		t.Fatalf("generated = %d, want 1", report.Generated)
	}
	if len(report.Skipped) != 2 {
		t.Fatalf("skipped = %+v, want 2 (syarat_masa_kerja + no_terbit_slip)", report.Skipped)
	}
	if len(store.created) != 1 {
		t.Fatalf("created = %d, want 1", len(store.created))
	}
	if store.created[0].TotalGaji != 11_000_000 {
		t.Fatalf("nominal = %v, want 11_000_000 (GAJI_KOTOR dari slip TERBIT terakhir)", store.created[0].TotalGaji)
	}
}

func TestGenerateKonflik(t *testing.T) {
	berlaku := time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC)
	masuk := time.Date(2020, 6, 16, 0, 0, 0, 0, time.UTC)

	store := &fakeStore{
		configs: map[int]models.ThrKonfigurasi{
			2027: {Tahun: 2027, BesaranTipe: domain.BesaranGajiKotor, SyaratMasaKerjaBulan: 3, BerlakuTanggal: berlaku},
		},
		candidates: []repository.Candidate{
			{UserID: "konfliktual", TanggalMasuk: &masuk, GajiPokok: 9_500_000, TotalGaji: 11_000_000, HasSlip: true},
		},
		conflictIDs: map[string]bool{"konfliktual": true},
	}

	service := NewService(store, testPolicy())
	report, err := service.Generate(context.Background(), auth.Principal{UserID: "admin-1", RoleName: "admin"}, GenerateInput{Tahun: 2027})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if report.Generated != 0 || len(report.Skipped) != 1 || report.Skipped[0].Reason != "konflik_slip" {
		t.Fatalf("report = %+v, want 1 conflict skipped", report)
	}
}

func TestGenerateConfigNotFound(t *testing.T) {
	store := &fakeStore{configs: map[int]models.ThrKonfigurasi{}}
	service := NewService(store, testPolicy())
	_, err := service.Generate(context.Background(), auth.Principal{UserID: "admin-1", RoleName: "admin"}, GenerateInput{Tahun: 2027})
	if !errors.Is(err, domain.ErrConfigNotFound) {
		t.Fatalf("err = %v, want ErrConfigNotFound", err)
	}
}

func TestGenerateForbidden(t *testing.T) {
	store := &fakeStore{configs: map[int]models.ThrKonfigurasi{}}
	service := NewService(store, testPolicy())
	_, err := service.Generate(context.Background(), auth.Principal{UserID: "pegawai-1", RoleName: "pegawai"}, GenerateInput{Tahun: 2027})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("err = %v, want ErrForbidden", err)
	}
}

func TestPlanMemisahkanKandidatDanYangDilewati(t *testing.T) {
	berlaku := time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC)
	masukEligible := time.Date(2020, 6, 16, 0, 0, 0, 0, time.UTC)
	masukNewbie := time.Date(2027, 1, 15, 0, 0, 0, 0, time.UTC)

	store := &fakeStore{
		configs: map[int]models.ThrKonfigurasi{
			2027: {Tahun: 2027, BesaranTipe: domain.BesaranGajiKotor, SyaratMasaKerjaBulan: 3, BerlakuTanggal: berlaku},
		},
		candidates: []repository.Candidate{
			{UserID: "eligibel", Nama: "Pegawai Eligibel", TanggalMasuk: &masukEligible, GajiPokok: 9_500_000, TotalGaji: 11_000_000, HasSlip: true},
			{UserID: "baru", Nama: "Pegawai Baru", TanggalMasuk: &masukNewbie, GajiPokok: 9_500_000, TotalGaji: 11_000_000, HasSlip: true},
			{UserID: "tanpa-slip", Nama: "Tanpa Slip", HasSlip: false},
		},
	}

	service := NewService(store, testPolicy())
	plan, err := service.Plan(context.Background(), auth.Principal{UserID: "admin-1", RoleName: "admin"}, 2027)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if len(plan.Items) != 1 || plan.Items[0].UserID != "eligibel" {
		t.Fatalf("items = %+v, want 1 (eligibel)", plan.Items)
	}
	if plan.Items[0].NominalOtomatis != 11_000_000 {
		t.Fatalf("nominal otomatis = %v, want 11_000_000", plan.Items[0].NominalOtomatis)
	}
	if plan.Items[0].MasaKerjaTahun != 6 {
		t.Fatalf("masa kerja = %d, want 6", plan.Items[0].MasaKerjaTahun)
	}
	if len(plan.Lewat) != 2 {
		t.Fatalf("lewat = %+v, want 2", plan.Lewat)
	}
	reasons := map[string]bool{}
	for _, item := range plan.Lewat {
		reasons[item.Reason] = true
	}
	if !reasons["syarat_masa_kerja"] || !reasons["no_terbit_slip"] {
		t.Fatalf("alasan lewat = %+v", reasons)
	}
}

func TestGenerateMemakaiNominalInputAdmin(t *testing.T) {
	berlaku := time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC)
	masuk := time.Date(2020, 6, 16, 0, 0, 0, 0, time.UTC)

	store := &fakeStore{
		configs: map[int]models.ThrKonfigurasi{
			2027: {Tahun: 2027, BesaranTipe: domain.BesaranGajiKotor, SyaratMasaKerjaBulan: 3, BerlakuTanggal: berlaku},
		},
		candidates: []repository.Candidate{
			{UserID: "eligibel", Nama: "Pegawai Eligibel", TanggalMasuk: &masuk, GajiPokok: 9_500_000, TotalGaji: 11_000_000, HasSlip: true},
		},
	}

	service := NewService(store, testPolicy())
	report, err := service.Generate(context.Background(), auth.Principal{UserID: "admin-1", RoleName: "admin"}, GenerateInput{
		Tahun:     2027,
		Overrides: map[string]float64{"eligibel": 5_500_000},
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if report.Generated != 1 {
		t.Fatalf("generated = %d, want 1", report.Generated)
	}
	if len(store.created) != 1 || store.created[0].TotalGaji != 5_500_000 {
		t.Fatalf("slip = %+v, want nominal 5_500_000", store.created)
	}
	if len(report.Created) != 1 || report.Created[0].Sumber != "input admin (prorata)" {
		t.Fatalf("created = %+v, want sumber input admin", report.Created)
	}
	if len(store.sources) != 1 || store.sources[0] != "input admin (prorata)" {
		t.Fatalf("sumber audit = %+v", store.sources)
	}
}
