// Package app memuat use case pemberhentian pegawai: pengajuan
// (operator/kabiro unit), persetujuan/penolakan (admin kantor pusat),
// rincian dan riwayat.
package app

import (
	"context"
	"strings"
	"time"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/models"

	"simpeg/backend/internal/pemberhentian/domain"
	"simpeg/backend/internal/pemberhentian/repository"
)

// ScopeResolver dipakai untuk memetakan homebaseId ke pohon subtree (unit
// beserta turunannya). Diimplementasikan oleh repository direktori.
type ScopeResolver interface {
	ListUnitScopeIDs(ctx context.Context, rootID string) ([]string, error)
}

// Store adalah port persistensi yang dibutuhkan use case.
type Store interface {
	EmployeeHomebaseID(ctx context.Context, userID string) (*string, error)
	// EmployeeExists melaporkan apakah pegawai dikenal; dipakai agar pengajuan
	// dengan UUID yang salah ketik ditolak 404, bukan gagal constraint database.
	EmployeeExists(ctx context.Context, userID string) (bool, error)
	EmployeeStartDate(ctx context.Context, userID string) (*time.Time, error)
	EmployeeKategoriStatusID(ctx context.Context, userID string) (*string, error)
	PendingForUser(ctx context.Context, userID string) (bool, error)
	RulesFor(ctx context.Context, kategoriStatusID string) (map[int]float64, error)
	Create(ctx context.Context, actor string, input repository.ProposeInput, unitID *string) (models.Pemberhentian, error)
	Get(ctx context.Context, id string) (models.Pemberhentian, error)
	List(ctx context.Context, options repository.ListOptions) ([]models.Pemberhentian, int64, error)
	Decide(ctx context.Context, actor, id string, approve bool, kompensasiBulan float64, note *string, unitID *string) (models.Pemberhentian, error)
}

// Service menjalankan use case pemberhentian pegawai.
type Service struct {
	repo   Store
	policy auth.Policy
	scope  ScopeResolver
}

// NewService membuat Service.
func NewService(repo Store, policy auth.Policy, scope ScopeResolver) *Service {
	return &Service{repo: repo, policy: policy, scope: scope}
}

// Propose membuat pengajuan pemberhentian.
//
// Operator hanya dapat mengajukan untuk pegawai pada unit penugasannya
// beserta turunannya. Admin dapat mengajukan untuk siapa pun (kantor pusat).
func (s *Service) Propose(ctx context.Context, principal auth.Principal, input repository.ProposeInput) (models.Pemberhentian, error) {
	alasan := domain.NormalizeAlasan(input.Alasan)
	if !domain.ValidAlasan(alasan) {
		return models.Pemberhentian{}, domain.ErrInvalidAlasan
	}
	input.Alasan = alasan
	if input.Tanggal.IsZero() {
		return models.Pemberhentian{}, domain.ErrInvalidDate
	}
	userID := strings.TrimSpace(input.UserID)
	if userID == "" {
		return models.Pemberhentian{}, domain.ErrInvalidUserID
	}
	input.UserID = userID

	needsScope := allows(ctx, s.policy, principal, "pemberhentian:propose:homebase") && !allows(ctx, s.policy, principal, "pemberhentian:propose:any")
	if !allows(ctx, s.policy, principal, "pemberhentian:propose:any") && !needsScope {
		return models.Pemberhentian{}, domain.ErrForbidden
	}

	homebaseID, err := s.repo.EmployeeHomebaseID(ctx, userID)
	if err != nil {
		return models.Pemberhentian{}, err
	}
	exists, err := s.repo.EmployeeExists(ctx, userID)
	if err != nil {
		return models.Pemberhentian{}, err
	}
	if !exists {
		return models.Pemberhentian{}, domain.ErrNotFound
	}
	var scopeIDs []string
	if needsScope {
		if principal.HomebaseID == "" {
			return models.Pemberhentian{}, domain.ErrForbidden
		}
		ids, err := s.scope.ListUnitScopeIDs(ctx, principal.HomebaseID)
		if err != nil {
			return models.Pemberhentian{}, err
		}
		scopeIDs = ids
		found := false
		if homebaseID != nil && *homebaseID != "" {
			for _, id := range scopeIDs {
				if id == *homebaseID {
					found = true
					break
				}
			}
		}
		if !found {
			return models.Pemberhentian{}, domain.ErrForbidden
		}
	}

	// JenisFromAlasan menurunkan jenis pemberhentian dari alasan; dilakukan
	// sebelum alur otorisasi agar pengajuan dengan alasan tak dikenal
	// ditolak lebih awal.
	jenis, err := domain.JenisFromAlasan(alasan)
	if err != nil {
		return models.Pemberhentian{}, err
	}
	input.Jenis = jenis

	pending, err := s.repo.PendingForUser(ctx, userID)
	if err != nil {
		return models.Pemberhentian{}, err
	}
	if pending {
		return models.Pemberhentian{}, domain.ErrPending
	}

	var unitID *string
	if homebaseID != nil && *homebaseID != "" {
		unitID = homebaseID
	} else if principal.HomebaseID != "" {
		unitID = &principal.HomebaseID
	}

	return s.repo.Create(ctx, principal.UserID, input, unitID)
}

// List mengembalikan daftar pengajuan/keputusan; admin melihat seluruh
// organisasi, operator hanya unit penugasannya beserta turunannya.
func (s *Service) List(ctx context.Context, principal auth.Principal, options repository.ListOptions) ([]models.Pemberhentian, int64, error) {
	if allows(ctx, s.policy, principal, "pemberhentian:read:any") {
		// Administrative view: seluruh organisasi.
	} else if allows(ctx, s.policy, principal, "pemberhentian:read:homebase") {
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

// Get mengembalikan satu catatan pemberhentian sesuai cakupan.
func (s *Service) Get(ctx context.Context, principal auth.Principal, id string) (models.Pemberhentian, error) {
	item, err := s.repo.Get(ctx, id)
	if err != nil {
		return models.Pemberhentian{}, err
	}
	if allows(ctx, s.policy, principal, "pemberhentian:read:any") {
		return item, nil
	}
	if allows(ctx, s.policy, principal, "pemberhentian:read:homebase") && principal.HomebaseID != "" {
		ids, err := s.scope.ListUnitScopeIDs(ctx, principal.HomebaseID)
		if err != nil {
			return models.Pemberhentian{}, err
		}
		homebaseID, err := s.repo.EmployeeHomebaseID(ctx, item.UserId)
		if err != nil {
			return models.Pemberhentian{}, err
		}
		if homebaseID != nil {
			for _, id := range ids {
				if id == *homebaseID {
					return item, nil
				}
			}
		}
	}
	return models.Pemberhentian{}, domain.ErrForbidden
}

// Approve menyetujui pengajuan dan menerbitkan SK PHK. Hanya admin.
func (s *Service) Approve(ctx context.Context, principal auth.Principal, id string, note *string) (models.Pemberhentian, error) {
	if !allows(ctx, s.policy, principal, "pemberhentian:approve") {
		return models.Pemberhentian{}, domain.ErrForbidden
	}
	item, err := s.repo.Get(ctx, id)
	if err != nil {
		return models.Pemberhentian{}, err
	}
	if item.Status != models.StatusMenunggu {
		return models.Pemberhentian{}, domain.ErrNotPending
	}

	homebaseID, err := s.repo.EmployeeHomebaseID(ctx, item.UserId)
	if err != nil {
		return models.Pemberhentian{}, err
	}
	var unitID *string
	if homebaseID != nil && *homebaseID != "" {
		unitID = homebaseID
	}

	masaKerja := 0
	if start, err := s.repo.EmployeeStartDate(ctx, item.UserId); err == nil && start != nil {
		masaKerja = yearsBetween(*start, item.Tanggal)
	}

	alasan := ""
	if item.Alasan != nil {
		alasan = *item.Alasan
	}
	honoured := domain.Honoured(alasan)

	kompensasiBulan := 0.0
	if honoured {
		katID, err := s.repo.EmployeeKategoriStatusID(ctx, item.UserId)
		if err == nil && katID != nil && *katID != "" {
			if rules, err := s.repo.RulesFor(ctx, *katID); err == nil {
				kompensasi := domain.HitungCompensation(rules, masaKerja, true)
				kompensasiBulan = kompensasi.TotalBulan
			}
		}
	}

	return s.repo.Decide(ctx, principal.UserID, id, true, kompensasiBulan, note, unitID)
}

// Reject menolak pengajuan. Hanya admin.
func (s *Service) Reject(ctx context.Context, principal auth.Principal, id string, note *string) (models.Pemberhentian, error) {
	if !allows(ctx, s.policy, principal, "pemberhentian:approve") {
		return models.Pemberhentian{}, domain.ErrForbidden
	}
	return s.repo.Decide(ctx, principal.UserID, id, false, 0, note, nil)
}

// CompensasiPreview adalah pratinjau kompensasi pemberhentian yang
// ditampilkan sebelum approve (`DEC-083`). Diisi atasan sebelum keputusan.
type CompensasiPreview struct {
	Honoured       bool    `json:"honoured"`
	Tersedia       bool    `json:"tersedia"`
	MasaKerjaTahun int     `json:"masaKerjaTahun"`
	PesangonBulan  float64 `json:"pesangonBulan"`
	TambahanBulan  float64 `json:"tambahanBulan"`
	TotalBulan     float64 `json:"totalBulan"`
}

// CompensasiPreview menghitung pratinjau kompensasi pengajuan; tidak
// menuliskan apa pun. Hanya admin (kabiro pusat) yang dapat memanggilnya —
// sama seperti approve.
func (s *Service) CompensasiPreview(ctx context.Context, principal auth.Principal, id string) (CompensasiPreview, error) {
	if !allows(ctx, s.policy, principal, "pemberhentian:approve") {
		return CompensasiPreview{}, domain.ErrForbidden
	}
	item, err := s.repo.Get(ctx, id)
	if err != nil {
		return CompensasiPreview{}, err
	}
	alasan := ""
	if item.Alasan != nil {
		alasan = *item.Alasan
	}
	honoured := domain.Honoured(alasan)

	preview := CompensasiPreview{Honoured: honoured}
	if !honoured {
		return preview, nil
	}
	if start, err := s.repo.EmployeeStartDate(ctx, item.UserId); err == nil && start != nil {
		preview.MasaKerjaTahun = yearsBetween(*start, item.Tanggal)
	}
	kategoriID, err := s.repo.EmployeeKategoriStatusID(ctx, item.UserId)
	if err != nil || kategoriID == nil || *kategoriID == "" {
		return preview, nil
	}
	rules, err := s.repo.RulesFor(ctx, *kategoriID)
	if err != nil {
		return CompensasiPreview{}, err
	}
	comp := domain.HitungCompensation(rules, preview.MasaKerjaTahun, true)
	preview.PesangonBulan = comp.PesangonBulan
	preview.TambahanBulan = comp.TambahanBulan
	preview.TotalBulan = comp.TotalBulan
	preview.Tersedia = comp.Tersedia
	return preview, nil
}

// yearsBetween menghitung tahun penuh antara dua tanggal.
func yearsBetween(start, end time.Time) int {
	years := end.Year() - start.Year()
	if end.Month() < start.Month() || (end.Month() == start.Month() && end.Day() < start.Day()) {
		years--
	}
	if years < 0 {
		years = 0
	}
	return years
}

// allows melaporkan apakah principal memiliki permission tertentu.
func allows(ctx context.Context, policy auth.Policy, principal auth.Principal, permission string) bool {
	if policy == nil {
		return false
	}
	if sp, ok := policy.(auth.ScopedPolicy); ok {
		return sp.AllowsScoped(ctx, principal, auth.Permission(permission))
	}
	return policy.Allows(principal.RoleName, auth.Permission(permission))
}
