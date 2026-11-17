// Package app memuat use case penerbitan SP1 (peringatan) pegawai: pembuatan
// catatan peringatan oleh admin/kabiro, pembacaan riwayat dengan cakupan
// unit. SP1 hanya pencatatan peringatan dan tidak memengaruhi slip gaji.
package app

import (
	"context"
	"strings"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/models"

	"simpeg/backend/internal/peringatan/domain"
	"simpeg/backend/internal/peringatan/repository"
)

// ScopeResolver dipakai untuk memetakan homebaseId ke pohon subtree.
type ScopeResolver interface {
	ListUnitScopeIDs(ctx context.Context, rootID string) ([]string, error)
}

// Store adalah port persistensi yang dibutuhkan use case.
type Store interface {
	repository.Store
}

// Service menjalankan use case penerbitan peringatan.
type Service struct {
	repo   repository.Store
	policy auth.Policy
	scope  ScopeResolver
}

// NewService membuat Service.
func NewService(repo repository.Store, policy auth.Policy, scope ScopeResolver) *Service {
	return &Service{repo: repo, policy: policy, scope: scope}
}

// Create menerbitkan SP1. Hanya pemegang permission peringatan:sp1:manage.
func (s *Service) Create(ctx context.Context, principal auth.Principal, input repository.CreateInput) (models.Peringatan, error) {
	if !allows(ctx, s.policy, principal, "peringatan:sp1:manage") {
		return models.Peringatan{}, domain.ErrForbidden
	}
	userID := strings.TrimSpace(input.UserID)
	if userID == "" {
		return models.Peringatan{}, domain.ErrInvalidUserID
	}
	input.UserID = userID
	input.Alasan = strings.TrimSpace(input.Alasan)
	input.Penerbit = strings.TrimSpace(input.Penerbit)
	if input.NomorSurat != nil {
		v := strings.TrimSpace(*input.NomorSurat)
		if v == "" {
			input.NomorSurat = nil
		} else {
			input.NomorSurat = &v
		}
	}
	if err := domain.ValidateInput(input.UserID, input.Alasan, input.Penerbit, input.Tingkat, input.Tanggal, input.Sanksi); err != nil {
		return models.Peringatan{}, err
	}

	exists, err := s.repo.EmployeeExists(ctx, userID)
	if err != nil {
		return models.Peringatan{}, err
	}
	if !exists {
		return models.Peringatan{}, domain.ErrNotFound
	}

	homebaseID, err := s.repo.EmployeeHomebaseID(ctx, userID)
	if err != nil {
		return models.Peringatan{}, err
	}
	var unitID *string
	if homebaseID != nil && *homebaseID != "" {
		unitID = homebaseID
	} else if principal.HomebaseID != "" {
		unitID = &principal.HomebaseID
	}

	return s.repo.Create(ctx, principal.UserID, input, unitID)
}

// List mengembalikan daftar peringatan; admin melihat seluruh organisasi,
// operator/atasan hanya unit penugasannya beserta turunannya.
func (s *Service) List(ctx context.Context, principal auth.Principal, options repository.ListOptions) ([]models.Peringatan, int64, error) {
	if allows(ctx, s.policy, principal, "peringatan:read:any") {
		// seluruh organisasi
	} else if allows(ctx, s.policy, principal, "peringatan:read:homebase") {
		if principal.HomebaseID == "" {
			return nil, 0, domain.ErrForbidden
		}
		ids, err := s.scope.ListUnitScopeIDs(ctx, principal.HomebaseID)
		if err != nil {
			return nil, 0, err
		}
		options.HomebaseIDs = ids
	} else {
		return nil, 0, domain.ErrForbidden
	}
	return s.repo.List(ctx, options)
}

// Get mengembalikan satu peringatan sesuai cakupan.
func (s *Service) Get(ctx context.Context, principal auth.Principal, id string) (models.Peringatan, error) {
	item, err := s.repo.Get(ctx, id)
	if err != nil {
		return models.Peringatan{}, err
	}
	if allows(ctx, s.policy, principal, "peringatan:read:any") {
		return item, nil
	}
	if allows(ctx, s.policy, principal, "peringatan:read:homebase") && principal.HomebaseID != "" {
		ids, err := s.scope.ListUnitScopeIDs(ctx, principal.HomebaseID)
		if err != nil {
			return models.Peringatan{}, err
		}
		homebaseID, err := s.repo.EmployeeHomebaseID(ctx, item.UserId)
		if err != nil {
			return models.Peringatan{}, err
		}
		if homebaseID != nil {
			for _, u := range ids {
				if u == *homebaseID {
					return item, nil
				}
			}
		}
	}
	return models.Peringatan{}, domain.ErrForbidden
}

func allows(ctx context.Context, policy auth.Policy, principal auth.Principal, permission string) bool {
	if policy == nil {
		return false
	}
	if sp, ok := policy.(auth.ScopedPolicy); ok {
		return sp.AllowsScoped(ctx, principal, auth.Permission(permission))
	}
	return policy.Allows(principal.RoleName, auth.Permission(permission))
}
