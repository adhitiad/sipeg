package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	. "simpeg/backend/internal/auth"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestJWTVerifierAcceptsValidToken(t *testing.T) {
	verifier := NewJWTVerifier("secret-123")
	tokenString := signJWT(t, "secret-123", BetterAuthClaims{
		UserID:     "00000000-0000-4000-8000-000000000001",
		Email:      "person@example.test",
		RoleName:   "operator",
		HomebaseID: "unit-a",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	})

	claims, err := verifier.VerifyToken(tokenString)
	if err != nil {
		t.Fatalf("expected valid token, got error: %v", err)
	}
	if claims.UserID != "00000000-0000-4000-8000-000000000001" || claims.RoleName != "operator" || claims.HomebaseID != "unit-a" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestJWTVerifierRejectsExpiredToken(t *testing.T) {
	verifier := NewJWTVerifier("secret-123")
	tokenString := signJWT(t, "secret-123", BetterAuthClaims{
		UserID: "00000000-0000-4000-8000-000000000001",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
		},
	})

	_, err := verifier.VerifyToken(tokenString)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestJWTVerifierRejectsWrongSecret(t *testing.T) {
	verifier := NewJWTVerifier("secret-123")
	tokenString := signJWT(t, "wrong-secret", BetterAuthClaims{
		UserID: "00000000-0000-4000-8000-000000000001",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	})

	_, err := verifier.VerifyToken(tokenString)
	if err == nil {
		t.Fatal("expected error for wrong secret")
	}
}

type fakeJWTPrincipalResolver struct {
	principal Principal
	err       error
}

func (r fakeJWTPrincipalResolver) ResolveUserPrincipal(context.Context, string) (Principal, error) {
	return r.principal, r.err
}

func TestJWTAuthenticateMiddlewareUsesDatabasePrincipal(t *testing.T) {
	verifier := NewJWTVerifier("secret-123")
	tokenString := signJWT(t, "secret-123", BetterAuthClaims{
		UserID:     "00000000-0000-4000-8000-000000000001",
		Email:      "stale@example.test",
		RoleName:   "admin",
		HomebaseID: "stale-unit",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	})

	resolved := Principal{
		UserID:     "00000000-0000-4000-8000-000000000001",
		Email:      "person@example.test",
		RoleName:   "pegawai",
		HomebaseID: "unit-db",
	}
	handler := JWTAuthenticateWithResolver(verifier, fakeJWTPrincipalResolver{principal: resolved}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := PrincipalFromContext(r.Context())
		if !ok {
			t.Fatal("principal not set in context")
		}
		if principal != resolved {
			t.Fatalf("resolver principal must override stale JWT claims: got %+v want %+v", principal, resolved)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.Code)
	}
}

func TestJWTAuthenticateFailsClosedWithoutDatabaseResolver(t *testing.T) {
	verifier := NewJWTVerifier("secret-123")
	tokenString := signJWT(t, "secret-123", BetterAuthClaims{
		UserID: "00000000-0000-4000-8000-000000000001",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	})
	handler := JWTAuthenticate(verifier, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler must not run without an authoritative principal resolver")
	}))
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 without resolver, got %d", res.Code)
	}
}
func TestJWTAuthenticateRejectsMissingToken(t *testing.T) {
	verifier := NewJWTVerifier("secret-123")
	handler := JWTAuthenticate(verifier, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler must not run without token")
	}))
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", res.Code)
	}
}

func TestRequireAnyPermissionAllowsAnyMatched(t *testing.T) {
	policy := StaticPolicy{"operator": {"employee:read:homebase": true}}
	principal := Principal{UserID: "00000000-0000-4000-8000-000000000001", RoleName: "operator"}
	nextCalled := false
	handler := RequireAnyPermission(policy, []Permission{"employee:read:any", "employee:read:homebase"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/employees", nil)
	req = req.WithContext(WithPrincipal(req.Context(), principal))
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if !nextCalled {
		t.Fatal("next handler should be called when any permission matches")
	}
	if res.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.Code)
	}
}

func TestRequireAnyPermissionDeniesWhenNoneMatch(t *testing.T) {
	policy := StaticPolicy{"operator": {"employee:read:homebase": true}}
	principal := Principal{UserID: "00000000-0000-4000-8000-000000000001", RoleName: "operator"}
	handler := RequireAnyPermission(policy, []Permission{"employee:read:any", "employee:create"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler must not run when no permission matches")
	}))
	req := httptest.NewRequest(http.MethodGet, "/employees", nil)
	req = req.WithContext(WithPrincipal(req.Context(), principal))
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.Code)
	}
}

func signJWT(t *testing.T, secret string, claims BetterAuthClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("failed to sign JWT: %v", err)
	}
	return tokenString
}

func TestCombinedSessionResolverRequiresDatabaseBackedJWTPrincipal(t *testing.T) {
	verifier := NewJWTVerifier("secret-123")
	tokenString := signJWT(t, "secret-123", BetterAuthClaims{
		UserID:     "00000000-0000-4000-8000-000000000002",
		RoleName:   "admin",
		HomebaseID: "unit-jwt",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	})
	resolver := NewCombinedSessionResolver(verifier, nil)

	_, err := resolver.ResolvePrincipal(context.TODO(), tokenString)
	if err != ErrInvalidSession {
		t.Fatalf("JWT without authoritative database principal must be rejected, got: %v", err)
	}
}

func TestCombinedSessionResolverFallsBackToGORM(t *testing.T) {
	verifier := NewJWTVerifier("secret-123")
	resolver := NewCombinedSessionResolver(verifier, nil)

	// Token yang bukan JWT (bukan 3 bagian) → fallback ke GORM (nil DB → ErrInvalidSession)
	_, err := resolver.ResolvePrincipal(context.TODO(), "opaque-session-token")
	if err != ErrInvalidSession {
		t.Fatalf("expected ErrInvalidSession for nil DB, got: %v", err)
	}
}
