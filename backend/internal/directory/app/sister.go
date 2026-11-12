// Package app memuat logika aplikasi untuk pemetaan ID eksternal SISTER/PDDIKTI.
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

// SisterService menjalankan use case pemetaan ID eksternal.
type SisterService struct {
	repo   repository.SisterRepository
	policy auth.Policy
	scope  scopeResolver
}

// NewSisterService membuat service pemetaan ID eksternal.
func NewSisterService(repo repository.SisterRepository, policy auth.Policy, scopeRepo repository.ScopeRepository) *SisterService {
	return &SisterService{repo: repo, policy: policy, scope: newScopeResolver(scopeRepo, policy)}
}

func (s *SisterService) List(ctx context.Context, principal auth.Principal, options listing.Options) (listing.Page[models.SisterSyncMap], error) {
	switch {
	case allows(s.policy, principal, "sister:read:any"):
	case allows(s.policy, principal, "sister:read:homebase"):
		scope, err := s.scope.resolve(ctx, principal)
		if err != nil {
			return listing.Page[models.SisterSyncMap]{}, err
		}
		if scope.Empty() {
			return listing.EmptyPage[models.SisterSyncMap](options.Page, options.PageSize), nil
		}
		options.ScopeHomebaseIDs = scope.List()
	case allows(s.policy, principal, "sister:read:self"):
		options.UserID = principal.UserID
	default:
		return listing.Page[models.SisterSyncMap]{}, domain.ErrForbidden
	}
	items, total, err := s.repo.ListSister(ctx, options)
	if err != nil {
		return listing.Page[models.SisterSyncMap]{}, err
	}
	return listing.Page[models.SisterSyncMap]{
		Items: items, Page: options.Page, PageSize: options.PageSize,
		TotalItems: total, TotalPages: listing.PageCount(total, options.PageSize),
	}, nil
}

func (s *SisterService) Get(ctx context.Context, principal auth.Principal, id string) (models.SisterSyncMap, error) {
	needsScope := allows(s.policy, principal, "sister:read:homebase") && !allows(s.policy, principal, "sister:read:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.SisterSyncMap{}, err
	}
	item, err := s.repo.GetSister(ctx, id)
	if err != nil {
		return models.SisterSyncMap{}, err
	}
	if !s.canRead(scoped, item) {
		return models.SisterSyncMap{}, domain.ErrForbidden
	}
	return item, nil
}

func (s *SisterService) Create(ctx context.Context, principal auth.Principal, input domain.CreateSisterSyncMap) (models.SisterSyncMap, error) {
	if err := input.Validate(); err != nil {
		return models.SisterSyncMap{}, err
	}
	needsScope := allows(s.policy, principal, "sister:create:homebase") && !allows(s.policy, principal, "sister:create")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.SisterSyncMap{}, err
	}
	if !s.canCreate(scoped, &input.UserID) {
		return models.SisterSyncMap{}, domain.ErrForbidden
	}
	item := models.SisterSyncMap{
		UserId:          input.UserID,
		SistemEksternal: input.SistemEksternal,
		IDEksternal:     input.IDEksternal,
		Status:          input.Status,
	}
	return s.repo.CreateSister(ctx, principal.UserID, item)
}

func (s *SisterService) Update(ctx context.Context, principal auth.Principal, id string, patch domain.SisterSyncMapPatch) (models.SisterSyncMap, error) {
	if !patch.HasChanges() {
		return models.SisterSyncMap{}, domain.ErrValidation
	}
	if err := patch.Validate(); err != nil {
		return models.SisterSyncMap{}, err
	}
	needsScope := allows(s.policy, principal, "sister:update:homebase") && !allows(s.policy, principal, "sister:update:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.SisterSyncMap{}, err
	}
	item, err := s.repo.GetSister(ctx, id)
	if err != nil {
		return models.SisterSyncMap{}, err
	}
	if !s.canWrite(scoped, item) {
		return models.SisterSyncMap{}, domain.ErrForbidden
	}
	return s.repo.UpdateSister(ctx, principal.UserID, id, patch)
}

func (s *SisterService) Archive(ctx context.Context, principal auth.Principal, id string) error {
	needsScope := allows(s.policy, principal, "sister:archive:homebase") && !allows(s.policy, principal, "sister:archive")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return err
	}
	item, err := s.repo.GetSister(ctx, id)
	if err != nil {
		return err
	}
	if !s.canArchive(scoped, item) {
		return domain.ErrForbidden
	}
	return s.repo.ArchiveSister(ctx, principal.UserID, id)
}

func (s *SisterService) scopedPrincipal(ctx context.Context, principal auth.Principal, needsScope bool) (scopedPrincipal, error) {
	if !needsScope {
		return scopedPrincipal{principal: principal}, nil
	}
	scope, err := s.scope.resolve(ctx, principal)
	if err != nil {
		return scopedPrincipal{}, err
	}
	return scopedPrincipal{principal: principal, scope: scope}, nil
}

func (s *SisterService) canRead(scoped scopedPrincipal, item models.SisterSyncMap) bool {
	for _, permission := range []auth.Permission{"sister:read:any", "sister:read:homebase", "sister:read:self"} {
		if auth.CanAccessUserInScope(scoped.principal, item.UserId, item.UserId, permission, s.policy, scoped.scope) {
			return true
		}
	}
	return false
}

func (s *SisterService) canCreate(scoped scopedPrincipal, userID *string) bool {
	if allows(s.policy, scoped.principal, "sister:create") {
		return true
	}
	if !allows(s.policy, scoped.principal, "sister:create:homebase") {
		return false
	}
	if userID == nil || *userID == "" {
		return false
	}
	return scoped.scope.Contains(*userID)
}

func (s *SisterService) canWrite(scoped scopedPrincipal, item models.SisterSyncMap) bool {
	for _, permission := range []auth.Permission{"sister:update:any", "sister:update:homebase"} {
		if auth.CanAccessUserInScope(scoped.principal, item.UserId, item.UserId, permission, s.policy, scoped.scope) {
			return true
		}
	}
	return false
}

func (s *SisterService) canArchive(scoped scopedPrincipal, item models.SisterSyncMap) bool {
	for _, permission := range []auth.Permission{"sister:archive", "sister:archive:homebase"} {
		if auth.CanAccessUserInScope(scoped.principal, item.UserId, item.UserId, permission, s.policy, scoped.scope) {
			return true
		}
	}
	return false
}

// SyncNow memperbarui waktu sinkronisasi terakhir.
func (s *SisterService) SyncNow(ctx context.Context, principal auth.Principal, id string) (models.SisterSyncMap, error) {
	if !allows(s.policy, principal, "sister:manage") {
		return models.SisterSyncMap{}, domain.ErrForbidden
	}
	now := time.Now()
	return s.repo.UpdateSisterSyncTime(ctx, principal.UserID, id, now)
}
