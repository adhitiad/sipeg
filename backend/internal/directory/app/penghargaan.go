// Package app memuat logika aplikasi untuk riwayat penghargaan pegawai.
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

// PenghargaanService menjalankan use case riwayat penghargaan pegawai.
type PenghargaanService struct {
	repo   repository.PenghargaanRepository
	policy auth.Policy
	scope  scopeResolver
}

// NewPenghargaanService membuat service riwayat penghargaan.
func NewPenghargaanService(repo repository.PenghargaanRepository, policy auth.Policy, scopeRepo repository.ScopeRepository) *PenghargaanService {
	return &PenghargaanService{repo: repo, policy: policy, scope: newScopeResolver(scopeRepo, policy)}
}

func (s *PenghargaanService) List(ctx context.Context, principal auth.Principal, options listing.Options) (listing.Page[models.RiwayatPenghargaan], error) {
	switch {
	case allows(s.policy, principal, "penghargaan:read:any"):
	case allows(s.policy, principal, "penghargaan:read:homebase"):
		scope, err := s.scope.resolve(ctx, principal)
		if err != nil {
			return listing.Page[models.RiwayatPenghargaan]{}, err
		}
		if scope.Empty() {
			return listing.EmptyPage[models.RiwayatPenghargaan](options.Page, options.PageSize), nil
		}
		options.ScopeHomebaseIDs = scope.List()
	case allows(s.policy, principal, "penghargaan:read:self"):
		options.UserID = principal.UserID
	default:
		return listing.Page[models.RiwayatPenghargaan]{}, domain.ErrForbidden
	}
	items, total, err := s.repo.ListPenghargaan(ctx, options)
	if err != nil {
		return listing.Page[models.RiwayatPenghargaan]{}, err
	}
	return listing.Page[models.RiwayatPenghargaan]{
		Items: items, Page: options.Page, PageSize: options.PageSize,
		TotalItems: total, TotalPages: listing.PageCount(total, options.PageSize),
	}, nil
}

func (s *PenghargaanService) Get(ctx context.Context, principal auth.Principal, id string) (models.RiwayatPenghargaan, error) {
	needsScope := allows(s.policy, principal, "penghargaan:read:homebase") && !allows(s.policy, principal, "penghargaan:read:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.RiwayatPenghargaan{}, err
	}
	item, err := s.repo.GetPenghargaan(ctx, id)
	if err != nil {
		return models.RiwayatPenghargaan{}, err
	}
	if !s.canRead(scoped, item) {
		return models.RiwayatPenghargaan{}, domain.ErrForbidden
	}
	return item, nil
}

func (s *PenghargaanService) Create(ctx context.Context, principal auth.Principal, input domain.CreateRiwayatPenghargaan) (models.RiwayatPenghargaan, error) {
	if err := input.Validate(); err != nil {
		return models.RiwayatPenghargaan{}, err
	}
	needsScope := allows(s.policy, principal, "penghargaan:create:homebase") && !allows(s.policy, principal, "penghargaan:create")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.RiwayatPenghargaan{}, err
	}
	if !s.canCreate(scoped, &input.UserID) {
		return models.RiwayatPenghargaan{}, domain.ErrForbidden
	}
	tanggalTerima, err := time.Parse("2006-01-02", input.TanggalTerima)
	if err != nil {
		return models.RiwayatPenghargaan{}, domain.ErrValidation
	}
	item := models.RiwayatPenghargaan{
		UserId:           input.UserID,
		NamaPenghargaan:  input.NamaPenghargaan,
		JenisPenghargaan: input.JenisPenghargaan,
		Penyelenggara:    input.Penyelenggara,
		TanggalTerima:    tanggalTerima,
		NomorSk:          input.NomorSk,
	}
	return s.repo.CreatePenghargaan(ctx, principal.UserID, item)
}

func (s *PenghargaanService) Update(ctx context.Context, principal auth.Principal, id string, patch domain.RiwayatPenghargaanPatch) (models.RiwayatPenghargaan, error) {
	if !patch.HasChanges() {
		return models.RiwayatPenghargaan{}, domain.ErrValidation
	}
	if err := patch.Validate(); err != nil {
		return models.RiwayatPenghargaan{}, err
	}
	needsScope := allows(s.policy, principal, "penghargaan:update:homebase") && !allows(s.policy, principal, "penghargaan:update:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.RiwayatPenghargaan{}, err
	}
	item, err := s.repo.GetPenghargaan(ctx, id)
	if err != nil {
		return models.RiwayatPenghargaan{}, err
	}
	if !s.canWrite(scoped, item) {
		return models.RiwayatPenghargaan{}, domain.ErrForbidden
	}
	return s.repo.UpdatePenghargaan(ctx, principal.UserID, id, patch)
}

func (s *PenghargaanService) Archive(ctx context.Context, principal auth.Principal, id string) error {
	needsScope := allows(s.policy, principal, "penghargaan:archive:homebase") && !allows(s.policy, principal, "penghargaan:archive")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return err
	}
	item, err := s.repo.GetPenghargaan(ctx, id)
	if err != nil {
		return err
	}
	if !s.canArchive(scoped, item) {
		return domain.ErrForbidden
	}
	return s.repo.ArchivePenghargaan(ctx, principal.UserID, id)
}

func (s *PenghargaanService) scopedPrincipal(ctx context.Context, principal auth.Principal, needsScope bool) (scopedPrincipal, error) {
	if !needsScope {
		return scopedPrincipal{principal: principal}, nil
	}
	scope, err := s.scope.resolve(ctx, principal)
	if err != nil {
		return scopedPrincipal{}, err
	}
	return scopedPrincipal{principal: principal, scope: scope}, nil
}

func (s *PenghargaanService) canRead(scoped scopedPrincipal, item models.RiwayatPenghargaan) bool {
	for _, permission := range []auth.Permission{"penghargaan:read:any", "penghargaan:read:homebase", "penghargaan:read:self"} {
		if auth.CanAccessUserInScope(scoped.principal, item.UserId, item.UserId, permission, s.policy, scoped.scope) {
			return true
		}
	}
	return false
}

func (s *PenghargaanService) canCreate(scoped scopedPrincipal, userID *string) bool {
	if allows(s.policy, scoped.principal, "penghargaan:create") {
		return true
	}
	if !allows(s.policy, scoped.principal, "penghargaan:create:homebase") {
		return false
	}
	if userID == nil || *userID == "" {
		return false
	}
	return scoped.scope.Contains(*userID)
}

func (s *PenghargaanService) canWrite(scoped scopedPrincipal, item models.RiwayatPenghargaan) bool {
	for _, permission := range []auth.Permission{"penghargaan:update:any", "penghargaan:update:homebase"} {
		if auth.CanAccessUserInScope(scoped.principal, item.UserId, item.UserId, permission, s.policy, scoped.scope) {
			return true
		}
	}
	return false
}

func (s *PenghargaanService) canArchive(scoped scopedPrincipal, item models.RiwayatPenghargaan) bool {
	for _, permission := range []auth.Permission{"penghargaan:archive", "penghargaan:archive:homebase"} {
		if auth.CanAccessUserInScope(scoped.principal, item.UserId, item.UserId, permission, s.policy, scoped.scope) {
			return true
		}
	}
	return false
}
