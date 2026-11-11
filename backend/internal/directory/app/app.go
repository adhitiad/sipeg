// Package app memuat logika aplikasi (use case) untuk area direktori.
//
// Lapisan ini menyusun aturan biznis: ia memeriksa permission principal,
// membatasi data pada cakupan unitnya, lalu memanggil port repository. Ia
// tidak tahu HTTP maupun SQL: bentuk permintaan dan bentuk respons ditangani
// oleh lapisan transport.
package app

import (
	"context"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/directory/repository"
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
// seluruh turunan pada pohon Homebase. Cakupan kosong berarti principal tidak
// ditugaskan pada unit mana pun.
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
