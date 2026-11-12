// Package app memuat logika aplikasi untuk komponen tunjangan pegawai.
package app

import (
	"context"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
	"simpeg/backend/internal/payroll/domain"
	"simpeg/backend/internal/payroll/repository"
)

// TunjanganService menjalankan use case komponen tunjangan pegawai.
type TunjanganService struct {
	repo   repository.TunjanganRepository
	policy auth.Policy
	scope  scopeResolver
}

// NewTunjanganService membuat service komponen tunjangan.
func NewTunjanganService(repo repository.TunjanganRepository, policy auth.Policy, scopeRepo ScopeRepository) *TunjanganService {
	return &TunjanganService{repo: repo, policy: policy, scope: newScopeResolver(scopeRepo)}
}

func (s *TunjanganService) List(ctx context.Context, principal auth.Principal, options listing.Options) (listing.Page[models.KomponenTunjangan], error) {
	switch {
	case allows(s.policy, principal, "tunjangan:read:any"):
	case allows(s.policy, principal, "tunjangan:read:homebase"):
		scope, err := s.scope.resolve(ctx, principal)
		if err != nil {
			return listing.Page[models.KomponenTunjangan]{}, err
		}
		if scope.Empty() {
			return listing.EmptyPage[models.KomponenTunjangan](options.Page, options.PageSize), nil
		}
		options.ScopeHomebaseIDs = scope.List()
	case allows(s.policy, principal, "tunjangan:read:self"):
		options.UserID = principal.UserID
	default:
		return listing.Page[models.KomponenTunjangan]{}, domain.ErrForbidden
	}
	items, total, err := s.repo.ListTunjangan(ctx, options)
	if err != nil {
		return listing.Page[models.KomponenTunjangan]{}, err
	}
	return listing.Page[models.KomponenTunjangan]{
		Items: items, Page: options.Page, PageSize: options.PageSize,
		TotalItems: total, TotalPages: listing.PageCount(total, options.PageSize),
	}, nil
}

func (s *TunjanganService) Get(ctx context.Context, principal auth.Principal, id string) (models.KomponenTunjangan, error) {
	needsScope := allows(s.policy, principal, "tunjangan:read:homebase") && !allows(s.policy, principal, "tunjangan:read:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.KomponenTunjangan{}, err
	}
	item, err := s.repo.GetTunjangan(ctx, id)
	if err != nil {
		return models.KomponenTunjangan{}, err
	}
	if !s.canRead(scoped, item) {
		return models.KomponenTunjangan{}, domain.ErrForbidden
	}
	return item, nil
}

func (s *TunjanganService) Create(ctx context.Context, principal auth.Principal, input domain.CreateKomponenTunjangan) (models.KomponenTunjangan, error) {
	if err := input.Validate(); err != nil {
		return models.KomponenTunjangan{}, err
	}
	needsScope := allows(s.policy, principal, "tunjangan:create:homebase") && !allows(s.policy, principal, "tunjangan:create")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.KomponenTunjangan{}, err
	}
	if !s.canCreate(scoped, &input.UserID) {
		return models.KomponenTunjangan{}, domain.ErrForbidden
	}
	item := models.KomponenTunjangan{
		UserId:         input.UserID,
		JenisTunjangan: models.JenisTunjangan(input.JenisTunjangan),
		Nilai:          input.Nilai,
		Aktif:          input.Aktif,
	}
	return s.repo.CreateTunjangan(ctx, principal.UserID, item)
}

func (s *TunjanganService) Update(ctx context.Context, principal auth.Principal, id string, patch domain.KomponenTunjanganPatch) (models.KomponenTunjangan, error) {
	if !patch.HasChanges() {
		return models.KomponenTunjangan{}, domain.ErrValidation
	}
	if err := patch.Validate(); err != nil {
		return models.KomponenTunjangan{}, err
	}
	needsScope := allows(s.policy, principal, "tunjangan:update:homebase") && !allows(s.policy, principal, "tunjangan:update:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.KomponenTunjangan{}, err
	}
	item, err := s.repo.GetTunjangan(ctx, id)
	if err != nil {
		return models.KomponenTunjangan{}, err
	}
	if !s.canWrite(scoped, item) {
		return models.KomponenTunjangan{}, domain.ErrForbidden
	}
	return s.repo.UpdateTunjangan(ctx, principal.UserID, id, patch)
}

func (s *TunjanganService) Archive(ctx context.Context, principal auth.Principal, id string) error {
	needsScope := allows(s.policy, principal, "tunjangan:archive:homebase") && !allows(s.policy, principal, "tunjangan:archive")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return err
	}
	item, err := s.repo.GetTunjangan(ctx, id)
	if err != nil {
		return err
	}
	if !s.canArchive(scoped, item) {
		return domain.ErrForbidden
	}
	return s.repo.ArchiveTunjangan(ctx, principal.UserID, id)
}

func (s *TunjanganService) scopedPrincipal(ctx context.Context, principal auth.Principal, needsScope bool) (scopedPrincipal, error) {
	if !needsScope {
		return scopedPrincipal{principal: principal}, nil
	}
	scope, err := s.scope.resolve(ctx, principal)
	if err != nil {
		return scopedPrincipal{}, err
	}
	return scopedPrincipal{principal: principal, scope: scope}, nil
}

func (s *TunjanganService) canRead(scoped scopedPrincipal, item models.KomponenTunjangan) bool {
	for _, permission := range []auth.Permission{"tunjangan:read:any", "tunjangan:read:homebase", "tunjangan:read:self"} {
		if auth.CanAccessUserInScope(scoped.principal, item.UserId, item.UserId, permission, s.policy, scoped.scope) {
			return true
		}
	}
	return false
}

func (s *TunjanganService) canCreate(scoped scopedPrincipal, userID *string) bool {
	if allows(s.policy, scoped.principal, "tunjangan:create") {
		return true
	}
	if !allows(s.policy, scoped.principal, "tunjangan:create:homebase") {
		return false
	}
	if userID == nil || *userID == "" {
		return false
	}
	return scoped.scope.Contains(*userID)
}

func (s *TunjanganService) canWrite(scoped scopedPrincipal, item models.KomponenTunjangan) bool {
	for _, permission := range []auth.Permission{"tunjangan:update:any", "tunjangan:update:homebase"} {
		if auth.CanAccessUserInScope(scoped.principal, item.UserId, item.UserId, permission, s.policy, scoped.scope) {
			return true
		}
	}
	return false
}

func (s *TunjanganService) canArchive(scoped scopedPrincipal, item models.KomponenTunjangan) bool {
	for _, permission := range []auth.Permission{"tunjangan:archive", "tunjangan:archive:homebase"} {
		if auth.CanAccessUserInScope(scoped.principal, item.UserId, item.UserId, permission, s.policy, scoped.scope) {
			return true
		}
	}
	return false
}

// scopedPrincipal memuat principal beserta cakupan unit efektifnya.
type scopedPrincipal struct {
	principal auth.Principal
	scope     auth.UnitScope
}
