package app

import (
	"context"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
	"simpeg/backend/internal/tridharma/domain"
)

// ListTridharmaPenunjang mengembalikan daftar tridharma penunjang sesuai cakupan.
func (s *TridharmaService) ListTridharmaPenunjang(ctx context.Context, principal auth.Principal, options listing.Options) (listing.Page[models.TridharmaPenunjang], error) {
	switch {
	case allows(s.policy, principal, "tridharma:penunjang:read:any"):
	case allows(s.policy, principal, "tridharma:penunjang:read:homebase"):
		scope, err := s.scope.resolve(ctx, principal)
		if err != nil {
			return listing.Page[models.TridharmaPenunjang]{}, err
		}
		if scope.Empty() {
			return listing.EmptyPage[models.TridharmaPenunjang](options.Page, options.PageSize), nil
		}
	case allows(s.policy, principal, "tridharma:penunjang:read:self"):
		options.UserID = principal.UserID
	default:
		return listing.Page[models.TridharmaPenunjang]{}, domain.ErrForbidden
	}

	items, total, err := s.repo.ListTridharmaPenunjang(ctx, options)
	if err != nil {
		return listing.Page[models.TridharmaPenunjang]{}, err
	}
	return listing.Page[models.TridharmaPenunjang]{
		Items:      items,
		Page:       options.Page,
		PageSize:   options.PageSize,
		TotalItems: total,
		TotalPages: listing.PageCount(total, options.PageSize),
	}, nil
}

// GetTridharmaPenunjang mengembalikan satu tridharma penunjang bila principal berhak.
func (s *TridharmaService) GetTridharmaPenunjang(ctx context.Context, principal auth.Principal, id string) (models.TridharmaPenunjang, error) {
	needsScope := allows(s.policy, principal, "tridharma:penunjang:read:homebase") && !allows(s.policy, principal, "tridharma:penunjang:read:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.TridharmaPenunjang{}, err
	}

	item, err := s.repo.GetTridharmaPenunjang(ctx, id)
	if err != nil {
		return models.TridharmaPenunjang{}, err
	}
	if !s.canAccessPenunjang(scoped, principal, item) {
		return models.TridharmaPenunjang{}, domain.ErrForbidden
	}
	return item, nil
}

// CreateTridharmaPenunjang membuat tridharma penunjang baru.
func (s *TridharmaService) CreateTridharmaPenunjang(ctx context.Context, principal auth.Principal, item models.TridharmaPenunjang) (models.TridharmaPenunjang, error) {
	if !allows(s.policy, principal, "tridharma:penunjang:manage:any") && !allows(s.policy, principal, "tridharma:penunjang:manage:homebase") {
		return models.TridharmaPenunjang{}, domain.ErrForbidden
	}
	return s.repo.CreateTridharmaPenunjang(ctx, principal.UserID, item)
}

// UpdateTridharmaPenunjang mengubah field tridharma penunjang.
func (s *TridharmaService) UpdateTridharmaPenunjang(ctx context.Context, principal auth.Principal, id string, updates map[string]any) (models.TridharmaPenunjang, error) {
	needsScope := allows(s.policy, principal, "tridharma:penunjang:manage:homebase") && !allows(s.policy, principal, "tridharma:penunjang:manage:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.TridharmaPenunjang{}, err
	}

	item, err := s.repo.GetTridharmaPenunjang(ctx, id)
	if err != nil {
		return models.TridharmaPenunjang{}, err
	}
	if !s.canAccessPenunjang(scoped, principal, item) {
		return models.TridharmaPenunjang{}, domain.ErrForbidden
	}
	return s.repo.UpdateTridharmaPenunjang(ctx, principal.UserID, id, updates)
}

// DeleteTridharmaPenunjang menghapus tridharma penunjang.
func (s *TridharmaService) DeleteTridharmaPenunjang(ctx context.Context, principal auth.Principal, id string) error {
	needsScope := allows(s.policy, principal, "tridharma:penunjang:manage:homebase") && !allows(s.policy, principal, "tridharma:penunjang:manage:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return err
	}

	item, err := s.repo.GetTridharmaPenunjang(ctx, id)
	if err != nil {
		return err
	}
	if !s.canAccessPenunjang(scoped, principal, item) {
		return domain.ErrForbidden
	}
	return s.repo.DeleteTridharmaPenunjang(ctx, principal.UserID, id)
}
