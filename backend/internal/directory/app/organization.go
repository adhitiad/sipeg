package app

import (
	"context"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/directory/domain"
	"simpeg/backend/internal/directory/repository"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
)

// OrganizationService menjalankan use case unit organisasi: daftar, baca,
// buat, ubah, dan arsip.
type OrganizationService struct {
	repo   repository.OrganizationRepository
	policy auth.Policy
	scope  scopeResolver
}

// NewOrganizationService membuat service unit organisasi.
func NewOrganizationService(repo repository.OrganizationRepository, policy auth.Policy, scopeRepo repository.ScopeRepository) *OrganizationService {
	return &OrganizationService{repo: repo, policy: policy, scope: newScopeResolver(scopeRepo, policy)}
}

// List mengembalikan daftar unit sesuai filter yang sudah diminta.
func (s *OrganizationService) List(ctx context.Context, principal auth.Principal, options listing.Options) (listing.Page[models.Homebase], error) {
	if !allows(s.policy, principal, "organization:read") {
		scope, err := s.scope.resolve(ctx, principal)
		if err != nil {
			return listing.Page[models.Homebase]{}, err
		}
		if scope.Empty() {
			return listing.EmptyPage[models.Homebase](options.Page, options.PageSize), nil
		}
		options.ScopeHomebaseIDs = scope.List()
	}

	organizations, total, err := s.repo.ListOrganizations(ctx, options)
	if err != nil {
		return listing.Page[models.Homebase]{}, err
	}
	return listing.Page[models.Homebase]{
		Items:      organizations,
		Page:       options.Page,
		PageSize:   options.PageSize,
		TotalItems: total,
		TotalPages: listing.PageCount(total, options.PageSize),
	}, nil
}

// Get mengembalikan satu unit bila principal berhak membacanya.
func (s *OrganizationService) Get(ctx context.Context, principal auth.Principal, id string) (models.Homebase, error) {
	organization, err := s.repo.GetOrganization(ctx, id)
	if err != nil {
		return models.Homebase{}, err
	}
	permitted, err := s.canAccess(ctx, principal, organization)
	if err != nil {
		return models.Homebase{}, err
	}
	if !permitted {
		return models.Homebase{}, domain.ErrForbidden
	}
	return organization, nil
}

// Create membuat satu unit organisasi baru.
func (s *OrganizationService) Create(ctx context.Context, principal auth.Principal, input domain.CreateOrganization) (models.Homebase, error) {
	if err := input.Validate(); err != nil {
		return models.Homebase{}, err
	}
	permitted, err := s.canCreate(ctx, principal, input.ParentID)
	if err != nil {
		return models.Homebase{}, err
	}
	if !permitted {
		return models.Homebase{}, domain.ErrForbidden
	}

	organization := models.Homebase{
		Name:        input.Name,
		Description: domain.CleanOptional(input.Description),
		Jenis:       models.JenisOrganisasi(input.Jenis),
		ParentId:    input.ParentID,
		Aktif:       true,
	}
	return s.repo.CreateOrganization(ctx, principal.UserID, organization)
}

// Update memperbarui unit organisasi bila principal berwenang.
func (s *OrganizationService) Update(ctx context.Context, principal auth.Principal, id string, patch domain.OrganizationPatch) (models.Homebase, error) {
	if !patch.HasChanges() {
		return models.Homebase{}, domain.ErrValidation
	}
	if err := patch.Validate(); err != nil {
		return models.Homebase{}, err
	}

	organization, err := s.repo.GetOrganization(ctx, id)
	if err != nil {
		return models.Homebase{}, err
	}
	permitted, err := s.canManage(ctx, principal, organization)
	if err != nil {
		return models.Homebase{}, err
	}
	if !permitted {
		return models.Homebase{}, domain.ErrForbidden
	}
	return s.repo.UpdateOrganization(ctx, principal.UserID, id, patch)
}

// Archive menonaktifkan unit yang tidak lagi memiliki anggota maupun anak aktif.
func (s *OrganizationService) Archive(ctx context.Context, principal auth.Principal, id string) error {
	organization, err := s.repo.GetOrganization(ctx, id)
	if err != nil {
		return err
	}
	permitted, err := s.canManage(ctx, principal, organization)
	if err != nil {
		return err
	}
	if !permitted {
		return domain.ErrForbidden
	}
	return s.repo.ArchiveOrganization(ctx, principal.UserID, id)
}

// canAccess memeriksa apakah principal boleh membaca sebuah unit. Role dengan
// cakupan global tidak dibatasi.
func (s *OrganizationService) canAccess(ctx context.Context, principal auth.Principal, organization models.Homebase) (bool, error) {
	if allows(s.policy, principal, "organization:read") || allows(s.policy, principal, "organization:manage") {
		return true, nil
	}
	if !allows(s.policy, principal, "organization:read:homebase") && !allows(s.policy, principal, "organization:manage:homebase") {
		return false, nil
	}
	scope, err := s.scope.resolve(ctx, principal)
	if err != nil {
		return false, err
	}
	return scope.Contains(organization.ID), nil
}

// canCreate memeriksa apakah principal boleh membuat unit di bawah induk
// tertentu. Role berscope hanya boleh membuat unit turunan dari unit yang
// sudah berada dalam cakupan.
func (s *OrganizationService) canCreate(ctx context.Context, principal auth.Principal, parentID *string) (bool, error) {
	if allows(s.policy, principal, "organization:manage") {
		return true, nil
	}
	if !allows(s.policy, principal, "organization:manage:homebase") {
		return false, nil
	}
	if parentID == nil {
		// Unit akar berada di luar setiap cakupan unit.
		return false, nil
	}
	scope, err := s.scope.resolve(ctx, principal)
	if err != nil {
		return false, err
	}
	return scope.Contains(*parentID), nil
}

// canManage memeriksa apakah principal boleh mengubah atau mengarsipkan sebuah
// unit.
func (s *OrganizationService) canManage(ctx context.Context, principal auth.Principal, organization models.Homebase) (bool, error) {
	if allows(s.policy, principal, "organization:manage") {
		return true, nil
	}
	if !allows(s.policy, principal, "organization:manage:homebase") {
		return false, nil
	}
	scope, err := s.scope.resolve(ctx, principal)
	if err != nil {
		return false, err
	}
	return scope.Contains(organization.ID), nil
}
