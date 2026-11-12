package observability

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecurityEventMiddlewareLogsAuthenticationFailure(t *testing.T) {
	var buffer bytes.Buffer
	logger := New(&buffer)
	handler := SecurityEventMiddleware(SecurityEventOptions{
		Logger:    logger,
		AuthPaths: []string{"/api/v1/auth"},
	})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))

	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/session?token=rahasia", nil)
	request.Header.Set("Authorization", "Bearer token-rahasia")
	request.RemoteAddr = "10.0.0.5:5555"
	handler.ServeHTTP(httptest.NewRecorder(), request)

	var entry map[string]any
	if err := json.Unmarshal(buffer.Bytes(), &entry); err != nil {
		t.Fatalf("expected JSON entry, got %q: %v", buffer.String(), err)
	}
	if entry["msg"] != "auth_failed" || entry["level"] != "warn" {
		t.Fatalf("unexpected entry: %#v", entry)
	}
	if entry["status"] != float64(http.StatusUnauthorized) {
		t.Fatalf("unexpected status: %#v", entry["status"])
	}
	if entry["auth_surface"] != true {
		t.Fatalf("expected auth surface marker: %#v", entry)
	}
	if entry["client_ip"] != "10.0.0.5" {
		t.Fatalf("unexpected client ip: %#v", entry["client_ip"])
	}
}

func TestSecurityEventMiddlewareLogsAccessDeniedAndRateLimit(t *testing.T) {
	cases := []struct {
		status int
		event  string
	}{
		{status: http.StatusForbidden, event: "access_denied"},
		{status: http.StatusTooManyRequests, event: "rate_limited"},
	}
	for _, testCase := range cases {
		var buffer bytes.Buffer
		logger := New(&buffer)
		handler := SecurityEventMiddleware(SecurityEventOptions{Logger: logger})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(testCase.status)
		}))

		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/employees", nil))

		var entry map[string]any
		if err := json.Unmarshal(buffer.Bytes(), &entry); err != nil {
			t.Fatalf("status %d: expected JSON entry, got %q", testCase.status, buffer.String())
		}
		if entry["msg"] != testCase.event {
			t.Fatalf("status %d: event = %v, want %s", testCase.status, entry["msg"], testCase.event)
		}
		if _, ok := entry["auth_surface"]; ok {
			t.Fatalf("status %d: auth surface marker must only appear on auth paths", testCase.status)
		}
	}
}

func TestSecurityEventMiddlewareIgnoresSuccessfulResponses(t *testing.T) {
	var buffer bytes.Buffer
	logger := New(&buffer)
	handler := SecurityEventMiddleware(SecurityEventOptions{Logger: logger})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/employees", nil))

	if buffer.Len() != 0 {
		t.Fatalf("unexpected security event: %s", buffer.String())
	}
}

func TestSecurityEventDoesNotRecordCredentialsOrQueryValues(t *testing.T) {
	var buffer bytes.Buffer
	logger := New(&buffer)
	handler := SecurityEventMiddleware(SecurityEventOptions{Logger: logger})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/employees?search=rahasia", nil)
	request.Header.Set("Authorization", "Bearer token-rahasia")
	handler.ServeHTTP(httptest.NewRecorder(), request)

	logged := buffer.String()
	if strings.Contains(logged, "token-rahasia") {
		t.Fatalf("credential leaked into log: %s", logged)
	}
	if strings.Contains(logged, "rahasia") {
		t.Fatalf("query value leaked into log: %s", logged)
	}
}
