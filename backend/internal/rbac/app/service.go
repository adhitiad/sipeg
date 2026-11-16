// Package app memuat use case RBAC berbasis scoped assignment
// (`DEC-087`): evaluasi izin terhadap permission efektif, tier role, dan
// blacklist.
package app

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/models"
	"simpeg/backend/internal/rbac/domain"
	"simpeg/backend/internal/rbac/repository"
)

// Service menjalankan evaluasi akses RBAC.
type Service struct {
	repo repository.Store

	// cache data RBAC (permissions, rolePermissions, roles) dan assignment
	// per user dengan TTL singkat untuk menghindari query berulang pada setiap
	// permintaan (DEC-097). RBAC berubah jarang; TTL 60 detik cukup.
	mu         sync.Mutex
	dataCache  *rbacCachedData
	dataExpiry time.Time
	userCache  map[string]userCacheEntry
}

type rbacCachedData struct {
	permissions     []models.RbacPermission
	rolePermissions []models.RbacRolePermission
	roles           []models.RbacRole
}

type userCacheEntry struct {
	assignments []models.RbacUserRoleScope
	expiry      time.Time
}

// NewService membuat Service.
func NewService(repo repository.Store) *Service {
	return &Service{repo: repo, userCache: make(map[string]userCacheEntry)}
}

// CanAccess menilai apakah principal dapat melakukan modul:aksi pada
// cakupan `scope`. Alur: blacklist -> tier -> matriks permission -> scope.
// Kunci matriks adalah `<module>:<action>:<scope>` (mis.
// `payroll:manage:homebase`); permission efektif dihitung lewat
// ParsePermission sehingga permission berskala ganda atau multi-aksi
// (`skp:periode:read:any`) dipetakan secara konsisten.
func (s *Service) CanAccess(ctx context.Context, principal auth.Principal, module, action string, scope string) (domain.Decision, error) {
	scopePerm := domain.ScopePermission(strings.ToLower(strings.TrimSpace(scope)))
	if scopePerm != domain.ScopeAny && scopePerm != domain.ScopeHomebase && scopePerm != domain.ScopeSelf {
		scopePerm = domain.ScopeAny
	}
	base := strings.ToLower(strings.TrimSpace(module)) + ":" + strings.ToLower(strings.TrimSpace(action))
	key := base + ":" + string(scopePerm)
	eff := domain.NormalizePermission(module, action, scopePerm)

	assignments, err := s.cachedUserRoleScopes(ctx, principal.UserID)
	if err != nil {
		return domain.Deny("gagal memuat assignment"), err
	}
	if len(assignments) == 0 {
		return domain.NoAssignment(), nil
	}
	data, err := s.cachedRBACData(ctx)
	if err != nil {
		return domain.Deny("gagal memuat data RBAC"), err
	}
	permissions := data.permissions
	rolePermissions := data.rolePermissions
	roles := data.roles

	permissionsByID := make(map[uint]domain.EffectivePermission, len(permissions))
	for _, permission := range permissions {
		permissionsByID[permission.ID] = domain.NormalizePermission(permission.Module, permission.Action, domain.ScopeAny)
	}

	permissionsByRole := make(map[uint]map[string]bool, len(roles))
	for _, row := range rolePermissions {
		permission, ok := permissionsByID[row.PermissionID]
		if !ok {
			continue
		}
		for _, candidateScope := range []domain.ScopePermission{domain.ScopeAny, domain.ScopeHomebase, domain.ScopeSelf} {
			key := strings.ToLower(permission.Module) + ":" + strings.ToLower(permission.Action) + ":" + string(candidateScope)
			if permissionsByRole[row.RoleID] == nil {
				permissionsByRole[row.RoleID] = make(map[string]bool)
			}
			permissionsByRole[row.RoleID][key] = true
		}
	}

	rolesByID := make(map[uint]models.RbacRole, len(roles))
	for _, role := range roles {
		rolesByID[role.ID] = role
	}

	for _, assignment := range assignments {
		if scopePerm == domain.ScopeAny && isScopedAssignment(assignment) {
			// Assignment yang terbatas pada scope tertentu (unit/fakultas/
			// jurusan/biro) tidak pernah mengizinkan permintaan lintas
			// organisasi — kendati role-nya punya grant `:any` di matriks.
			continue
		}
		role, ok := rolesByID[assignment.RoleID]
		if !ok {
			continue
		}
		tier := domain.TierRole(role.TierLevel)

		if tier == domain.TierSuperadmin {
			return domain.Allow(), nil
		}
		if isBlacklisted(assignment.DeniedPermissions, eff.Value, base, key) {
			return domain.Deny(domain.ErrBlacklisted.Error()), nil
		}
		if tier == domain.TierOperatorInduk {
			return domain.Allow(), nil
		}
		if allowed := permissionsByRole[assignment.RoleID]; allowed != nil && allowed[key] {
			return domain.Allow(), nil
		}
	}

	return domain.Deny(domain.ErrForbidden.Error()), nil
}

// cachedRBACData mengembalikan permissions, rolePermissions, roles dengan cache
// TTL 60 detik (DEC-097).
func (s *Service) cachedRBACData(ctx context.Context) (*rbacCachedData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.dataCache != nil && time.Now().Before(s.dataExpiry) {
		return s.dataCache, nil
	}
	permissions, err := s.repo.ListPermissions(ctx)
	if err != nil {
		return nil, err
	}
	rolePermissions, err := s.repo.ListRolePermissions(ctx)
	if err != nil {
		return nil, err
	}
	roles, err := s.repo.ListRoles(ctx)
	if err != nil {
		return nil, err
	}
	s.dataCache = &rbacCachedData{
		permissions:     permissions,
		rolePermissions: rolePermissions,
		roles:           roles,
	}
	s.dataExpiry = time.Now().Add(60 * time.Second)
	return s.dataCache, nil
}

// cachedUserRoleScopes mengembalikan assignment user dengan cache TTL 60 detik.
func (s *Service) cachedUserRoleScopes(ctx context.Context, userID string) ([]models.RbacUserRoleScope, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if entry, ok := s.userCache[userID]; ok && time.Now().Before(entry.expiry) {
		return entry.assignments, nil
	}
	assignments, err := s.repo.ListUserRoleScopes(ctx, userID)
	if err != nil {
		return nil, err
	}
	s.userCache[userID] = userCacheEntry{assignments: assignments, expiry: time.Now().Add(60 * time.Second)}
	return assignments, nil
}

// isScopedAssignment melaporkan apakah assignment dibatasi pada scope tertentu.
func isScopedAssignment(assignment models.RbacUserRoleScope) bool {
	return assignment.UnitID != nil || assignment.FakultasID != nil || assignment.JurusanID != nil || assignment.BiroCode != nil
}

// isBlacklisted mengecek daftar permission yang dilarang pada assignment
// terhadap seluruh representasi nilai permission (base, ber-scope, dan
// bentuk efektif) agar blacklist menolak grant terkait.
func isBlacklisted(denied []byte, values ...string) bool {
	if len(denied) == 0 {
		return false
	}
	var list []string
	if err := json.Unmarshal(denied, &list); err != nil || len(list) == 0 {
		return false
	}
	for _, candidate := range list {
		for _, value := range values {
			if strings.EqualFold(strings.TrimSpace(candidate), value) {
				return true
			}
		}
	}
	return false
}
