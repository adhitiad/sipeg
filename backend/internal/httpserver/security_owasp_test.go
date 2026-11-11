package httpserver

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"simpeg/backend/internal/apierror"
	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/models"
)

// errDatabaseDown meniru kegagalan dependency internal.
var errDatabaseDown = errors.New("dial tcp 10.0.0.5:5432: connection refused")

// --- A01 Broken Access Control ---

func TestIDORDeniesReadingAnotherEmployeeRecord(t *testing.T) {
	store := &fakeDirectoryStore{
		scopeIDs: []string{unitA},
		byID: map[string]models.User{
			luarID: {
				BaseModel: models.BaseModel{IDModel: models.IDModel{ID: luarID}},
				Nama:      "Milik Orang Lain", HomebaseId: pointer(unitB), Aktif: true,
			},
		},
	}
	mux := scopeMux(store, auth.Principal{UserID: pegawaiID, RoleName: "pegawai", HomebaseID: unitA},
		map[string]map[auth.Permission]bool{"pegawai": {"employee:read:self": true}})

	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/employees/"+luarID, nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", response.Code)
	}
}

func TestUnauthenticatedRequestGetsNoPrincipal(t *testing.T) {
	mux := scopeMux(&fakeDirectoryStore{}, auth.Principal{},
		map[string]map[auth.Permission]bool{"admin": {"employee:read:any": true}})

	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/employees", nil)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", response.Code)
	}
	if strings.Contains(response.Body.String(), "items") {
		t.Fatalf("unauthenticated response leaked data: %s", response.Body)
	}
}

func TestRequestHeaderCannotElevateRole(t *testing.T) {
	store := &fakeDirectoryStore{scopeIDs: []string{unitA}}
	mux := scopeMux(store, auth.Principal{UserID: pegawaiID, RoleName: "pegawai", HomebaseID: unitA},
		map[string]map[auth.Permission]bool{
			"pegawai": {"employee:read:self": true},
			"admin":   {"employee:read:any": true},
		})

	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/employees", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	request.Header.Set("X-Role", "admin")
	request.Header.Set("X-User-Id", "someone-else")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
	if len(store.listOptions.ScopeHomebaseIDs) != 0 {
		t.Fatal("request headers must not elevate the resolved scope")
	}
}

// --- A05 Injection ---

func TestSearchInputIsRejectedWhenNotBounded(t *testing.T) {
	store := &fakeDirectoryStore{}
	mux := scopeMux(store, auth.Principal{UserID: atasanID, RoleName: "admin"},
		map[string]map[auth.Permission]bool{"admin": {"employee:read:any": true}})

	long := strings.Repeat("a", 101)
	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/employees?search="+long, nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", response.Code)
	}
}

func TestSortInputOutsideAllowlistIsRejected(t *testing.T) {
	store := &fakeDirectoryStore{}
	mux := scopeMux(store, auth.Principal{UserID: atasanID, RoleName: "admin"},
		map[string]map[auth.Permission]bool{"admin": {"employee:read:any": true}})

	for _, value := range []string{"roleId", "nama; DROP TABLE \"user\"", "1=1"} {
		request := httptest.NewRequest(http.MethodGet, APIPrefix+"/employees?sort="+url.QueryEscape(value), nil)
		request.Header.Set("Authorization", "Bearer valid-session")
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)

		if response.Code != http.StatusBadRequest {
			t.Errorf("sort=%q: status = %d, want 400", value, response.Code)
		}
	}
}

func TestMalformedJSONBodyIsRejected(t *testing.T) {
	store := &fakeDirectoryStore{}
	mux := scopeMux(store, auth.Principal{UserID: atasanID, RoleName: "admin"},
		map[string]map[auth.Permission]bool{"admin": {"employee:create": true}})

	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/employees", strings.NewReader(`{"nama": `))
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", response.Code)
	}
}

func TestUnknownJSONFieldIsRejected(t *testing.T) {
	store := &fakeDirectoryStore{}
	mux := scopeMux(store, auth.Principal{UserID: atasanID, RoleName: "admin"},
		map[string]map[auth.Permission]bool{"admin": {"employee:create": true}})

	body := `{"nama":"Pegawai","email":"a@example.test","roleId":"11111111-1111-4111-8111-111111111111"}`
	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/employees", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body = %s", response.Code, response.Body)
	}
}

// --- A07 Authentication Failures ---

func TestInvalidSessionTokenIsRejected(t *testing.T) {
	// Resolver mengembalikan ErrInvalidSession untuk token yang tidak dikenal.
	mux := NewMux(testSessionResolver{})
	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/me", nil)
	request.Header.Set("Authorization", "Bearer invalid-token")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", response.Code)
	}
	if !strings.Contains(response.Body.String(), apierror.CodeUnauthorized) {
		t.Fatalf("unexpected body: %s", response.Body)
	}
}

func TestNonBearerAuthorizationSchemeIsRejected(t *testing.T) {
	mux := NewMux(nil)
	for _, header := range []string{"Basic dXNlcjpwYXNz", "valid-session", "Bearer a b"} {
		request := httptest.NewRequest(http.MethodGet, APIPrefix+"/me", nil)
		request.Header.Set("Authorization", header)
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)

		if response.Code != http.StatusUnauthorized {
			t.Errorf("authorization %q: status = %d, want 401", header, response.Code)
		}
	}
}

// --- A10 Mishandling of Exceptional Conditions ---

func TestDependencyFailureIsReportedAsServiceUnavailable(t *testing.T) {
	store := &fakeDirectoryStore{err: errDatabaseDown}
	mux := scopeMux(store, auth.Principal{UserID: atasanID, RoleName: "admin"},
		map[string]map[auth.Permission]bool{"admin": {"employee:read:any": true}})

	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/employees", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", response.Code)
	}
	if strings.Contains(response.Body.String(), "connection string") {
		t.Fatalf("internal detail leaked: %s", response.Body)
	}
}

func TestOversizedRequestBodyIsRejected(t *testing.T) {
	store := &fakeDirectoryStore{}
	mux := scopeMux(store, auth.Principal{UserID: atasanID, RoleName: "admin"},
		map[string]map[auth.Permission]bool{"admin": {"employee:create": true}})

	body := `{"nama":"` + strings.Repeat("x", 2<<20) + `"}`
	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/employees", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code == http.StatusCreated {
		t.Fatal("oversized body must not be accepted")
	}
}

func TestInvalidIdentifierIsRejectedBeforeStoreLookup(t *testing.T) {
	store := &fakeDirectoryStore{}
	mux := scopeMux(store, auth.Principal{UserID: atasanID, RoleName: "admin"},
		map[string]map[auth.Permission]bool{"admin": {"employee:read:any": true}})

	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/employees/bukan-uuid", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", response.Code)
	}
}
