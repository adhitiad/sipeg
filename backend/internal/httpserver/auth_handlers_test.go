package httpserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"simpeg/backend/internal/auth"
)

type fakeSessionRevoker struct {
	revokedToken string
	err          error
}

func (r *fakeSessionRevoker) RevokeSession(_ context.Context, token string, _ auth.Principal) error {
	r.revokedToken = token
	return r.err
}

func logoutMux(revoker auth.SessionRevoker) *http.ServeMux {
	mux := http.NewServeMux()
	RegisterLogoutRoute(mux, testSessionResolver{}, revoker)
	return mux
}

func TestLogoutRevokesCurrentSession(t *testing.T) {
	revoker := &fakeSessionRevoker{}
	mux := logoutMux(revoker)

	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/auth/logout", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204, body = %s", response.Code, response.Body)
	}
	if revoker.revokedToken != "valid-session" {
		t.Fatalf("revoked token = %q, want valid-session", revoker.revokedToken)
	}
}

func TestLogoutRequiresAuthenticatedSession(t *testing.T) {
	revoker := &fakeSessionRevoker{}
	mux := logoutMux(revoker)

	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/auth/logout", nil)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", response.Code)
	}
	if revoker.revokedToken != "" {
		t.Fatal("session must not be revoked without an authenticated session")
	}
}

func TestLogoutIsIdempotentWhenSessionAlreadyGone(t *testing.T) {
	revoker := &fakeSessionRevoker{err: auth.ErrSessionNotFound}
	mux := logoutMux(revoker)

	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/auth/logout", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204, body = %s", response.Code, response.Body)
	}
}

func TestLogoutReportsRevokeFailureWithoutLeakingDetails(t *testing.T) {
	revoker := &fakeSessionRevoker{err: errors.New("db connection string leaked")}
	mux := logoutMux(revoker)

	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/auth/logout", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", response.Code)
	}
	if body := response.Body.String(); body == "" || strings.Contains(body, "connection string") {
		t.Fatalf("internal error leaked: %s", body)
	}
}

func TestLogoutRouteIsNotRegisteredWithoutRevoker(t *testing.T) {
	mux := http.NewServeMux()
	RegisterLogoutRoute(mux, testSessionResolver{}, nil)

	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/auth/logout", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code == http.StatusNoContent {
		t.Fatal("logout route must not be registered without a revoker")
	}
}
