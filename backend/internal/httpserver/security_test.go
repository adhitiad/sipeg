package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func corsHandler(options CORSOptions) http.Handler {
	return CORSMiddleware(options)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
}

func TestCORSAllowsTrustedOrigin(t *testing.T) {
	handler := corsHandler(CORSOptions{
		TrustedOrigins: []string{"https://simpeg.unisubang.ac.id"},
		AllowedMethods: DefaultCORSMethods(),
		AllowedHeaders: DefaultCORSHeaders(),
	})

	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	request.Header.Set("Origin", "https://simpeg.unisubang.ac.id")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "https://simpeg.unisubang.ac.id" {
		t.Fatalf("allow origin = %q", got)
	}
	if response.Header().Get("Vary") != "Origin" {
		t.Fatal("response must vary on Origin")
	}
}

func TestCORSRejectsUntrustedOrigin(t *testing.T) {
	handler := corsHandler(CORSOptions{
		TrustedOrigins: []string{"https://simpeg.unisubang.ac.id"},
		AllowedMethods: DefaultCORSMethods(),
		AllowedHeaders: DefaultCORSHeaders(),
	})

	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	request.Header.Set("Origin", "https://penyerang.example")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("untrusted origin must not be allowed, got %q", got)
	}
}

func TestCORSPreflightFromUntrustedOriginIsRejected(t *testing.T) {
	handler := corsHandler(CORSOptions{
		TrustedOrigins: []string{"https://simpeg.unisubang.ac.id"},
		AllowedMethods: DefaultCORSMethods(),
		AllowedHeaders: DefaultCORSHeaders(),
	})

	request := httptest.NewRequest(http.MethodOptions, "/api/v1/employees", nil)
	request.Header.Set("Origin", "https://penyerang.example")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", response.Code)
	}
}

func TestCORSPreflightFromTrustedOriginIsAllowed(t *testing.T) {
	handler := corsHandler(CORSOptions{
		TrustedOrigins: []string{"https://simpeg.unisubang.ac.id"},
		AllowedMethods: DefaultCORSMethods(),
		AllowedHeaders: DefaultCORSHeaders(),
	})

	request := httptest.NewRequest(http.MethodOptions, "/api/v1/employees", nil)
	request.Header.Set("Origin", "https://simpeg.unisubang.ac.id")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", response.Code)
	}
	if response.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Fatal("expected allowed methods header")
	}
}

func TestCORSWithoutTrustedOriginsBlocksCrossOrigin(t *testing.T) {
	handler := corsHandler(CORSOptions{
		AllowedMethods: DefaultCORSMethods(),
		AllowedHeaders: DefaultCORSHeaders(),
	})

	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	request.Header.Set("Origin", "https://penyerang.example")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("no trusted origin means no cross-origin access, got %q", got)
	}
}

func TestSecurityHeadersArePresent(t *testing.T) {
	handler := SecurityHeadersMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health", nil))

	for header, want := range map[string]string{
		"X-Content-Type-Options":     "nosniff",
		"X-Frame-Options":            "DENY",
		"Referrer-Policy":            "no-referrer",
		"Cross-Origin-Opener-Policy": "same-origin",
	} {
		if got := response.Header().Get(header); got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}
	if !strings.Contains(response.Header().Get("Permissions-Policy"), "camera=()") {
		t.Error("permissions policy must disable camera")
	}
}
