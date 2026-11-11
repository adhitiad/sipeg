package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"simpeg/backend/internal/attendance"
	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/models"
)

type fakeAttendanceReader struct {
	userID      string
	homebaseIDs []string
	records     []models.Presensi
	counts      []attendance.StatusCount
}

func (r *fakeAttendanceReader) ListMyRecords(_ context.Context, userID string, _ attendance.RecordFilter) ([]models.Presensi, int64, error) {
	r.userID = userID
	return r.records, int64(len(r.records)), nil
}

func (r *fakeAttendanceReader) ListUnitSummary(_ context.Context, homebaseIDs []string, _ attendance.RecordFilter) ([]attendance.StatusCount, error) {
	r.homebaseIDs = homebaseIDs
	return r.counts, nil
}

type fakeAttendanceScope struct{}

func (fakeAttendanceScope) ListUnitScopeIDs(context.Context, string) ([]string, error) {
	return []string{"unit-a", "unit-child"}, nil
}

func TestMyAttendanceRecordsReturnsOnlySafeDailyFields(t *testing.T) {
	location := "private location"
	reader := &fakeAttendanceReader{records: []models.Presensi{{
		BaseModel: models.BaseModel{IDModel: models.IDModel{ID: "record-1"}},
		Tanggal:   time.Date(2026, 11, 11, 0, 0, 0, 0, time.UTC),
		Status:    models.StatusPresensiHadir, Lokasi: &location,
	}}}
	policy := auth.StaticPolicy{"pegawai": {"attendance:read:self": true}}
	resolver := testPrincipalResolver(func(context.Context, string) (auth.Principal, error) {
		return auth.Principal{UserID: "employee-1", RoleName: "pegawai"}, nil
	})
	mux := NewMuxWithOptions(MuxOptions{
		Resolver: resolver, Policy: policy,
		AttendanceRead: attendance.NewReadService(reader, nil, policy),
	})
	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/attendance/my-records?from=2026-11-01&to=2026-11-30", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK || reader.userID != "employee-1" {
		t.Fatalf("status=%d user=%q body=%s", response.Code, reader.userID, response.Body)
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(response.Body.String(), "private location") || strings.Contains(response.Body.String(), "lokasi") {
		t.Fatalf("sensitive location exposed: %s", response.Body)
	}
	items := body["items"].([]any)
	item := items[0].(map[string]any)
	if item["date"] != "2026-11-11" || item["status"] != "H" {
		t.Fatalf("unexpected item: %#v", item)
	}
}

func TestUnitAttendanceSummaryUsesResolvedScope(t *testing.T) {
	reader := &fakeAttendanceReader{counts: []attendance.StatusCount{{
		Date:   time.Date(2026, 11, 11, 0, 0, 0, 0, time.UTC),
		Status: models.StatusPresensiHadir, Count: 2,
	}}}
	policy := auth.StaticPolicy{"atasan": {"attendance:read:homebase": true}}
	resolver := testPrincipalResolver(func(context.Context, string) (auth.Principal, error) {
		return auth.Principal{UserID: "manager-1", RoleName: "atasan", HomebaseID: "unit-a"}, nil
	})
	mux := NewMuxWithOptions(MuxOptions{
		Resolver: resolver, Policy: policy,
		AttendanceRead: attendance.NewReadService(reader, fakeAttendanceScope{}, policy),
	})
	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/attendance/unit-summary?from=2026-11-11&to=2026-11-11", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK || len(reader.homebaseIDs) != 2 {
		t.Fatalf("status=%d scope=%v body=%s", response.Code, reader.homebaseIDs, response.Body)
	}
}

func TestAttendanceReadRejectsInvalidDateRange(t *testing.T) {
	reader := &fakeAttendanceReader{}
	policy := auth.StaticPolicy{"pegawai": {"attendance:read:self": true}}
	resolver := testPrincipalResolver(func(context.Context, string) (auth.Principal, error) {
		return auth.Principal{UserID: "employee-1", RoleName: "pegawai"}, nil
	})
	mux := NewMuxWithOptions(MuxOptions{
		Resolver: resolver, Policy: policy,
		AttendanceRead: attendance.NewReadService(reader, nil, policy),
	})
	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/attendance/my-records?from=2026-11-12&to=2026-11-11", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", response.Code)
	}
}
