// Service master pensiun: konfigurasi per kategori status dan kepesertaan
// DPLK per pegawai (khusus admin — `pensiun:manage`).
package app

import (
	"context"
	"errors"
	"strings"
	"time"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/models"

	"simpeg/backend/internal/pensiun/domain"
	"simpeg/backend/internal/pensiun/repository"
)

// Service menjalankan use case pensiun.
type Service struct {
	store  repository.Store
	policy auth.Policy
}

// NewService membuat Service.
func NewService(store repository.Store, policy auth.Policy) *Service {
	return &Service{store: store, policy: policy}
}

// allows melaporkan apakah principal memiliki permission tertentu. Bila
// policy mengimplementasikan auth.ScopedPolicy (PolicyAdapter RBAC,
// `DEC-087`), evaluasi scoped dipakai dan boleh mencabut izin statis.
func allows(ctx context.Context, policy auth.Policy, principal auth.Principal, permission string) bool {
	if policy == nil {
		return false
	}
	if sp, ok := policy.(auth.ScopedPolicy); ok {
		return sp.AllowsScoped(ctx, principal, auth.Permission(permission))
	}
	return policy.Allows(principal.RoleName, auth.Permission(permission))
}

func (s *Service) canManage(ctx context.Context, principal auth.Principal) bool {
	return allows(ctx, s.policy, principal, "pensiun:manage")
}

// readAccess menilai akses baca master pensiun: nil scope berarti lintas
// organisasi (`pensiun:manage`, admin); slice non-nil berarti dibatasi pada
// unit cakupan (`pensiun:read:homebase`, operator/atasan); slice kosong
// berarti principal tanpa unit penugasan yang valid — daftar kosong.
func (s *Service) readAccess(ctx context.Context, principal auth.Principal) ([]string, error) {
	if allows(ctx, s.policy, principal, "pensiun:manage") {
		return nil, nil
	}
	if !allows(ctx, s.policy, principal, "pensiun:read:homebase") {
		return nil, domain.ErrForbidden
	}
	if principal.HomebaseID == "" {
		return []string{}, nil
	}
	ids, err := s.store.ListUnitScopeIDs(ctx, principal.HomebaseID)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return []string{}, nil
	}
	return ids, nil
}

// ListKonfigurasi membaca konfigurasi (admin penuh; operator/atasan dibaca
// tanpa batasan unit — konfigurasi adalah master referensi organisasi).
func (s *Service) ListKonfigurasi(ctx context.Context, principal auth.Principal, kategoriStatusID string) ([]models.PensiunKonfigurasi, error) {
	if _, err := s.readAccess(ctx, principal); err != nil {
		return nil, err
	}
	return s.store.ListKonfigurasi(ctx, strings.TrimSpace(kategoriStatusID))
}

// UpsertKonfigurasi menulis atau mengganti baris (kategoriStatusId, berlakuMulai).
func (s *Service) UpsertKonfigurasi(ctx context.Context, principal auth.Principal, input domain.KonfigurasiInput) (models.PensiunKonfigurasi, error) {
	if !s.canManage(ctx, principal) {
		return models.PensiunKonfigurasi{}, domain.ErrForbidden
	}
	if err := input.Validate(); err != nil {
		return models.PensiunKonfigurasi{}, err
	}
	item := toModelKonfigurasi(input)
	return s.store.UpsertKonfigurasi(ctx, principal.UserID, item)
}

// UpdateKonfigurasi mengganti seluruh isian mutable satu baris konfigurasi.
func (s *Service) UpdateKonfigurasi(ctx context.Context, principal auth.Principal, id string, input domain.KonfigurasiInput) (models.PensiunKonfigurasi, error) {
	if !s.canManage(ctx, principal) {
		return models.PensiunKonfigurasi{}, domain.ErrForbidden
	}
	if strings.TrimSpace(id) == "" {
		return models.PensiunKonfigurasi{}, domain.ErrValidation
	}
	if err := input.Validate(); err != nil {
		return models.PensiunKonfigurasi{}, err
	}
	item := toModelKonfigurasi(input)
	updated, err := s.store.UpdateKonfigurasi(ctx, principal.UserID, id, item)
	if errors.Is(err, repository.ErrNotFoundKonfigurasi) {
		return models.PensiunKonfigurasi{}, domain.ErrConfigNotFound
	}
	return updated, err
}

// DeleteKonfigurasi menghapus satu baris konfigurasi.
func (s *Service) DeleteKonfigurasi(ctx context.Context, principal auth.Principal, id string) error {
	if !s.canManage(ctx, principal) {
		return domain.ErrForbidden
	}
	if strings.TrimSpace(id) == "" {
		return domain.ErrValidation
	}
	err := s.store.DeleteKonfigurasi(ctx, principal.UserID, id)
	if errors.Is(err, repository.ErrNotFoundKonfigurasi) {
		return domain.ErrConfigNotFound
	}
	return err
}

// ListKepesertaan membaca kepesertaan beserta nama pegawai; operator/atasan
// hanya melihat pegawai pada cakupan unitnya beserta seluruh turunan.
func (s *Service) ListKepesertaan(ctx context.Context, principal auth.Principal) ([]repository.KepesertaanRow, error) {
	scope, err := s.readAccess(ctx, principal)
	if err != nil {
		return nil, err
	}
	if scope != nil && len(scope) == 0 {
		return []repository.KepesertaanRow{}, nil
	}
	return s.store.ListKepesertaan(ctx, scope)
}

// UpsertKepesertaan menulis atau mengganti kepesertaan berdasarkan userId.
// Bila TanggalMulaiPensiun kosong, use case menghitung dari tanggal lahir
// pegawai + BUP pada konfigurasi aktif kategori status yang berlaku; bila
// data tidak lengkap, tanggal dibiarkan kosong (admin dapat mengisi manual).
func (s *Service) UpsertKepesertaan(ctx context.Context, principal auth.Principal, input domain.KepesertaanInput) (models.PensiunKepesertaan, error) {
	if !s.canManage(ctx, principal) {
		return models.PensiunKepesertaan{}, domain.ErrForbidden
	}
	if err := input.Validate(); err != nil {
		return models.PensiunKepesertaan{}, err
	}
	exists, err := s.store.ExistsUser(ctx, input.UserID)
	if err != nil {
		return models.PensiunKepesertaan{}, err
	}
	if !exists {
		return models.PensiunKepesertaan{}, domain.ErrUserNotFound
	}
	tanggal := input.TanggalMulaiPensiun
	if tanggal == nil {
		if computed := s.deriveTanggalPensiun(ctx, input.UserID, input.Mulai); computed != nil {
			tanggal = computed
		}
	}
	item := toModelKepesertaan(input, tanggal)
	return s.store.UpsertKepesertaan(ctx, principal.UserID, item)
}

// UpdateKepesertaan mengganti seluruh isian mutable kepesertaan.
func (s *Service) UpdateKepesertaan(ctx context.Context, principal auth.Principal, id string, input domain.KepesertaanInput) (models.PensiunKepesertaan, error) {
	if !s.canManage(ctx, principal) {
		return models.PensiunKepesertaan{}, domain.ErrForbidden
	}
	if strings.TrimSpace(id) == "" {
		return models.PensiunKepesertaan{}, domain.ErrValidation
	}
	if err := input.Validate(); err != nil {
		return models.PensiunKepesertaan{}, err
	}
	item := toModelKepesertaan(input, input.TanggalMulaiPensiun)
	updated, err := s.store.UpdateKepesertaan(ctx, principal.UserID, id, item)
	if errors.Is(err, repository.ErrNotFoundKepesertaan) {
		return models.PensiunKepesertaan{}, domain.ErrKepesertaanNotFound
	}
	return updated, err
}

// DeactivateKepesertaan menonaktifkan kepesertaan (riwayat tersimpan).
func (s *Service) DeactivateKepesertaan(ctx context.Context, principal auth.Principal, id string) error {
	if !s.canManage(ctx, principal) {
		return domain.ErrForbidden
	}
	if strings.TrimSpace(id) == "" {
		return domain.ErrValidation
	}
	err := s.store.DeactivateKepesertaan(ctx, principal.UserID, id)
	if errors.Is(err, repository.ErrNotFoundKepesertaan) {
		return domain.ErrKepesertaanNotFound
	}
	return err
}

// deriveTanggalPensiun menghitung tanggal pensiun = tanggal lahir + BUP dari
// konfigurasi aktif pada saat kepesertaan dimulai. Mengembalikan nil bila
// data tidak lengkap (tanggal lahir / kategori status / konfigurasi aktif).
func (s *Service) deriveTanggalPensiun(ctx context.Context, userID string, mulai time.Time) *time.Time {
	lahir, kategoriStatusID, err := s.store.UserBirthInfo(ctx, userID)
	if err != nil || lahir == nil || kategoriStatusID == nil || strings.TrimSpace(*kategoriStatusID) == "" {
		return nil
	}
	cfg, err := s.store.ActiveKonfigurasi(ctx, *kategoriStatusID, mulai)
	if err != nil {
		return nil
	}
	tanggal := domain.TanggalDariUsia(*lahir, cfg.BupUmur)
	return &tanggal
}

func toModelKonfigurasi(input domain.KonfigurasiInput) models.PensiunKonfigurasi {
	var penyedia *string
	if trimmed := strings.TrimSpace(input.PenyediaDplk); trimmed != "" {
		penyedia = &trimmed
	}
	return models.PensiunKonfigurasi{
		KategoriStatusId:     strings.TrimSpace(input.KategoriStatusID),
		BupUmur:              input.BupUmur,
		DplkWajib:            input.DplkWajib,
		IuranPegawaiPct:      input.IuranPegawaiPct,
		IuranPemberiKerjaPct: input.IuranPemberiKerjaPct,
		PenyediaDplk:         penyedia,
		BerlakuMulai:         input.BerlakuMulai,
		Aktif:                input.Aktif,
	}
}

func toModelKepesertaan(input domain.KepesertaanInput, tanggal *time.Time) models.PensiunKepesertaan {
	var nomor *string
	if trimmed := strings.TrimSpace(input.NomorKepesertaan); trimmed != "" {
		nomor = &trimmed
	}
	return models.PensiunKepesertaan{
		UserId:              strings.TrimSpace(input.UserID),
		PenyediaDplk:        strings.TrimSpace(input.PenyediaDplk),
		NomorKepesertaan:    nomor,
		TanggalMulaiPensiun: tanggal,
		Mulai:               input.Mulai,
		Aktif:               input.Aktif,
	}
}
