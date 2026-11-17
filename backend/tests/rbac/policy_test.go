package rbac

import (
	"context"
	"errors"
	"testing"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/models"
	rbacapp "simpeg/backend/internal/rbac/app"
	"simpeg/backend/internal/rbac/repository"
)

type failingAssignmentStore struct {
	repository.Store
}

func (failingAssignmentStore) ListUserRoleScopes(context.Context, string) ([]models.RbacUserRoleScope, error) {
	return nil, errors.New("postgres unavailable")
}

type noAssignmentStore struct {
	repository.Store
}

func (noAssignmentStore) ListUserRoleScopes(context.Context, string) ([]models.RbacUserRoleScope, error) {
	return nil, nil
}

func TestPolicyAdapterFailsClosedWhenRBACCannotBeRead(t *testing.T) {
	static := auth.StaticPolicy{"admin": {"rbac:read": true}}
	service := rbacapp.NewService(failingAssignmentStore{})
	adapter := NewPolicyAdapter(static, service)

	allowed := adapter.AllowsScoped(context.Background(), auth.Principal{
		UserID:   "00000000-0000-4000-8000-000000000001",
		RoleName: "admin",
	}, "rbac:read")
	if allowed {
		t.Fatal("RBAC storage failure must not fall back to a permissive static grant")
	}
}

func TestPolicyAdapterUsesStaticFallbackOnlyForExplicitNoAssignment(t *testing.T) {
	static := auth.StaticPolicy{"admin": {"rbac:read": true}}
	service := rbacapp.NewService(noAssignmentStore{})
	adapter := NewPolicyAdapter(static, service)

	allowed := adapter.AllowsScoped(context.Background(), auth.Principal{
		UserID: "00000000-0000-4000-8000-000000000001",
		RoleName: "admin",
	}, "rbac:read")
	if !allowed {
		t.Fatal("static migration fallback is allowed only when RBAC confirms there is no assignment")
	}
}
