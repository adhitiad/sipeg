// Package app memuat logika aplikasi (use case) untuk modul SKP & Log Aktivitas.
//
// Lapisan ini menyusun aturan biznis: memeriksa permission principal,
// membatasi data pada cakupan unitnya, lalu memanggil port repository. Ia
// tidak tahu HTTP maupun SQL.
package app

import (
	"context"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
	"simpeg/backend/internal/skp/domain"
	"simpeg/backend/internal/skp/repository"
)

// scopeResolver menyelesaikan cakupan unit seorang principal.
type scopeResolver struct {
	repo   repository.ScopeRepository
	policy auth.Policy
}

// newScopeResolver membuat resolver cakupan unit.
func newScopeResolver(repo repository.ScopeRepository, policy auth.Policy) scopeResolver {
	return scopeResolver{repo: repo, policy: policy}
}

// resolve mengembalikan cakupan unit principal: unit penugasannya beserta
// seluruh turunan pada pohon Homebase.
func (s scopeResolver) resolve(ctx context.Context, principal auth.Principal) (auth.UnitScope, error) {
	if principal.HomebaseID == "" {
		return auth.NewUnitScope("", nil), nil
	}
	if s.repo == nil {
		return auth.UnitScope{}, nil
	}
	unitIDs, err := s.repo.ListUnitScopeIDs(ctx, principal.HomebaseID)
	if err != nil {
		return auth.UnitScope{}, err
	}
	return auth.NewUnitScope(principal.HomebaseID, unitIDs), nil
}

// allows melaporkan apakah role principal memiliki permission tertentu.
func allows(policy auth.Policy, principal auth.Principal, permission auth.Permission) bool {
	return policy != nil && policy.Allows(principal.RoleName, permission)
}

// SKPService menjalankan use case SKP & Log Aktivitas: periode, target
// kinerja, log aktivitas harian, dan penilaian kinerja akhir.
type SKPService struct {
	repo   repository.Repository
	policy auth.Policy
	scope  scopeResolver
}

// NewSKPService membuat service SKP & Log Aktivitas.
func NewSKPService(repo repository.Repository, policy auth.Policy, scopeRepo repository.ScopeRepository) *SKPService {
	return &SKPService{repo: repo, policy: policy, scope: newScopeResolver(scopeRepo, policy)}
}

// ═══════════════════════════ SKP Periode ═══════════════════════════

// ListSKPPeriode mengembalikan daftar periode SKP sesuai cakupan principal.
func (s *SKPService) ListSKPPeriode(ctx context.Context, principal auth.Principal, options listing.Options) (listing.Page[models.SKPPeriode], error) {
	switch {
	case allows(s.policy, principal, "skp:periode:read:any"):
	case allows(s.policy, principal, "skp:periode:read:homebase"):
		scope, err := s.scope.resolve(ctx, principal)
		if err != nil {
			return listing.Page[models.SKPPeriode]{}, err
		}
		if scope.Empty() {
			return listing.EmptyPage[models.SKPPeriode](options.Page, options.PageSize), nil
		}
	case allows(s.policy, principal, "skp:periode:read:self"):
	default:
		return listing.Page[models.SKPPeriode]{}, domain.ErrForbidden
	}

	items, total, err := s.repo.ListSKPPeriode(ctx, options)
	if err != nil {
		return listing.Page[models.SKPPeriode]{}, err
	}
	return listing.Page[models.SKPPeriode]{
		Items:      items,
		Page:       options.Page,
		PageSize:   options.PageSize,
		TotalItems: total,
		TotalPages: listing.PageCount(total, options.PageSize),
	}, nil
}

// GetSKPPeriode mengembalikan satu periode SKP bila principal berhak.
func (s *SKPService) GetSKPPeriode(ctx context.Context, principal auth.Principal, id string) (models.SKPPeriode, error) {
	needsScope := allows(s.policy, principal, "skp:periode:read:homebase") && !allows(s.policy, principal, "skp:periode:read:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.SKPPeriode{}, err
	}

	periode, err := s.repo.GetSKPPeriode(ctx, id)
	if err != nil {
		return models.SKPPeriode{}, err
	}
	if !s.canAccessPeriode(scoped, principal, periode) {
		return models.SKPPeriode{}, domain.ErrForbidden
	}
	return periode, nil
}

// CreateSKPPeriode membuat periode SKP baru.
func (s *SKPService) CreateSKPPeriode(ctx context.Context, principal auth.Principal, periode models.SKPPeriode) (models.SKPPeriode, error) {
	if !allows(s.policy, principal, "skp:periode:manage:any") && !allows(s.policy, principal, "skp:periode:manage:homebase") {
		return models.SKPPeriode{}, domain.ErrForbidden
	}
	return s.repo.CreateSKPPeriode(ctx, principal.UserID, periode)
}

// UpdateSKPPeriode mengubah field periode SKP.
func (s *SKPService) UpdateSKPPeriode(ctx context.Context, principal auth.Principal, id string, updates map[string]any) (models.SKPPeriode, error) {
	needsScope := allows(s.policy, principal, "skp:periode:manage:homebase") && !allows(s.policy, principal, "skp:periode:manage:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.SKPPeriode{}, err
	}

	periode, err := s.repo.GetSKPPeriode(ctx, id)
	if err != nil {
		return models.SKPPeriode{}, err
	}
	if !s.canAccessPeriode(scoped, principal, periode) {
		return models.SKPPeriode{}, domain.ErrForbidden
	}
	return s.repo.UpdateSKPPeriode(ctx, principal.UserID, id, updates)
}

// DeleteSKPPeriode menghapus periode SKP.
func (s *SKPService) DeleteSKPPeriode(ctx context.Context, principal auth.Principal, id string) error {
	needsScope := allows(s.policy, principal, "skp:periode:manage:homebase") && !allows(s.policy, principal, "skp:periode:manage:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return err
	}

	periode, err := s.repo.GetSKPPeriode(ctx, id)
	if err != nil {
		return err
	}
	if !s.canAccessPeriode(scoped, principal, periode) {
		return domain.ErrForbidden
	}
	return s.repo.DeleteSKPPeriode(ctx, principal.UserID, id)
}

// ═══════════════════════════ SKP Target Kinerja ═══════════════════════════

// ListSKPTarget mengembalikan daftar target kinerja SKP sesuai cakupan.
func (s *SKPService) ListSKPTarget(ctx context.Context, principal auth.Principal, options listing.Options) (listing.Page[models.SKPTargetKinerja], error) {
	switch {
	case allows(s.policy, principal, "skp:target:read:any"):
	case allows(s.policy, principal, "skp:target:read:homebase"):
		scope, err := s.scope.resolve(ctx, principal)
		if err != nil {
			return listing.Page[models.SKPTargetKinerja]{}, err
		}
		if scope.Empty() {
			return listing.EmptyPage[models.SKPTargetKinerja](options.Page, options.PageSize), nil
		}
	case allows(s.policy, principal, "skp:target:read:self"):
		options.UserID = principal.UserID
	default:
		return listing.Page[models.SKPTargetKinerja]{}, domain.ErrForbidden
	}

	items, total, err := s.repo.ListSKPTarget(ctx, options)
	if err != nil {
		return listing.Page[models.SKPTargetKinerja]{}, err
	}
	return listing.Page[models.SKPTargetKinerja]{
		Items:      items,
		Page:       options.Page,
		PageSize:   options.PageSize,
		TotalItems: total,
		TotalPages: listing.PageCount(total, options.PageSize),
	}, nil
}

// GetSKPTarget mengembalikan satu target kinerja SKP bila principal berhak.
func (s *SKPService) GetSKPTarget(ctx context.Context, principal auth.Principal, id string) (models.SKPTargetKinerja, error) {
	needsScope := allows(s.policy, principal, "skp:target:read:homebase") && !allows(s.policy, principal, "skp:target:read:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.SKPTargetKinerja{}, err
	}

	target, err := s.repo.GetSKPTarget(ctx, id)
	if err != nil {
		return models.SKPTargetKinerja{}, err
	}
	if !s.canAccessTarget(scoped, principal, target) {
		return models.SKPTargetKinerja{}, domain.ErrForbidden
	}
	return target, nil
}

// CreateSKPTarget membuat target kinerja SKP baru.
func (s *SKPService) CreateSKPTarget(ctx context.Context, principal auth.Principal, target models.SKPTargetKinerja) (models.SKPTargetKinerja, error) {
	if !allows(s.policy, principal, "skp:target:manage:any") && !allows(s.policy, principal, "skp:target:manage:homebase") {
		return models.SKPTargetKinerja{}, domain.ErrForbidden
	}
	return s.repo.CreateSKPTarget(ctx, principal.UserID, target)
}

// UpdateSKPTarget mengubah field target kinerja SKP.
func (s *SKPService) UpdateSKPTarget(ctx context.Context, principal auth.Principal, id string, updates map[string]any) (models.SKPTargetKinerja, error) {
	needsScope := allows(s.policy, principal, "skp:target:manage:homebase") && !allows(s.policy, principal, "skp:target:manage:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.SKPTargetKinerja{}, err
	}

	target, err := s.repo.GetSKPTarget(ctx, id)
	if err != nil {
		return models.SKPTargetKinerja{}, err
	}
	if !s.canAccessTarget(scoped, principal, target) {
		return models.SKPTargetKinerja{}, domain.ErrForbidden
	}
	return s.repo.UpdateSKPTarget(ctx, principal.UserID, id, updates)
}

// DeleteSKPTarget menghapus target kinerja SKP.
func (s *SKPService) DeleteSKPTarget(ctx context.Context, principal auth.Principal, id string) error {
	needsScope := allows(s.policy, principal, "skp:target:manage:homebase") && !allows(s.policy, principal, "skp:target:manage:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return err
	}

	target, err := s.repo.GetSKPTarget(ctx, id)
	if err != nil {
		return err
	}
	if !s.canAccessTarget(scoped, principal, target) {
		return domain.ErrForbidden
	}
	return s.repo.DeleteSKPTarget(ctx, principal.UserID, id)
}
