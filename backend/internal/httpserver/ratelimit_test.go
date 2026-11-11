package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRateLimiterBlocksAfterLimitExceeded(t *testing.T) {
	limiter := NewRateLimiter(RateLimitOptions{Limit: 3, Window: time.Minute})
	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for attempt := 1; attempt <= 3; attempt++ {
		request := httptest.NewRequest(http.MethodGet, "/api/v1/employees", nil)
		request.RemoteAddr = "10.0.0.1:1234"
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)

		if response.Code != http.StatusOK {
			t.Fatalf("attempt %d: status = %d, want 200", attempt, response.Code)
		}
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/employees", nil)
	request.RemoteAddr = "10.0.0.1:1234"
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", response.Code)
	}
	if response.Header().Get("Retry-After") == "" {
		t.Fatal("expected Retry-After header")
	}
	if !strings.Contains(response.Body.String(), `"code":"rate_limited"`) {
		t.Fatalf("unexpected body: %s", response.Body)
	}
}

func TestRateLimiterCountsPerClient(t *testing.T) {
	limiter := NewRateLimiter(RateLimitOptions{Limit: 1, Window: time.Minute})
	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	first := httptest.NewRequest(http.MethodGet, "/api/v1/employees", nil)
	first.RemoteAddr = "10.0.0.1:1234"
	firstResponse := httptest.NewRecorder()
	handler.ServeHTTP(firstResponse, first)

	second := httptest.NewRequest(http.MethodGet, "/api/v1/employees", nil)
	second.RemoteAddr = "10.0.0.2:1234"
	secondResponse := httptest.NewRecorder()
	handler.ServeHTTP(secondResponse, second)

	if firstResponse.Code != http.StatusOK || secondResponse.Code != http.StatusOK {
		t.Fatalf("clients must have independent budgets, got %d and %d", firstResponse.Code, secondResponse.Code)
	}
}

func TestRateLimiterWindowResets(t *testing.T) {
	now := time.Now()
	limiter := NewRateLimiter(RateLimitOptions{Limit: 1, Window: time.Minute, Now: func() time.Time { return now }})
	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	send := func() int {
		request := httptest.NewRequest(http.MethodGet, "/api/v1/employees", nil)
		request.RemoteAddr = "10.0.0.1:1234"
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response.Code
	}

	if send() != http.StatusOK {
		t.Fatal("first request must be allowed")
	}
	if send() != http.StatusTooManyRequests {
		t.Fatal("second request within window must be blocked")
	}
	now = now.Add(2 * time.Minute)
	if send() != http.StatusOK {
		t.Fatal("request after window must be allowed again")
	}
}

func TestRateLimiterAppliesStricterLimitToSensitivePaths(t *testing.T) {
	limiter := NewRateLimiter(RateLimitOptions{
		Limit:        100,
		Window:       time.Minute,
		StrictPaths:  []string{APIPrefix + "/auth"},
		StrictLimit:  1,
		StrictWindow: time.Minute,
	})
	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	send := func(path string) int {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		request.RemoteAddr = "10.0.0.1:1234"
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response.Code
	}

	if send(APIPrefix+"/auth/logout") != http.StatusOK {
		t.Fatal("first request to sensitive path must be allowed")
	}
	if send(APIPrefix+"/auth/logout") != http.StatusTooManyRequests {
		t.Fatal("second request to sensitive path must be blocked")
	}
	if send(APIPrefix+"/employees") != http.StatusOK {
		t.Fatal("regular path must keep the general limit")
	}
}

func TestStatementTimeoutMiddlewareSetsDeadline(t *testing.T) {
	handler := StatementTimeoutMiddleware(50 * time.Millisecond)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deadline, ok := r.Context().Deadline()
		if !ok {
			t.Fatal("expected deadline on request context")
		}
		if time.Until(deadline) <= 0 {
			t.Fatal("deadline must be in the future")
		}
		w.WriteHeader(http.StatusOK)
	}))

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/employees", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
}

func TestStatementTimeoutMiddlewareSkippedWhenNotPositive(t *testing.T) {
	handler := StatementTimeoutMiddleware(0)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := r.Context().Deadline(); ok {
			t.Fatal("no deadline expected when timeout is not positive")
		}
		w.WriteHeader(http.StatusOK)
	}))

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/employees", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
}
