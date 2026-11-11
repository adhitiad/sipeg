package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"simpeg/backend/internal/auth"
)

type testSessionResolver struct{}

func (testSessionResolver) ResolvePrincipal(_ context.Context, token string) (auth.Principal, error) {
	if token != "valid-session" {
		return auth.Principal{}, auth.ErrInvalidSession
	}
	return auth.Principal{UserID: "user-1", Email: "person@example.test", RoleName: "operator"}, nil
}

func TestHealthEndpointReturnsOk(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	res := httptest.NewRecorder()

	HandleHealth(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected JSON body, got %q: %v", res.Body.String(), err)
	}
	if body["status"] != "ok" {
		t.Fatalf("unexpected body: %s", res.Body.String())
	}
}

func TestSystemEndpointsRejectNonGetMethods(t *testing.T) {
	mux := NewMux(nil)
	for _, path := range []string{"/health", "/ready"} {
		req := httptest.NewRequest(http.MethodPost, path, nil)
		res := httptest.NewRecorder()
		mux.ServeHTTP(res, req)
		if res.Code != http.StatusMethodNotAllowed {
			t.Errorf("POST %s: expected 405, got %d", path, res.Code)
		}
	}
}

func TestMeRequiresSessionAndReturnsSafeProfile(t *testing.T) {
	mux := NewMux(testSessionResolver{})

	unauthorized := httptest.NewRecorder()
	mux.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, APIPrefix+"/me", nil))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthenticated request to return 401, got %d", unauthorized.Code)
	}
	req := httptest.NewRequest(http.MethodGet, APIPrefix+"/me", nil)
	req.Header.Set("Authorization", "Bearer valid-session")
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected authenticated request to return 200, got %d", res.Code)
	}
	if !strings.Contains(res.Body.String(), `"userId":"user-1"`) || !strings.Contains(res.Body.String(), `"roleName":"operator"`) {
		t.Fatalf("expected safe profile response, got %s", res.Body.String())
	}

	wrongMethod := httptest.NewRequest(http.MethodPost, APIPrefix+"/me", nil)
	wrongMethod.Header.Set("Authorization", "Bearer valid-session")
	wrongMethodRes := httptest.NewRecorder()
	mux.ServeHTTP(wrongMethodRes, wrongMethod)
	if wrongMethodRes.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected POST to return 405, got %d", wrongMethodRes.Code)
	}
}
