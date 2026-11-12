// Package app memuat logika aplikasi untuk arsip dokumen pegawai.
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

// ArsipService menjalankan use case arsip dokumen pegawai.
type ArsipService struct {
	repo   repository.ArsipRepository
	policy auth.Policy
	scope  scopeResolver
}

// NewArsipService membuat service arsip dokumen.
func NewArsipService(repo repository.ArsipRepository, policy auth.Policy, scopeRepo repository.ScopeRepository) *ArsipService {
	return &ArsipService{repo: repo, policy: policy, scope: newScopeResolver(scopeRepo, policy)}
}

func (s *ArsipService) List(ctx context.Context, principal auth.Principal, options listing.Options) (listing.Page[models.ArsipDokumenPegawai], error) {
	switch {
	case allows(s.policy, principal, "arsip:read:any"):
	case allows(s.policy, principal, "arsip:read:homebase"):
		scope, err := s.scope.resolve(ctx, principal)
		if err != nil {
			return listing.Page[models.ArsipDokumenPegawai]{}, err
		}
		if scope.Empty() {
			return listing.EmptyPage[models.ArsipDokumenPegawai](options.Page, options.PageSize), nil
		}
		options.ScopeHomebaseIDs = scope.List()
	case allows(s.policy, principal, "arsip:read:self"):
		options.UserID = principal.UserID
	default:
		return listing.Page[models.ArsipDokumenPegawai]{}, domain.ErrForbidden
	}
	items, total, err := s.repo.ListArsip(ctx, options)
	if err != nil {
		return listing.Page[models.ArsipDokumenPegawai]{}, err
	}
	return listing.Page[models.ArsipDokumenPegawai]{
		Items: items, Page: options.Page, PageSize: options.PageSize,
		TotalItems: total, TotalPages: listing.PageCount(total, options.PageSize),
	}, nil
}

func (s *ArsipService) Get(ctx context.Context, principal auth.Principal, id string) (models.ArsipDokumenPegawai, error) {
	needsScope := allows(s.policy, principal, "arsip:read:homebase") && !allows(s.policy, principal, "arsip:read:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.ArsipDokumenPegawai{}, err
	}
	item, err := s.repo.GetArsip(ctx, id)
	if err != nil {
		return models.ArsipDokumenPegawai{}, err
	}
	if !s.canRead(scoped, item) {
		return models.ArsipDokumenPegawai{}, domain.ErrForbidden
	}
	return item, nil
}

func (s *ArsipService) Create(ctx context.Context, principal auth.Principal, input domain.CreateArsipDokumenPegawai) (models.ArsipDokumenPegawai, error) {
	if err := input.Validate(); err != nil {
		return models.ArsipDokumenPegawai{}, err
	}
	needsScope := allows(s.policy, principal, "arsip:create:homebase") && !allows(s.policy, principal, "arsip:create")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.ArsipDokumenPegawai{}, err
	}
	if !s.canCreate(scoped, &input.UserID) {
		return models.ArsipDokumenPegawai{}, domain.ErrForbidden
	}
	var berlakuSampai *time.Time
	if input.BerlakuSampai != nil {
		t, err := time.Parse("2006-01-02", *input.BerlakuSampai)
		if err != nil {
			return models.ArsipDokumenPegawai{}, domain.ErrValidation
		}
		berlakuSampai = &t
	}
	item := models.ArsipDokumenPegawai{
		UserId:             input.UserID,
		JenisDokumen:       input.JenisDokumen,
		Judul:              input.Judul,
		FileUrl:            input.FileURL,
		FileSize:           input.FileSize,
		MimeType:           input.MimeType,
		Keterangan:         input.Keterangan,
		BerlakuSampai:      berlakuSampai,
		CloudinaryPublicId: input.CloudinaryPublicId,
	}
	return s.repo.CreateArsip(ctx, principal.UserID, item)
}

func (s *ArsipService) Update(ctx context.Context, principal auth.Principal, id string, patch domain.ArsipDokumenPatch) (models.ArsipDokumenPegawai, error) {
	if !patch.HasChanges() {
		return models.ArsipDokumenPegawai{}, domain.ErrValidation
	}
	if err := patch.Validate(); err != nil {
		return models.ArsipDokumenPegawai{}, err
	}
	needsScope := allows(s.policy, principal, "arsip:update:homebase") && !allows(s.policy, principal, "arsip:update:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.ArsipDokumenPegawai{}, err
	}
	item, err := s.repo.GetArsip(ctx, id)
	if err != nil {
		return models.ArsipDokumenPegawai{}, err
	}
	if !s.canWrite(scoped, item) {
		return models.ArsipDokumenPegawai{}, domain.ErrForbidden
	}
	return s.repo.UpdateArsip(ctx, principal.UserID, id, patch)
}

func (s *ArsipService) Archive(ctx context.Context, principal auth.Principal, id string) error {
	needsScope := allows(s.policy, principal, "arsip:archive:homebase") && !allows(s.policy, principal, "arsip:archive")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return err
	}
	item, err := s.repo.GetArsip(ctx, id)
	if err != nil {
		return err
	}
	if !s.canArchive(scoped, item) {
		return domain.ErrForbidden
	}
	return s.repo.ArchiveArsip(ctx, principal.UserID, id)
}

func (s *ArsipService) scopedPrincipal(ctx context.Context, principal auth.Principal, needsScope bool) (scopedPrincipal, error) {
	if !needsScope {
		return scopedPrincipal{principal: principal}, nil
	}
	scope, err := s.scope.resolve(ctx, principal)
	if err != nil {
		return scopedPrincipal{}, err
	}
	return scopedPrincipal{principal: principal, scope: scope}, nil
}

func (s *ArsipService) canRead(scoped scopedPrincipal, item models.ArsipDokumenPegawai) bool {
	for _, permission := range []auth.Permission{"arsip:read:any", "arsip:read:homebase", "arsip:read:self"} {
		if auth.CanAccessUserInScope(scoped.principal, item.UserId, item.UserId, permission, s.policy, scoped.scope) {
			return true
		}
	}
	return false
}

func (s *ArsipService) canCreate(scoped scopedPrincipal, userID *string) bool {
	if allows(s.policy, scoped.principal, "arsip:create") {
		return true
	}
	if !allows(s.policy, scoped.principal, "arsip:create:homebase") {
		return false
	}
	if userID == nil || *userID == "" {
		return false
	}
	return scoped.scope.Contains(*userID)
}

func (s *ArsipService) canWrite(scoped scopedPrincipal, item models.ArsipDokumenPegawai) bool {
	for _, permission := range []auth.Permission{"arsip:update:any", "arsip:update:homebase"} {
		if auth.CanAccessUserInScope(scoped.principal, item.UserId, item.UserId, permission, s.policy, scoped.scope) {
			return true
		}
	}
	return false
}

func (s *ArsipService) canArchive(scoped scopedPrincipal, item models.ArsipDokumenPegawai) bool {
	for _, permission := range []auth.Permission{"arsip:archive", "arsip:archive:homebase"} {
		if auth.CanAccessUserInScope(scoped.principal, item.UserId, item.UserId, permission, s.policy, scoped.scope) {
			return true
		}
	}
	return false
}
