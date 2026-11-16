// Package app memuat logika aplikasi (use case) untuk area BPJS.
package app

import (
	"context"
	"strconv"
	"strings"
	"time"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/bpjs/domain"
	"simpeg/backend/internal/bpjs/repository"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
)

// ScopeRepository adalah port resolusi cakupan unit pada pohon organisasi.
type ScopeRepository interface {
	ListUnitScopeIDs(context.Context, string) ([]string, error)
}

// Service menjalankan use case BPJS Kesehatan/Ketenagakerjaan.
type Service struct {
	repo   repository.Repository
	policy auth.Policy
	scope  ScopeRepository
}

// NewService membuat service BPJS.
func NewService(repo repository.Repository, policy auth.Policy, scope ScopeRepository) *Service {
	return &Service{repo: repo, policy: policy, scope: scope}
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

func (s *Service) canReadReference(ctx context.Context, principal auth.Principal) bool {
	return allows(ctx, s.policy, principal, "payroll:read:any") ||
		allows(ctx, s.policy, principal, "payroll:read:homebase") ||
		allows(ctx, s.policy, principal, "payroll:manage")
}

func (s *Service) checkManage(ctx context.Context, principal auth.Principal) error {
	if !allows(ctx, s.policy, principal, "payroll:manage") {
		return domain.ErrForbidden
	}
	return nil
}

func (s *Service) scopeFor(ctx context.Context, principal auth.Principal) ([]string, error) {
	if allows(ctx, s.policy, principal, "payroll:read:any") || allows(ctx, s.policy, principal, "payroll:manage") {
		return nil, nil
	}
	if !allows(ctx, s.policy, principal, "payroll:read:homebase") {
		return nil, domain.ErrForbidden
	}
	if s.scope == nil {
		return []string{}, nil
	}
	return s.scope.ListUnitScopeIDs(ctx, principal.HomebaseID)
}

// ListKonfigurasi mengembalikan seluruh tarif iuran BPJS.
func (s *Service) ListKonfigurasi(ctx context.Context, principal auth.Principal) ([]models.BpjsKonfigurasi, error) {
	if !s.canReadReference(ctx, principal) {
		return nil, domain.ErrForbidden
	}
	return s.repo.ListKonfigurasi(ctx)
}

// CreateKonfigurasi membuat tarif iuran BPJS baru.
func (s *Service) CreateKonfigurasi(ctx context.Context, principal auth.Principal, input domain.CreateKonfigurasi) (models.BpjsKonfigurasi, error) {
	if err := s.checkManage(ctx, principal); err != nil {
		return models.BpjsKonfigurasi{}, err
	}
	if err := input.Validate(); err != nil {
		return models.BpjsKonfigurasi{}, err
	}
	return s.repo.CreateKonfigurasi(ctx, principal.UserID, models.BpjsKonfigurasi{
		Jenis:                models.JenisBPJS(input.Jenis),
		IuranPegawaiPct:      input.IuranPegawaiPct,
		IuranPemberiKerjaPct: input.IuranPemberiKerjaPct,
		Dasar:                input.Dasar,
		BatasAtas:            input.BatasAtas,
		BerlakuMulai:         input.BerlakuMulai,
		Aktif:                true,
	})
}

// UpdateKonfigurasi menyimpan perubahan tarif iuran BPJS.
func (s *Service) UpdateKonfigurasi(ctx context.Context, principal auth.Principal, id string, patch domain.KonfigurasiPatch) (models.BpjsKonfigurasi, error) {
	if err := s.checkManage(ctx, principal); err != nil {
		return models.BpjsKonfigurasi{}, err
	}
	if err := patch.Validate(); err != nil {
		return models.BpjsKonfigurasi{}, err
	}
	if !patch.HasChanges() {
		return models.BpjsKonfigurasi{}, domain.ErrValidation
	}
	updates := make(map[string]any)
	if patch.IuranPegawaiPct.Present {
		if v, err := strconv.ParseFloat(strings.TrimSpace(patch.IuranPegawaiPct.Value), 64); err == nil {
			updates["iuranPegawaiPct"] = v
		}
	}
	if patch.IuranPemberiKerjaPct.Present {
		if v, err := strconv.ParseFloat(strings.TrimSpace(patch.IuranPemberiKerjaPct.Value), 64); err == nil {
			updates["iuranPemberiKerjaPct"] = v
		}
	}
	if patch.Dasar.Present {
		updates["dasar"] = strings.TrimSpace(patch.Dasar.Value)
	}
	if patch.BatasAtas.Present {
		value := strings.TrimSpace(patch.BatasAtas.Value)
		if patch.BatasAtas.Null || value == "" {
			updates["batasAtas"] = nil
		} else if v, err := strconv.ParseFloat(value, 64); err == nil {
			updates["batasAtas"] = v
		}
	}
	if patch.BerlakuMulai.Present {
		if t, err := time.Parse("2006-01-02", strings.TrimSpace(patch.BerlakuMulai.Value)); err == nil {
			updates["berlakuMulai"] = t
		}
	}
	if patch.Aktif.Present {
		active, err := strconv.ParseBool(patch.Aktif.Value)
		if err != nil {
			active = !strings.EqualFold(patch.Aktif.Value, "false")
		}
		updates["aktif"] = active
	}
	return s.repo.UpdateKonfigurasi(ctx, principal.UserID, id, updates)
}

// ListPeserta mengembalikan baris kepesertaan BPJS sesuai cakupan principal.
func (s *Service) ListPeserta(ctx context.Context, principal auth.Principal, options listing.Options) (listing.Page[models.BpjsPeserta], error) {
	switch {
	case allows(ctx, s.policy, principal, "payroll:read:self") && !allows(ctx, s.policy, principal, "payroll:read:any"):
		options.UserID = principal.UserID
	case allows(ctx, s.policy, principal, "payroll:read:any") || allows(ctx, s.policy, principal, "payroll:manage"):
	case allows(ctx, s.policy, principal, "payroll:read:homebase"):
		ids, err := s.scopeFor(ctx, principal)
		if err != nil {
			return listing.Page[models.BpjsPeserta]{}, err
		}
		if len(ids) == 0 {
			return listing.EmptyPage[models.BpjsPeserta](options.Page, options.PageSize), nil
		}
		options.ScopeHomebaseIDs = ids
	default:
		return listing.Page[models.BpjsPeserta]{}, domain.ErrForbidden
	}
	rows, total, err := s.repo.ListPeserta(ctx, options)
	if err != nil {
		return listing.Page[models.BpjsPeserta]{}, err
	}
	return listing.Page[models.BpjsPeserta]{
		Items:      rows,
		Page:       options.Page,
		PageSize:   options.PageSize,
		TotalItems: total,
		TotalPages: listing.PageCount(total, options.PageSize),
	}, nil
}

// GetPeserta mengembalikan satu baris kepesertaan BPJS sesuai cakupan principal.
func (s *Service) GetPeserta(ctx context.Context, principal auth.Principal, id string) (models.BpjsPeserta, error) {
	row, err := s.repo.GetPeserta(ctx, id)
	if err != nil {
		return models.BpjsPeserta{}, err
	}
	if allows(ctx, s.policy, principal, "payroll:read:any") || allows(ctx, s.policy, principal, "payroll:manage") {
		return row, nil
	}
	if allows(ctx, s.policy, principal, "payroll:read:self") && row.UserId == principal.UserID {
		return row, nil
	}
	if allows(ctx, s.policy, principal, "payroll:read:homebase") {
		ids, err := s.scopeFor(ctx, principal)
		if err != nil {
			return models.BpjsPeserta{}, err
		}
		if len(ids) > 0 {
			rows, _, err := s.repo.ListPeserta(ctx, listing.Options{
				Page: 1, PageSize: 1, UserID: row.UserId, ScopeHomebaseIDs: ids,
			})
			if err == nil && len(rows) > 0 {
				return row, nil
			}
		}
	}
	return models.BpjsPeserta{}, domain.ErrForbidden
}

// CreatePeserta membuat baris kepesertaan BPJS baru.
func (s *Service) CreatePeserta(ctx context.Context, principal auth.Principal, input domain.CreatePeserta) (models.BpjsPeserta, error) {
	if err := s.checkManage(ctx, principal); err != nil {
		return models.BpjsPeserta{}, err
	}
	if err := input.Validate(); err != nil {
		return models.BpjsPeserta{}, err
	}
	return s.repo.CreatePeserta(ctx, principal.UserID, models.BpjsPeserta{
		UserId:           input.UserId,
		Jenis:            models.JenisBPJS(input.Jenis),
		NomorKepesertaan: input.NomorKepesertaan,
		Mulai:            input.Mulai,
		Selesai:          input.Selesai,
		Aktif:            true,
	})
}

// UpdatePeserta menyimpan perubahan kepesertaan BPJS.
func (s *Service) UpdatePeserta(ctx context.Context, principal auth.Principal, id string, patch domain.PesertaPatch) (models.BpjsPeserta, error) {
	if err := s.checkManage(ctx, principal); err != nil {
		return models.BpjsPeserta{}, err
	}
	if err := patch.Validate(); err != nil {
		return models.BpjsPeserta{}, err
	}
	if !patch.HasChanges() {
		return models.BpjsPeserta{}, domain.ErrValidation
	}
	updates := make(map[string]any)
	if patch.NomorKepesertaan.Present {
		if patch.NomorKepesertaan.Null || patch.NomorKepesertaan.Value == "" {
			updates["nomorKepesertaan"] = nil
		} else {
			updates["nomorKepesertaan"] = patch.NomorKepesertaan.Value
		}
	}
	if patch.Mulai.Present {
		if t, err := time.Parse("2006-01-02", strings.TrimSpace(patch.Mulai.Value)); err == nil {
			updates["mulai"] = t
		}
	}
	if patch.Selesai.Present {
		value := strings.TrimSpace(patch.Selesai.Value)
		if patch.Selesai.Null || value == "" {
			updates["selesai"] = nil
		} else if t, err := time.Parse("2006-01-02", value); err == nil {
			updates["selesai"] = t
		}
	}
	if patch.Aktif.Present {
		active, err := strconv.ParseBool(patch.Aktif.Value)
		if err != nil {
			active = !strings.EqualFold(patch.Aktif.Value, "false")
		}
		updates["aktif"] = active
	}
	return s.repo.UpdatePeserta(ctx, principal.UserID, id, updates)
}
