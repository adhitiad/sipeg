// Package app memuat logika aplikasi untuk riwayat golongan pegawai.
package app

import (
	"context"
	"time"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/directory/domain"
	"simpeg/backend/internal/directory/repository"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
)

// GolonganService menjalankan use case riwayat golongan pegawai.
type GolonganService struct {
	repo   repository.GolonganRepository
	policy auth.Policy
	scope  scopeResolver
}

// NewGolonganService membuat service riwayat golongan.
func NewGolonganService(repo repository.GolonganRepository, policy auth.Policy, scopeRepo repository.ScopeRepository) *GolonganService {
	return &GolonganService{repo: repo, policy: policy, scope: newScopeResolver(scopeRepo, policy)}
}

func (s *GolonganService) List(ctx context.Context, principal auth.Principal, options listing.Options) (listing.Page[models.RiwayatGolongan], error) {
	switch {
	case allows(s.policy, principal, "golongan:read:any"):
	case allows(s.policy, principal, "golongan:read:homebase"):
		scope, err := s.scope.resolve(ctx, principal)
		if err != nil {
			return listing.Page[models.RiwayatGolongan]{}, err
		}
		if scope.Empty() {
			return listing.EmptyPage[models.RiwayatGolongan](options.Page, options.PageSize), nil
		}
		options.ScopeHomebaseIDs = scope.List()
	case allows(s.policy, principal, "golongan:read:self"):
		options.UserID = principal.UserID
	default:
		return listing.Page[models.RiwayatGolongan]{}, domain.ErrForbidden
	}
	items, total, err := s.repo.ListGolongan(ctx, options)
	if err != nil {
		return listing.Page[models.RiwayatGolongan]{}, err
	}
	return listing.Page[models.RiwayatGolongan]{
		Items: items, Page: options.Page, PageSize: options.PageSize,
		TotalItems: total, TotalPages: listing.PageCount(total, options.PageSize),
	}, nil
}

func (s *GolonganService) Get(ctx context.Context, principal auth.Principal, id string) (models.RiwayatGolongan, error) {
	needsScope := allows(s.policy, principal, "golongan:read:homebase") && !allows(s.policy, principal, "golongan:read:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.RiwayatGolongan{}, err
	}
	item, err := s.repo.GetGolongan(ctx, id)
	if err != nil {
		return models.RiwayatGolongan{}, err
	}
	if !s.canRead(scoped, item) {
		return models.RiwayatGolongan{}, domain.ErrForbidden
	}
	return item, nil
}

func (s *GolonganService) Create(ctx context.Context, principal auth.Principal, input domain.CreateRiwayatGolongan) (models.RiwayatGolongan, error) {
	if err := input.Validate(); err != nil {
		return models.RiwayatGolongan{}, err
	}
	needsScope := allows(s.policy, principal, "golongan:create:homebase") && !allows(s.policy, principal, "golongan:create")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.RiwayatGolongan{}, err
	}
	if !s.canCreate(scoped, &input.UserID) {
		return models.RiwayatGolongan{}, domain.ErrForbidden
	}
	tmt, err := time.Parse("2006-01-02", input.TMT)
	if err != nil {
		return models.RiwayatGolongan{}, domain.ErrValidation
	}
	item := models.RiwayatGolongan{
		UserId:      input.UserID,
		Golongan:    input.Golongan,
		Ruang:       input.Ruang,
		TMT:         tmt,
		AngkaKredit: input.AngkaKredit,
		NomorSk:     input.NomorSk,
	}
	return s.repo.CreateGolongan(ctx, principal.UserID, item)
}

func (s *GolonganService) Update(ctx context.Context, principal auth.Principal, id string, patch domain.RiwayatGolonganPatch) (models.RiwayatGolongan, error) {
	if !patch.HasChanges() {
		return models.RiwayatGolongan{}, domain.ErrValidation
	}
	if err := patch.Validate(); err != nil {
		return models.RiwayatGolongan{}, err
	}
	needsScope := allows(s.policy, principal, "golongan:update:homebase") && !allows(s.policy, principal, "golongan:update:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.RiwayatGolongan{}, err
	}
	item, err := s.repo.GetGolongan(ctx, id)
	if err != nil {
		return models.RiwayatGolongan{}, err
	}
	if !s.canWrite(scoped, item) {
		return models.RiwayatGolongan{}, domain.ErrForbidden
	}
	return s.repo.UpdateGolongan(ctx, principal.UserID, id, patch)
}

func (s *GolonganService) Archive(ctx context.Context, principal auth.Principal, id string) error {
	needsScope := allows(s.policy, principal, "golongan:archive:homebase") && !allows(s.policy, principal, "golongan:archive")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return err
	}
	item, err := s.repo.GetGolongan(ctx, id)
	if err != nil {
		return err
	}
	if !s.canArchive(scoped, item) {
		return domain.ErrForbidden
	}
	return s.repo.ArchiveGolongan(ctx, principal.UserID, id)
}

func (s *GolonganService) scopedPrincipal(ctx context.Context, principal auth.Principal, needsScope bool) (scopedPrincipal, error) {
	if !needsScope {
		return scopedPrincipal{principal: principal}, nil
	}
	scope, err := s.scope.resolve(ctx, principal)
	if err != nil {
		return scopedPrincipal{}, err
	}
	return scopedPrincipal{principal: principal, scope: scope}, nil
}

func (s *GolonganService) canRead(scoped scopedPrincipal, item models.RiwayatGolongan) bool {
	for _, permission := range []auth.Permission{"golongan:read:any", "golongan:read:homebase", "golongan:read:self"} {
		if auth.CanAccessUserInScope(scoped.principal, item.UserId, item.UserId, permission, s.policy, scoped.scope) {
			return true
		}
	}
	return false
}

func (s *GolonganService) canCreate(scoped scopedPrincipal, userID *string) bool {
	if allows(s.policy, scoped.principal, "golongan:create") {
		return true
	}
	if !allows(s.policy, scoped.principal, "golongan:create:homebase") {
		return false
	}
	if userID == nil || *userID == "" {
		return false
	}
	return scoped.scope.Contains(*userID)
}

func (s *GolonganService) canWrite(scoped scopedPrincipal, item models.RiwayatGolongan) bool {
	for _, permission := range []auth.Permission{"golongan:update:any", "golongan:update:homebase"} {
		if auth.CanAccessUserInScope(scoped.principal, item.UserId, item.UserId, permission, s.policy, scoped.scope) {
			return true
		}
	}
	return false
}

func (s *GolonganService) canArchive(scoped scopedPrincipal, item models.RiwayatGolongan) bool {
	for _, permission := range []auth.Permission{"golongan:archive", "golongan:archive:homebase"} {
		if auth.CanAccessUserInScope(scoped.principal, item.UserId, item.UserId, permission, s.policy, scoped.scope) {
			return true
		}
	}
	return false
}
