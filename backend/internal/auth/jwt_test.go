package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestJWTVerifierAcceptsValidToken(t *testing.T) {
	verifier := NewJWTVerifier("secret-123")
	tokenString := signJWT(t, "secret-123", BetterAuthClaims{
		UserID:     "user-1",
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
	if claims.UserID != "user-1" || claims.RoleName != "operator" || claims.HomebaseID != "unit-a" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestJWTVerifierRejectsExpiredToken(t *testing.T) {
	verifier := NewJWTVerifier("secret-123")
	tokenString := signJWT(t, "secret-123", BetterAuthClaims{
		UserID: "user-1",
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
		UserID: "user-1",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	})

	_, err := verifier.VerifyToken(tokenString)
	if err == nil {
		t.Fatal("expected error for wrong secret")
	}
}

func TestJWTAuthenticateMiddleware(t *testing.T) {
	verifier := NewJWTVerifier("secret-123")
	tokenString := signJWT(t, "secret-123", BetterAuthClaims{
		UserID:     "user-1",
		Email:      "person@example.test",
		RoleName:   "operator",
		HomebaseID: "unit-a",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	})

	handler := JWTAuthenticate(verifier, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := PrincipalFromContext(r.Context())
		if !ok {
			t.Fatal("principal not set in context")
		}
		if principal.UserID != "user-1" || principal.RoleName != "operator" {
			t.Fatalf("unexpected principal: %+v", principal)
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
	principal := Principal{UserID: "user-1", RoleName: "operator"}
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
	principal := Principal{UserID: "user-1", RoleName: "operator"}
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

func TestCombinedSessionResolverJWTFirst(t *testing.T) {
	verifier := NewJWTVerifier("secret-123")
	tokenString := signJWT(t, "secret-123", BetterAuthClaims{
		UserID:     "user-jwt",
		RoleName:   "admin",
		HomebaseID: "unit-jwt",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	})
	resolver := NewCombinedSessionResolver(verifier, nil)

	principal, err := resolver.ResolvePrincipal(context.TODO(), tokenString)
	if err != nil {
		t.Fatalf("expected valid principal from JWT, got error: %v", err)
	}
	if principal.UserID != "user-jwt" || principal.RoleName != "admin" {
		t.Fatalf("unexpected principal: %+v", principal)
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
