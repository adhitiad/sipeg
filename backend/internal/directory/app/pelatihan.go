// Package app memuat logika aplikasi untuk riwayat pelatihan pegawai.
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

// PelatihanService menjalankan use case riwayat pelatihan pegawai.
type PelatihanService struct {
	repo   repository.PelatihanRepository
	policy auth.Policy
	scope  scopeResolver
}

// NewPelatihanService membuat service riwayat pelatihan.
func NewPelatihanService(repo repository.PelatihanRepository, policy auth.Policy, scopeRepo repository.ScopeRepository) *PelatihanService {
	return &PelatihanService{repo: repo, policy: policy, scope: newScopeResolver(scopeRepo, policy)}
}

func (s *PelatihanService) List(ctx context.Context, principal auth.Principal, options listing.Options) (listing.Page[models.RiwayatPelatihan], error) {
	switch {
	case allows(s.policy, principal, "pelatihan:read:any"):
	case allows(s.policy, principal, "pelatihan:read:homebase"):
		scope, err := s.scope.resolve(ctx, principal)
		if err != nil {
			return listing.Page[models.RiwayatPelatihan]{}, err
		}
		if scope.Empty() {
			return listing.EmptyPage[models.RiwayatPelatihan](options.Page, options.PageSize), nil
		}
		options.ScopeHomebaseIDs = scope.List()
	case allows(s.policy, principal, "pelatihan:read:self"):
		options.UserID = principal.UserID
	default:
		return listing.Page[models.RiwayatPelatihan]{}, domain.ErrForbidden
	}
	items, total, err := s.repo.ListPelatihan(ctx, options)
	if err != nil {
		return listing.Page[models.RiwayatPelatihan]{}, err
	}
	return listing.Page[models.RiwayatPelatihan]{
		Items: items, Page: options.Page, PageSize: options.PageSize,
		TotalItems: total, TotalPages: listing.PageCount(total, options.PageSize),
	}, nil
}

func (s *PelatihanService) Get(ctx context.Context, principal auth.Principal, id string) (models.RiwayatPelatihan, error) {
	needsScope := allows(s.policy, principal, "pelatihan:read:homebase") && !allows(s.policy, principal, "pelatihan:read:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.RiwayatPelatihan{}, err
	}
	item, err := s.repo.GetPelatihan(ctx, id)
	if err != nil {
		return models.RiwayatPelatihan{}, err
	}
	if !s.canRead(scoped, item) {
		return models.RiwayatPelatihan{}, domain.ErrForbidden
	}
	return item, nil
}

func (s *PelatihanService) Create(ctx context.Context, principal auth.Principal, input domain.CreateRiwayatPelatihan) (models.RiwayatPelatihan, error) {
	if err := input.Validate(); err != nil {
		return models.RiwayatPelatihan{}, err
	}
	needsScope := allows(s.policy, principal, "pelatihan:create:homebase") && !allows(s.policy, principal, "pelatihan:create")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.RiwayatPelatihan{}, err
	}
	if !s.canCreate(scoped, &input.UserID) {
		return models.RiwayatPelatihan{}, domain.ErrForbidden
	}
	tanggalMulai, err := time.Parse("2006-01-02", input.TanggalMulai)
	if err != nil {
		return models.RiwayatPelatihan{}, domain.ErrValidation
	}
	tanggalSelesai, err := time.Parse("2006-01-02", input.TanggalSelesai)
	if err != nil {
		return models.RiwayatPelatihan{}, domain.ErrValidation
	}
	item := models.RiwayatPelatihan{
		UserId:             input.UserID,
		NamaPelatihan:      input.NamaPelatihan,
		JenisPelatihan:     input.JenisPelatihan,
		Penyelenggara:      input.Penyelenggara,
		TanggalMulai:       tanggalMulai,
		TanggalSelesai:     tanggalSelesai,
		DurasiJam:          input.DurasiJam,
		MemilikiSertifikat: input.MemilikiSertifikat,
	}
	return s.repo.CreatePelatihan(ctx, principal.UserID, item)
}

func (s *PelatihanService) Update(ctx context.Context, principal auth.Principal, id string, patch domain.RiwayatPelatihanPatch) (models.RiwayatPelatihan, error) {
	if !patch.HasChanges() {
		return models.RiwayatPelatihan{}, domain.ErrValidation
	}
	if err := patch.Validate(); err != nil {
		return models.RiwayatPelatihan{}, err
	}
	needsScope := allows(s.policy, principal, "pelatihan:update:homebase") && !allows(s.policy, principal, "pelatihan:update:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.RiwayatPelatihan{}, err
	}
	item, err := s.repo.GetPelatihan(ctx, id)
	if err != nil {
		return models.RiwayatPelatihan{}, err
	}
	if !s.canWrite(scoped, item) {
		return models.RiwayatPelatihan{}, domain.ErrForbidden
	}
	return s.repo.UpdatePelatihan(ctx, principal.UserID, id, patch)
}

func (s *PelatihanService) Archive(ctx context.Context, principal auth.Principal, id string) error {
	needsScope := allows(s.policy, principal, "pelatihan:archive:homebase") && !allows(s.policy, principal, "pelatihan:archive")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return err
	}
	item, err := s.repo.GetPelatihan(ctx, id)
	if err != nil {
		return err
	}
	if !s.canArchive(scoped, item) {
		return domain.ErrForbidden
	}
	return s.repo.ArchivePelatihan(ctx, principal.UserID, id)
}

func (s *PelatihanService) scopedPrincipal(ctx context.Context, principal auth.Principal, needsScope bool) (scopedPrincipal, error) {
	if !needsScope {
		return scopedPrincipal{principal: principal}, nil
	}
	scope, err := s.scope.resolve(ctx, principal)
	if err != nil {
		return scopedPrincipal{}, err
	}
	return scopedPrincipal{principal: principal, scope: scope}, nil
}

func (s *PelatihanService) canRead(scoped scopedPrincipal, item models.RiwayatPelatihan) bool {
	for _, permission := range []auth.Permission{"pelatihan:read:any", "pelatihan:read:homebase", "pelatihan:read:self"} {
		if auth.CanAccessUserInScope(scoped.principal, item.UserId, item.UserId, permission, s.policy, scoped.scope) {
			return true
		}
	}
	return false
}

func (s *PelatihanService) canCreate(scoped scopedPrincipal, userID *string) bool {
	if allows(s.policy, scoped.principal, "pelatihan:create") {
		return true
	}
	if !allows(s.policy, scoped.principal, "pelatihan:create:homebase") {
		return false
	}
	if userID == nil || *userID == "" {
		return false
	}
	return scoped.scope.Contains(*userID)
}

func (s *PelatihanService) canWrite(scoped scopedPrincipal, item models.RiwayatPelatihan) bool {
	for _, permission := range []auth.Permission{"pelatihan:update:any", "pelatihan:update:homebase"} {
		if auth.CanAccessUserInScope(scoped.principal, item.UserId, item.UserId, permission, s.policy, scoped.scope) {
			return true
		}
	}
	return false
}

func (s *PelatihanService) canArchive(scoped scopedPrincipal, item models.RiwayatPelatihan) bool {
	for _, permission := range []auth.Permission{"pelatihan:archive", "pelatihan:archive:homebase"} {
		if auth.CanAccessUserInScope(scoped.principal, item.UserId, item.UserId, permission, s.policy, scoped.scope) {
			return true
		}
	}
	return false
}
