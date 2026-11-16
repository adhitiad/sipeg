package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	. "simpeg/backend/internal/httpserver"
	"testing"

	"simpeg/backend/internal/attendance"
	"simpeg/backend/internal/auth"
)

func attendanceRecordsMux(reader *fakeAttendanceReader) http.Handler {
	policy := auth.StaticPolicy{"pegawai": {"attendance:read:self": true}}
	resolver := testPrincipalResolver(func(context.Context, string) (auth.Principal, error) {
		return auth.Principal{UserID: "employee-1", RoleName: "pegawai"}, nil
	})
	return NewMuxWithOptions(MuxOptions{
		Resolver:       resolver,
		Policy:         policy,
		AttendanceRead: attendance.NewReadService(reader, nil, policy),
	})
}

func TestMyAttendanceRecordsRejectsSortOutsideAllowlist(t *testing.T) {
	reader := &fakeAttendanceReader{}
	mux := attendanceRecordsMux(reader)

	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/attendance/my-records?from=2026-11-01&to=2026-11-30&sort=lokasi", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", response.Code, response.Body)
	}
	if got := response.Body.String(); !contains(got, "invalid_query") {
		t.Fatalf("body=%s", got)
	}
}

func TestMyAttendanceRecordsRejectsUnknownStatus(t *testing.T) {
	reader := &fakeAttendanceReader{}
	mux := attendanceRecordsMux(reader)

	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/attendance/my-records?from=2026-11-01&to=2026-11-30&status=Z", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", response.Code, response.Body)
	}
	if got := response.Body.String(); !contains(got, "invalid_query") {
		t.Fatalf("body=%s", got)
	}
}

func TestMyAttendanceRecordsPassesFilterToRepository(t *testing.T) {
	reader := &fakeAttendanceReader{}
	mux := attendanceRecordsMux(reader)

	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/attendance/my-records?from=2026-11-01&to=2026-11-30&sort=status&order=asc&status=H&search=Budi", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body)
	}
	filter := reader.filter
	if filter.Sort != "status" || filter.Order != "asc" {
		t.Fatalf("sort=%q order=%q", filter.Sort, filter.Order)
	}
	if filter.Status != "H" || filter.Search != "Budi" {
		t.Fatalf("status=%q search=%q", filter.Status, filter.Search)
	}
}

func TestMyAttendanceRecordsKeepsNewestFirstByDefault(t *testing.T) {
	reader := &fakeAttendanceReader{}
	mux := attendanceRecordsMux(reader)

	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/attendance/my-records?from=2026-11-01&to=2026-11-30", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body)
	}
	if reader.filter.Order != "desc" || reader.filter.Sort != "tanggal" {
		t.Fatalf("sort=%q order=%q", reader.filter.Sort, reader.filter.Order)
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (haystack == needle || indexOf(haystack, needle) >= 0)
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
