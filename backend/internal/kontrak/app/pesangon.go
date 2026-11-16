// Service master pesangonAturan: baca, buat/ganti, ubah, hapus (admin).
package app

import (
	"context"
	"strings"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/models"

	"simpeg/backend/internal/kontrak/domain"
)

// PesangonStore adalah port yang dibutuhkan service master pesangonAturan.
type PesangonStore interface {
	ListRules(ctx context.Context, kategoriStatusID string) ([]models.PesangonAturan, error)
	CreateRule(ctx context.Context, rule models.PesangonAturan) (models.PesangonAturan, error)
	UpdateRule(ctx context.Context, id string, rule models.PesangonAturan) (models.PesangonAturan, error)
	DeleteRule(ctx context.Context, id string) error
}

// PesangonService menjalankan use case master pesangonAturan.
type PesangonService struct {
	repo   PesangonStore
	policy auth.Policy
}

// NewPesangonService membuat PesangonService.
func NewPesangonService(repo PesangonStore, policy auth.Policy) *PesangonService {
	return &PesangonService{repo: repo, policy: policy}
}

// canManage abreviates a local policy check.
func (s *PesangonService) canManage(ctx context.Context, principal auth.Principal) bool {
	return allows(ctx, s.policy, principal, "pesangon:manage")
}

// List membaca aturan per kategori status.
func (s *PesangonService) List(ctx context.Context, principal auth.Principal, kategoriStatusID string) ([]models.PesangonAturan, error) {
	if !s.canManage(ctx, principal) {
		return nil, domain.ErrForbidden
	}
	return s.repo.ListRules(ctx, strings.TrimSpace(kategoriStatusID))
}

// Upsert menulis atau mengganti satu baris aturan.
func (s *PesangonService) Upsert(ctx context.Context, principal auth.Principal, input domain.PesangonAturanInput) (models.PesangonAturan, error) {
	if !s.canManage(ctx, principal) {
		return models.PesangonAturan{}, domain.ErrForbidden
	}
	if err := input.Validate(); err != nil {
		return models.PesangonAturan{}, err
	}
	sum := strings.TrimSpace(input.SumberRegulasi)
	if sum == "" {
		sum = "PP 35/2021"
	}
	return s.repo.CreateRule(ctx, models.PesangonAturan{
		KategoriStatusId:   input.KategoriStatusID,
		MasaKerjaTahun:     input.MasaKerjaTahun,
		PesangonBulan:      input.PesangonBulan,
		PenghargaanBulan:   input.PenghargaanBulan,
		GantiKerugianBulan: input.GantiKerugianBulan,
		SumberRegulasi:     sum,
	})
}

// Update mengubah satu baris aturan.
func (s *PesangonService) Update(ctx context.Context, principal auth.Principal, id string, input domain.PesangonAturanInput) (models.PesangonAturan, error) {
	if !s.canManage(ctx, principal) {
		return models.PesangonAturan{}, domain.ErrForbidden
	}
	if id == "" {
		return models.PesangonAturan{}, domain.ErrValidation
	}
	if err := input.Validate(); err != nil {
		return models.PesangonAturan{}, err
	}
	return s.repo.UpdateRule(ctx, id, models.PesangonAturan{
		PesangonBulan:      input.PesangonBulan,
		PenghargaanBulan:   input.PenghargaanBulan,
		GantiKerugianBulan: input.GantiKerugianBulan,
		SumberRegulasi:     strings.TrimSpace(input.SumberRegulasi),
	})
}

// Delete menghapus satu baris aturan.
func (s *PesangonService) Delete(ctx context.Context, principal auth.Principal, id string) error {
	if !s.canManage(ctx, principal) {
		return domain.ErrForbidden
	}
	return s.repo.DeleteRule(ctx, id)
}
