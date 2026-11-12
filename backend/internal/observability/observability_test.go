package observability

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestLoggerWritesStructuredJSONEntries(t *testing.T) {
	var buffer bytes.Buffer
	logger := New(&buffer)
	logger.now = func() time.Time { return time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC) }

	logger.Info("http_request", Fields{"status": 200, "path": "/api/employees"})

	var entry map[string]any
	if err := json.Unmarshal(buffer.Bytes(), &entry); err != nil {
		t.Fatalf("expected JSON entry, got %q: %v", buffer.String(), err)
	}
	if entry["level"] != "info" || entry["msg"] != "http_request" {
		t.Fatalf("unexpected entry: %#v", entry)
	}
	if entry["ts"] != "2026-11-10T09:00:00Z" {
		t.Fatalf("unexpected timestamp: %v", entry["ts"])
	}
	if entry["status"] != float64(200) || entry["path"] != "/api/employees" {
		t.Fatalf("unexpected fields: %#v", entry)
	}
}

func TestLoggerErrorLevelAndNilSafety(t *testing.T) {
	var buffer bytes.Buffer
	logger := New(&buffer)
	logger.Error("boom", Fields{"error": "x"})
	if !strings.Contains(buffer.String(), `"level":"error"`) {
		t.Fatalf("expected error level, got %s", buffer.String())
	}

	// Logger nil tidak boleh panic.
	var nilLogger *Logger
	nilLogger.Info("ignored", nil)
	nilLogger.Error("ignored", nil)
}

func TestRequestIDMiddlewareUsesClientValueWhenSafe(t *testing.T) {
	var seen string
	handler := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = RequestIDFromContext(r.Context())
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	request.Header.Set(RequestIDHeader, "req-abc-123")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if seen != "req-abc-123" {
		t.Fatalf("request id = %q, want req-abc-123", seen)
	}
	if response.Header().Get(RequestIDHeader) != "req-abc-123" {
		t.Fatalf("response header = %q", response.Header().Get(RequestIDHeader))
	}
}

func TestRequestIDMiddlewareReplacesUnsafeClientValue(t *testing.T) {
	for _, unsafe := range []string{"has space", "with\nnewline", strings.Repeat("a", 200), "<script>"} {
		var seen string
		handler := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			seen = RequestIDFromContext(r.Context())
		}))
		request := httptest.NewRequest(http.MethodGet, "/api/me", nil)
		request.Header.Set(RequestIDHeader, unsafe)
		handler.ServeHTTP(httptest.NewRecorder(), request)

		if seen == unsafe {
			t.Fatalf("unsafe request id %q was accepted", unsafe)
		}
		if len(seen) != 36 {
			t.Fatalf("expected generated uuid, got %q", seen)
		}
	}
}

func TestRequestIDMiddlewareGeneratesIDWhenMissing(t *testing.T) {
	var seen string
	handler := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = RequestIDFromContext(r.Context())
	}))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/me", nil))
	if len(seen) != 36 {
		t.Fatalf("expected generated uuid, got %q", seen)
	}
}

func TestAccessLogDoesNotRecordCredentialsOrQueryValues(t *testing.T) {
	var buffer bytes.Buffer
	logger := New(&buffer)
	handler := AccessLogMiddleware(AccessLogOptions{Logger: logger})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sink, _ := ActorSinkFromContext(r.Context())
		sink.Observe("user-1", "operator")
		w.WriteHeader(http.StatusOK)
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/employees?search=rahasia", nil)
	request.Header.Set("Authorization", "Bearer token-rahasia")
	handler.ServeHTTP(httptest.NewRecorder(), request)

	logged := buffer.String()
	if strings.Contains(logged, "token-rahasia") {
		t.Fatalf("credential leaked into log: %s", logged)
	}
	if strings.Contains(logged, "rahasia") {
		t.Fatalf("query value leaked into log: %s", logged)
	}
	if !strings.Contains(logged, `"user_id":"user-1"`) || !strings.Contains(logged, `"role":"operator"`) {
		t.Fatalf("expected principal in log: %s", logged)
	}
	if !strings.Contains(logged, `"status":200`) {
		t.Fatalf("expected status in log: %s", logged)
	}
}
