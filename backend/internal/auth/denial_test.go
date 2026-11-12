package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeDenialRecorder menyimpan penolakan yang diteruskan ke lapisan auth.
type fakeDenialRecorder struct {
	denials []Denial
	err     error
}

func (r *fakeDenialRecorder) RecordDenial(_ context.Context, denial Denial) error {
	r.denials = append(r.denials, denial)
	return r.err
}

func TestRequirePermissionRecordsDenial(t *testing.T) {
	recorder := &fakeDenialRecorder{}
	policy := StaticPolicy{"pegawai": {"employee:read:self": true}}
	handler := DenialRecorderMiddleware(recorder)(RequirePermission(policy, "employee:read:any",
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			t.Fatal("handler must not run when permission is missing")
		})))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/employees", nil)
	request.RemoteAddr = "10.0.0.5:5555"
	request = request.WithContext(WithPrincipal(request.Context(), Principal{
		UserID: "user-1", RoleName: "pegawai", HomebaseID: "unit-a",
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", response.Code)
	}
	if len(recorder.denials) != 1 {
		t.Fatalf("recorded denials = %d, want 1", len(recorder.denials))
	}
	denial := recorder.denials[0]
	if denial.Actor.UserID != "user-1" || denial.Actor.HomebaseID != "unit-a" {
		t.Fatalf("unexpected actor: %#v", denial.Actor)
	}
	if len(denial.Permissions) != 1 || denial.Permissions[0] != "employee:read:any" {
		t.Fatalf("unexpected permissions: %#v", denial.Permissions)
	}
	if denial.IP != "10.0.0.5" {
		t.Fatalf("ip = %q, want 10.0.0.5", denial.IP)
	}
	if !strings.Contains(denial.Route, "/api/v1/employees") {
		t.Fatalf("route = %q", denial.Route)
	}
}

func TestRequireAnyPermissionRecordsAllRequestedPermissions(t *testing.T) {
	recorder := &fakeDenialRecorder{}
	policy := StaticPolicy{"pegawai": {"employee:read:self": true}}
	permissions := []Permission{"employee:read:any", "employee:read:homebase"}
	handler := DenialRecorderMiddleware(recorder)(RequireAnyPermission(policy, permissions,
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			t.Fatal("handler must not run when no permission is granted")
		})))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/employees", nil)
	request = request.WithContext(WithPrincipal(request.Context(), Principal{UserID: "user-1", RoleName: "pegawai"}))
	handler.ServeHTTP(httptest.NewRecorder(), request)

	if len(recorder.denials) != 1 {
		t.Fatalf("recorded denials = %d, want 1", len(recorder.denials))
	}
	if len(recorder.denials[0].Permissions) != 2 {
		t.Fatalf("unexpected permissions: %#v", recorder.denials[0].Permissions)
	}
}

func TestGrantedPermissionIsNotRecorded(t *testing.T) {
	recorder := &fakeDenialRecorder{}
	policy := StaticPolicy{"admin": {"employee:read:any": true}}
	handler := DenialRecorderMiddleware(recorder)(RequirePermission(policy, "employee:read:any",
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/employees", nil)
	request = request.WithContext(WithPrincipal(request.Context(), Principal{UserID: "user-1", RoleName: "admin"}))
	handler.ServeHTTP(httptest.NewRecorder(), request)

	if len(recorder.denials) != 0 {
		t.Fatalf("granted access must not be recorded: %#v", recorder.denials)
	}
}

func TestDenialWithoutRecorderStillReturnsForbidden(t *testing.T) {
	policy := StaticPolicy{}
	handler := RequirePermission(policy, "employee:read:any", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/employees", nil)
	request = request.WithContext(WithPrincipal(request.Context(), Principal{UserID: "user-1", RoleName: "pegawai"}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", response.Code)
	}
}

func TestDenialRecorderMiddlewareIgnoresNilRecorder(t *testing.T) {
	handler := DenialRecorderMiddleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/employees", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
}

func TestDenialNoteCarriesPermissionAndRouteOnly(t *testing.T) {
	note := denialNote(Denial{
		Permissions: []Permission{"employee:read:any", "employee:read:homebase"},
		Route:       "GET /api/v1/employees/{employeeId}",
	})

	if !strings.Contains(note, "employee:read:any") || !strings.Contains(note, "employee:read:homebase") {
		t.Fatalf("note misses permission: %s", note)
	}
	if !strings.Contains(note, "GET /api/v1/employees/{employeeId}") {
		t.Fatalf("note misses route: %s", note)
	}
}

func TestGormDenialRecorderWritesAuditEntry(t *testing.T) {
	db, mock := newIssuerDB(t)
	mock.ExpectBegin()
	expectInsertAuditLog(mock)
	mock.ExpectCommit()

	recorder := GormDenialRecorder{DB: db}
	err := recorder.RecordDenial(context.Background(), Denial{
		Actor:       Principal{UserID: "user-1", RoleName: "pegawai", HomebaseID: "unit-a"},
		Permissions: []Permission{"employee:read:any"},
		Method:      http.MethodGet,
		Route:       "GET /api/v1/employees",
		IP:          "10.0.0.5",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestGormDenialRecorderRejectsMissingDatabase(t *testing.T) {
	recorder := GormDenialRecorder{}
	if err := recorder.RecordDenial(context.Background(), Denial{}); err == nil {
		t.Fatal("expected error when database handle is missing")
	}
}
