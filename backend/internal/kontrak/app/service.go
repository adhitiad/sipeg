// Package app memuat use case perpanjangan masa kerja dosen pasca-BUP dan
// daftar kontrak kerja (DEC-083).
package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/kontrak/domain"
	"simpeg/backend/internal/kontrak/repository"
	"simpeg/backend/internal/models"
)

// ScopeResolver memetakan homebaseId ke pohon unit (unit beserta turunannya).
type ScopeResolver interface {
	ListUnitScopeIDs(ctx context.Context, rootID string) ([]string, error)
}

// Store adalah port persistensi yang dibutuhkan use case.
type Store interface {
	EmployeeHomebaseID(ctx context.Context, userID string) (*string, error)
	EmployeeExists(ctx context.Context, userID string) (bool, error)
	EmployeeKategoriStatusID(ctx context.Context, userID string) (*string, error)
	CountByTypeYear(ctx context.Context, jenis models.JenisKontrak, year int) (int64, error)
	ActiveOverlapping(ctx context.Context, userID string, mulai, selesai time.Time) (bool, error)
	Create(ctx context.Context, actor string, input repository.KontrakInput, nomorSk string, newKategoriStatusID string, unitID *string) (models.KontrakKerja, error)
	List(ctx context.Context, filter repository.KontrakFilter) ([]models.KontrakKerja, int64, error)
}

// Service menjalankan use case perpanjangan + kontrak.
type Service struct {
	repo   Store
	policy auth.Policy
	scope  ScopeResolver
}

// NewService membuat Service.
func NewService(repo Store, policy auth.Policy, scope ScopeResolver) *Service {
	return &Service{repo: repo, policy: policy, scope: scope}
}

// Extend memproses perpanjangan masa kerja dosen pasca-BUP sebagai
// perjanjian kerja tahunan. Admin (kabiro pusat) sendiri yang memicu;
// syarat pangkat diverifikasi di luar sistem (DEC-083). Perjanjian baru
// yang tumpang tindih dengan kontrak AKTIF pegawai ditolak 409;
// `allowOverlap` memaksa perjanjian dibuat dan tercatat di audit.
func (s *Service) Extend(ctx context.Context, principal auth.Principal, userID string, mulai *time.Time, durasiBulan int, allowOverlap bool) (models.KontrakKerja, error) {
	if !allows(ctx, s.policy, principal, "perpanjangan:manage") {
		return models.KontrakKerja{}, domain.ErrForbidden
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return models.KontrakKerja{}, domain.ErrInvalidInput
	}
	exists, err := s.repo.EmployeeExists(ctx, userID)
	if err != nil {
		return models.KontrakKerja{}, err
	}
	if !exists {
		return models.KontrakKerja{}, domain.ErrNotFound
	}

	start := time.Now().UTC()
	if mulai != nil {
		start = *mulai
	}
	start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)

	durasi := durasiBulan
	if durasi <= 0 {
		durasi = domain.DurasiBulanPerjanjian
	}
	if durasi > 60 {
		return models.KontrakKerja{}, domain.ErrInvalidInput
	}

	selesai := start.AddDate(0, durasi, 0)
	overlap, err := s.repo.ActiveOverlapping(ctx, userID, start, selesai)
	if err != nil {
		return models.KontrakKerja{}, err
	}
	if overlap && !allowOverlap {
		return models.KontrakKerja{}, domain.ErrOverlap
	}

	seq, err := s.repo.CountByTypeYear(ctx, models.JenisKontrakPKWT, start.Year())
	if err != nil {
		return models.KontrakKerja{}, err
	}
	nomorSk := fmt.Sprintf("PKWT-%d-%04d", start.Year(), seq+1)

	homebaseID, err := s.repo.EmployeeHomebaseID(ctx, userID)
	if err != nil {
		return models.KontrakKerja{}, err
	}
	var unitID *string
	if homebaseID != nil && *homebaseID != "" {
		unitID = homebaseID
	}

	return s.repo.Create(ctx, principal.UserID, repository.KontrakInput{
		UserID:       userID,
		Mulai:        start,
		Durasi:       durasi,
		AllowOverlap: allowOverlap,
	}, nomorSk, domain.KategoriDosenTidakTetapID, unitID)
}

// List mengembalikan daftar kontrak kerja; admin melihat seluruh organisasi,
// operator hanya unit penugasannya beserta turunannya, pegawai hanya
// kontrak miliknya sendiri.
func (s *Service) List(ctx context.Context, principal auth.Principal, filter repository.KontrakFilter) ([]models.KontrakKerja, int64, error) {
	switch {
	case allows(ctx, s.policy, principal, "kontrak:read:any"):
	case allows(ctx, s.policy, principal, "kontrak:read:homebase"):
		if principal.HomebaseID == "" {
			return nil, 0, domain.ErrForbidden
		}
		ids, err := s.scope.ListUnitScopeIDs(ctx, principal.HomebaseID)
		if err != nil {
			return nil, 0, err
		}
		filter.HomebaseIDs = ids
	case allows(ctx, s.policy, principal, "kontrak:read:self"):
		filter.UserID = principal.UserID
	default:
		return nil, 0, domain.ErrForbidden
	}
	return s.repo.List(ctx, filter)
}

// allows melaporkan apakah principal memiliki permission tertentu. Bila policy
// mengimplementasikan auth.ScopedPolicy, evaluasi scoped dipakai (DEC-091).
func allows(ctx context.Context, policy auth.Policy, principal auth.Principal, permission string) bool {
	if policy == nil {
		return false
	}
	if sp, ok := policy.(auth.ScopedPolicy); ok {
		return sp.AllowsScoped(ctx, principal, auth.Permission(permission))
	}
	return policy.Allows(principal.RoleName, auth.Permission(permission))
}
