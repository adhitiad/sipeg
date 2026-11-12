package httpserver

import (
	"encoding/csv"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"simpeg/backend/internal/audit"
	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/models"
)

func TestAuditExportWritesCSVWithHeadersAndRows(t *testing.T) {
	store := &fakeDirectoryStore{
		auditLogs: []models.AuditLog{
			{IDModel: models.IDModel{ID: "log-1"}, Aksi: "employee.updated", Keterangan: pointer("diperbarui")},
			{IDModel: models.IDModel{ID: "log-2"}, Aksi: "session.issued"},
		},
	}
	mux := scopeMux(store, auth.Principal{UserID: atasanID, RoleName: "operator", HomebaseID: unitA},
		map[string]map[auth.Permission]bool{"operator": {"audit:read:homebase": true}})

	// Parameter pagination diabaikan oleh ekspor.
	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/audit-logs/export?page=2&pageSize=5", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
	contentType := response.Header().Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/csv") {
		t.Fatalf("content type = %q", contentType)
	}
	if !strings.Contains(response.Header().Get("Content-Disposition"), "attachment; filename=") {
		t.Fatalf("content disposition = %q", response.Header().Get("Content-Disposition"))
	}
	body := response.Body.String()
	if !strings.HasPrefix(body, "\xef\xbb\xbf") {
		t.Fatalf("expected UTF-8 BOM, body starts with %q", body[:3])
	}
	records, err := csv.NewReader(strings.NewReader(strings.TrimPrefix(body, "\xef\xbb\xbf"))).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 3 {
		t.Fatalf("expected header + 2 rows, got %d records", len(records))
	}
	wantHeader := []string{"id", "createdAt", "aktorId", "aksi", "sasaranId", "unitId", "keterangan", "ip"}
	if !slices.Equal(records[0], wantHeader) {
		t.Fatalf("unexpected header: %v", records[0])
	}
	if records[1][3] != "employee.updated" || records[2][3] != "session.issued" {
		t.Fatalf("unexpected rows: %v %v", records[1], records[2])
	}
	if len(store.recordedAudits) != 1 || store.recordedAudits[0].Action != audit.ActionAuditExported {
		t.Fatalf("expected audit.exported entry, got %v", store.recordedAudits)
	}
}

func TestAuditExportRejectedWithoutPermission(t *testing.T) {
	store := &fakeDirectoryStore{}
	mux := testDirectoryMux(store, map[string]map[auth.Permission]bool{})

	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/audit-logs/export", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403, body = %s", response.Code, response.Body)
	}
}
