package app

import (
	"context"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
	"simpeg/backend/internal/skp/domain"
)

// ═══════════════════════════ Log Aktivitas Harian ═══════════════════════════

// ListLogAktivitas mengembalikan daftar log aktivitas harian sesuai cakupan.
func (s *SKPService) ListLogAktivitas(ctx context.Context, principal auth.Principal, options listing.Options) (listing.Page[models.LogAktivitasHarian], error) {
	switch {
	case allows(s.policy, principal, "skp:log:read:any"):
	case allows(s.policy, principal, "skp:log:read:homebase"):
		scope, err := s.scope.resolve(ctx, principal)
		if err != nil {
			return listing.Page[models.LogAktivitasHarian]{}, err
		}
		if scope.Empty() {
			return listing.EmptyPage[models.LogAktivitasHarian](options.Page, options.PageSize), nil
		}
	case allows(s.policy, principal, "skp:log:read:self"):
		options.UserID = principal.UserID
	default:
		return listing.Page[models.LogAktivitasHarian]{}, domain.ErrForbidden
	}

	items, total, err := s.repo.ListLogAktivitas(ctx, options)
	if err != nil {
		return listing.Page[models.LogAktivitasHarian]{}, err
	}
	return listing.Page[models.LogAktivitasHarian]{
		Items:      items,
		Page:       options.Page,
		PageSize:   options.PageSize,
		TotalItems: total,
		TotalPages: listing.PageCount(total, options.PageSize),
	}, nil
}

// GetLogAktivitas mengembalikan satu log aktivitas harian bila principal berhak.
func (s *SKPService) GetLogAktivitas(ctx context.Context, principal auth.Principal, id string) (models.LogAktivitasHarian, error) {
	needsScope := allows(s.policy, principal, "skp:log:read:homebase") && !allows(s.policy, principal, "skp:log:read:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.LogAktivitasHarian{}, err
	}

	log, err := s.repo.GetLogAktivitas(ctx, id)
	if err != nil {
		return models.LogAktivitasHarian{}, err
	}
	if !s.canAccessLog(scoped, principal, log) {
		return models.LogAktivitasHarian{}, domain.ErrForbidden
	}
	return log, nil
}

// CreateLogAktivitas membuat log aktivitas harian baru.
func (s *SKPService) CreateLogAktivitas(ctx context.Context, principal auth.Principal, log models.LogAktivitasHarian) (models.LogAktivitasHarian, error) {
	if !allows(s.policy, principal, "skp:log:manage:any") && !allows(s.policy, principal, "skp:log:manage:homebase") {
		return models.LogAktivitasHarian{}, domain.ErrForbidden
	}
	return s.repo.CreateLogAktivitas(ctx, principal.UserID, log)
}

// UpdateLogAktivitas mengubah field log aktivitas harian.
func (s *SKPService) UpdateLogAktivitas(ctx context.Context, principal auth.Principal, id string, updates map[string]any) (models.LogAktivitasHarian, error) {
	needsScope := allows(s.policy, principal, "skp:log:manage:homebase") && !allows(s.policy, principal, "skp:log:manage:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.LogAktivitasHarian{}, err
	}

	log, err := s.repo.GetLogAktivitas(ctx, id)
	if err != nil {
		return models.LogAktivitasHarian{}, err
	}
	if !s.canAccessLog(scoped, principal, log) {
		return models.LogAktivitasHarian{}, domain.ErrForbidden
	}
	return s.repo.UpdateLogAktivitas(ctx, principal.UserID, id, updates)
}

// DeleteLogAktivitas menghapus log aktivitas harian.
func (s *SKPService) DeleteLogAktivitas(ctx context.Context, principal auth.Principal, id string) error {
	needsScope := allows(s.policy, principal, "skp:log:manage:homebase") && !allows(s.policy, principal, "skp:log:manage:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return err
	}

	log, err := s.repo.GetLogAktivitas(ctx, id)
	if err != nil {
		return err
	}
	if !s.canAccessLog(scoped, principal, log) {
		return domain.ErrForbidden
	}
	return s.repo.DeleteLogAktivitas(ctx, principal.UserID, id)
}

// ═══════════════════════════ Penilaian Kinerja Akhir ═══════════════════════════

// ListPenilaian mengembalikan daftar penilaian kinerja akhir sesuai cakupan.
func (s *SKPService) ListPenilaian(ctx context.Context, principal auth.Principal, options listing.Options) (listing.Page[models.PenilaianKinerjaAkhir], error) {
	switch {
	case allows(s.policy, principal, "skp:penilaian:read:any"):
	case allows(s.policy, principal, "skp:penilaian:read:homebase"):
		scope, err := s.scope.resolve(ctx, principal)
		if err != nil {
			return listing.Page[models.PenilaianKinerjaAkhir]{}, err
		}
		if scope.Empty() {
			return listing.EmptyPage[models.PenilaianKinerjaAkhir](options.Page, options.PageSize), nil
		}
	case allows(s.policy, principal, "skp:penilaian:read:self"):
		options.UserID = principal.UserID
	default:
		return listing.Page[models.PenilaianKinerjaAkhir]{}, domain.ErrForbidden
	}

	items, total, err := s.repo.ListPenilaian(ctx, options)
	if err != nil {
		return listing.Page[models.PenilaianKinerjaAkhir]{}, err
	}
	return listing.Page[models.PenilaianKinerjaAkhir]{
		Items:      items,
		Page:       options.Page,
		PageSize:   options.PageSize,
		TotalItems: total,
		TotalPages: listing.PageCount(total, options.PageSize),
	}, nil
}

// GetPenilaian mengembalikan satu penilaian kinerja akhir bila principal berhak.
func (s *SKPService) GetPenilaian(ctx context.Context, principal auth.Principal, id string) (models.PenilaianKinerjaAkhir, error) {
	needsScope := allows(s.policy, principal, "skp:penilaian:read:homebase") && !allows(s.policy, principal, "skp:penilaian:read:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.PenilaianKinerjaAkhir{}, err
	}

	penilaian, err := s.repo.GetPenilaian(ctx, id)
	if err != nil {
		return models.PenilaianKinerjaAkhir{}, err
	}
	if !s.canAccessPenilaian(scoped, principal, penilaian) {
		return models.PenilaianKinerjaAkhir{}, domain.ErrForbidden
	}
	return penilaian, nil
}

// CreatePenilaian membuat penilaian kinerja akhir baru.
func (s *SKPService) CreatePenilaian(ctx context.Context, principal auth.Principal, penilaian models.PenilaianKinerjaAkhir) (models.PenilaianKinerjaAkhir, error) {
	if !allows(s.policy, principal, "skp:penilaian:manage:any") && !allows(s.policy, principal, "skp:penilaian:manage:homebase") {
		return models.PenilaianKinerjaAkhir{}, domain.ErrForbidden
	}
	return s.repo.CreatePenilaian(ctx, principal.UserID, penilaian)
}

// UpdatePenilaian mengubah field penilaian kinerja akhir.
func (s *SKPService) UpdatePenilaian(ctx context.Context, principal auth.Principal, id string, updates map[string]any) (models.PenilaianKinerjaAkhir, error) {
	needsScope := allows(s.policy, principal, "skp:penilaian:manage:homebase") && !allows(s.policy, principal, "skp:penilaian:manage:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.PenilaianKinerjaAkhir{}, err
	}

	penilaian, err := s.repo.GetPenilaian(ctx, id)
	if err != nil {
		return models.PenilaianKinerjaAkhir{}, err
	}
	if !s.canAccessPenilaian(scoped, principal, penilaian) {
		return models.PenilaianKinerjaAkhir{}, domain.ErrForbidden
	}
	return s.repo.UpdatePenilaian(ctx, principal.UserID, id, updates)
}

// DeletePenilaian menghapus penilaian kinerja akhir.
func (s *SKPService) DeletePenilaian(ctx context.Context, principal auth.Principal, id string) error {
	needsScope := allows(s.policy, principal, "skp:penilaian:manage:homebase") && !allows(s.policy, principal, "skp:penilaian:manage:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return err
	}

	penilaian, err := s.repo.GetPenilaian(ctx, id)
	if err != nil {
		return err
	}
	if !s.canAccessPenilaian(scoped, principal, penilaian) {
		return domain.ErrForbidden
	}
	return s.repo.DeletePenilaian(ctx, principal.UserID, id)
}

// ═══════════════════════════ scope helpers ════════════════════════════

// scopedPrincipal menyelesaikan cakupan unit principal bila dibutuhkan.
func (s *SKPService) scopedPrincipal(ctx context.Context, principal auth.Principal, needsScope bool) (auth.UnitScope, error) {
	if !needsScope {
		return auth.UnitScope{}, nil
	}
	return s.scope.resolve(ctx, principal)
}

// canAccessPeriode memeriksa apakah principal berhak mengakses periode SKP.
func (s *SKPService) canAccessPeriode(scope auth.UnitScope, principal auth.Principal, periode models.SKPPeriode) bool {
	if allows(s.policy, principal, "skp:periode:read:any") || allows(s.policy, principal, "skp:periode:manage:any") {
		return true
	}
	return scope.Contains("")
}

// canAccessTarget memeriksa apakah principal berhak mengakses target kinerja.
func (s *SKPService) canAccessTarget(scope auth.UnitScope, principal auth.Principal, target models.SKPTargetKinerja) bool {
	if allows(s.policy, principal, "skp:target:read:any") || allows(s.policy, principal, "skp:target:manage:any") {
		return true
	}
	if principal.UserID == target.UserId {
		return true
	}
	return scope.Contains("")
}

// canAccessLog memeriksa apakah principal berhak mengakses log aktivitas.
func (s *SKPService) canAccessLog(scope auth.UnitScope, principal auth.Principal, log models.LogAktivitasHarian) bool {
	if allows(s.policy, principal, "skp:log:read:any") || allows(s.policy, principal, "skp:log:manage:any") {
		return true
	}
	if principal.UserID == log.UserId {
		return true
	}
	return scope.Contains("")
}

// canAccessPenilaian memeriksa apakah principal berhak mengakses penilaian.
func (s *SKPService) canAccessPenilaian(scope auth.UnitScope, principal auth.Principal, penilaian models.PenilaianKinerjaAkhir) bool {
	if allows(s.policy, principal, "skp:penilaian:read:any") || allows(s.policy, principal, "skp:penilaian:manage:any") {
		return true
	}
	if principal.UserID == penilaian.UserId {
		return true
	}
	return scope.Contains("")
}
