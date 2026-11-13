package app

import (
	"context"
	"strings"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/directory/domain"
	"simpeg/backend/internal/directory/repository"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
)

// JabatanService menjalankan use case master jabatan: daftar, baca, buat,
// ubah, dan arsip. Data referensi ini tidak bercakupan unit; akses baca
// diberikan ke seluruh role yang sudah terautentikasi, sedangkan pengelolaan
// dibatasi untuk admin.
type JabatanService struct {
	repo   repository.JabatanRepository
	policy auth.Policy
}

// NewJabatanService membuat service jabatan.
func NewJabatanService(repo repository.JabatanRepository, policy auth.Policy) *JabatanService {
	return &JabatanService{repo: repo, policy: policy}
}

func (s *JabatanService) canRead(principal auth.Principal) bool {
	for _, permission := range []auth.Permission{"jabatan:read", "jabatan:manage"} {
		if allows(s.policy, principal, permission) {
			return true
		}
	}
	return false
}

// List mengembalikan halaman master jabatan sesuai filter yang diminta.
func (s *JabatanService) List(ctx context.Context, principal auth.Principal, options listing.Options) (listing.Page[models.Jabatan], error) {
	if !s.canRead(principal) {
		return listing.Page[models.Jabatan]{}, domain.ErrForbidden
	}
	jabatans, total, err := s.repo.ListJabatans(ctx, options)
	if err != nil {
		return listing.Page[models.Jabatan]{}, err
	}
	return listing.Page[models.Jabatan]{
		Items:      jabatans,
		Page:       options.Page,
		PageSize:   options.PageSize,
		TotalItems: total,
		TotalPages: listing.PageCount(total, options.PageSize),
	}, nil
}

// Get mengembalikan satu master jabatan.
func (s *JabatanService) Get(ctx context.Context, principal auth.Principal, id string) (models.Jabatan, error) {
	if !s.canRead(principal) {
		return models.Jabatan{}, domain.ErrForbidden
	}
	return s.repo.GetJabatan(ctx, id)
}

// Create membuat master jabatan baru (khusus admin).
func (s *JabatanService) Create(ctx context.Context, principal auth.Principal, input domain.CreateJabatan) (models.Jabatan, error) {
	if !allows(s.policy, principal, "jabatan:manage") {
		return models.Jabatan{}, domain.ErrForbidden
	}
	input.Nama = strings.TrimSpace(input.Nama)
	input.Jenis = domain.NormalizeJabatanType(input.Jenis)
	if input.Jenis == "" {
		input.Jenis = domain.DefaultJabatanType
	}
	if err := input.Validate(); err != nil {
		return models.Jabatan{}, err
	}
	return s.repo.CreateJabatan(ctx, principal.UserID, models.Jabatan{
		Nama:  input.Nama,
		Jenis: models.JenisJabatan(input.Jenis),
		Aktif: true,
	})
}

// Update memperbarui master jabatan (khusus admin).
func (s *JabatanService) Update(ctx context.Context, principal auth.Principal, id string, patch domain.JabatanPatch) (models.Jabatan, error) {
	if !allows(s.policy, principal, "jabatan:manage") {
		return models.Jabatan{}, domain.ErrForbidden
	}
	if err := patch.Validate(); err != nil {
		return models.Jabatan{}, err
	}
	return s.repo.UpdateJabatan(ctx, principal.UserID, id, patch)
}

// Archive menonaktifkan master jabatan (khusus admin).
func (s *JabatanService) Archive(ctx context.Context, principal auth.Principal, id string) error {
	if !allows(s.policy, principal, "jabatan:manage") {
		return domain.ErrForbidden
	}
	return s.repo.ArchiveJabatan(ctx, principal.UserID, id)
}
