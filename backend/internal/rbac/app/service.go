// Package app memuat use case RBAC berbasis scoped assignment
// (`DEC-087`): evaluasi izin terhadap permission efektif, tier role, dan
// blacklist.
package app

import (
	"context"
	"encoding/json"
	"strings"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/models"
	"simpeg/backend/internal/rbac/domain"
	"simpeg/backend/internal/rbac/repository"
)

// Service menjalankan evaluasi akses RBAC.
type Service struct {
	repo repository.Store
}

type rbacCachedData struct {
	permissions     []models.RbacPermission
	rolePermissions []models.RbacRolePermission
	roles           []models.RbacRole
}

// NewService membuat Service.
func NewService(repo repository.Store) *Service {
	return &Service{repo: repo}
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

	if s.repo == nil || strings.TrimSpace(principal.UserID) == "" {
		return domain.Deny("principal atau repository RBAC tidak valid"), nil
	}
	assignments, err := s.repo.ListUserRoleScopes(ctx, principal.UserID)
	if err != nil {
		return domain.Deny("gagal memuat assignment"), err
	}
	if len(assignments) == 0 {
		return domain.NoAssignment(), nil
	}
	data, err := s.loadRBACData(ctx)
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

	superadmin := false
	operatorInduk := false
	granted := false

	for _, assignment := range assignments {
		if !assignmentMatchesRequestedScope(assignment, scopePerm, principal.HomebaseID) {
			continue
		}
		role, ok := rolesByID[assignment.RoleID]
		if !ok {
			continue
		}
		tier := domain.TierRole(role.TierLevel)

		if tier == domain.TierSuperadmin {
			// Superadmin adalah satu-satunya tier yang secara eksplisit
			// melewati blacklist. Scope assignment tetap harus cocok.
			superadmin = true
			continue
		}
		if isBlacklisted(assignment.DeniedPermissions, eff.Value, base, key) {
			return domain.Deny(domain.ErrBlacklisted.Error()), nil
		}
		if tier == domain.TierOperatorInduk {
			operatorInduk = true
			continue
		}
		if allowed := permissionsByRole[assignment.RoleID]; allowed != nil && allowed[key] {
			granted = true
		}
	}

	if superadmin || operatorInduk || granted {
		return domain.Allow(), nil
	}
	return domain.Deny(domain.ErrForbidden.Error()), nil
}

// loadRBACData membaca matriks dan role langsung dari database. Keputusan
// authorization tidak di-cache agar pencabutan izin berlaku pada request
// berikutnya, bukan tertunda oleh TTL.
func (s *Service) loadRBACData(ctx context.Context) (*rbacCachedData, error) {
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
	return &rbacCachedData{
		permissions:     permissions,
		rolePermissions: rolePermissions,
		roles:           roles,
	}, nil
}

// assignmentMatchesRequestedScope memastikan assignment RBAC benar-benar
// sejalan dengan scope yang dapat ditegakkan oleh resolver data saat ini.
// Scope fakultas/jurusan/biro belum dapat dipetakan aman ke seluruh repository,
// sehingga ditolak daripada diperluas menjadi akses homebase.
func assignmentMatchesRequestedScope(assignment models.RbacUserRoleScope, requested domain.ScopePermission, principalHomebaseID string) bool {
	if !isScopedAssignment(assignment) {
		return true
	}

	switch requested {
	case domain.ScopeAny:
		return false
	case domain.ScopeSelf:
		return true
	case domain.ScopeHomebase:
		if assignment.FakultasID != nil || assignment.JurusanID != nil || assignment.BiroCode != nil {
			return false
		}
		if assignment.UnitID == nil || strings.TrimSpace(*assignment.UnitID) == "" || strings.TrimSpace(principalHomebaseID) == "" {
			return false
		}
		return strings.EqualFold(strings.TrimSpace(*assignment.UnitID), strings.TrimSpace(principalHomebaseID))
	default:
		return false
	}
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
	if err := json.Unmarshal(denied, &list); err != nil {
		// JSON blacklist yang rusak tidak boleh berubah menjadi allow.
		return true
	}
	if len(list) == 0 {
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
