package app

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/models"
	payrollDomain "simpeg/backend/internal/payroll/domain"

	. "simpeg/backend/internal/payroll/app"
)

// fakeGenerateRepository adalah GenerateRepository sintetis untuk uji mesin slip.
type fakeGenerateRepository struct {
	ptkp         string
	ptkpErr      error
	homebaseID   *string
	kategoriID   string
	kategoriKode string
	alpaCount    int64
	salary       float64
	salaryErr    error
	periode      models.PeriodeAkademik
	periodeErr   error
	sakitPersen  float64
	sakitBulanKe int
	sakitFound   bool
	honor        models.HonorSksPeriode
	honorFound   bool
	tunjangan    []models.KomponenTunjangan
	potongan     []models.KomponenPotongan
	bpjs         []models.BpjsKonfigurasi
	ter          []models.Pph21TerTarif

	aktifPtr      *bool
	pemberhentian *models.Pemberhentian

	upserted       models.Gaji
	upsertPotongan []models.GajiPotongan
	upsertNote     *string
	upsertErr      error
}

func (r *fakeGenerateRepository) GetEmployeeStatusPtkp(context.Context, string) (string, error) {
	return r.ptkp, r.ptkpErr
}

func (r *fakeGenerateRepository) GetEmployeeAktif(context.Context, string) (*bool, error) {
	return r.aktifPtr, nil
}

func (r *fakeGenerateRepository) ApprovedPemberhentianFor(context.Context, string) (*models.Pemberhentian, error) {
	return r.pemberhentian, nil
}

func (r *fakeGenerateRepository) GetEmployeeHomebaseID(context.Context, string) (*string, error) {
	return r.homebaseID, nil
}

func (r *fakeGenerateRepository) GetEmployeeKategoriStatus(context.Context, string) (string, string, error) {
	return r.kategoriID, r.kategoriKode, nil
}

func (r *fakeGenerateRepository) LookupGajiPokok(context.Context, string, int, time.Time) (models.SkalaGaji, models.SkalaGajiDetail, error) {
	if r.salaryErr != nil {
		return models.SkalaGaji{}, models.SkalaGajiDetail{}, r.salaryErr
	}
	return models.SkalaGaji{}, models.SkalaGajiDetail{Nominal: r.salary}, nil
}

func (r *fakeGenerateRepository) FindPeriodeAkademik(context.Context, time.Time, time.Time) (models.PeriodeAkademik, error) {
	return r.periode, r.periodeErr
}

func (r *fakeGenerateRepository) GetPeriodeAkademikByID(context.Context, string) (models.PeriodeAkademik, error) {
	return r.periode, r.periodeErr
}

func (r *fakeGenerateRepository) ListKomponenTunjangan(context.Context, string) ([]models.KomponenTunjangan, error) {
	return r.tunjangan, nil
}

func (r *fakeGenerateRepository) ListKomponenPotongan(context.Context, string) ([]models.KomponenPotongan, error) {
	return r.potongan, nil
}

func (r *fakeGenerateRepository) LookupHonor(context.Context, string, string) (models.HonorSksPeriode, error) {
	if !r.honorFound {
		return models.HonorSksPeriode{}, payrollDomain.ErrNotFound
	}
	return r.honor, nil
}

func (r *fakeGenerateRepository) LatestBpjsConfig(context.Context, time.Time) ([]models.BpjsKonfigurasi, error) {
	return r.bpjs, nil
}

func (r *fakeGenerateRepository) TerOn(context.Context, time.Time) ([]models.Pph21TerTarif, error) {
	return r.ter, nil
}

func (r *fakeGenerateRepository) CountPresensiAlpa(context.Context, string, time.Time, time.Time) (int64, error) {
	return r.alpaCount, nil
}

func (r *fakeGenerateRepository) PeriodeSakitFor(context.Context, string, time.Time) (float64, int, bool, error) {
	if !r.sakitFound {
		return 100, 0, false, nil
	}
	return r.sakitPersen, r.sakitBulanKe, true, nil
}

func (r *fakeGenerateRepository) UpsertSlip(_ context.Context, _ string, slip models.Gaji, tunjangan []models.GajiTunjangan, potongan []models.GajiPotongan, _ *string, auditNote *string) (models.Gaji, error) {
	if r.upsertErr != nil {
		return models.Gaji{}, r.upsertErr
	}
	r.upserted = slip
	r.upsertPotongan = potongan
	r.upsertNote = auditNote
	slip.ID = "slip-generated"
	slip.NomorSlip = "SL-2026-06-0001"
	return slip, nil
}

// TerOnRows membantu uji dengan TER sintetis (lapisan tunggal 0%).
func TerOnRows() []models.Pph21TerTarif {
	return []models.Pph21TerTarif{
		{Kategori: "A", Lapisan: 1, BatasBawah: 0, BatasAtas: nil, TarifPct: 0, BerlakuMulai: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Kategori: "B", Lapisan: 1, BatasBawah: 0, BatasAtas: nil, TarifPct: 0, BerlakuMulai: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
}

func generatePolicy() auth.Policy {
	return auth.StaticPolicy{
		"admin":   {"payroll:manage": true},
		"pegawai": {},
	}
}

func generateInput() GenerateSlipInput {
	return GenerateSlipInput{
		UserID:           "user-1",
		Periode:          "2026-06",
		KategoriStatusID: "d0000000-0000-0000-0000-000000000004",
		Tingkat:          1,
	}
}

func principal() auth.Principal {
	return auth.Principal{UserID: "admin-1", RoleName: "admin"}
}

func potonganByJenis(items []models.GajiPotongan, jenis string) (models.GajiPotongan, bool) {
	for _, item := range items {
		if item.Jenis == jenis {
			return item, true
		}
	}
	return models.GajiPotongan{}, false
}

func TestGenerateSlipMemotongAlpa(t *testing.T) {
	homebase := "e0000000-0000-0000-0000-000000000001"
	fake := &fakeGenerateRepository{
		ptkp:         "TK/0",
		homebaseID:   &homebase,
		kategoriKode: "TENDIK_TETAP",
		alpaCount:    3,
		salary:       5000000,
		periodeErr:   payrollDomain.ErrNotFound, // periode akademik tidak memuat periode gaji: honor diabaikan
		ter:          TerOnRows(),
	}
	service := NewGenerateSlipService(fake, generatePolicy())
	slip, err := service.GenerateSlip(context.Background(), principal(), generateInput())
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if slip.ID != "slip-generated" {
		t.Fatalf("slip id = %q", slip.ID)
	}
	alpa, found := potonganByJenis(fake.upsertPotongan, payrollDomain.JenisPotonganAlpa)
	if !found {
		t.Fatal("komponen ALPA tidak ditemukan pada potongan slip")
	}
	if alpa.Nominal != 75000 {
		t.Fatalf("potongan ALPA = %v, want 75000 (3 hari x 25000)", alpa.Nominal)
	}
}

func TestGenerateSlipMemotongAlpaSampaiBatasMaksimal(t *testing.T) {
	homebase := "e0000000-0000-0000-0000-000000000001"
	fake := &fakeGenerateRepository{
		ptkp:         "TK/0",
		homebaseID:   &homebase,
		kategoriKode: "TENDIK_KONTRAK",
		alpaCount:    40,
		salary:       5000000,
		periodeErr:   payrollDomain.ErrNotFound,
		ter:          TerOnRows(),
	}
	service := NewGenerateSlipService(fake, generatePolicy())
	if _, err := service.GenerateSlip(context.Background(), principal(), generateInput()); err != nil {
		t.Fatalf("generate: %v", err)
	}
	alpa, found := potonganByJenis(fake.upsertPotongan, payrollDomain.JenisPotonganAlpa)
	if !found {
		t.Fatal("komponen ALPA tidak ditemukan pada potongan slip")
	}
	if alpa.Nominal != 750000 {
		t.Fatalf("potongan ALPA = %v, want 750000 (batas maksimal)", alpa.Nominal)
	}
}

func TestGenerateSlipTidakMemotongAlpaUntukKategoriTidakTercakup(t *testing.T) {
	homebase := "e0000000-0000-0000-0000-000000000001"
	for _, kode := range []string{"DOSEN_LB", "DOSEN_DPK", ""} {
		fake := &fakeGenerateRepository{
			ptkp:         "TK/0",
			homebaseID:   &homebase,
			kategoriKode: kode,
			alpaCount:    40,
			salary:       5000000,
			periodeErr:   payrollDomain.ErrNotFound,
			ter:          TerOnRows(),
		}
		service := NewGenerateSlipService(fake, generatePolicy())
		if _, err := service.GenerateSlip(context.Background(), principal(), generateInput()); err != nil {
			t.Fatalf("generate (%q): %v", kode, err)
		}
		if _, found := potonganByJenis(fake.upsertPotongan, payrollDomain.JenisPotonganAlpa); found {
			t.Fatalf("kategori %q tidak boleh tercakup potongan alpa", kode)
		}
	}
}

func TestGenerateSlipTanpaAlpaTidakMenambahkanKomponen(t *testing.T) {
	homebase := "e0000000-0000-0000-0000-000000000001"
	fake := &fakeGenerateRepository{
		ptkp:         "TK/0",
		homebaseID:   &homebase,
		kategoriKode: "DOSEN_TETAP_YAYASAN",
		alpaCount:    0,
		salary:       5000000,
		periodeErr:   payrollDomain.ErrNotFound,
		ter:          TerOnRows(),
	}
	service := NewGenerateSlipService(fake, generatePolicy())
	if _, err := service.GenerateSlip(context.Background(), principal(), generateInput()); err != nil {
		t.Fatalf("generate: %v", err)
	}
	if _, found := potonganByJenis(fake.upsertPotongan, payrollDomain.JenisPotonganAlpa); found {
		t.Fatal("komponen ALPA tidak boleh ada bila tidak ada hari alpa")
	}
}

func TestGenerateSlipSakitBerkepanjanganMemotongGajiPokokSesuaiPersenBerjalan(t *testing.T) {
	homebase := "e0000000-0000-0000-0000-000000000001"
	fake := &fakeGenerateRepository{
		ptkp:         "TK/0",
		homebaseID:   &homebase,
		kategoriKode: "TENDIK_TETAP",
		salary:       4_000_000,
		periodeErr:   payrollDomain.ErrNotFound,
		ter:          TerOnRows(),
		sakitPersen:  75,
		sakitBulanKe: 6,
		sakitFound:   true,
	}
	service := NewGenerateSlipService(fake, generatePolicy())
	slip, err := service.GenerateSlip(context.Background(), principal(), generateInput())
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if slip.GajiPokok != 3_000_000 {
		t.Fatalf("gaji pokok = %v, want 3000000 (75%% x 4000000)", slip.GajiPokok)
	}
	if fake.upsertNote == nil || !strings.Contains(*fake.upsertNote, "sakit berkepanjangan bulan ke-6") {
		t.Fatalf("auditNote = %v, want mengandung keterangan sakit berkepanjangan", fake.upsertNote)
	}
}

func TestGenerateSlipTanpaPeriodeSakitGajiPokokPenuh(t *testing.T) {
	homebase := "e0000000-0000-0000-0000-000000000001"
	fake := &fakeGenerateRepository{
		ptkp:         "TK/0",
		homebaseID:   &homebase,
		kategoriKode: "TENDIK_TETAP",
		salary:       4_000_000,
		periodeErr:   payrollDomain.ErrNotFound,
		ter:          TerOnRows(),
	}
	service := NewGenerateSlipService(fake, generatePolicy())
	slip, err := service.GenerateSlip(context.Background(), principal(), generateInput())
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if slip.GajiPokok != 4_000_000 {
		t.Fatalf("gaji pokok = %v, want 4000000", slip.GajiPokok)
	}
	if fake.upsertNote != nil {
		t.Fatalf("auditNote = %v, want nil", fake.upsertNote)
	}
}

func TestGenerateSlipMenolakPegawai(t *testing.T) {
	fake := &fakeGenerateRepository{ptkpErr: errors.New("tidak boleh dipanggil")}
	service := NewGenerateSlipService(fake, generatePolicy())
	_, err := service.GenerateSlip(context.Background(), auth.Principal{UserID: "user-1", RoleName: "pegawai"}, generateInput())
	if !errors.Is(err, payrollDomain.ErrForbidden) {
		t.Fatalf("error = %v, want ErrForbidden", err)
	}
}
