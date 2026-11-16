package rbacapp

import (
	"context"
	"testing"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/models"
	rbacapp "simpeg/backend/internal/rbac/app"
	"simpeg/backend/internal/rbac/domain"
)

type fakeStore struct {
	permissions []models.RbacPermission
	roles       []models.RbacRole
	rolePerms   []models.RbacRolePermission
	assignments []models.RbacUserRoleScope
}

func (f *fakeStore) ListPermissions(ctx context.Context) ([]models.RbacPermission, error) {
	return f.permissions, nil
}

func (f *fakeStore) ListRoles(ctx context.Context) ([]models.RbacRole, error) {
	return f.roles, nil
}

func (f *fakeStore) ListRolePermissions(ctx context.Context) ([]models.RbacRolePermission, error) {
	return f.rolePerms, nil
}

func (f *fakeStore) ListUserRoleScopes(ctx context.Context, userID string) ([]models.RbacUserRoleScope, error) {
	items := []models.RbacUserRoleScope{}
	for _, assignment := range f.assignments {
		if assignment.UserID == userID {
			items = append(items, assignment)
		}
	}
	return items, nil
}

func (f *fakeStore) CreateRole(ctx context.Context, role models.RbacRole) (models.RbacRole, error) {
	return role, nil
}

func (f *fakeStore) UpdateRole(ctx context.Context, id uint, role models.RbacRole) (models.RbacRole, error) {
	role.ID = id
	return role, nil
}

func (f *fakeStore) DeleteRole(ctx context.Context, id uint) error { return nil }

func (f *fakeStore) CreatePermission(ctx context.Context, perm models.RbacPermission) (models.RbacPermission, error) {
	return perm, nil
}

func (f *fakeStore) DeletePermission(ctx context.Context, id uint) error { return nil }

func (f *fakeStore) AssignPermissionToRole(ctx context.Context, roleID, permID uint) error {
	return nil
}

func (f *fakeStore) RevokePermissionFromRole(ctx context.Context, roleID, permID uint) error {
	return nil
}

func (f *fakeStore) CreateUserRoleScope(ctx context.Context, scope models.RbacUserRoleScope) (models.RbacUserRoleScope, error) {
	return scope, nil
}

func (f *fakeStore) DeleteUserRoleScope(ctx context.Context, id uint) error { return nil }

func (f *fakeStore) ListDelegasi(ctx context.Context, userID string) ([]models.Delegasi, error) {
	return nil, nil
}

func (f *fakeStore) CreateDelegasi(ctx context.Context, d models.Delegasi) (models.Delegasi, error) {
	return d, nil
}

func (f *fakeStore) RevokeDelegasi(ctx context.Context, id uint) error { return nil }

func TestCanAccessOperatorIndukBypassBlacklist(t *testing.T) {
	store := &fakeStore{
		assignments: []models.RbacUserRoleScope{
			{UserID: "operator-1", RoleID: 2},
		},
		roles: []models.RbacRole{
			{ID: 2, Code: "OPERATOR_INDUK", Name: "Operator Induk", TierLevel: 2},
		},
	}
	service := rbacapp.NewService(store)
	decision, err := service.CanAccess(context.Background(), auth.Principal{UserID: "operator-1", RoleName: "OPERATOR_INDUK"}, "payroll", "manage", "any")
	if err != nil {
		t.Fatalf("canaccess: %v", err)
	}
	if !decision.Allowed {
		t.Fatalf("operator induk tanpa blacklist harus diizinkan, decision = %+v", decision)
	}
}

func TestCanAccessOperatorIndukBlacklistDeny(t *testing.T) {
	store := &fakeStore{
		assignments: []models.RbacUserRoleScope{
			{UserID: "operator-1", RoleID: 2, DeniedPermissions: []byte(`["payroll:manage"]`)},
		},
		roles: []models.RbacRole{
			{ID: 2, Code: "OPERATOR_INDUK", Name: "Operator Induk", TierLevel: 2},
		},
	}
	service := rbacapp.NewService(store)
	decision, err := service.CanAccess(context.Background(), auth.Principal{UserID: "operator-1", RoleName: "OPERATOR_INDUK"}, "payroll", "manage", "any")
	if err != nil {
		t.Fatalf("canaccess: %v", err)
	}
	if decision.Allowed {
		t.Fatal("operator induk dengan permission di-blacklist harus ditolak")
	}
}

func TestCanAccessAdminMatriksPermission(t *testing.T) {
	store := &fakeStore{
		assignments: []models.RbacUserRoleScope{
			{UserID: "admin-1", RoleID: 3},
		},
		permissions: []models.RbacPermission{
			{ID: 10, Module: "payroll", Action: "manage"},
		},
		rolePerms: []models.RbacRolePermission{
			{RoleID: 3, PermissionID: 10},
		},
		roles: []models.RbacRole{
			{ID: 3, Code: "ADMIN_UNIT", Name: "Admin Unit", TierLevel: 3},
		},
	}
	service := rbacapp.NewService(store)
	decision, err := service.CanAccess(context.Background(), auth.Principal{UserID: "admin-1", RoleName: "ADMIN_UNIT"}, "payroll", "manage", "any")
	if err != nil {
		t.Fatalf("canaccess: %v", err)
	}
	if !decision.Allowed || !decision.Authoritative {
		t.Fatalf("admin dengan akses matriks harus diizinkan (otoritatif), decision = %+v", decision)
	}

	decision, err = service.CanAccess(context.Background(), auth.Principal{UserID: "admin-1", RoleName: "ADMIN_UNIT"}, "payroll", "delete", "any")
	if err != nil {
		t.Fatalf("canaccess: %v", err)
	}
	if decision.Allowed || !decision.Authoritative {
		t.Fatalf("admin tanpa akses delete harus ditolak secara otoritatif, decision = %+v", decision)
	}
}

func TestCanAccessSuperadminBypass(t *testing.T) {
	store := &fakeStore{
		assignments: []models.RbacUserRoleScope{
			{UserID: "super-1", RoleID: 1},
		},
		roles: []models.RbacRole{
			{ID: 1, Code: "SUPERADMIN", Name: "Superadmin", TierLevel: 1},
		},
	}
	service := rbacapp.NewService(store)
	decision, err := service.CanAccess(context.Background(), auth.Principal{UserID: "super-1", RoleName: "SUPERADMIN"}, "system", "manage", "any")
	if err != nil {
		t.Fatalf("canaccess: %v", err)
	}
	if !decision.Allowed || !decision.Authoritative {
		t.Fatalf("superadmin harus bypass (otoritatif), decision = %+v", decision)
	}
}

func TestCanAccessNoAssignmentNotAuthoritative(t *testing.T) {
	store := &fakeStore{}
	service := rbacapp.NewService(store)
	decision, err := service.CanAccess(context.Background(), auth.Principal{UserID: "no-rbac"}, "payroll", "manage", "any")
	if err != nil {
		t.Fatalf("canaccess: %v", err)
	}
	if decision.Allowed || decision.Authoritative {
		t.Fatalf("tanpa assignment harus non-otoritatif (fallback statis), decision = %+v", decision)
	}
}

func TestCanAccessAdminMatriksPermissionHomebaseScope(t *testing.T) {
	store := &fakeStore{
		assignments: []models.RbacUserRoleScope{
			{UserID: "admin-homebase", RoleID: 4},
		},
		permissions: []models.RbacPermission{
			{ID: 20, Module: "payroll", Action: "read"},
		},
		rolePerms: []models.RbacRolePermission{
			{RoleID: 4, PermissionID: 20},
		},
		roles: []models.RbacRole{
			{ID: 4, Code: "ADMIN", Name: "Admin", TierLevel: 3},
		},
	}
	service := rbacapp.NewService(store)
	decision, err := service.CanAccess(context.Background(), auth.Principal{UserID: "admin-homebase", RoleName: "ADMIN"}, "payroll", "read", "homebase")
	if err != nil {
		t.Fatalf("canaccess: %v", err)
	}
	if !decision.Allowed || !decision.Authoritative {
		t.Fatalf("matriks harus mengizinkan scope homebase, decision = %+v", decision)
	}

	decision, err = service.CanAccess(context.Background(), auth.Principal{UserID: "admin-homebase", RoleName: "ADMIN"}, "payroll", "read", "self")
	if err != nil {
		t.Fatalf("canaccess: %v", err)
	}
	if !decision.Allowed {
		t.Fatalf("matriks harus mengizinkan scope self selama terdaftar, decision = %+v", decision)
	}
}

func TestCanAccessAdminMatriksMultiSegmentAction(t *testing.T) {
	store := &fakeStore{
		assignments: []models.RbacUserRoleScope{
			{UserID: "skp-admin", RoleID: 5},
		},
		permissions: []models.RbacPermission{
			{ID: 30, Module: "skp", Action: "periode:read"},
		},
		rolePerms: []models.RbacRolePermission{
			{RoleID: 5, PermissionID: 30},
		},
		roles: []models.RbacRole{
			{ID: 5, Code: "SKP_ADMIN", Name: "SKP Admin", TierLevel: 3},
		},
	}
	service := rbacapp.NewService(store)
	module, action, scope := domain.ParsePermission("skp:periode:read:any")
	if module != "skp" || action != "periode:read" || scope != domain.ScopeAny {
		t.Fatalf("parse permission multi-aksi = module %q action %q scope %q", module, action, scope)
	}
	decision, err := service.CanAccess(context.Background(), auth.Principal{UserID: "skp-admin", RoleName: "SKP_ADMIN"}, module, action, string(scope))
	if err != nil {
		t.Fatalf("canaccess: %v", err)
	}
	if !decision.Allowed {
		t.Fatalf("matriks multi-aksi harus mengizinkan, decision = %+v", decision)
	}
}

func TestCanAccessScopedAssignmentCannotAuthorizeAnyScope(t *testing.T) {
	unit := "20000000-0000-0000-0000-000000000001"
	store := &fakeStore{
		assignments: []models.RbacUserRoleScope{
			{UserID: "operator-any", RoleID: 7, UnitID: &unit},
		},
		permissions: []models.RbacPermission{
			{ID: 50, Module: "payroll", Action: "read"},
		},
		rolePerms: []models.RbacRolePermission{
			{RoleID: 7, PermissionID: 50},
		},
		roles: []models.RbacRole{
			{ID: 7, Code: "OPERATOR", Name: "Operator", TierLevel: 4},
		},
	}
	service := rbacapp.NewService(store)
	decision, err := service.CanAccess(context.Background(), auth.Principal{UserID: "operator-any", RoleName: "OPERATOR"}, "payroll", "read", "any")
	if err != nil {
		t.Fatalf("canaccess: %v", err)
	}
	if decision.Allowed || !decision.Authoritative {
		t.Fatalf("assignment scoped tidak boleh mengizinkan cakupan lintas organisasi, decision = %+v", decision)
	}

	decision, err = service.CanAccess(context.Background(), auth.Principal{UserID: "operator-any", RoleName: "OPERATOR"}, "payroll", "read", "homebase")
	if err != nil {
		t.Fatalf("canaccess: %v", err)
	}
	if !decision.Allowed || !decision.Authoritative {
		t.Fatalf("assignment scoped boleh mengizinkan cakupan homebase, decision = %+v", decision)
	}
}

func TestCanAccessUnscopedAssignmentAuthorizesAnyScope(t *testing.T) {
	store := &fakeStore{
		assignments: []models.RbacUserRoleScope{
			{UserID: "admin-any", RoleID: 8},
		},
		permissions: []models.RbacPermission{
			{ID: 51, Module: "payroll", Action: "read"},
		},
		rolePerms: []models.RbacRolePermission{
			{RoleID: 8, PermissionID: 51},
		},
		roles: []models.RbacRole{
			{ID: 8, Code: "ADMIN", Name: "Admin", TierLevel: 3},
		},
	}
	service := rbacapp.NewService(store)
	decision, err := service.CanAccess(context.Background(), auth.Principal{UserID: "admin-any", RoleName: "ADMIN"}, "payroll", "read", "any")
	if err != nil {
		t.Fatalf("canaccess: %v", err)
	}
	if !decision.Allowed || !decision.Authoritative {
		t.Fatalf("assignment tanpa cakupan harus mengizinkan cakupan lintas organisasi, decision = %+v", decision)
	}
}

func TestCanAccessBlacklistDeniesScopedGrant(t *testing.T) {
	store := &fakeStore{
		assignments: []models.RbacUserRoleScope{
			{UserID: "operator-scope", RoleID: 6, DeniedPermissions: []byte(`["payroll:manage"]`)},
		},
		permissions: []models.RbacPermission{
			{ID: 40, Module: "payroll", Action: "manage"},
		},
		rolePerms: []models.RbacRolePermission{
			{RoleID: 6, PermissionID: 40},
		},
		roles: []models.RbacRole{
			{ID: 6, Code: "OPERATOR_UNIT", Name: "Operator Unit", TierLevel: 4},
		},
	}
	service := rbacapp.NewService(store)
	decision, err := service.CanAccess(context.Background(), auth.Principal{UserID: "operator-scope", RoleName: "OPERATOR_UNIT"}, "payroll", "manage", "homebase")
	if err != nil {
		t.Fatalf("canaccess: %v", err)
	}
	if decision.Allowed {
		t.Fatal("blacklist `payroll:manage` harus menolak grant `payroll:manage:homebase`")
	}
}
