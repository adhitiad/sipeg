package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"simpeg/backend/internal/apierror"
)

func versionedMux() *http.ServeMux {
	mux := NewMux(nil)
	return mux
}

func TestVersionGuardRejectsUnversionedPath(t *testing.T) {
	handler := VersionGuard(versionedMux())

	request := httptest.NewRequest(http.MethodGet, "/api/employees", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body = %s", response.Code, response.Body)
	}
	if !strings.Contains(response.Body.String(), apierror.CodeUnversionedPath) {
		t.Fatalf("unexpected body: %s", response.Body)
	}
	if !strings.Contains(response.Body.String(), APIPrefix) {
		t.Fatalf("error must point to the versioned prefix: %s", response.Body)
	}
}

func TestVersionGuardRejectsUnknownVersion(t *testing.T) {
	handler := VersionGuard(versionedMux())

	request := httptest.NewRequest(http.MethodGet, "/api/v2/employees", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404, body = %s", response.Code, response.Body)
	}
}

func TestVersionGuardAllowsVersionedPath(t *testing.T) {
	handler := VersionGuard(versionedMux())

	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/employees", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	// Path versi v1 yang tidak dikenal harus 404, bukan 400 tanpa versi.
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404, body = %s", response.Code, response.Body)
	}
	if strings.Contains(response.Body.String(), apierror.CodeUnversionedPath) {
		t.Fatalf("versioned path must not report unversioned error: %s", response.Body)
	}
}

func TestVersionGuardLeavesOperationalPathsUntouched(t *testing.T) {
	handler := VersionGuard(versionedMux())

	for _, path := range []string{"/health", "/ready", "/tidak-ada"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)

		if strings.Contains(response.Body.String(), apierror.CodeUnversionedPath) {
			t.Errorf("%s must not be treated as an API path: %s", path, response.Body)
		}
	}
}

func TestVersionMiddlewareAddsVersionHeader(t *testing.T) {
	handler := VersionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health", nil))

	if got := response.Header().Get(VersionHeader); got != Version {
		t.Fatalf("version header = %q, want %q", got, Version)
	}
}

func TestVersionedRouteStillReturnsMethodNotAllowed(t *testing.T) {
	handler := JSONErrorMiddleware(VersionGuard(versionedMux()))

	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/me", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405, body = %s", response.Code, response.Body)
	}
	if !strings.Contains(response.Body.String(), apierror.CodeMethodNotAllowed) {
		t.Fatalf("unexpected body: %s", response.Body)
	}
}

func TestAPIPrefixCarriesVersion(t *testing.T) {
	if APIPrefix != "/api/"+Version {
		t.Fatalf("APIPrefix = %q, does not carry version %q", APIPrefix, Version)
	}
}
