package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeResolver struct {
	principal Principal
	err       error
}

func (r fakeResolver) ResolvePrincipal(context.Context, string) (Principal, error) {
	return r.principal, r.err
}

func TestAuthenticateRejectsMissingAndInvalidSessions(t *testing.T) {
	resolver := fakeResolver{err: ErrInvalidSession}
	handler := Authenticate(resolver)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler must not run without a valid session")
	}))

	for _, authorization := range []string{"", "Basic abc", "Bearer invalid"} {
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", authorization)
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		if res.Code != http.StatusUnauthorized {
			t.Errorf("authorization %q: expected 401, got %d", authorization, res.Code)
		}
	}
}

func TestAuthenticateReportsResolverUnavailable(t *testing.T) {
	handler := Authenticate(fakeResolver{err: errors.New("database unavailable")})(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler must not run when session resolution fails")
	}))
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer valid-session")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 for resolver failure, got %d", res.Code)
	}
	if res.Body.String() != "{\"error\":{\"code\":\"service_unavailable\"}}\n" {
		t.Fatalf("unexpected error response: %s", res.Body.String())
	}
}

func TestAuthenticateReportsMissingResolverAsUnavailable(t *testing.T) {
	handler := Authenticate(nil)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler must not run without a session resolver")
	}))
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer valid-session")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 for missing resolver, got %d", res.Code)
	}
}

func TestRequirePermissionDeniesByDefault(t *testing.T) {
	principal := Principal{UserID: "user-1", RoleName: "operator"}
	handler := RequirePermission(StaticPolicy{}, "employee:write", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler must not run when permission is not granted")
	}))
	req := httptest.NewRequest(http.MethodPost, "/employees", nil)
	req = req.WithContext(WithPrincipal(req.Context(), principal))
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.Code)
	}
}

func TestRequirePermissionRejectsMissingPrincipal(t *testing.T) {
	handler := RequirePermission(StaticPolicy{"operator": {"employee:write": true}}, "employee:write", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler must not run without an authenticated principal")
	}))
	req := httptest.NewRequest(http.MethodPost, "/employees", nil)
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without principal, got %d", res.Code)
	}
}

func TestRequirePermissionAllowsExplicitRoleGrant(t *testing.T) {
	principal := Principal{UserID: "user-1", RoleName: "operator"}
	policy := StaticPolicy{"operator": {"employee:write": true}}
	handler := RequirePermission(policy, "employee:write", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodPost, "/employees", nil)
	req = req.WithContext(WithPrincipal(req.Context(), principal))
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.Code)
	}
}

func TestAuthenticateSetsPrincipalFromResolvedSession(t *testing.T) {
	want := Principal{UserID: "user-1", Email: "person@example.test", RoleName: "operator"}
	handler := Authenticate(fakeResolver{principal: want})(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		got, ok := PrincipalFromContext(req.Context())
		if !ok || got != want {
			t.Errorf("expected principal %v, got %v", want, got)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer opaque-session")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.Code)
	}
}

func TestAuthorizationUsesResolvedRoleNotRequestHeaders(t *testing.T) {
	policy := StaticPolicy{
		"operator":   {"employee:write": false},
		"superadmin": {"employee:write": true},
	}
	handler := Authenticate(fakeResolver{
		principal: Principal{UserID: "user-1", RoleName: "operator"},
	})(RequirePermission(policy, "employee:write", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("request header must not elevate the resolved role")
	})))
	req := httptest.NewRequest(http.MethodPost, "/employees", nil)
	req.Header.Set("Authorization", "Bearer valid-session")
	req.Header.Set("X-Role", "superadmin")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for ungranted resolved role, got %d", res.Code)
	}
}

func TestCanAccessHomebaseRequiresSelfOrExplicitScope(t *testing.T) {
	principal := Principal{UserID: "user-1", RoleName: "supervisor", HomebaseID: "unit-a"}
	policy := StaticPolicy{"supervisor": {
		"employee:read:self":     true,
		"employee:read:homebase": true,
	}}
	// Cakupan unit "unit-a" beserta turunannya "unit-a-child".
	scope := NewUnitScope("unit-a", []string{"unit-a", "unit-a-child"})

	if !CanAccessUserInScope(principal, "user-1", "unit-b", "employee:read:self", policy, scope) {
		t.Fatal("user must be able to access their own record")
	}
	if !CanAccessUserInScope(principal, "user-2", "unit-a", "employee:read:homebase", policy, scope) {
		t.Fatal("role must be able to access records in its assigned homebase")
	}
	if !CanAccessUserInScope(principal, "user-2", "unit-a-child", "employee:read:homebase", policy, scope) {
		t.Fatal("role must be able to access records in descendant units")
	}
	if CanAccessUserInScope(principal, "user-2", "unit-b", "employee:read:homebase", policy, scope) {
		t.Fatal("role must not access records outside its assigned homebase")
	}
	if CanAccessUserInScope(principal, "user-2", "unit-a", "employee:read:any", policy, scope) {
		t.Fatal("ungranted cross-user permission must be denied")
	}
}

func TestUnitScopeContainsAndEmpty(t *testing.T) {
	scope := NewUnitScope("unit-a", []string{"unit-a", "unit-a-child"})
	if !scope.Contains("unit-a") || !scope.Contains("unit-a-child") {
		t.Fatal("expected both units in scope")
	}
	if scope.Contains("unit-b") || scope.Contains("") {
		t.Fatal("unit outside scope must not be contained")
	}
	if scope.Empty() {
		t.Fatal("scope with units must not be empty")
	}
	if !NewUnitScope("", nil).Empty() {
		t.Fatal("scope without units must be empty")
	}
	if len(scope.List()) != 2 {
		t.Fatalf("unexpected scope list: %v", scope.List())
	}
}

func TestCanAccessUserInScopeDeniesWithoutPolicy(t *testing.T) {
	principal := Principal{UserID: "user-1", RoleName: "operator"}
	scope := NewUnitScope("unit-a", []string{"unit-a"})
	if CanAccessUserInScope(principal, "user-2", "unit-a", "employee:read:homebase", nil, scope) {
		t.Fatal("missing policy must deny access")
	}
}
