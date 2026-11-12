// Package app memuat logika aplikasi untuk data keluarga pegawai.
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

// KeluargaService menjalankan use case data keluarga pegawai.
type KeluargaService struct {
	repo   repository.KeluargaRepository
	policy auth.Policy
	scope  scopeResolver
}

// NewKeluargaService membuat service keluarga pegawai.
func NewKeluargaService(repo repository.KeluargaRepository, policy auth.Policy, scopeRepo repository.ScopeRepository) *KeluargaService {
	return &KeluargaService{repo: repo, policy: policy, scope: newScopeResolver(scopeRepo, policy)}
}

// List mengembalikan daftar keluarga pegawai sesuai cakupan principal.
func (s *KeluargaService) List(ctx context.Context, principal auth.Principal, options listing.Options) (listing.Page[models.KeluargaPegawai], error) {
	switch {
	case allows(s.policy, principal, "keluarga:read:any"):
	case allows(s.policy, principal, "keluarga:read:homebase"):
		scope, err := s.scope.resolve(ctx, principal)
		if err != nil {
			return listing.Page[models.KeluargaPegawai]{}, err
		}
		if scope.Empty() {
			return listing.EmptyPage[models.KeluargaPegawai](options.Page, options.PageSize), nil
		}
		options.ScopeHomebaseIDs = scope.List()
	case allows(s.policy, principal, "keluarga:read:self"):
		options.UserID = principal.UserID
	default:
		return listing.Page[models.KeluargaPegawai]{}, domain.ErrForbidden
	}

	items, total, err := s.repo.ListKeluarga(ctx, options)
	if err != nil {
		return listing.Page[models.KeluargaPegawai]{}, err
	}
	return listing.Page[models.KeluargaPegawai]{
		Items:      items,
		Page:       options.Page,
		PageSize:   options.PageSize,
		TotalItems: total,
		TotalPages: listing.PageCount(total, options.PageSize),
	}, nil
}

// Get mengembalikan satu catatan keluarga bila principal berhak membacanya.
func (s *KeluargaService) Get(ctx context.Context, principal auth.Principal, id string) (models.KeluargaPegawai, error) {
	needsScope := allows(s.policy, principal, "keluarga:read:homebase") && !allows(s.policy, principal, "keluarga:read:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.KeluargaPegawai{}, err
	}

	item, err := s.repo.GetKeluarga(ctx, id)
	if err != nil {
		return models.KeluargaPegawai{}, err
	}
	if !s.canRead(scoped, item) {
		return models.KeluargaPegawai{}, domain.ErrForbidden
	}
	return item, nil
}

// Create membuat catatan keluarga pegawai baru.
func (s *KeluargaService) Create(ctx context.Context, principal auth.Principal, input domain.CreateKeluargaPegawai) (models.KeluargaPegawai, error) {
	if err := input.Validate(); err != nil {
		return models.KeluargaPegawai{}, err
	}

	needsScope := allows(s.policy, principal, "keluarga:create:homebase") && !allows(s.policy, principal, "keluarga:create")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.KeluargaPegawai{}, err
	}
	if !s.canCreate(scoped, &input.UserID) {
		return models.KeluargaPegawai{}, domain.ErrForbidden
	}

	tanggalLahir, err := time.Parse("2006-01-02", input.TanggalLahir)
	if err != nil {
		return models.KeluargaPegawai{}, domain.ErrValidation
	}

	item := models.KeluargaPegawai{
		UserId:           input.UserID,
		Hubungan:         input.Hubungan,
		Nama:             input.Nama,
		Nik:              input.NIK,
		TanggalLahir:     tanggalLahir,
		Pekerjaan:        input.Pekerjaan,
		StatusTanggungan: input.StatusTanggungan,
	}
	return s.repo.CreateKeluarga(ctx, principal.UserID, item)
}

// Update memperbarui catatan keluarga pegawai bila principal berhak mengubahnya.
func (s *KeluargaService) Update(ctx context.Context, principal auth.Principal, id string, patch domain.KeluargaPatch) (models.KeluargaPegawai, error) {
	if !patch.HasChanges() {
		return models.KeluargaPegawai{}, domain.ErrValidation
	}
	if err := patch.Validate(); err != nil {
		return models.KeluargaPegawai{}, err
	}

	needsScope := allows(s.policy, principal, "keluarga:update:homebase") && !allows(s.policy, principal, "keluarga:update:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.KeluargaPegawai{}, err
	}

	item, err := s.repo.GetKeluarga(ctx, id)
	if err != nil {
		return models.KeluargaPegawai{}, err
	}
	if !s.canWrite(scoped, item) {
		return models.KeluargaPegawai{}, domain.ErrForbidden
	}
	return s.repo.UpdateKeluarga(ctx, principal.UserID, id, patch)
}

// Archive menonaktifkan catatan keluarga pegawai (soft delete) bila principal berwenang.
func (s *KeluargaService) Archive(ctx context.Context, principal auth.Principal, id string) error {
	needsScope := allows(s.policy, principal, "keluarga:archive:homebase") && !allows(s.policy, principal, "keluarga:archive")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return err
	}

	item, err := s.repo.GetKeluarga(ctx, id)
	if err != nil {
		return err
	}
	if !s.canArchive(scoped, item) {
		return domain.ErrForbidden
	}
	return s.repo.ArchiveKeluarga(ctx, principal.UserID, id)
}

// scopedPrincipal menyusun principal beserta cakupan unitnya.
func (s *KeluargaService) scopedPrincipal(ctx context.Context, principal auth.Principal, needsScope bool) (scopedPrincipal, error) {
	if !needsScope {
		return scopedPrincipal{principal: principal}, nil
	}
	scope, err := s.scope.resolve(ctx, principal)
	if err != nil {
		return scopedPrincipal{}, err
	}
	return scopedPrincipal{principal: principal, scope: scope}, nil
}

// canRead memeriksa akses baca ke satu catatan keluarga.
func (s *KeluargaService) canRead(scoped scopedPrincipal, item models.KeluargaPegawai) bool {
	for _, permission := range []auth.Permission{"keluarga:read:any", "keluarga:read:homebase", "keluarga:read:self"} {
		if auth.CanAccessUserInScope(scoped.principal, item.UserId, item.UserId, permission, s.policy, scoped.scope) {
			return true
		}
	}
	return false
}

// canCreate memeriksa akses membuat catatan keluarga.
func (s *KeluargaService) canCreate(scoped scopedPrincipal, userID *string) bool {
	if allows(s.policy, scoped.principal, "keluarga:create") {
		return true
	}
	if !allows(s.policy, scoped.principal, "keluarga:create:homebase") {
		return false
	}
	if userID == nil || *userID == "" {
		return false
	}
	return scoped.scope.Contains(*userID)
}

// canWrite memeriksa akses ubah satu catatan keluarga.
func (s *KeluargaService) canWrite(scoped scopedPrincipal, item models.KeluargaPegawai) bool {
	for _, permission := range []auth.Permission{"keluarga:update:any", "keluarga:update:homebase"} {
		if auth.CanAccessUserInScope(scoped.principal, item.UserId, item.UserId, permission, s.policy, scoped.scope) {
			return true
		}
	}
	return false
}

// canArchive memeriksa akses arsip satu catatan keluarga.
func (s *KeluargaService) canArchive(scoped scopedPrincipal, item models.KeluargaPegawai) bool {
	for _, permission := range []auth.Permission{"keluarga:archive", "keluarga:archive:homebase"} {
		if auth.CanAccessUserInScope(scoped.principal, item.UserId, item.UserId, permission, s.policy, scoped.scope) {
			return true
		}
	}
	return false
}
