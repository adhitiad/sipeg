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
	"simpeg/backend/internal/outbox"
)

type fakeOutboxStats struct {
	stats outbox.Stats
	err   error
	calls int
}

func (f *fakeOutboxStats) Stats(_ context.Context) (outbox.Stats, error) {
	f.calls++
	return f.stats, f.err
}

type opsResolver struct{ role string }

func (r opsResolver) ResolvePrincipal(_ context.Context, token string) (auth.Principal, error) {
	if token != "admin-session" && token != "valid-session" {
		return auth.Principal{}, auth.ErrInvalidSession
	}
	return auth.Principal{UserID: "user-1", Email: "person@example.test", RoleName: r.role}, nil
}

func opsMux(reader OutboxStatsReader, grants map[string]map[auth.Permission]bool, role string) *http.ServeMux {
	mux := http.NewServeMux()
	RegisterOpsRoutes(mux, opsResolver{role: role}, auth.StaticPolicy(grants), reader)
	return mux
}

func TestOpsStatusRequiresPermission(t *testing.T) {
	reader := &fakeOutboxStats{stats: outbox.Stats{FailedByKind: map[string]int64{}}}

	denied := opsMux(reader, map[string]map[auth.Permission]bool{
		"operator": {"audit:read:homebase": true},
	}, "operator")
	req := httptest.NewRequest(http.MethodGet, APIPrefix+"/admin/ops/status", nil)
	req.Header.Set("Authorization", "Bearer valid-session")
	resp := httptest.NewRecorder()
	denied.ServeHTTP(resp, req)
	if resp.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.Code)
	}
	if reader.calls != 0 {
		t.Fatalf("reader dipanggil untuk role tanpa izin: %d kali", reader.calls)
	}

	granted := opsMux(reader, map[string]map[auth.Permission]bool{
		"admin": {"ops:read": true},
	}, "admin")
	okReq := httptest.NewRequest(http.MethodGet, APIPrefix+"/admin/ops/status", nil)
	okReq.Header.Set("Authorization", "Bearer admin-session")
	okResp := httptest.NewRecorder()
	granted.ServeHTTP(okResp, okReq)
	if okResp.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", okResp.Code, okResp.Body.String())
	}
}

func TestOpsStatusRequiresAuthentication(t *testing.T) {
	mux := opsMux(&fakeOutboxStats{stats: outbox.Stats{FailedByKind: map[string]int64{}}}, map[string]map[auth.Permission]bool{
		"admin": {"ops:read": true},
	}, "admin")
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, APIPrefix+"/admin/ops/status", nil))
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.Code)
	}
}

func TestOpsStatusReturnsOutboxAggregate(t *testing.T) {
	oldest := time.Date(2026, 11, 14, 1, 0, 0, 0, time.UTC)
	reader := &fakeOutboxStats{stats: outbox.Stats{
		Pending:         3,
		Processing:      1,
		Done:            10,
		Failed:          2,
		OldestPendingAt: &oldest,
		FailedByKind:    map[string]int64{"email.slip-tersedia": 1, "email.gaji-terbayar": 1},
	}}
	mux := opsMux(reader, map[string]map[auth.Permission]bool{
		"admin": {"ops:read": true},
	}, "admin")
	req := httptest.NewRequest(http.MethodGet, APIPrefix+"/admin/ops/status", nil)
	req.Header.Set("Authorization", "Bearer admin-session")
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.Code, resp.Body.String())
	}
	var body OpsStatusDTO
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Outbox.Pending != 3 || body.Outbox.Failed != 2 {
		t.Fatalf("unexpected payload: %+v", body.Outbox)
	}
	if body.Outbox.OldestPendingAt == nil || body.Outbox.OldestPendingAgeMs == nil {
		t.Fatalf("expected oldest pending age: %+v", body.Outbox)
	}
	if body.Outbox.FailedByKind["email.slip-tersedia"] != 1 {
		t.Fatalf("unexpected failedByKind: %+v", body.Outbox.FailedByKind)
	}
}

func TestOpsStatusReportsRepositoryFailure(t *testing.T) {
	reader := &fakeOutboxStats{err: errors.New("database tidak dapat dijangkau")}
	mux := opsMux(reader, map[string]map[auth.Permission]bool{
		"admin": {"ops:read": true},
	}, "admin")
	req := httptest.NewRequest(http.MethodGet, APIPrefix+"/admin/ops/status", nil)
	req.Header.Set("Authorization", "Bearer admin-session")
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	if resp.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", resp.Code)
	}
	if strings.Contains(resp.Body.String(), "database") {
		t.Fatalf("respons membocorkan detail kegagalan: %s", resp.Body.String())
	}
}

func TestOpsRoutesAreNotRegisteredWithoutReader(t *testing.T) {
	mux := http.NewServeMux()
	RegisterOpsRoutes(mux, opsResolver{}, auth.StaticPolicy{}, nil)
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, APIPrefix+"/admin/ops/status", nil))
	if resp.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.Code)
	}
}
