// Package app memuat use case penggajian inti: pembuatan slip otomatis.
package app

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/calculasi/domain"
	"simpeg/backend/internal/models"
	payrollDomain "simpeg/backend/internal/payroll/domain"
	"simpeg/backend/internal/payroll/repository"
)

// GenerateSlipService menjalankan use case pembuatan slip otomatis.
type GenerateSlipService struct {
	repo   repository.GenerateRepository
	policy auth.Policy
}

// NewGenerateSlipService membuat service pembuatan slip otomatis.
func NewGenerateSlipService(repo repository.GenerateRepository, policy auth.Policy) *GenerateSlipService {
	return &GenerateSlipService{repo: repo, policy: policy}
}

// GenerateSlipInput memuat input pembuatan slip. KategoriStatusID dan Tingkat
// boleh kosong; bila diisi, gaji pokok diambil dari skala efektif.
type GenerateSlipInput struct {
	UserID           string
	Periode          string
	KategoriStatusID string
	Tingkat          int
}

// GenerateSlip membuat atau mengganti slip DRAF milik pegawai pada periode
// tertentu, lalu menyimpan komponennya (penerimaan dan potongan) dalam satu
// transaksi.
func (s *GenerateSlipService) GenerateSlip(ctx context.Context, principal auth.Principal, input GenerateSlipInput) (models.Gaji, error) {
	if !allows(ctx, s.policy, principal, "payroll:manage") {
		return models.Gaji{}, payrollDomain.ErrForbidden
	}

	period, err := payrollDomain.ParsePeriod(input.Periode)
	if err != nil {
		return models.Gaji{}, payrollDomain.ErrValidation
	}

	// Pemberhentian menonaktifkan pegawai (`DEC-083`): slip hanya boleh
	// dibuat untuk periode yang seusia periode pemberhentian atau lebih
	// lama (pegawai masih bekerja pada periode itu). Pada bulan
	// pemberhentian, kompensasi pemberhentian (pesangon = bulan × gaji
	// pokok berjalan) ikut ditampilkan sebagai komponen PESANGON pada
	// slip dan dikenakan tarif TER bulan berjalan.
	pesangonBulan := 0.0
	pesangonNote := ""
	pesangonNominal := 0.0
	if aktif, err := s.repo.GetEmployeeAktif(ctx, input.UserID); err != nil {
		return models.Gaji{}, err
	} else if aktif != nil && !*aktif {
		pem, err := s.repo.ApprovedPemberhentianFor(ctx, input.UserID)
		if err != nil {
			return models.Gaji{}, err
		}
		if pem == nil {
			return models.Gaji{}, payrollDomain.ErrPegawaiTidakAktif
		}
		periodStart := time.Date(period.Year, period.Month, 1, 0, 0, 0, 0, time.UTC)
		if pem.Tanggal.UTC().Before(periodStart) {
			return models.Gaji{}, payrollDomain.ErrPegawaiTidakAktif
		}
		if pem.Tanggal.UTC().Year() == period.Year && pem.Tanggal.UTC().Month() == period.Month && pem.KompensasiBulan > 0 {
			pesangonBulan = pem.KompensasiBulan
			pesangonNote = fmt.Sprintf("pesangon pemberhentian: %.0f bulan tercatat pada slip ini", pem.KompensasiBulan)
		}
	}

	periodEnd := time.Date(period.Year, period.Month, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 1, -1)

	ptkp, err := s.repo.GetEmployeeStatusPtkp(ctx, input.UserID)
	if err != nil {
		return models.Gaji{}, err
	}
	unitID, err := s.repo.GetEmployeeHomebaseID(ctx, input.UserID)
	if err != nil {
		return models.Gaji{}, err
	}
	_, kategoriKode, err := s.repo.GetEmployeeKategoriStatus(ctx, input.UserID)
	if err != nil {
		return models.Gaji{}, err
	}

	// Potongan alpa (DEC-076): Rp 25.000 per hari alpa tanpa keterangan,
	// maksimal Rp 750.000 per bulan, hanya untuk kategori status pegawai
	// yang tercakup aturan (dosen tetap yayasan, tendik tetap, tendik
	// kontrak).
	alpaDeduction := 0.0
	if payrollDomain.EligibleForAlpaDeduction(kategoriKode) {
		monthStart := time.Date(period.Year, period.Month, 1, 0, 0, 0, 0, time.UTC)
		nextMonth := monthStart.AddDate(0, 1, 0)
		alpaCount, err := s.repo.CountPresensiAlpa(ctx, input.UserID, monthStart, nextMonth)
		if err != nil {
			return models.Gaji{}, err
		}
		alpaDeduction = payrollDomain.AlpaDeduction(alpaCount)
	}

	salary := 0.0
	if strings.TrimSpace(input.KategoriStatusID) != "" {
		_, detail, err := s.repo.LookupGajiPokok(ctx, input.KategoriStatusID, input.Tingkat, periodEnd)
		if err != nil {
			return models.Gaji{}, err
		}
		salary = detail.Nominal
	}

	// Cuti sakit berkepanjangan (DEC-079): gaji pokok dibayar sesuai persen
	// berjalan dari PeriodeSakit yang mencakup periode slip (bulan
	// 5–8 = 75%, dst.). Potongan berbasis persen gaji pokok, BPJS, dan
	// perhitungan PPh 21 TER mengikuti nominal yang berjalan.
	sakitNote := ""
	if pct, bulanKe, ok, err := s.repo.PeriodeSakitFor(ctx, input.UserID, periodEnd); err != nil {
		return models.Gaji{}, err
	} else if ok && pct < 100 {
		salary = math.Round(salary * pct / 100)
		sakitNote = fmt.Sprintf("sakit berkepanjangan bulan ke-%d; gaji pokok dibayar %.0f%%", bulanKe, pct)
	}

	honorSks := 0.0
	ratePerSks := 0.0
	periodeAkademik, err := s.repo.FindPeriodeAkademik(ctx, time.Date(period.Year, period.Month, 1, 0, 0, 0, 0, time.UTC), periodEnd)
	if err == nil {
		if honor, err := s.repo.LookupHonor(ctx, input.UserID, periodeAkademik.ID); err == nil {
			honorSks = honor.Sks
			ratePerSks = honor.RatePerSks
		}
	} else if err != payrollDomain.ErrNotFound {
		return models.Gaji{}, err
	}

	tunjanganComponents, err := s.repo.ListKomponenTunjangan(ctx, input.UserID)
	if err != nil {
		return models.Gaji{}, err
	}
	potonganComponents, err := s.repo.ListKomponenPotongan(ctx, input.UserID)
	if err != nil {
		return models.Gaji{}, err
	}

	bpjsRates, err := s.repo.LatestBpjsConfig(ctx, periodEnd)
	if err != nil {
		return models.Gaji{}, err
	}

	terRows, err := s.repo.TerOn(ctx, periodEnd)
	if err != nil {
		return models.Gaji{}, err
	}

	calculasiInput := domain.SlipInput{
		Periode:    period.Key(),
		GajiPokok:  salary,
		HonorSks:   honorSks,
		RatePerSks: ratePerSks,
		Tunjangan:  []domain.Komponen{},
		Potongan:   []domain.Komponen{},
		Bpjs:       []domain.BpjsRate{},
		Ter:        []domain.TerRow{},
		StatusPtkp: ptkp,
	}
	for _, row := range tunjanganComponents {
		calculasiInput.Tunjangan = append(calculasiInput.Tunjangan, domain.Komponen{
			Jenis:   string(row.JenisTunjangan),
			Nominal: row.Nilai,
		})
	}
	if pesangonBulan > 0 && salary > 0 {
		pesangonNominal = math.Round(pesangonBulan * salary)
		calculasiInput.Tunjangan = append(calculasiInput.Tunjangan, domain.Komponen{
			Jenis:   "PESANGON",
			Nominal: pesangonNominal,
		})
	}
	for _, row := range potonganComponents {
		nominal := row.Nominal
		if row.Nominal == 0 && row.Persentase != nil {
			nominal = (*row.Persentase / 100) * salary
		}
		calculasiInput.Potongan = append(calculasiInput.Potongan, domain.Komponen{
			Jenis:   string(row.JenisPotongan),
			Nominal: nominal,
		})
	}
	if alpaDeduction > 0 {
		calculasiInput.Potongan = append(calculasiInput.Potongan, domain.Komponen{
			Jenis:   payrollDomain.JenisPotonganAlpa,
			Nominal: alpaDeduction,
		})
	}
	for _, row := range bpjsRates {
		bpjs := domain.BpjsRate{
			Jenis:                string(row.Jenis),
			IuranPegawaiPct:      row.IuranPegawaiPct,
			IuranPemberiKerjaPct: row.IuranPemberiKerjaPct,
			Dasar:                row.Dasar,
		}
		if row.BatasAtas != nil {
			bpjs.BatasAtas = row.BatasAtas
		}
		calculasiInput.Bpjs = append(calculasiInput.Bpjs, bpjs)
	}
	for _, row := range terRows {
		calculasiInput.Ter = append(calculasiInput.Ter, domain.TerRow{
			Kategori:   string(row.Kategori),
			Lapisan:    row.Lapisan,
			BatasBawah: row.BatasBawah,
			BatasAtas:  row.BatasAtas,
			TarifPct:   row.TarifPct,
		})
	}

	result, err := domain.Calculate(calculasiInput)
	if err != nil {
		return models.Gaji{}, err
	}

	slip := models.Gaji{
		UserId:        input.UserID,
		Periode:       time.Date(period.Year, period.Month, 1, 0, 0, 0, 0, time.UTC),
		TanggalGaji:   time.Date(period.Year, period.Month, 1, 0, 0, 0, 0, time.UTC),
		GajiPokok:     result.GajiPokok,
		TotalGaji:     result.Bruto,
		Status:        models.StatusSlipDraf,
		PotonganPph21: result.Pph21,
	}

	var tunjangan []models.GajiTunjangan
	var potongan []models.GajiPotongan
	for _, komponen := range tunjanganComponents {
		tunjangan = append(tunjangan, models.GajiTunjangan{
			Jenis:   string(komponen.JenisTunjangan),
			Nominal: komponen.Nilai,
		})
	}
	if result.HonorTotal > 0 {
		tunjangan = append(tunjangan, models.GajiTunjangan{
			Jenis:   "HONOR_SKS",
			Nominal: result.HonorTotal,
		})
	}
	if pesangonNominal > 0 {
		tunjangan = append(tunjangan, models.GajiTunjangan{
			Jenis:   "PESANGON",
			Nominal: pesangonNominal,
		})
	}
	for _, komponen := range potonganComponents {
		nominal := komponen.Nominal
		if komponen.Nominal == 0 && komponen.Persentase != nil {
			nominal = (*komponen.Persentase / 100) * salary
		}
		potongan = append(potongan, models.GajiPotongan{
			Jenis:   string(komponen.JenisPotongan),
			Nominal: nominal,
		})
	}
	if alpaDeduction > 0 {
		potongan = append(potongan, models.GajiPotongan{
			Jenis:   payrollDomain.JenisPotonganAlpa,
			Nominal: alpaDeduction,
		})
	}
	if result.BpjsKesehatan > 0 {
		potongan = append(potongan, models.GajiPotongan{
			Jenis:   "BPJS_KESEHATAN",
			Nominal: result.BpjsKesehatan,
		})
	}
	if result.BpjsKetenagakerjaan > 0 {
		potongan = append(potongan, models.GajiPotongan{
			Jenis:   "BPJS_KETENAGAKERJAAN",
			Nominal: result.BpjsKetenagakerjaan,
		})
	}
	if result.Pph21 > 0 {
		potongan = append(potongan, models.GajiPotongan{
			Jenis:   "PPH21",
			Nominal: result.Pph21,
		})
	}

	var auditNote *string
	var notes []string
	if sakitNote != "" {
		notes = append(notes, sakitNote)
	}
	if pesangonNote != "" {
		notes = append(notes, pesangonNote)
	}
	if len(notes) > 0 {
		note := strings.Join(notes, "; ")
		auditNote = &note
	}
	slip, err = s.repo.UpsertSlip(ctx, principal.UserID, slip, tunjangan, potongan, unitID, auditNote)
	if err != nil {
		return models.Gaji{}, err
	}

	return slip, nil
}
