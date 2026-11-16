// Service baca Surat Keputusan (SK). Modul ini **baca saja**: SK terbit dari
// alur kerja (mis. SK-PHK otomatis saat persetujuan pemberhentian) dan belum
// ada alur penerbitan manual — keputusannya masih terbuka (`DEC-091`).
package app

import (
	"context"
	"errors"
	"strings"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/listing"

	"simpeg/backend/internal/sk/domain"
	"simpeg/backend/internal/sk/repository"
)

// ScopeResolver memetakan homebaseId ke pohon subtree (unit beserta
// turunannya). Diimplementasikan oleh repository direktori.
type ScopeResolver interface {
	ListUnitScopeIDs(ctx context.Context, rootID string) ([]string, error)
}

// Service menjalankan use case baca SK.
type Service struct {
	store  repository.Store
	policy auth.Policy
	scope  ScopeResolver
}

// NewService membuat Service.
func NewService(store repository.Store, policy auth.Policy, scope ScopeResolver) *Service {
	return &Service{store: store, policy: policy, scope: scope}
}

// allows melaporkan apakah principal memiliki permission tertentu. Bila
// policy mengimplementasikan auth.ScopedPolicy (PolicyAdapter RBAC), evaluasi
// scoped dipakai sehingga pencabutan RBAC berlaku juga di use case.
func allows(ctx context.Context, policy auth.Policy, principal auth.Principal, permission string) bool {
	if policy == nil {
		return false
	}
	if sp, ok := policy.(auth.ScopedPolicy); ok {
		return sp.AllowsScoped(ctx, principal, auth.Permission(permission))
	}
	return policy.Allows(principal.RoleName, auth.Permission(permission))
}

// List membaca daftar SK sesuai filter. Admin (`:any`) melihat seluruh SK;
// operator/atasan (`:homebase`) hanya SK pegawai pada cakupan unitnya.
func (s *Service) List(ctx context.Context, principal auth.Principal, filter domain.Filter) (listing.Page[domain.Row], error) {
	filter = filter.Normalize()
	switch {
	case allows(ctx, s.policy, principal, "sk:read:any"):
	case allows(ctx, s.policy, principal, "sk:read:homebase"):
		if principal.HomebaseID == "" {
			return listing.EmptyPage[domain.Row](filter.Page, filter.PageSize), nil
		}
		ids, err := s.scope.ListUnitScopeIDs(ctx, principal.HomebaseID)
		if err != nil {
			return listing.Page[domain.Row]{}, err
		}
		if len(ids) == 0 {
			return listing.EmptyPage[domain.Row](filter.Page, filter.PageSize), nil
		}
		filter.ScopeHomebaseIDs = ids
	default:
		return listing.Page[domain.Row]{}, domain.ErrForbidden
	}
	if filter.Jenis != "" && !domain.JenisTerdaftar(filter.Jenis) {
		return listing.Page[domain.Row]{}, domain.ErrValidation
	}
	if filter.Penerbit != "" && !domain.PenerbitTerdaftar(filter.Penerbit) {
		return listing.Page[domain.Row]{}, domain.ErrValidation
	}
	return s.store.List(ctx, filter)
}

// Get membaca satu SK; untuk principal ber-cakupan unit, SK di luar cakupan
// diperlakukan sebagai tidak ditemukan.
func (s *Service) Get(ctx context.Context, principal auth.Principal, id string) (domain.Row, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.Row{}, domain.ErrValidation
	}
	row, err := s.store.Get(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return domain.Row{}, domain.ErrNotFound
		}
		return domain.Row{}, err
	}
	switch {
	case allows(ctx, s.policy, principal, "sk:read:any"):
		return row, nil
	case allows(ctx, s.policy, principal, "sk:read:homebase"):
		if principal.HomebaseID == "" || row.HomebaseID == nil {
			return domain.Row{}, domain.ErrNotFound
		}
		ids, err := s.scope.ListUnitScopeIDs(ctx, principal.HomebaseID)
		if err != nil {
			return domain.Row{}, err
		}
		for _, id := range ids {
			if id == *row.HomebaseID {
				return row, nil
			}
		}
		return domain.Row{}, domain.ErrNotFound
	default:
		return domain.Row{}, domain.ErrForbidden
	}
}
