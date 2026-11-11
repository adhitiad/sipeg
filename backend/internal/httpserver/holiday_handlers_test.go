package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/holiday"
)

// fakeHolidayStore adalah penyimpanan kalender hari libur sintetis.
type fakeHolidayStore struct {
	entries []holiday.Entry
}

func (s *fakeHolidayStore) ReplaceYear(_ context.Context, _ string, _ int, entries []holiday.Entry) (int, error) {
	s.entries = append([]holiday.Entry{}, entries...)
	return len(entries), nil
}

func (s *fakeHolidayStore) ListYear(_ context.Context, _ int) ([]holiday.Entry, error) {
	return append([]holiday.Entry{}, s.entries...), nil
}

func holidayTestMux(role string, grants map[string]map[auth.Permission]bool, calendar *holiday.Service) *http.ServeMux {
	policy := auth.StaticPolicy(grants)
	return NewMuxWithOptions(MuxOptions{
		Resolver: principalResolver{principal: auth.Principal{UserID: "user-1", RoleName: role}},
		Policy:   policy,
		Holiday:  calendar,
	})
}

func TestHolidayRoutesRequireAuthenticationAndPermission(t *testing.T) {
	calendar := holiday.NewService(&fakeHolidayStore{}, nil, nil)
	mux := holidayTestMux("pegawai", nil, calendar)
	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/holidays?year=2026", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", response.Code)
	}
}

func TestListHolidaysReturnsStoredCalendar(t *testing.T) {
	store := &fakeHolidayStore{entries: []holiday.Entry{
		{Tanggal: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Nama: "Tahun Baru 2026"},
	}}
	calendar := holiday.NewService(store, nil, nil)
	policy := map[string]map[auth.Permission]bool{"operator": {"holiday:read": true}}
	mux := holidayTestMux("operator", policy, calendar)

	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/holidays?year=2026", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
	if !strings.Contains(response.Body.String(), "Tahun Baru 2026") || !strings.Contains(response.Body.String(), "2026-01-01") {
		t.Fatalf("body = %s", response.Body)
	}
}

func TestListHolidaysRejectsInvalidYear(t *testing.T) {
	calendar := holiday.NewService(&fakeHolidayStore{}, nil, nil)
	policy := map[string]map[auth.Permission]bool{"admin": {"holiday:read": true}}
	mux := holidayTestMux("admin", policy, calendar)

	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/holidays?year=1800", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", response.Code)
	}
}

func TestRefreshHolidaysRequiresManagePermission(t *testing.T) {
	calendar := holiday.NewService(&fakeHolidayStore{}, nil, nil)
	policy := map[string]map[auth.Permission]bool{"operator": {"holiday:read": true}}
	mux := holidayTestMux("operator", policy, calendar)

	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/holidays/2026/refresh", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", response.Code)
	}
}

func TestRefreshHolidaysWithoutProviderReturns503(t *testing.T) {
	calendar := holiday.NewService(&fakeHolidayStore{}, nil, nil)
	policy := map[string]map[auth.Permission]bool{"admin": {"holiday:manage": true, "holiday:read": true}}
	mux := holidayTestMux("admin", policy, calendar)

	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/holidays/2026/refresh", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", response.Code)
	}
}

func TestRefreshHolidaysWithProviderReplacesCalendar(t *testing.T) {
	store := &fakeHolidayStore{}
	provider := fakeHolidayProvider{entries: []holiday.Entry{
		{Tanggal: time.Date(2026, 8, 17, 0, 0, 0, 0, time.UTC), Nama: "Hari Kemerdekaan RI"},
	}}
	calendar := holiday.NewService(store, provider, nil)
	policy := map[string]map[auth.Permission]bool{"admin": {"holiday:manage": true, "holiday:read": true}}
	mux := holidayTestMux("admin", policy, calendar)

	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/holidays/2026/refresh", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
	if !strings.Contains(response.Body.String(), `"refreshed":1`) {
		t.Fatalf("body = %s", response.Body)
	}
	if len(store.entries) != 1 || store.entries[0].Nama != "Hari Kemerdekaan RI" {
		t.Fatalf("entries = %+v", store.entries)
	}
}

type fakeHolidayProvider struct {
	entries []holiday.Entry
}

func (p fakeHolidayProvider) FetchYear(_ context.Context, _ int) ([]holiday.Entry, error) {
	return append([]holiday.Entry{}, p.entries...), nil
}
