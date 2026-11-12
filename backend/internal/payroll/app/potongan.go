// Package app memuat logika aplikasi untuk komponen potongan pegawai.
package app

import (
	"context"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
	"simpeg/backend/internal/payroll/domain"
	"simpeg/backend/internal/payroll/repository"
)

// PotonganService menjalankan use case komponen potongan pegawai.
type PotonganService struct {
	repo   repository.PotonganRepository
	policy auth.Policy
	scope  scopeResolver
}

// NewPotonganService membuat service komponen potongan.
func NewPotonganService(repo repository.PotonganRepository, policy auth.Policy, scopeRepo ScopeRepository) *PotonganService {
	return &PotonganService{repo: repo, policy: policy, scope: newScopeResolver(scopeRepo)}
}

func (s *PotonganService) List(ctx context.Context, principal auth.Principal, options listing.Options) (listing.Page[models.KomponenPotongan], error) {
	switch {
	case allows(s.policy, principal, "potongan:read:any"):
	case allows(s.policy, principal, "potongan:read:homebase"):
		scope, err := s.scope.resolve(ctx, principal)
		if err != nil {
			return listing.Page[models.KomponenPotongan]{}, err
		}
		if scope.Empty() {
			return listing.EmptyPage[models.KomponenPotongan](options.Page, options.PageSize), nil
		}
		options.ScopeHomebaseIDs = scope.List()
	case allows(s.policy, principal, "potongan:read:self"):
		options.UserID = principal.UserID
	default:
		return listing.Page[models.KomponenPotongan]{}, domain.ErrForbidden
	}
	items, total, err := s.repo.ListPotongan(ctx, options)
	if err != nil {
		return listing.Page[models.KomponenPotongan]{}, err
	}
	return listing.Page[models.KomponenPotongan]{
		Items: items, Page: options.Page, PageSize: options.PageSize,
		TotalItems: total, TotalPages: listing.PageCount(total, options.PageSize),
	}, nil
}

func (s *PotonganService) Get(ctx context.Context, principal auth.Principal, id string) (models.KomponenPotongan, error) {
	needsScope := allows(s.policy, principal, "potongan:read:homebase") && !allows(s.policy, principal, "potongan:read:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.KomponenPotongan{}, err
	}
	item, err := s.repo.GetPotongan(ctx, id)
	if err != nil {
		return models.KomponenPotongan{}, err
	}
	if !s.canRead(scoped, item) {
		return models.KomponenPotongan{}, domain.ErrForbidden
	}
	return item, nil
}

func (s *PotonganService) Create(ctx context.Context, principal auth.Principal, input domain.CreateKomponenPotongan) (models.KomponenPotongan, error) {
	if err := input.Validate(); err != nil {
		return models.KomponenPotongan{}, err
	}
	needsScope := allows(s.policy, principal, "potongan:create:homebase") && !allows(s.policy, principal, "potongan:create")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.KomponenPotongan{}, err
	}
	if !s.canCreate(scoped, &input.UserID) {
		return models.KomponenPotongan{}, domain.ErrForbidden
	}
	item := models.KomponenPotongan{
		UserId:         input.UserID,
		JenisPotongan:  models.JenisPotongan(input.JenisPotongan),
		Persentase:     input.Persentase,
		Nominal:        input.Nominal,
		PorsiInstitusi: input.PorsiInstitusi,
		PorsiPegawai:   input.PorsiPegawai,
		Aktif:          input.Aktif,
	}
	return s.repo.CreatePotongan(ctx, principal.UserID, item)
}

func (s *PotonganService) Update(ctx context.Context, principal auth.Principal, id string, patch domain.KomponenPotonganPatch) (models.KomponenPotongan, error) {
	if !patch.HasChanges() {
		return models.KomponenPotongan{}, domain.ErrValidation
	}
	if err := patch.Validate(); err != nil {
		return models.KomponenPotongan{}, err
	}
	needsScope := allows(s.policy, principal, "potongan:update:homebase") && !allows(s.policy, principal, "potongan:update:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.KomponenPotongan{}, err
	}
	item, err := s.repo.GetPotongan(ctx, id)
	if err != nil {
		return models.KomponenPotongan{}, err
	}
	if !s.canWrite(scoped, item) {
		return models.KomponenPotongan{}, domain.ErrForbidden
	}
	return s.repo.UpdatePotongan(ctx, principal.UserID, id, patch)
}

func (s *PotonganService) Archive(ctx context.Context, principal auth.Principal, id string) error {
	needsScope := allows(s.policy, principal, "potongan:archive:homebase") && !allows(s.policy, principal, "potongan:archive")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return err
	}
	item, err := s.repo.GetPotongan(ctx, id)
	if err != nil {
		return err
	}
	if !s.canArchive(scoped, item) {
		return domain.ErrForbidden
	}
	return s.repo.ArchivePotongan(ctx, principal.UserID, id)
}

func (s *PotonganService) scopedPrincipal(ctx context.Context, principal auth.Principal, needsScope bool) (scopedPrincipal, error) {
	if !needsScope {
		return scopedPrincipal{principal: principal}, nil
	}
	scope, err := s.scope.resolve(ctx, principal)
	if err != nil {
		return scopedPrincipal{}, err
	}
	return scopedPrincipal{principal: principal, scope: scope}, nil
}

func (s *PotonganService) canRead(scoped scopedPrincipal, item models.KomponenPotongan) bool {
	for _, permission := range []auth.Permission{"potongan:read:any", "potongan:read:homebase", "potongan:read:self"} {
		if auth.CanAccessUserInScope(scoped.principal, item.UserId, item.UserId, permission, s.policy, scoped.scope) {
			return true
		}
	}
	return false
}

func (s *PotonganService) canCreate(scoped scopedPrincipal, userID *string) bool {
	if allows(s.policy, scoped.principal, "potongan:create") {
		return true
	}
	if !allows(s.policy, scoped.principal, "potongan:create:homebase") {
		return false
	}
	if userID == nil || *userID == "" {
		return false
	}
	return scoped.scope.Contains(*userID)
}

func (s *PotonganService) canWrite(scoped scopedPrincipal, item models.KomponenPotongan) bool {
	for _, permission := range []auth.Permission{"potongan:update:any", "potongan:update:homebase"} {
		if auth.CanAccessUserInScope(scoped.principal, item.UserId, item.UserId, permission, s.policy, scoped.scope) {
			return true
		}
	}
	return false
}

func (s *PotonganService) canArchive(scoped scopedPrincipal, item models.KomponenPotongan) bool {
	for _, permission := range []auth.Permission{"potongan:archive", "potongan:archive:homebase"} {
		if auth.CanAccessUserInScope(scoped.principal, item.UserId, item.UserId, permission, s.policy, scoped.scope) {
			return true
		}
	}
	return false
}
