package app

import (
	"context"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
	"simpeg/backend/internal/tridharma/domain"
)

// ListTridharmaPendidikan mengembalikan daftar tridharma pendidikan sesuai cakupan.
func (s *TridharmaService) ListTridharmaPendidikan(ctx context.Context, principal auth.Principal, options listing.Options) (listing.Page[models.TridharmaPendidikan], error) {
	switch {
	case allows(s.policy, principal, "tridharma:pendidikan:read:any"):
	case allows(s.policy, principal, "tridharma:pendidikan:read:homebase"):
		scope, err := s.scope.resolve(ctx, principal)
		if err != nil {
			return listing.Page[models.TridharmaPendidikan]{}, err
		}
		if scope.Empty() {
			return listing.EmptyPage[models.TridharmaPendidikan](options.Page, options.PageSize), nil
		}
	case allows(s.policy, principal, "tridharma:pendidikan:read:self"):
		options.UserID = principal.UserID
	default:
		return listing.Page[models.TridharmaPendidikan]{}, domain.ErrForbidden
	}

	items, total, err := s.repo.ListTridharmaPendidikan(ctx, options)
	if err != nil {
		return listing.Page[models.TridharmaPendidikan]{}, err
	}
	return listing.Page[models.TridharmaPendidikan]{
		Items:      items,
		Page:       options.Page,
		PageSize:   options.PageSize,
		TotalItems: total,
		TotalPages: listing.PageCount(total, options.PageSize),
	}, nil
}

// GetTridharmaPendidikan mengembalikan satu tridharma pendidikan bila principal berhak.
func (s *TridharmaService) GetTridharmaPendidikan(ctx context.Context, principal auth.Principal, id string) (models.TridharmaPendidikan, error) {
	needsScope := allows(s.policy, principal, "tridharma:pendidikan:read:homebase") && !allows(s.policy, principal, "tridharma:pendidikan:read:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.TridharmaPendidikan{}, err
	}

	item, err := s.repo.GetTridharmaPendidikan(ctx, id)
	if err != nil {
		return models.TridharmaPendidikan{}, err
	}
	if !s.canAccessPendidikan(scoped, principal, item) {
		return models.TridharmaPendidikan{}, domain.ErrForbidden
	}
	return item, nil
}

// CreateTridharmaPendidikan membuat tridharma pendidikan baru.
func (s *TridharmaService) CreateTridharmaPendidikan(ctx context.Context, principal auth.Principal, item models.TridharmaPendidikan) (models.TridharmaPendidikan, error) {
	if !allows(s.policy, principal, "tridharma:pendidikan:manage:any") && !allows(s.policy, principal, "tridharma:pendidikan:manage:homebase") {
		return models.TridharmaPendidikan{}, domain.ErrForbidden
	}
	return s.repo.CreateTridharmaPendidikan(ctx, principal.UserID, item)
}

// UpdateTridharmaPendidikan mengubah field tridharma pendidikan.
func (s *TridharmaService) UpdateTridharmaPendidikan(ctx context.Context, principal auth.Principal, id string, updates map[string]any) (models.TridharmaPendidikan, error) {
	needsScope := allows(s.policy, principal, "tridharma:pendidikan:manage:homebase") && !allows(s.policy, principal, "tridharma:pendidikan:manage:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.TridharmaPendidikan{}, err
	}

	item, err := s.repo.GetTridharmaPendidikan(ctx, id)
	if err != nil {
		return models.TridharmaPendidikan{}, err
	}
	if !s.canAccessPendidikan(scoped, principal, item) {
		return models.TridharmaPendidikan{}, domain.ErrForbidden
	}
	return s.repo.UpdateTridharmaPendidikan(ctx, principal.UserID, id, updates)
}

// DeleteTridharmaPendidikan menghapus tridharma pendidikan.
func (s *TridharmaService) DeleteTridharmaPendidikan(ctx context.Context, principal auth.Principal, id string) error {
	needsScope := allows(s.policy, principal, "tridharma:pendidikan:manage:homebase") && !allows(s.policy, principal, "tridharma:pendidikan:manage:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return err
	}

	item, err := s.repo.GetTridharmaPendidikan(ctx, id)
	if err != nil {
		return err
	}
	if !s.canAccessPendidikan(scoped, principal, item) {
		return domain.ErrForbidden
	}
	return s.repo.DeleteTridharmaPendidikan(ctx, principal.UserID, id)
}

// ListTridharmaPenelitian mengembalikan daftar tridharma penelitian sesuai cakupan.
func (s *TridharmaService) ListTridharmaPenelitian(ctx context.Context, principal auth.Principal, options listing.Options) (listing.Page[models.TridharmaPenelitian], error) {
	switch {
	case allows(s.policy, principal, "tridharma:penelitian:read:any"):
	case allows(s.policy, principal, "tridharma:penelitian:read:homebase"):
		scope, err := s.scope.resolve(ctx, principal)
		if err != nil {
			return listing.Page[models.TridharmaPenelitian]{}, err
		}
		if scope.Empty() {
			return listing.EmptyPage[models.TridharmaPenelitian](options.Page, options.PageSize), nil
		}
	case allows(s.policy, principal, "tridharma:penelitian:read:self"):
		options.UserID = principal.UserID
	default:
		return listing.Page[models.TridharmaPenelitian]{}, domain.ErrForbidden
	}

	items, total, err := s.repo.ListTridharmaPenelitian(ctx, options)
	if err != nil {
		return listing.Page[models.TridharmaPenelitian]{}, err
	}
	return listing.Page[models.TridharmaPenelitian]{
		Items:      items,
		Page:       options.Page,
		PageSize:   options.PageSize,
		TotalItems: total,
		TotalPages: listing.PageCount(total, options.PageSize),
	}, nil
}

// GetTridharmaPenelitian mengembalikan satu tridharma penelitian bila principal berhak.
func (s *TridharmaService) GetTridharmaPenelitian(ctx context.Context, principal auth.Principal, id string) (models.TridharmaPenelitian, error) {
	needsScope := allows(s.policy, principal, "tridharma:penelitian:read:homebase") && !allows(s.policy, principal, "tridharma:penelitian:read:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.TridharmaPenelitian{}, err
	}

	item, err := s.repo.GetTridharmaPenelitian(ctx, id)
	if err != nil {
		return models.TridharmaPenelitian{}, err
	}
	if !s.canAccessPenelitian(scoped, principal, item) {
		return models.TridharmaPenelitian{}, domain.ErrForbidden
	}
	return item, nil
}

// CreateTridharmaPenelitian membuat tridharma penelitian baru.
func (s *TridharmaService) CreateTridharmaPenelitian(ctx context.Context, principal auth.Principal, item models.TridharmaPenelitian) (models.TridharmaPenelitian, error) {
	if !allows(s.policy, principal, "tridharma:penelitian:manage:any") && !allows(s.policy, principal, "tridharma:penelitian:manage:homebase") {
		return models.TridharmaPenelitian{}, domain.ErrForbidden
	}
	return s.repo.CreateTridharmaPenelitian(ctx, principal.UserID, item)
}

// UpdateTridharmaPenelitian mengubah field tridharma penelitian.
func (s *TridharmaService) UpdateTridharmaPenelitian(ctx context.Context, principal auth.Principal, id string, updates map[string]any) (models.TridharmaPenelitian, error) {
	needsScope := allows(s.policy, principal, "tridharma:penelitian:manage:homebase") && !allows(s.policy, principal, "tridharma:penelitian:manage:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.TridharmaPenelitian{}, err
	}

	item, err := s.repo.GetTridharmaPenelitian(ctx, id)
	if err != nil {
		return models.TridharmaPenelitian{}, err
	}
	if !s.canAccessPenelitian(scoped, principal, item) {
		return models.TridharmaPenelitian{}, domain.ErrForbidden
	}
	return s.repo.UpdateTridharmaPenelitian(ctx, principal.UserID, id, updates)
}

// DeleteTridharmaPenelitian menghapus tridharma penelitian.
func (s *TridharmaService) DeleteTridharmaPenelitian(ctx context.Context, principal auth.Principal, id string) error {
	needsScope := allows(s.policy, principal, "tridharma:penelitian:manage:homebase") && !allows(s.policy, principal, "tridharma:penelitian:manage:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return err
	}

	item, err := s.repo.GetTridharmaPenelitian(ctx, id)
	if err != nil {
		return err
	}
	if !s.canAccessPenelitian(scoped, principal, item) {
		return domain.ErrForbidden
	}
	return s.repo.DeleteTridharmaPenelitian(ctx, principal.UserID, id)
}

// ListTridharmaPengabdian mengembalikan daftar tridharma pengabdian sesuai cakupan.
func (s *TridharmaService) ListTridharmaPengabdian(ctx context.Context, principal auth.Principal, options listing.Options) (listing.Page[models.TridharmaPengabdian], error) {
	switch {
	case allows(s.policy, principal, "tridharma:pengabdian:read:any"):
	case allows(s.policy, principal, "tridharma:pengabdian:read:homebase"):
		scope, err := s.scope.resolve(ctx, principal)
		if err != nil {
			return listing.Page[models.TridharmaPengabdian]{}, err
		}
		if scope.Empty() {
			return listing.EmptyPage[models.TridharmaPengabdian](options.Page, options.PageSize), nil
		}
	case allows(s.policy, principal, "tridharma:pengabdian:read:self"):
		options.UserID = principal.UserID
	default:
		return listing.Page[models.TridharmaPengabdian]{}, domain.ErrForbidden
	}

	items, total, err := s.repo.ListTridharmaPengabdian(ctx, options)
	if err != nil {
		return listing.Page[models.TridharmaPengabdian]{}, err
	}
	return listing.Page[models.TridharmaPengabdian]{
		Items:      items,
		Page:       options.Page,
		PageSize:   options.PageSize,
		TotalItems: total,
		TotalPages: listing.PageCount(total, options.PageSize),
	}, nil
}

// GetTridharmaPengabdian mengembalikan satu tridharma pengabdian bila principal berhak.
func (s *TridharmaService) GetTridharmaPengabdian(ctx context.Context, principal auth.Principal, id string) (models.TridharmaPengabdian, error) {
	needsScope := allows(s.policy, principal, "tridharma:pengabdian:read:homebase") && !allows(s.policy, principal, "tridharma:pengabdian:read:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.TridharmaPengabdian{}, err
	}

	item, err := s.repo.GetTridharmaPengabdian(ctx, id)
	if err != nil {
		return models.TridharmaPengabdian{}, err
	}
	if !s.canAccessPengabdian(scoped, principal, item) {
		return models.TridharmaPengabdian{}, domain.ErrForbidden
	}
	return item, nil
}

// CreateTridharmaPengabdian membuat tridharma pengabdian baru.
func (s *TridharmaService) CreateTridharmaPengabdian(ctx context.Context, principal auth.Principal, item models.TridharmaPengabdian) (models.TridharmaPengabdian, error) {
	if !allows(s.policy, principal, "tridharma:pengabdian:manage:any") && !allows(s.policy, principal, "tridharma:pengabdian:manage:homebase") {
		return models.TridharmaPengabdian{}, domain.ErrForbidden
	}
	return s.repo.CreateTridharmaPengabdian(ctx, principal.UserID, item)
}

// UpdateTridharmaPengabdian mengubah field tridharma pengabdian.
func (s *TridharmaService) UpdateTridharmaPengabdian(ctx context.Context, principal auth.Principal, id string, updates map[string]any) (models.TridharmaPengabdian, error) {
	needsScope := allows(s.policy, principal, "tridharma:pengabdian:manage:homebase") && !allows(s.policy, principal, "tridharma:pengabdian:manage:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.TridharmaPengabdian{}, err
	}

	item, err := s.repo.GetTridharmaPengabdian(ctx, id)
	if err != nil {
		return models.TridharmaPengabdian{}, err
	}
	if !s.canAccessPengabdian(scoped, principal, item) {
		return models.TridharmaPengabdian{}, domain.ErrForbidden
	}
	return s.repo.UpdateTridharmaPengabdian(ctx, principal.UserID, id, updates)
}

// DeleteTridharmaPengabdian menghapus tridharma pengabdian.
func (s *TridharmaService) DeleteTridharmaPengabdian(ctx context.Context, principal auth.Principal, id string) error {
	needsScope := allows(s.policy, principal, "tridharma:pengabdian:manage:homebase") && !allows(s.policy, principal, "tridharma:pengabdian:manage:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return err
	}

	item, err := s.repo.GetTridharmaPengabdian(ctx, id)
	if err != nil {
		return err
	}
	if !s.canAccessPengabdian(scoped, principal, item) {
		return domain.ErrForbidden
	}
	return s.repo.DeleteTridharmaPengabdian(ctx, principal.UserID, id)
}
