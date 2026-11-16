// Package app memuat logika aplikasi (use case) untuk honor mengajar per
// SKS (dosen LB).
package app

import (
	"context"
	"strconv"
	"strings"
	"time"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/honorsks/domain"
	"simpeg/backend/internal/honorsks/repository"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
)

// ScopeRepository adalah port resolusi cakupan unit pada pohon organisasi.
type ScopeRepository interface {
	ListUnitScopeIDs(context.Context, string) ([]string, error)
}

// Service menjalankan use case honor mengajar per SKS.
type Service struct {
	repo   repository.Repository
	policy auth.Policy
	scope  ScopeRepository
}

// NewService membuat service honor mengajar per SKS.
func NewService(repo repository.Repository, policy auth.Policy, scope ScopeRepository) *Service {
	return &Service{repo: repo, policy: policy, scope: scope}
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

func (s *Service) scopeFor(ctx context.Context, principal auth.Principal) ([]string, error) {
	if allows(ctx, s.policy, principal, "payroll:read:any") || allows(ctx, s.policy, principal, "payroll:manage") {
		return nil, nil
	}
	if !allows(ctx, s.policy, principal, "payroll:read:homebase") {
		return nil, domain.ErrForbidden
	}
	if s.scope == nil {
		return []string{}, nil
	}
	return s.scope.ListUnitScopeIDs(ctx, principal.HomebaseID)
}

// ListRates mengembalikan seluruh tarif honor per SKS.
func (s *Service) ListRates(ctx context.Context, principal auth.Principal) ([]models.HonorSksRate, error) {
	if !s.canReadReference(ctx, principal) {
		return nil, domain.ErrForbidden
	}
	return s.repo.ListRates(ctx)
}

// CurrentRate mengembalikan tarif berlaku pada tanggal tertentu (asOf YYYY-MM-DD).
func (s *Service) CurrentRate(ctx context.Context, principal auth.Principal, asOf string) (models.HonorSksRate, error) {
	if !s.canReadReference(ctx, principal) {
		return models.HonorSksRate{}, domain.ErrForbidden
	}
	return s.repo.LatestRateOn(ctx, asOf)
}

// CreateRate membuat tarif honor per SKS baru.
func (s *Service) CreateRate(ctx context.Context, principal auth.Principal, input domain.CreateHonorRate) (models.HonorSksRate, error) {
	if err := s.checkManage(ctx, principal); err != nil {
		return models.HonorSksRate{}, err
	}
	if err := input.Validate(); err != nil {
		return models.HonorSksRate{}, err
	}
	return s.repo.CreateRate(ctx, principal.UserID, models.HonorSksRate{
		BerlakuMulai: input.BerlakuMulai,
		PerSks:       input.PerSks,
		Aktif:        true,
	})
}

// UpdateRate menyimpan perubahan tarif honor per SKS.
func (s *Service) UpdateRate(ctx context.Context, principal auth.Principal, id string, patch domain.HonorRatePatch) (models.HonorSksRate, error) {
	if err := s.checkManage(ctx, principal); err != nil {
		return models.HonorSksRate{}, err
	}
	if err := patch.Validate(); err != nil {
		return models.HonorSksRate{}, err
	}
	if !patch.HasChanges() {
		return models.HonorSksRate{}, domain.ErrValidation
	}
	updates := make(map[string]any)
	if patch.BerlakuMulai.Present {
		if t, err := time.Parse("2006-01-02", strings.TrimSpace(patch.BerlakuMulai.Value)); err == nil {
			updates["berlakuMulai"] = t
		}
	}
	if patch.PerSks.Present {
		if v, err := strconv.ParseFloat(strings.TrimSpace(patch.PerSks.Value), 64); err == nil {
			updates["perSks"] = v
		}
	}
	if patch.Aktif.Present {
		active, err := strconv.ParseBool(patch.Aktif.Value)
		if err != nil {
			active = !strings.EqualFold(patch.Aktif.Value, "false")
		}
		updates["aktif"] = active
	}
	return s.repo.UpdateRate(ctx, principal.UserID, id, updates)
}

// ListPeriode mengembalikan baris beban mengajar sesuai cakupan principal.
//
// Pegawai hanya melihat baris miliknya; admin melihat seluruhnya; operator
// melihat baris untuk pegawai pada unit penugasannya beserta turunannya.
func (s *Service) ListPeriode(ctx context.Context, principal auth.Principal, options listing.Options) (listing.Page[models.HonorSksPeriode], error) {
	switch {
	case allows(ctx, s.policy, principal, "payroll:read:self") && !allows(ctx, s.policy, principal, "payroll:read:any"):
		options.UserID = principal.UserID
	case allows(ctx, s.policy, principal, "payroll:read:any") || allows(ctx, s.policy, principal, "payroll:manage"):
		// seluruh baris
	case allows(ctx, s.policy, principal, "payroll:read:homebase"):
		ids, err := s.scopeFor(ctx, principal)
		if err != nil {
			return listing.Page[models.HonorSksPeriode]{}, err
		}
		if len(ids) == 0 {
			return listing.EmptyPage[models.HonorSksPeriode](options.Page, options.PageSize), nil
		}
		options.ScopeHomebaseIDs = ids
	default:
		return listing.Page[models.HonorSksPeriode]{}, domain.ErrForbidden
	}
	rows, total, err := s.repo.ListPeriode(ctx, options)
	if err != nil {
		return listing.Page[models.HonorSksPeriode]{}, err
	}
	return listing.Page[models.HonorSksPeriode]{
		Items:      rows,
		Page:       options.Page,
		PageSize:   options.PageSize,
		TotalItems: total,
		TotalPages: listing.PageCount(total, options.PageSize),
	}, nil
}

// GetPeriode mengembalikan satu baris beban mengajar sesuai cakupan principal.
func (s *Service) GetPeriode(ctx context.Context, principal auth.Principal, id string) (models.HonorSksPeriode, error) {
	row, err := s.repo.GetPeriode(ctx, id)
	if err != nil {
		return models.HonorSksPeriode{}, err
	}
	if allows(ctx, s.policy, principal, "payroll:read:any") || allows(ctx, s.policy, principal, "payroll:manage") {
		return row, nil
	}
	if allows(ctx, s.policy, principal, "payroll:read:self") && row.UserId == principal.UserID {
		return row, nil
	}
	if allows(ctx, s.policy, principal, "payroll:read:homebase") {
		ids, err := s.scopeFor(ctx, principal)
		if err != nil {
			return models.HonorSksPeriode{}, err
		}
		if len(ids) > 0 {
			page, _, err := s.repo.ListPeriode(ctx, listing.Options{
				Page: 1, PageSize: 1, UserID: row.UserId, ScopeHomebaseIDs: ids,
			})
			if err == nil && len(page) > 0 {
				return row, nil
			}
		}
	}
	return models.HonorSksPeriode{}, domain.ErrForbidden
}

// CreatePeriode membuat baris beban mengajar baru.
func (s *Service) CreatePeriode(ctx context.Context, principal auth.Principal, input domain.CreateHonorPeriode) (models.HonorSksPeriode, error) {
	if err := s.checkManage(ctx, principal); err != nil {
		return models.HonorSksPeriode{}, err
	}
	if err := input.Validate(); err != nil {
		return models.HonorSksPeriode{}, err
	}
	return s.repo.CreatePeriode(ctx, principal.UserID, models.HonorSksPeriode{
		UserId:            input.UserId,
		PeriodeAkademikId: input.PeriodeAkademikId,
		Sks:               input.Sks,
		RatePerSks:        input.RatePerSks,
		TotalHonor:        domain.TotalHonor(input.Sks, input.RatePerSks),
		SkId:              input.SkId,
	})
}

// UpdatePeriode menyimpan perubahan beban mengajar.
func (s *Service) UpdatePeriode(ctx context.Context, principal auth.Principal, id string, patch domain.HonorPeriodePatch) (models.HonorSksPeriode, error) {
	if err := s.checkManage(ctx, principal); err != nil {
		return models.HonorSksPeriode{}, err
	}
	if err := patch.Validate(); err != nil {
		return models.HonorSksPeriode{}, err
	}
	if !patch.HasChanges() {
		return models.HonorSksPeriode{}, domain.ErrValidation
	}
	updates := make(map[string]any)
	if patch.Sks.Present {
		if v, err := strconv.ParseFloat(strings.TrimSpace(patch.Sks.Value), 64); err == nil {
			updates["sks"] = v
		}
	}
	if patch.RatePerSks.Present {
		if v, err := strconv.ParseFloat(strings.TrimSpace(patch.RatePerSks.Value), 64); err == nil {
			updates["ratePerSks"] = v
		}
	}
	if patch.SkId.Present {
		if patch.SkId.Null || patch.SkId.Value == "" {
			updates["skId"] = nil
		} else {
			updates["skId"] = patch.SkId.Value
		}
	}
	return s.repo.UpdatePeriode(ctx, principal.UserID, id, updates)
}

// DeletePeriode menghapus baris beban mengajar.
func (s *Service) DeletePeriode(ctx context.Context, principal auth.Principal, id string) error {
	if err := s.checkManage(ctx, principal); err != nil {
		return err
	}
	return s.repo.DeletePeriode(ctx, principal.UserID, id)
}
