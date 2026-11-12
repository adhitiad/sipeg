package app

import (
	"context"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/models"
	"simpeg/backend/internal/tridharma/repository"
)

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

// TridharmaService menjalankan use case Tridharma & BKD: periode akademik,
// evaluasi BKD, dan kegiatan tridharma (pendidikan, penelitian, pengabdian,
// penunjang).
type TridharmaService struct {
	repo   repository.Repository
	policy auth.Policy
	scope  scopeResolver
}

// NewTridharmaService membuat service Tridharma & BKD.
func NewTridharmaService(repo repository.Repository, policy auth.Policy, scopeRepo repository.ScopeRepository) *TridharmaService {
	return &TridharmaService{repo: repo, policy: policy, scope: newScopeResolver(scopeRepo, policy)}
}

// scopedPrincipal menyelesaikan cakupan unit principal bila dibutuhkan.
func (s *TridharmaService) scopedPrincipal(ctx context.Context, principal auth.Principal, needsScope bool) (auth.UnitScope, error) {
	if !needsScope {
		return auth.UnitScope{}, nil
	}
	return s.scope.resolve(ctx, principal)
}

// canAccessPeriode memeriksa apakah principal berhak mengakses periode akademik.
func (s *TridharmaService) canAccessPeriode(scope auth.UnitScope, principal auth.Principal, periode models.PeriodeAkademik) bool {
	if allows(s.policy, principal, "tridharma:periode:read:any") || allows(s.policy, principal, "tridharma:periode:manage:any") {
		return true
	}
	return scope.Contains("")
}

// canAccessEvaluasi memeriksa apakah principal berhak mengakses evaluasi BKD.
func (s *TridharmaService) canAccessEvaluasi(scope auth.UnitScope, principal auth.Principal, evaluasi models.EvaluasiBKD) bool {
	if allows(s.policy, principal, "tridharma:evaluasi:read:any") || allows(s.policy, principal, "tridharma:evaluasi:manage:any") {
		return true
	}
	if principal.UserID == evaluasi.UserId {
		return true
	}
	return scope.Contains("")
}

// canAccessPendidikan memeriksa apakah principal berhak mengakses tridharma pendidikan.
func (s *TridharmaService) canAccessPendidikan(scope auth.UnitScope, principal auth.Principal, item models.TridharmaPendidikan) bool {
	if allows(s.policy, principal, "tridharma:pendidikan:read:any") || allows(s.policy, principal, "tridharma:pendidikan:manage:any") {
		return true
	}
	if principal.UserID == item.UserId {
		return true
	}
	return scope.Contains("")
}

// canAccessPenelitian memeriksa apakah principal berhak mengakses tridharma penelitian.
func (s *TridharmaService) canAccessPenelitian(scope auth.UnitScope, principal auth.Principal, item models.TridharmaPenelitian) bool {
	if allows(s.policy, principal, "tridharma:penelitian:read:any") || allows(s.policy, principal, "tridharma:penelitian:manage:any") {
		return true
	}
	if principal.UserID == item.UserId {
		return true
	}
	return scope.Contains("")
}

// canAccessPengabdian memeriksa apakah principal berhak mengakses tridharma pengabdian.
func (s *TridharmaService) canAccessPengabdian(scope auth.UnitScope, principal auth.Principal, item models.TridharmaPengabdian) bool {
	if allows(s.policy, principal, "tridharma:pengabdian:read:any") || allows(s.policy, principal, "tridharma:pengabdian:manage:any") {
		return true
	}
	if principal.UserID == item.UserId {
		return true
	}
	return scope.Contains("")
}

// canAccessPenunjang memeriksa apakah principal berhak mengakses tridharma penunjang.
func (s *TridharmaService) canAccessPenunjang(scope auth.UnitScope, principal auth.Principal, item models.TridharmaPenunjang) bool {
	if allows(s.policy, principal, "tridharma:penunjang:read:any") || allows(s.policy, principal, "tridharma:penunjang:manage:any") {
		return true
	}
	if principal.UserID == item.UserId {
		return true
	}
	return scope.Contains("")
}
