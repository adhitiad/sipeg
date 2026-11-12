// Package app memuat logika aplikasi (use case) untuk pengajuan cuti pegawai.
package app

import (
	"context"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/directory/domain"
	"simpeg/backend/internal/directory/repository"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
)

// LeaveService menjalankan use case pengajuan cuti pegawai.
type LeaveService struct {
	repo   repository.LeaveRepository
	policy auth.Policy
	scope  scopeResolver
}

// NewLeaveService membuat service pengajuan cuti.
func NewLeaveService(repo repository.LeaveRepository, policy auth.Policy, scopeRepo repository.ScopeRepository) *LeaveService {
	return &LeaveService{repo: repo, policy: policy, scope: newScopeResolver(scopeRepo, policy)}
}

// List mengembalikan daftar pengajuan cuti sesuai filter dan cakupan principal.
func (s *LeaveService) List(ctx context.Context, principal auth.Principal, options listing.Options) (listing.Page[models.Cuti], error) {
	switch {
	case allows(s.policy, principal, "cuti:read:any"):
	case allows(s.policy, principal, "cuti:read:homebase"):
		scope, err := s.scope.resolve(ctx, principal)
		if err != nil {
			return listing.Page[models.Cuti]{}, err
		}
		if scope.Empty() {
			return listing.EmptyPage[models.Cuti](options.Page, options.PageSize), nil
		}
		options.ScopeHomebaseIDs = scope.List()
	case allows(s.policy, principal, "cuti:read:self"):
		options.UserID = principal.UserID
	default:
		return listing.Page[models.Cuti]{}, domain.ErrForbidden
	}

	items, total, err := s.repo.ListCuti(ctx, options)
	if err != nil {
		return listing.Page[models.Cuti]{}, err
	}
	return listing.Page[models.Cuti]{
		Items:      items,
		Page:       options.Page,
		PageSize:   options.PageSize,
		TotalItems: total,
		TotalPages: listing.PageCount(total, options.PageSize),
	}, nil
}

// Get mengembalikan satu pengajuan cuti bila principal berhak membacanya.
func (s *LeaveService) Get(ctx context.Context, principal auth.Principal, id string) (models.Cuti, error) {
	needsScope := allows(s.policy, principal, "cuti:read:homebase") && !allows(s.policy, principal, "cuti:read:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.Cuti{}, err
	}

	item, err := s.repo.GetCuti(ctx, id)
	if err != nil {
		return models.Cuti{}, err
	}
	if !s.canRead(scoped, item) {
		return models.Cuti{}, domain.ErrForbidden
	}
	return item, nil
}

// Create membuat pengajuan cuti baru. Status awal selalu MENUNGGU.
func (s *LeaveService) Create(ctx context.Context, principal auth.Principal, input domain.CreateCuti) (models.Cuti, error) {
	needsScope := allows(s.policy, principal, "cuti:create:homebase") && !allows(s.policy, principal, "cuti:create")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.Cuti{}, err
	}
	if !s.canCreate(scoped, input.UserID) {
		return models.Cuti{}, domain.ErrForbidden
	}
	item := input.Record()
	if err := domain.ValidasiCuti(item); err != nil {
		return models.Cuti{}, err
	}
	return s.repo.CreateCuti(ctx, principal.UserID, item)
}

// Update mengubah pengajuan cuti yang masih MENUNGGU.
func (s *LeaveService) Update(ctx context.Context, principal auth.Principal, id string, patch domain.CutiPatch) (models.Cuti, error) {
	if !patch.HasChanges() {
		return models.Cuti{}, domain.ErrValidation
	}
	if err := patch.Validate(); err != nil {
		return models.Cuti{}, err
	}
	needsScope := allows(s.policy, principal, "cuti:update:homebase") && !allows(s.policy, principal, "cuti:update:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.Cuti{}, err
	}
	item, err := s.repo.GetCuti(ctx, id)
	if err != nil {
		return models.Cuti{}, err
	}
	if !s.canWrite(scoped, item) {
		return models.Cuti{}, domain.ErrForbidden
	}
	merged, err := domain.ApplyCutiPatch(item, patch)
	if err != nil {
		return models.Cuti{}, err
	}
	if err := domain.ValidasiCuti(merged); err != nil {
		return models.Cuti{}, err
	}
	return s.repo.UpdateCuti(ctx, principal.UserID, id, merged)
}

// Approve menyetujui pengajuan cuti. Hanya atasan/HRD dengan cakupan unit.
func (s *LeaveService) Approve(ctx context.Context, principal auth.Principal, id string, catatan *string) (models.Cuti, error) {
	needsScope := allows(s.policy, principal, "cuti:approve:homebase") && !allows(s.policy, principal, "cuti:approve:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.Cuti{}, err
	}
	item, err := s.repo.GetCuti(ctx, id)
	if err != nil {
		return models.Cuti{}, err
	}
	if !s.canApprove(scoped, item) {
		return models.Cuti{}, domain.ErrForbidden
	}
	// Eviden dicek ulang saat persetujuan supaya pengajuan yang sudah
	// tercatat tanpa bukti tidak dapat disetujui.
	if err := domain.ValidasiEviden(item); err != nil {
		return models.Cuti{}, err
	}
	return s.repo.ApproveCuti(ctx, principal.UserID, id, catatan)
}

// Reject menolak pengajuan cuti. Hanya atasan/HRD dengan cakupan unit.
func (s *LeaveService) Reject(ctx context.Context, principal auth.Principal, id string, catatan *string) (models.Cuti, error) {
	needsScope := allows(s.policy, principal, "cuti:approve:homebase") && !allows(s.policy, principal, "cuti:approve:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.Cuti{}, err
	}
	item, err := s.repo.GetCuti(ctx, id)
	if err != nil {
		return models.Cuti{}, err
	}
	if !s.canApprove(scoped, item) {
		return models.Cuti{}, domain.ErrForbidden
	}
	return s.repo.RejectCuti(ctx, principal.UserID, id, catatan)
}

func (s *LeaveService) scopedPrincipal(ctx context.Context, principal auth.Principal, needsScope bool) (scopedPrincipal, error) {
	if !needsScope {
		return scopedPrincipal{principal: principal}, nil
	}
	scope, err := s.scope.resolve(ctx, principal)
	if err != nil {
		return scopedPrincipal{}, err
	}
	return scopedPrincipal{principal: principal, scope: scope}, nil
}

func (s *LeaveService) canRead(scoped scopedPrincipal, item models.Cuti) bool {
	if allows(s.policy, scoped.principal, "cuti:read:any") {
		return true
	}
	if scoped.principal.UserID == item.UserId && allows(s.policy, scoped.principal, "cuti:read:self") {
		return true
	}
	if allows(s.policy, scoped.principal, "cuti:read:homebase") && !scoped.scope.Empty() {
		homebaseID, err := s.repo.UserHomebaseID(context.Background(), item.UserId)
		if err == nil && homebaseID != "" && scoped.scope.Contains(homebaseID) {
			return true
		}
	}
	return false
}

func (s *LeaveService) canCreate(scoped scopedPrincipal, userID string) bool {
	if allows(s.policy, scoped.principal, "cuti:create:any") {
		return true
	}
	if allows(s.policy, scoped.principal, "cuti:create") {
		// Tanpa suffix :any atau :homebase = hanya untuk diri sendiri.
		return scoped.principal.UserID == userID
	}
	if !allows(s.policy, scoped.principal, "cuti:create:homebase") {
		return false
	}
	if scoped.principal.UserID == userID {
		return true
	}
	if scoped.scope.Empty() {
		return false
	}
	homebaseID, err := s.repo.UserHomebaseID(context.Background(), userID)
	if err != nil || homebaseID == "" {
		return false
	}
	return scoped.scope.Contains(homebaseID)
}

func (s *LeaveService) canWrite(scoped scopedPrincipal, item models.Cuti) bool {
	if allows(s.policy, scoped.principal, "cuti:update:any") {
		return true
	}
	if scoped.principal.UserID == item.UserId && allows(s.policy, scoped.principal, "cuti:update:homebase") {
		return true
	}
	if allows(s.policy, scoped.principal, "cuti:update:homebase") && !scoped.scope.Empty() {
		homebaseID, err := s.repo.UserHomebaseID(context.Background(), item.UserId)
		if err == nil && homebaseID != "" && scoped.scope.Contains(homebaseID) {
			return true
		}
	}
	return false
}

func (s *LeaveService) canApprove(scoped scopedPrincipal, item models.Cuti) bool {
	if allows(s.policy, scoped.principal, "cuti:approve:any") {
		return true
	}
	if allows(s.policy, scoped.principal, "cuti:approve:homebase") && !scoped.scope.Empty() {
		homebaseID, err := s.repo.UserHomebaseID(context.Background(), item.UserId)
		if err == nil && homebaseID != "" && scoped.scope.Contains(homebaseID) {
			return true
		}
	}
	return false
}
