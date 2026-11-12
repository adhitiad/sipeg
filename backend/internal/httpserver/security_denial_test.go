package httpserver

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/observability"
)

// recordingDenialRecorder menyimpan penolakan yang tercatat dari route nyata.
type recordingDenialRecorder struct {
	denials []auth.Denial
}

func (r *recordingDenialRecorder) RecordDenial(_ context.Context, denial auth.Denial) error {
	r.denials = append(r.denials, denial)
	return nil
}

func TestForbiddenRequestRecordsDenialWithRoutePattern(t *testing.T) {
	recorder := &recordingDenialRecorder{}
	store := &fakeDirectoryStore{}
	// Role tanpa grant membuat penolakan terjadi di gerbang izin, bukan di
	// pemeriksaan cakupan pada use case.
	mux := scopeMux(store, auth.Principal{UserID: pegawaiID, RoleName: "atasan", HomebaseID: unitA},
		map[string]map[auth.Permission]bool{"atasan": {}})

	handler := auth.DenialRecorderMiddleware(recorder)(mux)
	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/employees/"+luarID, nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", response.Code)
	}
	if len(recorder.denials) != 1 {
		t.Fatalf("recorded denials = %d, want 1", len(recorder.denials))
	}
	denial := recorder.denials[0]
	if denial.Actor.UserID != pegawaiID || denial.Actor.HomebaseID != unitA {
		t.Fatalf("unexpected actor: %#v", denial.Actor)
	}
	if !strings.Contains(denial.Route, "{employeeId}") {
		t.Fatalf("route = %q, want route pattern with placeholder", denial.Route)
	}
	if strings.Contains(denial.Route, luarID) {
		t.Fatalf("route must not carry parameter values: %q", denial.Route)
	}
}

func TestAllowedRequestRecordsNoDenial(t *testing.T) {
	recorder := &recordingDenialRecorder{}
	store := &fakeDirectoryStore{}
	mux := scopeMux(store, auth.Principal{UserID: atasanID, RoleName: "admin"},
		map[string]map[auth.Permission]bool{"admin": {"employee:read:any": true}})

	handler := auth.DenialRecorderMiddleware(recorder)(mux)
	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/employees", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
	if len(recorder.denials) != 0 {
		t.Fatalf("granted access must not be recorded: %#v", recorder.denials)
	}
}

func TestAccessDeniedProducesSecurityEventWithActor(t *testing.T) {
	var buffer bytes.Buffer
	logger := observability.New(&buffer)
	store := &fakeDirectoryStore{}
	mux := scopeMux(store, auth.Principal{UserID: atasanID, RoleName: "atasan", HomebaseID: unitA},
		map[string]map[auth.Permission]bool{"atasan": {}})

	handler := observability.RequestIDMiddleware(
		observability.AccessLogMiddleware(observability.AccessLogOptions{Logger: logger})(
			observability.SecurityEventMiddleware(observability.SecurityEventOptions{Logger: logger})(mux)))

	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/employees", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", response.Code)
	}
	logged := buffer.String()
	if !strings.Contains(logged, `"msg":"access_denied"`) {
		t.Fatalf("expected access denied event: %s", logged)
	}
	if !strings.Contains(logged, `"user_id":"`+atasanID+`"`) {
		t.Fatalf("expected actor in security event: %s", logged)
	}
}
