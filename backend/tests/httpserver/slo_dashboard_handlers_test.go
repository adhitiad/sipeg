package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	. "simpeg/backend/internal/httpserver"
	"strings"
	"testing"
	"time"

	"simpeg/backend/internal/auth"
	sloapp "simpeg/backend/internal/slo/app"
)

// fakeSLODashboard menggantikan use case dashboard dengan hasil yang
// disiapkan pengujian.
type fakeSLODashboard struct {
	snapshot sloapp.Snapshot
	err      error
	lookback time.Duration
	calls    int
}

func (f *fakeSLODashboard) Snapshot(_ context.Context, lookback time.Duration) (sloapp.Snapshot, error) {
	f.calls++
	f.lookback = lookback
	return f.snapshot, f.err
}

// sloAdminResolver menghasilkan principal role tertentu untuk pengujian.
type sloAdminResolver struct{ role string }

func (r sloAdminResolver) ResolvePrincipal(_ context.Context, token string) (auth.Principal, error) {
	if token != "admin-session" && token != "valid-session" {
		return auth.Principal{}, auth.ErrInvalidSession
	}
	return auth.Principal{UserID: "user-1", Email: "person@example.test", RoleName: r.role}, nil
}

func sloAdminMux(dashboard SLODashboardReader, grants map[string]map[auth.Permission]bool, role string) *http.ServeMux {
	mux := http.NewServeMux()
	RegisterSLOAdminRoutes(mux, sloAdminResolver{role: role}, auth.StaticPolicy(grants), dashboard)
	return mux
}

func TestSLOMetricsRequiresPermission(t *testing.T) {
	dashboard := &fakeSLODashboard{}

	denied := sloAdminMux(dashboard, map[string]map[auth.Permission]bool{
		"operator": {"audit:read:homebase": true},
	}, "operator")
	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/admin/slo/metrics", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	denied.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", response.Code)
	}
	if dashboard.calls != 0 {
		t.Fatalf("dashboard dipanggil untuk role tanpa izin: %d kali", dashboard.calls)
	}

	granted := sloAdminMux(dashboard, map[string]map[auth.Permission]bool{
		"admin": {"slo:read": true},
	}, "admin")
	okRequest := httptest.NewRequest(http.MethodGet, APIPrefix+"/admin/slo/metrics", nil)
	okRequest.Header.Set("Authorization", "Bearer admin-session")
	okResponse := httptest.NewRecorder()
	granted.ServeHTTP(okResponse, okRequest)
	if okResponse.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", okResponse.Code, okResponse.Body.String())
	}
}

func TestSLOMetricsRequiresAuthentication(t *testing.T) {
	mux := sloAdminMux(&fakeSLODashboard{}, map[string]map[auth.Permission]bool{
		"admin": {"slo:read": true},
	}, "admin")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, APIPrefix+"/admin/slo/metrics", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", response.Code)
	}
}

func TestSLOMetricsReturnsAggregate(t *testing.T) {
	now := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	dashboard := &fakeSLODashboard{snapshot: sloapp.Snapshot{
		WindowStart: now.Add(-24 * time.Hour),
		WindowEnd:   now,
		Requests:    1250,
		Errors:      25,
		ErrorRate:   0.02,
		AvgMs:       42.5,
		P95Ms:       800,
		P99Ms:       1200,
		MaxMs:       1500,
		Routes: []sloapp.RouteSnapshot{{
			Route:     "/api/v1/employees",
			Method:    http.MethodGet,
			Requests:  1250,
			Errors:    25,
			ErrorRate: 0.02,
			AvgMs:     42.5,
			P95Ms:     800,
			P99Ms:     1200,
			MaxMs:     1500,
		}},
	}}
	mux := sloAdminMux(dashboard, map[string]map[auth.Permission]bool{
		"admin": {"slo:read": true},
	}, "admin")

	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/admin/slo/metrics?hours=72", nil)
	request.Header.Set("Authorization", "Bearer admin-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if dashboard.lookback != 72*time.Hour {
		t.Fatalf("lookback = %s, ingin 72h", dashboard.lookback)
	}
	var body SLOMetricsDTO
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Requests != 1250 || body.P95Ms != 800 || len(body.Routes) != 1 {
		t.Fatalf("unexpected payload: %+v", body)
	}
	if body.Routes[0].Route != "/api/v1/employees" {
		t.Fatalf("unexpected route: %+v", body.Routes[0])
	}
}

func TestSLOMetricsRejectsInvalidHours(t *testing.T) {
	dashboard := &fakeSLODashboard{}
	mux := sloAdminMux(dashboard, map[string]map[auth.Permission]bool{
		"admin": {"slo:read": true},
	}, "admin")

	for _, raw := range []string{"abc", "0", "-1", "169", "9999"} {
		request := httptest.NewRequest(http.MethodGet, APIPrefix+"/admin/slo/metrics?hours="+raw, nil)
		request.Header.Set("Authorization", "Bearer admin-session")
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest {
			t.Errorf("hours=%s: status = %d, want 400", raw, response.Code)
		}
	}
	if dashboard.calls != 0 {
		t.Fatalf("dashboard dipanggil untuk parameter tidak valid: %d kali", dashboard.calls)
	}
}

func TestSLOMetricsDefaultsTo24Hours(t *testing.T) {
	dashboard := &fakeSLODashboard{}
	mux := sloAdminMux(dashboard, map[string]map[auth.Permission]bool{
		"admin": {"slo:read": true},
	}, "admin")
	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/admin/slo/metrics", nil)
	request.Header.Set("Authorization", "Bearer admin-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if dashboard.lookback != 24*time.Hour {
		t.Fatalf("lookback = %s, ingin 24h", dashboard.lookback)
	}
}

func TestSLOMetricsReportsRepositoryFailure(t *testing.T) {
	dashboard := &fakeSLODashboard{err: errors.New("database tidak dapat dijangkau")}
	mux := sloAdminMux(dashboard, map[string]map[auth.Permission]bool{
		"admin": {"slo:read": true},
	}, "admin")
	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/admin/slo/metrics", nil)
	request.Header.Set("Authorization", "Bearer admin-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", response.Code)
	}
	if strings.Contains(response.Body.String(), "database") {
		t.Fatalf("respons membocorkan detail kegagalan: %s", response.Body.String())
	}
}

func TestSLOAdminRoutesAreNotRegisteredWithoutDashboard(t *testing.T) {
	mux := http.NewServeMux()
	RegisterSLOAdminRoutes(mux, testSessionResolver{}, auth.StaticPolicy{}, nil)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, APIPrefix+"/admin/slo/metrics", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", response.Code)
	}
}
