// Package app memuat logika aplikasi (use case) untuk area skala gaji
// pokok yayasan.
package app

import (
	"context"
	"strconv"
	"strings"
	"time"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
	"simpeg/backend/internal/payscale/domain"
	"simpeg/backend/internal/payscale/repository"
)

// Service menjalankan use case master skala gaji pokok yayasan: daftar, baca,
// buat, ubah, dan kelola baris nominal per tingkat dan kategori status.
type Service struct {
	repo   repository.Repository
	policy auth.Policy
}

// NewService membuat service skala gaji pokok yayasan.
func NewService(repo repository.Repository, policy auth.Policy) *Service {
	return &Service{repo: repo, policy: policy}
}

func allows(ctx context.Context, policy auth.Policy, principal auth.Principal, permission auth.Permission) bool {
	if policy == nil {
		return false
	}
	if sp, ok := policy.(auth.ScopedPolicy); ok {
		return sp.AllowsScoped(ctx, principal, permission)
	}
	return policy.Allows(principal.RoleName, permission)
}

func (s *Service) canRead(ctx context.Context, principal auth.Principal) bool {
	return allows(ctx, s.policy, principal, "payroll:read:any") || allows(ctx, s.policy, principal, "payroll:read:homebase") || allows(ctx, s.policy, principal, "payroll:manage")
}

func (s *Service) checkManage(ctx context.Context, principal auth.Principal) error {
	if !allows(ctx, s.policy, principal, "payroll:manage") {
		return domain.ErrForbidden
	}
	return nil
}

// List mengembalikan halaman skala gaji sesuai filter yang diminta.
func (s *Service) List(ctx context.Context, principal auth.Principal, options listing.Options) (listing.Page[models.SkalaGaji], error) {
	if !s.canRead(ctx, principal) {
		return listing.Page[models.SkalaGaji]{}, domain.ErrForbidden
	}
	rows, total, err := s.repo.ListSkala(ctx, options)
	if err != nil {
		return listing.Page[models.SkalaGaji]{}, err
	}
	return listing.Page[models.SkalaGaji]{
		Items:      rows,
		Page:       options.Page,
		PageSize:   options.PageSize,
		TotalItems: total,
		TotalPages: listing.PageCount(total, options.PageSize),
	}, nil
}

// Get mengembalikan satu skala gaji.
func (s *Service) Get(ctx context.Context, principal auth.Principal, id string) (models.SkalaGaji, error) {
	if !s.canRead(ctx, principal) {
		return models.SkalaGaji{}, domain.ErrForbidden
	}
	return s.repo.GetSkala(ctx, id)
}

// GetWithDetails mengembalikan satu skala gaji beserta seluruh baris nominalnya.
func (s *Service) GetWithDetails(ctx context.Context, principal auth.Principal, id string) (models.SkalaGaji, []models.SkalaGajiDetail, error) {
	if !s.canRead(ctx, principal) {
		return models.SkalaGaji{}, nil, domain.ErrForbidden
	}
	return s.repo.GetSkalaWithDetails(ctx, id)
}

// ListKategoriStatus membaca taxonomy kategori status pegawai.
func (s *Service) ListKategoriStatus(ctx context.Context) ([]models.KategoriStatusPegawai, error) {
	return s.repo.ListKategoriStatus(ctx)
}

// Create membuat skala gaji baru (khusus admin).
func (s *Service) Create(ctx context.Context, principal auth.Principal, input domain.CreateSkalaGaji) (models.SkalaGaji, error) {
	if err := s.checkManage(ctx, principal); err != nil {
		return models.SkalaGaji{}, err
	}
	input.Kode = strings.TrimSpace(input.Kode)
	input.Nama = strings.TrimSpace(input.Nama)
	if err := input.Validate(); err != nil {
		return models.SkalaGaji{}, err
	}
	return s.repo.CreateSkala(ctx, principal.UserID, models.SkalaGaji{
		Kode:           input.Kode,
		Nama:           input.Nama,
		BerlakuMulai:   input.BerlakuMulai,
		BerlakuSelesai: input.BerlakuSelesai,
		Aktif:          true,
	})
}

// Update menyimpan perubahan skala gaji (khusus admin).
//
// Nilai tanggal (berlakuMulai/berlakuSelesai) diterima sebagai string
// `YYYY-MM-DD` atau dikosongkan (`null`) untuk berlakuSelesai. Nilai
// kosong untuk field wajib lain ditolak validasi.
func (s *Service) Update(ctx context.Context, principal auth.Principal, id string, patch domain.SkalaGajiPatch) (models.SkalaGaji, error) {
	if err := s.checkManage(ctx, principal); err != nil {
		return models.SkalaGaji{}, err
	}
	if err := patch.Validate(); err != nil {
		return models.SkalaGaji{}, err
	}
	if !patch.HasChanges() {
		return models.SkalaGaji{}, domain.ErrValidation
	}
	updates := skalaColumns(patch)
	return s.repo.UpdateSkala(ctx, principal.UserID, id, updates)
}

// AddDetail menambahkan baris nominal pada sebuah skala gaji (khusus admin).
func (s *Service) AddDetail(ctx context.Context, principal auth.Principal, input domain.CreateSkalaGajiDetail) (models.SkalaGajiDetail, error) {
	if err := s.checkManage(ctx, principal); err != nil {
		return models.SkalaGajiDetail{}, err
	}
	if err := input.Validate(); err != nil {
		return models.SkalaGajiDetail{}, err
	}
	return s.repo.CreateDetail(ctx, principal.UserID, models.SkalaGajiDetail{
		SkalaGajiId:      input.SkalaGajiId,
		KategoriStatusId: input.KategoriStatusId,
		Tingkat:          input.Tingkat,
		Nominal:          input.Nominal,
	})
}

// UpdateDetail menyimpan perubahan baris nominal (khusus admin).
func (s *Service) UpdateDetail(ctx context.Context, principal auth.Principal, id string, patch domain.SkalaGajiDetailPatch) (models.SkalaGajiDetail, error) {
	if err := s.checkManage(ctx, principal); err != nil {
		return models.SkalaGajiDetail{}, err
	}
	if err := patch.Validate(); err != nil {
		return models.SkalaGajiDetail{}, err
	}
	if !patch.HasChanges() {
		return models.SkalaGajiDetail{}, domain.ErrValidation
	}
	updates := detailColumns(patch)
	return s.repo.UpdateDetail(ctx, principal.UserID, id, updates)
}

// DeleteDetail menghapus baris nominal (khusus admin).
func (s *Service) DeleteDetail(ctx context.Context, principal auth.Principal, id string) error {
	if err := s.checkManage(ctx, principal); err != nil {
		return err
	}
	return s.repo.DeleteDetail(ctx, principal.UserID, id)
}

func skalaColumns(patch domain.SkalaGajiPatch) map[string]any {
	updates := make(map[string]any)
	if patch.Kode.Present {
		updates["kode"] = patch.Kode.Value
	}
	if patch.Nama.Present {
		updates["nama"] = patch.Nama.Value
	}
	if patch.BerlakuMulai.Present {
		if t, err := time.Parse("2006-01-02", patch.BerlakuMulai.Value); err == nil {
			updates["berlakuMulai"] = t
		}
	}
	if patch.BerlakuSelesai.Present {
		if patch.BerlakuSelesai.Null || patch.BerlakuSelesai.Value == "" {
			updates["berlakuSelesai"] = nil
		} else if t, err := time.Parse("2006-01-02", patch.BerlakuSelesai.Value); err == nil {
			updates["berlakuSelesai"] = t
		}
	}
	if patch.Aktif.Present {
		active, err := strconv.ParseBool(patch.Aktif.Value)
		if err != nil {
			active = !strings.EqualFold(patch.Aktif.Value, "false")
		}
		updates["aktif"] = active
	}
	return updates
}

func detailColumns(patch domain.SkalaGajiDetailPatch) map[string]any {
	updates := make(map[string]any)
	if patch.KategoriStatusId.Present {
		updates["kategoriStatusId"] = patch.KategoriStatusId.Value
	}
	if patch.Tingkat.Present {
		if level, err := strconv.Atoi(strings.TrimSpace(patch.Tingkat.Value)); err == nil {
			updates["tingkat"] = level
		}
	}
	if patch.Nominal.Present {
		if amount, err := strconv.ParseFloat(strings.TrimSpace(patch.Nominal.Value), 64); err == nil {
			updates["nominal"] = amount
		}
	}
	return updates
}
