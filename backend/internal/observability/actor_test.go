package observability

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAccessLogIncludesActorObservedDuringRequest(t *testing.T) {
	var buffer bytes.Buffer
	logger := New(&buffer)
	handler := AccessLogMiddleware(AccessLogOptions{Logger: logger})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sink, ok := ActorSinkFromContext(r.Context())
		if !ok {
			t.Fatal("expected actor sink in request context")
		}
		sink.Observe("user-1", "operator")
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/employees", nil))

	logged := buffer.String()
	if !strings.Contains(logged, `"user_id":"user-1"`) || !strings.Contains(logged, `"role":"operator"`) {
		t.Fatalf("expected actor in access log: %s", logged)
	}
}

func TestSecurityEventIncludesActorForAccessDenied(t *testing.T) {
	var buffer bytes.Buffer
	logger := New(&buffer)
	handler := SecurityEventMiddleware(SecurityEventOptions{Logger: logger})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sink, _ := ActorSinkFromContext(r.Context())
		sink.Observe("user-2", "atasan")
		w.WriteHeader(http.StatusForbidden)
	}))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/employees", nil))

	logged := buffer.String()
	if !strings.Contains(logged, `"msg":"access_denied"`) {
		t.Fatalf("expected access denied event: %s", logged)
	}
	if !strings.Contains(logged, `"user_id":"user-2"`) || !strings.Contains(logged, `"role":"atasan"`) {
		t.Fatalf("expected actor in security event: %s", logged)
	}
}

func TestSecurityEventOmitsActorForFailedAuthentication(t *testing.T) {
	var buffer bytes.Buffer
	logger := New(&buffer)
	handler := SecurityEventMiddleware(SecurityEventOptions{Logger: logger})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/me", nil))

	logged := buffer.String()
	if strings.Contains(logged, "user_id") {
		t.Fatalf("failed authentication must not record an actor: %s", logged)
	}
}
