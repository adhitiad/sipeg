package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

type fakeSessionIssuer struct {
	issuedUserID string
	token        string
	expiresAt    time.Time
	err          error
}

func (i *fakeSessionIssuer) IssueSession(_ context.Context, userID string) (string, time.Time, error) {
	i.issuedUserID = userID
	return i.token, i.expiresAt, i.err
}

func logoutMux(revoker auth.SessionRevoker) *http.ServeMux {
	mux := http.NewServeMux()
	RegisterLogoutRoute(mux, testSessionResolver{}, revoker)
	return mux
}

func sessionIssueMux(resolver auth.SessionResolver, policy auth.Policy, issuer auth.SessionIssuer) *http.ServeMux {
	mux := http.NewServeMux()
	RegisterSessionIssueRoute(mux, resolver, policy, issuer)
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

func TestSessionIssueRequiresAuthenticatedSession(t *testing.T) {
	issuer := &fakeSessionIssuer{}
	mux := sessionIssueMux(testSessionResolver{}, auth.StaticPolicy{}, issuer)

	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/auth/session", nil)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without session, got %d", response.Code)
	}
	if issuer.issuedUserID != "" {
		t.Fatal("session must not be issued without an authenticated session")
	}
}

func TestSessionIssueDeniesWithoutPermission(t *testing.T) {
	issuer := &fakeSessionIssuer{}
	// testSessionResolver menghasilkan principal role "operator" yang tidak
	// memiliki grant auth:session:issue pada policy kosong.
	mux := sessionIssueMux(testSessionResolver{}, auth.StaticPolicy{}, issuer)

	body := strings.NewReader(`{"userId":"user-1"}`)
	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/auth/session", body)
	request.Header.Set("Authorization", "Bearer valid-session")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without permission, got %d", response.Code)
	}
	if issuer.issuedUserID != "" {
		t.Fatal("session must not be issued without permission")
	}
}

func TestSessionIssueIssuesSessionForValidTarget(t *testing.T) {
	issuer := &fakeSessionIssuer{token: "issued-token", expiresAt: time.Now().Add(time.Hour)}
	policy := auth.StaticPolicy{"operator": {"auth:session:issue": true}}
	mux := sessionIssueMux(testSessionResolver{}, policy, issuer)

	body := strings.NewReader(`{"userId":"user-1"}`)
	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/auth/session", body)
	request.Header.Set("Authorization", "Bearer valid-session")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d, body = %s", response.Code, response.Body)
	}
	if issuer.issuedUserID != "user-1" {
		t.Fatalf("issued userId = %q, want user-1", issuer.issuedUserID)
	}
	var resp map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &resp); err != nil {
		t.Fatalf("expected JSON body, got %q: %v", response.Body.String(), err)
	}
	if resp["token"] != "issued-token" {
		t.Fatalf("expected issued token in response, got %v", resp["token"])
	}
}

func TestSessionIssueRejectsInvalidBody(t *testing.T) {
	issuer := &fakeSessionIssuer{}
	policy := auth.StaticPolicy{"operator": {"auth:session:issue": true}}
	mux := sessionIssueMux(testSessionResolver{}, policy, issuer)

	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/auth/session", strings.NewReader(`not-json`))
	request.Header.Set("Authorization", "Bearer valid-session")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for invalid body, got %d", response.Code)
	}
	if issuer.issuedUserID != "" {
		t.Fatal("session must not be issued for invalid body")
	}
}

func TestSessionIssueRejectsMissingUserID(t *testing.T) {
	issuer := &fakeSessionIssuer{}
	policy := auth.StaticPolicy{"operator": {"auth:session:issue": true}}
	mux := sessionIssueMux(testSessionResolver{}, policy, issuer)

	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/auth/session", strings.NewReader(`{}`))
	request.Header.Set("Authorization", "Bearer valid-session")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for missing userId, got %d", response.Code)
	}
}

func TestSessionIssuePropagatesIssuerFailureWithoutLeakingDetails(t *testing.T) {
	issuer := &fakeSessionIssuer{err: errors.New("connection string leaked")}
	policy := auth.StaticPolicy{"operator": {"auth:session:issue": true}}
	mux := sessionIssueMux(testSessionResolver{}, policy, issuer)

	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/auth/session", strings.NewReader(`{"userId":"user-1"}`))
	request.Header.Set("Authorization", "Bearer valid-session")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 for issuer failure, got %d", response.Code)
	}
	if body := response.Body.String(); strings.Contains(body, "connection string") {
		t.Fatalf("internal error leaked: %s", body)
	}
}

func TestSessionIssueRouteIsNotRegisteredWithoutIssuer(t *testing.T) {
	mux := http.NewServeMux()
	RegisterSessionIssueRoute(mux, testSessionResolver{}, auth.StaticPolicy{}, nil)

	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/auth/session", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code == http.StatusCreated {
		t.Fatal("session issue route must not be registered without an issuer")
	}
}
