// Package app memuat use case THR: CRUD konfigurasi dan generate slip THR
// (`DEC-083`/`DEC-070`). Sumber nominal hanya dari slip TERBIT/TERBAYAR terakhir —
// tidak ada perhitungan ulang dari skala; pegawai tanpa slip TERBIT dilewati
// dan dilaporkan.
package app

import (
	"context"
	"strings"
	"time"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/models"

	"simpeg/backend/internal/thr/domain"
	"simpeg/backend/internal/thr/repository"
)

// Store adalah port persistensi yang dibutuhkan use case.
type Store interface {
	List(ctx context.Context) ([]models.ThrKonfigurasi, error)
	GetByTahun(ctx context.Context, tahun int) (models.ThrKonfigurasi, error)
	Create(ctx context.Context, actorID string, item models.ThrKonfigurasi) (models.ThrKonfigurasi, error)
	Update(ctx context.Context, actorID string, tahun int, item models.ThrKonfigurasi) (models.ThrKonfigurasi, error)
	Delete(ctx context.Context, actorID string, tahun int) error
	Candidates(ctx context.Context) ([]repository.Candidate, error)
	CreateSlip(ctx context.Context, actor string, unitID *string, tahun int, berlakunya time.Time, userID string, nominal float64, sumber string) (models.Gaji, error)
}

// Service menjalankan use case THR.
type Service struct {
	repo   Store
	policy auth.Policy
}

// NewService membuat Service.
func NewService(repo Store, policy auth.Policy) *Service {
	return &Service{repo: repo, policy: policy}
}

// Config adalah representasi konfigurasi THR untuk transport.
type Config = models.ThrKonfigurasi

// ConfigInput memuat isian konfigurasi THR.
type ConfigInput struct {
	Tahun                int
	BesaranTipe          string
	Persen               *float64
	SyaratMasaKerjaBulan int
	BerlakuTanggal       time.Time
}

// SkippedEntry menandai pegawai yang tidak menerima THR dan sebabnya.
type SkippedEntry struct {
	UserID string `json:"userId"`
	Nama   string `json:"nama"`
	Reason string `json:"reason"`
}

// PlanItem adalah satu kandidat THR beserta nominal yang dihitung otomatis.
// Admin dapat mengganti nominal (prorata) sebelum slip dibuat.
type PlanItem struct {
	UserID          string  `json:"userId"`
	Nama            string  `json:"nama"`
	MasaKerjaTahun  int     `json:"masaKerjaTahun"`
	NominalOtomatis float64 `json:"nominalOtomatis"`
	Sumber          string  `json:"sumber"`
}

// Plan adalah rencana THR untuk satu tahun: kandidat yang memenuhi syarat
// beserta nominal otomatis, dan pegawai yang dilewati beserta sebabnya.
type Plan struct {
	Tahun          int            `json:"tahun"`
	BesaranTipe    string         `json:"besaranTipe"`
	BerlakuTanggal string         `json:"berlakuTanggal"`
	Items          []PlanItem     `json:"items"`
	Lewat          []SkippedEntry `json:"lewat"`
}

// GenerateInput memuat tahun dan override nominal per pegawai (prorata yang
// diinput admin).
type GenerateInput struct {
	Tahun     int
	Overrides map[string]float64
}

// CreatedEntry menandai slip THR yang berhasil dibuat.
type CreatedEntry struct {
	UserID  string  `json:"userId"`
	Nama    string  `json:"nama"`
	Nominal float64 `json:"nominal"`
	Sumber  string  `json:"sumber"`
}

// GenerateReport memuat hasil pembuatan THR.
type GenerateReport struct {
	Tahun          int            `json:"tahun"`
	BesaranTipe    string         `json:"besaranTipe"`
	BerlakuTanggal string         `json:"berlakuTanggal"`
	Generated      int            `json:"generated"`
	Created        []CreatedEntry `json:"created"`
	Skipped        []SkippedEntry `json:"skipped"`
}

// List mengembalikan seluruh konfigurasi.
func (s *Service) List(ctx context.Context, principal auth.Principal) ([]models.ThrKonfigurasi, error) {
	if !allows(ctx, s.policy, principal, "thr:manage") {
		return nil, domain.ErrForbidden
	}
	return s.repo.List(ctx)
}

// Get mengembalikan satu konfigurasi tahun tersebut.
func (s *Service) Get(ctx context.Context, principal auth.Principal, tahun int) (models.ThrKonfigurasi, error) {
	if !allows(ctx, s.policy, principal, "thr:manage") {
		return models.ThrKonfigurasi{}, domain.ErrForbidden
	}
	return s.repo.GetByTahun(ctx, tahun)
}

// Upsert menulis atau mengganti konfigurasi (unik per tahun).
func (s *Service) Upsert(ctx context.Context, principal auth.Principal, input ConfigInput) (models.ThrKonfigurasi, error) {
	if !allows(ctx, s.policy, principal, "thr:manage") {
		return models.ThrKonfigurasi{}, domain.ErrForbidden
	}
	input.BesaranTipe = strings.TrimSpace(strings.ToUpper(input.BesaranTipe))
	if err := domain.ValidateConfig(input.Tahun, input.BesaranTipe, input.Persen, input.SyaratMasaKerjaBulan, input.BerlakuTanggal); err != nil {
		return models.ThrKonfigurasi{}, err
	}
	item := models.ThrKonfigurasi{
		Tahun:                input.Tahun,
		BesaranTipe:          input.BesaranTipe,
		Persen:               input.Persen,
		SyaratMasaKerjaBulan: input.SyaratMasaKerjaBulan,
		BerlakuTanggal:       input.BerlakuTanggal,
	}
	if existing, err := s.repo.GetByTahun(ctx, input.Tahun); err == nil {
		item.ID = existing.ID
		return s.repo.Update(ctx, principal.UserID, input.Tahun, item)
	}
	return s.repo.Create(ctx, principal.UserID, item)
}

// Delete menghapus konfigurasi tahun tersebut.
func (s *Service) Delete(ctx context.Context, principal auth.Principal, tahun int) error {
	if !allows(ctx, s.policy, principal, "thr:manage") {
		return domain.ErrForbidden
	}
	return s.repo.Delete(ctx, principal.UserID, tahun)
}

// Plan menyusun rencana THR untuk sebuah tahun: pegawai yang memenuhi
// syarat beserta nominal otomatis, dan pegawai yang dilewati beserta
// sebabnya. Plan tidak menuliskan apa pun — dipakai admin untuk menyesuaikan
// nominal (prorata) sebelum slip dibuat.
func (s *Service) Plan(ctx context.Context, principal auth.Principal, tahun int) (Plan, error) {
	if !allows(ctx, s.policy, principal, "thr:manage") {
		return Plan{}, domain.ErrForbidden
	}
	cfg, err := s.repo.GetByTahun(ctx, tahun)
	if err != nil {
		return Plan{}, err
	}
	candidates, err := s.repo.Candidates(ctx)
	if err != nil {
		return Plan{}, err
	}

	plan := Plan{
		Tahun:          tahun,
		BesaranTipe:    cfg.BesaranTipe,
		BerlakuTanggal: cfg.BerlakuTanggal.Format("2006-01-02"),
		Items:          []PlanItem{},
		Lewat:          []SkippedEntry{},
	}
	cutoff := domain.CutoffFor(cfg.BerlakuTanggal, cfg.SyaratMasaKerjaBulan)

	for _, candidate := range candidates {
		if !candidate.HasSlip {
			plan.Lewat = append(plan.Lewat, SkippedEntry{UserID: candidate.UserID, Nama: candidate.Nama, Reason: "no_terbit_slip"})
			continue
		}
		masaKerja := 0
		if candidate.TanggalMasuk != nil {
			masaKerja = yearsBetween(*candidate.TanggalMasuk, cfg.BerlakuTanggal)
		}
		if candidate.TanggalMasuk == nil || candidate.TanggalMasuk.UTC().After(cutoff.UTC()) {
			plan.Lewat = append(plan.Lewat, SkippedEntry{UserID: candidate.UserID, Nama: candidate.Nama, Reason: "syarat_masa_kerja"})
			continue
		}
		basis := domain.Basis{GajiPokok: candidate.GajiPokok, TotalGaji: candidate.TotalGaji, GajiBersih: candidate.GajiBersih}
		nominal, err := domain.Nominal(cfg.BesaranTipe, cfg.Persen, basis)
		if err != nil || nominal <= 0 {
			plan.Lewat = append(plan.Lewat, SkippedEntry{UserID: candidate.UserID, Nama: candidate.Nama, Reason: "nominal_nol"})
			continue
		}
		plan.Items = append(plan.Items, PlanItem{
			UserID:          candidate.UserID,
			Nama:            candidate.Nama,
			MasaKerjaTahun:  masaKerja,
			NominalOtomatis: nominal,
			Sumber:          cfg.BesaranTipe,
		})
	}
	return plan, nil
}

// Generate membuat slip THR untuk setiap pegawai aktif yang memenuhi
// syarat: tanggal masuk memenuhi syarat masa kerja (dihitung terhadap
// `berlakuTanggal`), dan memiliki slip TERBIT/TERBAYAR terakhir sebagai sumber
// nominal. Pegawai yang tidak memenuhi syarat dilaporkan pada `Skipped`.
//
// `input.Overrides` berisi nominal yang diinput admin (prorata) per pegawai;
// bila ada untuk seorang pegawai, nominal itu yang dipakai dan sumbernya
// tercatat pada jejak audit.
func (s *Service) Generate(ctx context.Context, principal auth.Principal, input GenerateInput) (GenerateReport, error) {
	if !allows(ctx, s.policy, principal, "thr:manage") {
		return GenerateReport{}, domain.ErrForbidden
	}
	tahun := input.Tahun
	cfg, err := s.repo.GetByTahun(ctx, tahun)
	if err != nil {
		return GenerateReport{}, err
	}

	candidates, err := s.repo.Candidates(ctx)
	if err != nil {
		return GenerateReport{}, err
	}

	report := GenerateReport{
		Tahun:          tahun,
		BesaranTipe:    cfg.BesaranTipe,
		BerlakuTanggal: cfg.BerlakuTanggal.Format("2006-01-02"),
		Created:        []CreatedEntry{},
		Skipped:        []SkippedEntry{},
	}
	cutoff := domain.CutoffFor(cfg.BerlakuTanggal, cfg.SyaratMasaKerjaBulan)

	for _, candidate := range candidates {
		if !candidate.HasSlip {
			report.Skipped = append(report.Skipped, SkippedEntry{UserID: candidate.UserID, Nama: candidate.Nama, Reason: "no_terbit_slip"})
			continue
		}
		if candidate.TanggalMasuk == nil || candidate.TanggalMasuk.UTC().After(cutoff.UTC()) {
			report.Skipped = append(report.Skipped, SkippedEntry{UserID: candidate.UserID, Nama: candidate.Nama, Reason: "syarat_masa_kerja"})
			continue
		}
		basis := domain.Basis{GajiPokok: candidate.GajiPokok, TotalGaji: candidate.TotalGaji, GajiBersih: candidate.GajiBersih}
		nominal, err := domain.Nominal(cfg.BesaranTipe, cfg.Persen, basis)
		if err != nil || nominal <= 0 {
			report.Skipped = append(report.Skipped, SkippedEntry{UserID: candidate.UserID, Nama: candidate.Nama, Reason: "nominal_nol"})
			continue
		}
		sumber := "otomatis"
		if override, ok := input.Overrides[candidate.UserID]; ok && override > 0 {
			nominal = override
			sumber = "input admin (prorata)"
		}
		var unitID *string
		if candidate.HomebaseID != nil && *candidate.HomebaseID != "" {
			unitID = candidate.HomebaseID
		}
		if _, err := s.repo.CreateSlip(ctx, principal.UserID, unitID, tahun, cfg.BerlakuTanggal, candidate.UserID, nominal, sumber); err != nil {
			if err == domain.ErrSlipConflict {
				report.Skipped = append(report.Skipped, SkippedEntry{UserID: candidate.UserID, Nama: candidate.Nama, Reason: "konflik_slip"})
				continue
			}
			return GenerateReport{}, err
		}
		report.Created = append(report.Created, CreatedEntry{UserID: candidate.UserID, Nama: candidate.Nama, Nominal: nominal, Sumber: sumber})
		report.Generated++
	}

	return report, nil
}

// yearsBetween menghitung tahun penuh antara dua tanggal (floor).
func yearsBetween(start, end time.Time) int {
	years := end.Year() - start.Year()
	if end.Month() < start.Month() || (end.Month() == start.Month() && end.Day() < start.Day()) {
		years--
	}
	if years < 0 {
		return 0
	}
	return years
}

// allows melaporkan apakah principal memiliki permission tertentu.
func allows(ctx context.Context, policy auth.Policy, principal auth.Principal, permission string) bool {
	if policy == nil {
		return false
	}
	if sp, ok := policy.(auth.ScopedPolicy); ok {
		return sp.AllowsScoped(ctx, principal, auth.Permission(permission))
	}
	return policy.Allows(principal.RoleName, auth.Permission(permission))
}
