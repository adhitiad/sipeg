package app

import (
	"context"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
	"simpeg/backend/internal/tridharma/domain"
)

// ListPeriodeAkademik mengembalikan daftar periode akademik sesuai cakupan principal.
func (s *TridharmaService) ListPeriodeAkademik(ctx context.Context, principal auth.Principal, options listing.Options) (listing.Page[models.PeriodeAkademik], error) {
	switch {
	case allows(s.policy, principal, "tridharma:periode:read:any"):
	case allows(s.policy, principal, "tridharma:periode:read:homebase"):
		scope, err := s.scope.resolve(ctx, principal)
		if err != nil {
			return listing.Page[models.PeriodeAkademik]{}, err
		}
		if scope.Empty() {
			return listing.EmptyPage[models.PeriodeAkademik](options.Page, options.PageSize), nil
		}
	case allows(s.policy, principal, "tridharma:periode:read:self"):
	default:
		return listing.Page[models.PeriodeAkademik]{}, domain.ErrForbidden
	}

	items, total, err := s.repo.ListPeriodeAkademik(ctx, options)
	if err != nil {
		return listing.Page[models.PeriodeAkademik]{}, err
	}
	return listing.Page[models.PeriodeAkademik]{
		Items:      items,
		Page:       options.Page,
		PageSize:   options.PageSize,
		TotalItems: total,
		TotalPages: listing.PageCount(total, options.PageSize),
	}, nil
}

// GetPeriodeAkademik mengembalikan satu periode akademik bila principal berhak.
func (s *TridharmaService) GetPeriodeAkademik(ctx context.Context, principal auth.Principal, id string) (models.PeriodeAkademik, error) {
	needsScope := allows(s.policy, principal, "tridharma:periode:read:homebase") && !allows(s.policy, principal, "tridharma:periode:read:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.PeriodeAkademik{}, err
	}

	periode, err := s.repo.GetPeriodeAkademik(ctx, id)
	if err != nil {
		return models.PeriodeAkademik{}, err
	}
	if !s.canAccessPeriode(scoped, principal, periode) {
		return models.PeriodeAkademik{}, domain.ErrForbidden
	}
	return periode, nil
}

// CreatePeriodeAkademik membuat periode akademik baru.
func (s *TridharmaService) CreatePeriodeAkademik(ctx context.Context, principal auth.Principal, periode models.PeriodeAkademik) (models.PeriodeAkademik, error) {
	if !allows(s.policy, principal, "tridharma:periode:manage:any") && !allows(s.policy, principal, "tridharma:periode:manage:homebase") {
		return models.PeriodeAkademik{}, domain.ErrForbidden
	}
	return s.repo.CreatePeriodeAkademik(ctx, principal.UserID, periode)
}

// UpdatePeriodeAkademik mengubah field periode akademik.
func (s *TridharmaService) UpdatePeriodeAkademik(ctx context.Context, principal auth.Principal, id string, updates map[string]any) (models.PeriodeAkademik, error) {
	needsScope := allows(s.policy, principal, "tridharma:periode:manage:homebase") && !allows(s.policy, principal, "tridharma:periode:manage:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.PeriodeAkademik{}, err
	}

	periode, err := s.repo.GetPeriodeAkademik(ctx, id)
	if err != nil {
		return models.PeriodeAkademik{}, err
	}
	if !s.canAccessPeriode(scoped, principal, periode) {
		return models.PeriodeAkademik{}, domain.ErrForbidden
	}
	return s.repo.UpdatePeriodeAkademik(ctx, principal.UserID, id, updates)
}

// DeletePeriodeAkademik menghapus periode akademik.
func (s *TridharmaService) DeletePeriodeAkademik(ctx context.Context, principal auth.Principal, id string) error {
	needsScope := allows(s.policy, principal, "tridharma:periode:manage:homebase") && !allows(s.policy, principal, "tridharma:periode:manage:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return err
	}

	periode, err := s.repo.GetPeriodeAkademik(ctx, id)
	if err != nil {
		return err
	}
	if !s.canAccessPeriode(scoped, principal, periode) {
		return domain.ErrForbidden
	}
	return s.repo.DeletePeriodeAkademik(ctx, principal.UserID, id)
}

// ListEvaluasiBKD mengembalikan daftar evaluasi BKD sesuai cakupan.
func (s *TridharmaService) ListEvaluasiBKD(ctx context.Context, principal auth.Principal, options listing.Options) (listing.Page[models.EvaluasiBKD], error) {
	switch {
	case allows(s.policy, principal, "tridharma:evaluasi:read:any"):
	case allows(s.policy, principal, "tridharma:evaluasi:read:homebase"):
		scope, err := s.scope.resolve(ctx, principal)
		if err != nil {
			return listing.Page[models.EvaluasiBKD]{}, err
		}
		if scope.Empty() {
			return listing.EmptyPage[models.EvaluasiBKD](options.Page, options.PageSize), nil
		}
	case allows(s.policy, principal, "tridharma:evaluasi:read:self"):
		options.UserID = principal.UserID
	default:
		return listing.Page[models.EvaluasiBKD]{}, domain.ErrForbidden
	}

	items, total, err := s.repo.ListEvaluasiBKD(ctx, options)
	if err != nil {
		return listing.Page[models.EvaluasiBKD]{}, err
	}
	return listing.Page[models.EvaluasiBKD]{
		Items:      items,
		Page:       options.Page,
		PageSize:   options.PageSize,
		TotalItems: total,
		TotalPages: listing.PageCount(total, options.PageSize),
	}, nil
}

// GetEvaluasiBKD mengembalikan satu evaluasi BKD bila principal berhak.
func (s *TridharmaService) GetEvaluasiBKD(ctx context.Context, principal auth.Principal, id string) (models.EvaluasiBKD, error) {
	needsScope := allows(s.policy, principal, "tridharma:evaluasi:read:homebase") && !allows(s.policy, principal, "tridharma:evaluasi:read:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.EvaluasiBKD{}, err
	}

	evaluasi, err := s.repo.GetEvaluasiBKD(ctx, id)
	if err != nil {
		return models.EvaluasiBKD{}, err
	}
	if !s.canAccessEvaluasi(scoped, principal, evaluasi) {
		return models.EvaluasiBKD{}, domain.ErrForbidden
	}
	return evaluasi, nil
}

// CreateEvaluasiBKD membuat evaluasi BKD baru.
func (s *TridharmaService) CreateEvaluasiBKD(ctx context.Context, principal auth.Principal, evaluasi models.EvaluasiBKD) (models.EvaluasiBKD, error) {
	if !allows(s.policy, principal, "tridharma:evaluasi:manage:any") && !allows(s.policy, principal, "tridharma:evaluasi:manage:homebase") {
		return models.EvaluasiBKD{}, domain.ErrForbidden
	}
	return s.repo.CreateEvaluasiBKD(ctx, principal.UserID, evaluasi)
}

// UpdateEvaluasiBKD mengubah field evaluasi BKD.
func (s *TridharmaService) UpdateEvaluasiBKD(ctx context.Context, principal auth.Principal, id string, updates map[string]any) (models.EvaluasiBKD, error) {
	needsScope := allows(s.policy, principal, "tridharma:evaluasi:manage:homebase") && !allows(s.policy, principal, "tridharma:evaluasi:manage:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.EvaluasiBKD{}, err
	}

	evaluasi, err := s.repo.GetEvaluasiBKD(ctx, id)
	if err != nil {
		return models.EvaluasiBKD{}, err
	}
	if !s.canAccessEvaluasi(scoped, principal, evaluasi) {
		return models.EvaluasiBKD{}, domain.ErrForbidden
	}
	return s.repo.UpdateEvaluasiBKD(ctx, principal.UserID, id, updates)
}

// DeleteEvaluasiBKD menghapus evaluasi BKD.
func (s *TridharmaService) DeleteEvaluasiBKD(ctx context.Context, principal auth.Principal, id string) error {
	needsScope := allows(s.policy, principal, "tridharma:evaluasi:manage:homebase") && !allows(s.policy, principal, "tridharma:evaluasi:manage:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return err
	}

	evaluasi, err := s.repo.GetEvaluasiBKD(ctx, id)
	if err != nil {
		return err
	}
	if !s.canAccessEvaluasi(scoped, principal, evaluasi) {
		return domain.ErrForbidden
	}
	return s.repo.DeleteEvaluasiBKD(ctx, principal.UserID, id)
}
