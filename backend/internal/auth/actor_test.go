package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"simpeg/backend/internal/observability"
)

func TestAuthenticateObservesPrincipalIntoActorSink(t *testing.T) {
	sink := &observability.ActorSink{}
	resolver := fakeResolver{principal: Principal{UserID: "user-1", RoleName: "operator"}}
	handler := Authenticate(resolver)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/employees", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	request = request.WithContext(observability.WithActorSink(request.Context(), sink))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
	userID, role, ok := sink.Load()
	if !ok {
		t.Fatal("expected principal to be observed")
	}
	if userID != "user-1" || role != "operator" {
		t.Fatalf("unexpected actor: %q / %q", userID, role)
	}
}

func TestAuthenticateLeavesActorSinkEmptyOnInvalidSession(t *testing.T) {
	sink := &observability.ActorSink{}
	handler := Authenticate(fakeResolver{err: ErrInvalidSession})(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler must not run without a valid session")
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/employees", nil)
	request.Header.Set("Authorization", "Bearer invalid")
	request = request.WithContext(observability.WithActorSink(request.Context(), sink))
	handler.ServeHTTP(httptest.NewRecorder(), request)

	if _, _, ok := sink.Load(); ok {
		t.Fatal("failed authentication must not record an actor")
	}
}
