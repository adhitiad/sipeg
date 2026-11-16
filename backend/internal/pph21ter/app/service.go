// Package app memuat logika aplikasi (use case) untuk PPh 21 TER.
package app

import (
	"context"
	"strconv"
	"strings"
	"time"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/models"
	"simpeg/backend/internal/pph21ter/domain"
	"simpeg/backend/internal/pph21ter/repository"
)

// Service menjalankan use case PPh 21 skema TER (PMK 168/2023).
type Service struct {
	repo   repository.Repository
	policy auth.Policy
}

// NewService membuat service PPh 21 TER.
func NewService(repo repository.Repository, policy auth.Policy) *Service {
	return &Service{repo: repo, policy: policy}
}

func allows(ctx context.Context, policy auth.Policy, principal auth.Principal, permission auth.Permission) bool {
	if policy == nil {
		return false
	}
	if sp, ok := policy.(auth.ScopedPolicy); ok {
		return sp.AllowsScoped(ctx, principal, permission)
	}
	return policy.Allows(principal.RoleName, permission)
}

func (s *Service) canReadReference(ctx context.Context, principal auth.Principal) bool {
	return allows(ctx, s.policy, principal, "payroll:read:any") ||
		allows(ctx, s.policy, principal, "payroll:read:homebase") ||
		allows(ctx, s.policy, principal, "payroll:manage")
}

func (s *Service) checkManage(ctx context.Context, principal auth.Principal) error {
	if !allows(ctx, s.policy, principal, "payroll:manage") {
		return domain.ErrForbidden
	}
	return nil
}

// ListTer mengembalikan seluruh baris tarif TER.
func (s *Service) ListTer(ctx context.Context, principal auth.Principal) ([]models.Pph21TerTarif, error) {
	if !s.canReadReference(ctx, principal) {
		return nil, domain.ErrForbidden
	}
	return s.repo.ListTer(ctx)
}

// ListTerOn mengembalikan baris tarif TER yang berlaku pada tanggal tertentu.
func (s *Service) ListTerOn(ctx context.Context, principal auth.Principal, asOf string) ([]models.Pph21TerTarif, error) {
	if !s.canReadReference(ctx, principal) {
		return nil, domain.ErrForbidden
	}
	return s.repo.ListTerOn(ctx, asOf)
}

// ResolveTarif mengembalikan tarif efektif (persen) untuk kategori dan
// penghasilan bruto bulanan tertentu, pada tarif berlaku terakhir sebelum asOf.
func (s *Service) ResolveTarif(ctx context.Context, principal auth.Principal, kategori string, grossMonthly float64, asOf string) (domain.TerRecord, error) {
	if !s.canReadReference(ctx, principal) {
		return domain.TerRecord{}, domain.ErrForbidden
	}
	day := asOf
	if strings.TrimSpace(day) == "" {
		day = time.Now().UTC().Format("2006-01-02")
	}
	rows, err := s.repo.ListTerOn(ctx, day)
	if err != nil {
		return domain.TerRecord{}, err
	}
	records := make([]domain.TerRecord, 0, len(rows))
	for _, row := range rows {
		records = append(records, domain.TerRecord{
			ID:           row.ID,
			Kategori:     string(row.Kategori),
			Lapisan:      row.Lapisan,
			BatasBawah:   row.BatasBawah,
			BatasAtas:    row.BatasAtas,
			TarifPct:     row.TarifPct,
			BerlakuMulai: row.BerlakuMulai,
		})
	}
	return domain.ResolveTarif(records, kategori, grossMonthly)
}

// CreateTer membuat baris tarif TER baru.
func (s *Service) CreateTer(ctx context.Context, principal auth.Principal, input domain.CreateTer) (models.Pph21TerTarif, error) {
	if err := s.checkManage(ctx, principal); err != nil {
		return models.Pph21TerTarif{}, err
	}
	if err := input.Validate(); err != nil {
		return models.Pph21TerTarif{}, err
	}
	return s.repo.CreateTer(ctx, principal.UserID, models.Pph21TerTarif{
		Kategori:     models.PtkpKategoriTER(input.Kategori),
		Lapisan:      input.Lapisan,
		BatasBawah:   input.BatasBawah,
		BatasAtas:    input.BatasAtas,
		TarifPct:     input.TarifPct,
		BerlakuMulai: input.BerlakuMulai,
	})
}

// UpdateTer menyimpan perubahan baris tarif TER.
func (s *Service) UpdateTer(ctx context.Context, principal auth.Principal, id string, patch domain.TerPatch) (models.Pph21TerTarif, error) {
	if err := s.checkManage(ctx, principal); err != nil {
		return models.Pph21TerTarif{}, err
	}
	if err := patch.Validate(); err != nil {
		return models.Pph21TerTarif{}, err
	}
	if !patch.HasChanges() {
		return models.Pph21TerTarif{}, domain.ErrValidation
	}
	updates := make(map[string]any)
	if patch.Kategori.Present {
		updates["kategori"] = patch.Kategori.Value
	}
	if patch.Lapisan.Present {
		if v, err := strconv.Atoi(strings.TrimSpace(patch.Lapisan.Value)); err == nil {
			updates["lapisan"] = v
		}
	}
	if patch.BatasBawah.Present {
		if v, err := strconv.ParseFloat(strings.TrimSpace(patch.BatasBawah.Value), 64); err == nil {
			updates["batasBawah"] = v
		}
	}
	if patch.BatasAtas.Present {
		value := strings.TrimSpace(patch.BatasAtas.Value)
		if patch.BatasAtas.Null || value == "" {
			updates["batasAtas"] = nil
		} else if v, err := strconv.ParseFloat(value, 64); err == nil {
			updates["batasAtas"] = v
		}
	}
	if patch.TarifPct.Present {
		if v, err := strconv.ParseFloat(strings.TrimSpace(patch.TarifPct.Value), 64); err == nil {
			updates["tarifPct"] = v
		}
	}
	if patch.BerlakuMulai.Present {
		if t, err := time.Parse("2006-01-02", strings.TrimSpace(patch.BerlakuMulai.Value)); err == nil {
			updates["berlakuMulai"] = t
		}
	}
	return s.repo.UpdateTer(ctx, principal.UserID, id, updates)
}

// DeleteTer menghapus satu baris tarif TER.
func (s *Service) DeleteTer(ctx context.Context, principal auth.Principal, id string) error {
	if err := s.checkManage(ctx, principal); err != nil {
		return err
	}
	return s.repo.DeleteTer(ctx, principal.UserID, id)
}
