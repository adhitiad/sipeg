package httpserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"simpeg/backend/internal/apierror"
)

type stubChecker struct {
	name  string
	err   error
	calls int
}

func (c *stubChecker) Name() string { return c.name }

func (c *stubChecker) Check(context.Context) error {
	c.calls++
	return c.err
}

func TestReadinessReportsReadyWhenAllDependenciesHealthy(t *testing.T) {
	database := &stubChecker{name: "postgres"}
	queue := &stubChecker{name: "queue"}
	handler := NewReadinessHandler([]DependencyChecker{database, queue}, time.Second)

	request := httptest.NewRequest(http.MethodGet, "/ready", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
	if !strings.Contains(response.Body.String(), `"status":"ready"`) {
		t.Fatalf("unexpected body: %s", response.Body)
	}
	if !strings.Contains(response.Body.String(), `"postgres":{"status":"ok"}`) {
		t.Fatalf("expected per-dependency status: %s", response.Body)
	}
	if !strings.Contains(response.Body.String(), `"checks":{`) {
		t.Fatalf("expected checks object: %s", response.Body)
	}
}

func TestReadinessReportsNotReadyWhenDependencyUnavailable(t *testing.T) {
	failing := &stubChecker{name: "postgres", err: errors.New("dial tcp 10.0.0.5:5432: connection refused")}
	handler := NewReadinessHandler([]DependencyChecker{failing}, time.Second)

	request := httptest.NewRequest(http.MethodGet, "/ready", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", response.Code)
	}
	body := response.Body.String()
	if !strings.Contains(body, apierror.CodeNotReady) {
		t.Fatalf("expected not_ready code: %s", body)
	}
	// Detail internal dependency tidak boleh bocor ke klien.
	if strings.Contains(body, "10.0.0.5") || strings.Contains(body, "connection refused") {
		t.Fatalf("internal error leaked: %s", body)
	}
}

func TestReadinessWithoutCheckersIsReady(t *testing.T) {
	handler := NewReadinessHandler(nil, time.Second)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/ready", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
}

func TestLivenessStaysUpWhenDependencyIsDown(t *testing.T) {
	// /health tidak bergantung pada dependency eksternal.
	response := httptest.NewRecorder()
	HandleHealth(response, httptest.NewRequest(http.MethodGet, "/health", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
}

func TestJSONErrorMiddlewareConvertsPlainNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("GET /known", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	handler := JSONErrorMiddleware(mux)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/unknown", nil))

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", response.Code)
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("content type = %q, want application/json", contentType)
	}
	if !strings.Contains(response.Body.String(), `"code":"`+apierror.CodeNotFound+`"`) {
		t.Fatalf("unexpected body: %s", response.Body)
	}
	if strings.Contains(response.Body.String(), "404 page not found") {
		t.Fatalf("plain text body was not replaced: %s", response.Body)
	}
}

func TestJSONErrorMiddlewareConvertsMethodNotAllowed(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("GET /known", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	handler := JSONErrorMiddleware(mux)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/known", nil))

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", response.Code)
	}
	if !strings.Contains(response.Body.String(), `"code":"`+apierror.CodeMethodNotAllowed+`"`) {
		t.Fatalf("unexpected body: %s", response.Body)
	}
}

func TestErrorResponsesUseUniformShape(t *testing.T) {
	mux := NewMux(nil)
	handler := JSONErrorMiddleware(mux)

	cases := []struct {
		method string
		path   string
		status int
		code   string
	}{
		{http.MethodGet, "/missing", http.StatusNotFound, apierror.CodeNotFound},
		{http.MethodPost, "/health", http.StatusMethodNotAllowed, apierror.CodeMethodNotAllowed},
		{http.MethodGet, APIPrefix + "/me", http.StatusUnauthorized, apierror.CodeUnauthorized},
	}
	for _, test := range cases {
		request := httptest.NewRequest(test.method, test.path, nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)

		if response.Code != test.status {
			t.Errorf("%s %s: status = %d, want %d", test.method, test.path, response.Code, test.status)
		}
		body := response.Body.String()
		if !strings.Contains(body, `"error":{`) || !strings.Contains(body, `"code":"`+test.code+`"`) {
			t.Errorf("%s %s: unexpected error shape: %s", test.method, test.path, body)
		}
	}
}
