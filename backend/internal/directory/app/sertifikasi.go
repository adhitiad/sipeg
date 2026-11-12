// Package app memuat logika aplikasi untuk riwayat sertifikasi pegawai.
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

// SertifikasiService menjalankan use case riwayat sertifikasi pegawai.
type SertifikasiService struct {
	repo   repository.SertifikasiRepository
	policy auth.Policy
	scope  scopeResolver
}

// NewSertifikasiService membuat service riwayat sertifikasi.
func NewSertifikasiService(repo repository.SertifikasiRepository, policy auth.Policy, scopeRepo repository.ScopeRepository) *SertifikasiService {
	return &SertifikasiService{repo: repo, policy: policy, scope: newScopeResolver(scopeRepo, policy)}
}

func (s *SertifikasiService) List(ctx context.Context, principal auth.Principal, options listing.Options) (listing.Page[models.RiwayatSertifikasi], error) {
	switch {
	case allows(s.policy, principal, "sertifikasi:read:any"):
	case allows(s.policy, principal, "sertifikasi:read:homebase"):
		scope, err := s.scope.resolve(ctx, principal)
		if err != nil {
			return listing.Page[models.RiwayatSertifikasi]{}, err
		}
		if scope.Empty() {
			return listing.EmptyPage[models.RiwayatSertifikasi](options.Page, options.PageSize), nil
		}
		options.ScopeHomebaseIDs = scope.List()
	case allows(s.policy, principal, "sertifikasi:read:self"):
		options.UserID = principal.UserID
	default:
		return listing.Page[models.RiwayatSertifikasi]{}, domain.ErrForbidden
	}
	items, total, err := s.repo.ListSertifikasi(ctx, options)
	if err != nil {
		return listing.Page[models.RiwayatSertifikasi]{}, err
	}
	return listing.Page[models.RiwayatSertifikasi]{
		Items: items, Page: options.Page, PageSize: options.PageSize,
		TotalItems: total, TotalPages: listing.PageCount(total, options.PageSize),
	}, nil
}

func (s *SertifikasiService) Get(ctx context.Context, principal auth.Principal, id string) (models.RiwayatSertifikasi, error) {
	needsScope := allows(s.policy, principal, "sertifikasi:read:homebase") && !allows(s.policy, principal, "sertifikasi:read:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.RiwayatSertifikasi{}, err
	}
	item, err := s.repo.GetSertifikasi(ctx, id)
	if err != nil {
		return models.RiwayatSertifikasi{}, err
	}
	if !s.canRead(scoped, item) {
		return models.RiwayatSertifikasi{}, domain.ErrForbidden
	}
	return item, nil
}

func (s *SertifikasiService) Create(ctx context.Context, principal auth.Principal, input domain.CreateRiwayatSertifikasi) (models.RiwayatSertifikasi, error) {
	if err := input.Validate(); err != nil {
		return models.RiwayatSertifikasi{}, err
	}
	needsScope := allows(s.policy, principal, "sertifikasi:create:homebase") && !allows(s.policy, principal, "sertifikasi:create")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.RiwayatSertifikasi{}, err
	}
	if !s.canCreate(scoped, &input.UserID) {
		return models.RiwayatSertifikasi{}, domain.ErrForbidden
	}
	tanggalTerbit, err := time.Parse("2006-01-02", input.TanggalTerbit)
	if err != nil {
		return models.RiwayatSertifikasi{}, domain.ErrValidation
	}
	var berlakuSampai *time.Time
	if input.BerlakuSampai != nil {
		t, err := time.Parse("2006-01-02", *input.BerlakuSampai)
		if err != nil {
			return models.RiwayatSertifikasi{}, domain.ErrValidation
		}
		berlakuSampai = &t
	}
	item := models.RiwayatSertifikasi{
		UserId:           input.UserID,
		JenisSertifikasi: input.JenisSertifikasi,
		NamaSertifikat:   input.NamaSertifikat,
		NomorRegistrasi:  input.NomorRegistrasi,
		Penyelenggara:    input.Penyelenggara,
		TanggalTerbit:    tanggalTerbit,
		BerlakuSampai:    berlakuSampai,
		Status:           input.Status,
	}
	return s.repo.CreateSertifikasi(ctx, principal.UserID, item)
}

func (s *SertifikasiService) Update(ctx context.Context, principal auth.Principal, id string, patch domain.RiwayatSertifikasiPatch) (models.RiwayatSertifikasi, error) {
	if !patch.HasChanges() {
		return models.RiwayatSertifikasi{}, domain.ErrValidation
	}
	if err := patch.Validate(); err != nil {
		return models.RiwayatSertifikasi{}, err
	}
	needsScope := allows(s.policy, principal, "sertifikasi:update:homebase") && !allows(s.policy, principal, "sertifikasi:update:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.RiwayatSertifikasi{}, err
	}
	item, err := s.repo.GetSertifikasi(ctx, id)
	if err != nil {
		return models.RiwayatSertifikasi{}, err
	}
	if !s.canWrite(scoped, item) {
		return models.RiwayatSertifikasi{}, domain.ErrForbidden
	}
	return s.repo.UpdateSertifikasi(ctx, principal.UserID, id, patch)
}

func (s *SertifikasiService) Archive(ctx context.Context, principal auth.Principal, id string) error {
	needsScope := allows(s.policy, principal, "sertifikasi:archive:homebase") && !allows(s.policy, principal, "sertifikasi:archive")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return err
	}
	item, err := s.repo.GetSertifikasi(ctx, id)
	if err != nil {
		return err
	}
	if !s.canArchive(scoped, item) {
		return domain.ErrForbidden
	}
	return s.repo.ArchiveSertifikasi(ctx, principal.UserID, id)
}

func (s *SertifikasiService) scopedPrincipal(ctx context.Context, principal auth.Principal, needsScope bool) (scopedPrincipal, error) {
	if !needsScope {
		return scopedPrincipal{principal: principal}, nil
	}
	scope, err := s.scope.resolve(ctx, principal)
	if err != nil {
		return scopedPrincipal{}, err
	}
	return scopedPrincipal{principal: principal, scope: scope}, nil
}

func (s *SertifikasiService) canRead(scoped scopedPrincipal, item models.RiwayatSertifikasi) bool {
	for _, permission := range []auth.Permission{"sertifikasi:read:any", "sertifikasi:read:homebase", "sertifikasi:read:self"} {
		if auth.CanAccessUserInScope(scoped.principal, item.UserId, item.UserId, permission, s.policy, scoped.scope) {
			return true
		}
	}
	return false
}

func (s *SertifikasiService) canCreate(scoped scopedPrincipal, userID *string) bool {
	if allows(s.policy, scoped.principal, "sertifikasi:create") {
		return true
	}
	if !allows(s.policy, scoped.principal, "sertifikasi:create:homebase") {
		return false
	}
	if userID == nil || *userID == "" {
		return false
	}
	return scoped.scope.Contains(*userID)
}

func (s *SertifikasiService) canWrite(scoped scopedPrincipal, item models.RiwayatSertifikasi) bool {
	for _, permission := range []auth.Permission{"sertifikasi:update:any", "sertifikasi:update:homebase"} {
		if auth.CanAccessUserInScope(scoped.principal, item.UserId, item.UserId, permission, s.policy, scoped.scope) {
			return true
		}
	}
	return false
}

func (s *SertifikasiService) canArchive(scoped scopedPrincipal, item models.RiwayatSertifikasi) bool {
	for _, permission := range []auth.Permission{"sertifikasi:archive", "sertifikasi:archive:homebase"} {
		if auth.CanAccessUserInScope(scoped.principal, item.UserId, item.UserId, permission, s.policy, scoped.scope) {
			return true
		}
	}
	return false
}
