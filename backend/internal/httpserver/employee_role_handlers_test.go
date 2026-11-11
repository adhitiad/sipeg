package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"simpeg/backend/internal/auth"
)

func TestEmployeeUpdateRejectsPrivilegeAndIdentityFields(t *testing.T) {
	store := &fakeDirectoryStore{}
	mux := testDirectoryMux(store, map[string]map[auth.Permission]bool{
		"operator": {"employee:update:any": true},
	})
	request := httptest.NewRequest(http.MethodPatch, APIPrefix+"/employees/11111111-1111-4111-8111-111111111111",
		strings.NewReader(`{"roleId":"22222222-2222-4222-8222-222222222222"}`))
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", response.Code)
	}
	if store.employeePatch.Nama.Present {
		t.Fatal("invalid update reached the store")
	}
}

func TestAssignEmployeeRoleIsAdminOnlyAndValidatesRole(t *testing.T) {
	path := APIPrefix + "/employees/11111111-1111-4111-8111-111111111111/role"
	adminResolver := testPrincipalResolver(func(context.Context, string) (auth.Principal, error) {
		return auth.Principal{UserID: "admin-1", RoleName: "admin"}, nil
	})
	store := &fakeDirectoryStore{}
	mux := testDirectoryMuxWithResolver(adminResolver, store, map[string]map[auth.Permission]bool{
		"admin": {"employee:role:manage": true},
	})
	request := httptest.NewRequest(http.MethodPut, path, strings.NewReader(`{"roleName":" ATASAN "}`))
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusOK || store.assignedRole != "atasan" || store.actorID != "admin-1" {
		t.Fatalf("status=%d role=%q actor=%q body=%s", response.Code, store.assignedRole, store.actorID, response.Body)
	}

	request = httptest.NewRequest(http.MethodPut, path, strings.NewReader(`{"roleName":"superadmin"}`))
	request.Header.Set("Authorization", "Bearer valid-session")
	response = httptest.NewRecorder()
	testDirectoryMuxWithResolver(adminResolver, &fakeDirectoryStore{}, map[string]map[auth.Permission]bool{
		"admin": {"employee:role:manage": true},
	}).ServeHTTP(response, request)
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("unknown role status = %d, want 422", response.Code)
	}

	request = httptest.NewRequest(http.MethodPut, path, strings.NewReader(`{"roleName":"pegawai"}`))
	request.Header.Set("Authorization", "Bearer valid-session")
	response = httptest.NewRecorder()
	testDirectoryMux(&fakeDirectoryStore{}, map[string]map[auth.Permission]bool{
		"atasan": {"employee:read:homebase": true},
	}).ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("non-admin status = %d, want 403", response.Code)
	}
}

type testPrincipalResolver func(context.Context, string) (auth.Principal, error)

func (resolver testPrincipalResolver) ResolvePrincipal(ctx context.Context, token string) (auth.Principal, error) {
	return resolver(ctx, token)
}
