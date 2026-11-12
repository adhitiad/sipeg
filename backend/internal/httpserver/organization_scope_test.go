package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"simpeg/backend/internal/audit"
	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/models"
)

func TestOperatorOrganizationListIsScopedToUnit(t *testing.T) {
	store := &fakeDirectoryStore{scopeIDs: []string{unitA, unitAChild}}
	mux := scopeMux(store, auth.Principal{UserID: atasanID, RoleName: "operator", HomebaseID: unitA},
		map[string]map[auth.Permission]bool{"operator": {"organization:read:homebase": true}})

	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/organizations", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
	if len(store.listOptions.ScopeHomebaseIDs) != 2 {
		t.Fatalf("scope units = %v, want unit + descendant", store.listOptions.ScopeHomebaseIDs)
	}
}

func TestOperatorCannotReadOrganizationOutsideScope(t *testing.T) {
	store := &fakeDirectoryStore{
		scopeIDs:     []string{unitA},
		byID:         map[string]models.User{},
		organization: models.Homebase{BaseModel: models.BaseModel{IDModel: models.IDModel{ID: luarID}}, Name: "Luar Unit", Aktif: true},
	}
	mux := scopeMux(store, auth.Principal{UserID: atasanID, RoleName: "operator", HomebaseID: unitA},
		map[string]map[auth.Permission]bool{"operator": {"organization:read:homebase": true}})

	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/organizations/"+luarID, nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403, body = %s", response.Code, response.Body)
	}
}

func TestOperatorCanManageOrganizationInsideScope(t *testing.T) {
	store := &fakeDirectoryStore{
		scopeIDs:     []string{unitA},
		organization: models.Homebase{BaseModel: models.BaseModel{IDModel: models.IDModel{ID: unitA}}, Name: "Unit A", Aktif: true},
	}
	mux := scopeMux(store, auth.Principal{UserID: atasanID, RoleName: "operator", HomebaseID: unitA},
		map[string]map[auth.Permission]bool{"operator": {"organization:manage:homebase": true}})

	request := httptest.NewRequest(http.MethodPatch, APIPrefix+"/organizations/"+unitA,
		strings.NewReader(`{"name":"Unit A Baru"}`))
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body = %s", response.Code, response.Body)
	}
}

func TestOperatorCannotCreateOrganizationOutsideScope(t *testing.T) {
	store := &fakeDirectoryStore{scopeIDs: []string{unitA}}
	mux := scopeMux(store, auth.Principal{UserID: atasanID, RoleName: "operator", HomebaseID: unitA},
		map[string]map[auth.Permission]bool{"operator": {"organization:manage:homebase": true}})

	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/organizations", strings.NewReader(
		`{"name":"Jurusan Luar","jenis":"JURUSAN","parentId":"`+unitB+`"}`))
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403, body = %s", response.Code, response.Body)
	}
	if store.organization.ID != "" {
		t.Fatal("forbidden create must not reach the store")
	}
}

func TestAdminOrganizationAccessIsNotScoped(t *testing.T) {
	store := &fakeDirectoryStore{scopeIDs: []string{unitA}}
	mux := scopeMux(store, auth.Principal{UserID: atasanID, RoleName: "admin"},
		map[string]map[auth.Permission]bool{"admin": {"organization:read": true}})

	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/organizations", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
	if len(store.listOptions.ScopeHomebaseIDs) != 0 {
		t.Fatalf("global role must not be scoped, got %v", store.listOptions.ScopeHomebaseIDs)
	}
}

func TestAuditListIsScopedToUnit(t *testing.T) {
	store := &fakeDirectoryStore{scopeIDs: []string{unitA, unitAChild}}
	mux := scopeMux(store, auth.Principal{UserID: atasanID, RoleName: "operator", HomebaseID: unitA},
		map[string]map[auth.Permission]bool{"operator": {"audit:read:homebase": true}})

	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/audit-logs", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
	if len(store.listOptions.ScopeHomebaseIDs) != 2 {
		t.Fatalf("audit scope units = %v, want unit + descendant", store.listOptions.ScopeHomebaseIDs)
	}
}

func TestAuditAccessIsRecordedWhenEntriesAreReturned(t *testing.T) {
	store := &fakeDirectoryStore{
		scopeIDs: []string{unitA},
		auditLogs: []models.AuditLog{
			{IDModel: models.IDModel{ID: "log-1"}, Aksi: "employee.updated"},
		},
	}
	mux := scopeMux(store, auth.Principal{UserID: atasanID, RoleName: "operator", HomebaseID: unitA},
		map[string]map[auth.Permission]bool{"operator": {"audit:read:homebase": true}})

	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/audit-logs", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
	if len(store.recordedAudits) != 1 {
		t.Fatalf("expected one audit access entry, got %d", len(store.recordedAudits))
	}
	entry := store.recordedAudits[0]
	if entry.Action != audit.ActionAuditAccessed {
		t.Fatalf("unexpected action: %q", entry.Action)
	}
	if entry.ActorID != atasanID {
		t.Fatalf("audit access actor = %q, want %q", entry.ActorID, atasanID)
	}
}

func TestAuditAccessIsNotRecordedForEmptyResult(t *testing.T) {
	store := &fakeDirectoryStore{scopeIDs: []string{unitA}}
	mux := scopeMux(store, auth.Principal{UserID: atasanID, RoleName: "operator", HomebaseID: unitA},
		map[string]map[auth.Permission]bool{"operator": {"audit:read:homebase": true}})

	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/audit-logs", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
	if len(store.recordedAudits) != 0 {
		t.Fatalf("empty read must not write audit access, got %d entries", len(store.recordedAudits))
	}
}

func TestAuditDTODoesNotExposeRawUnitColumn(t *testing.T) {
	// DTO audit tidak pernah memuat isi profil pegawai.
	unit := unitA
	target := luarID
	entry := models.AuditLog{
		IDModel:   models.IDModel{ID: "log-1"},
		Aksi:      "employee.updated",
		UnitId:    &unit,
		Sasaran:   &target,
		CreatedAt: time.Now(),
	}
	dto := auditLogDTO(entry)
	body, err := json.Marshal(dto)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `"aksi":"employee.updated"`) {
		t.Fatalf("unexpected DTO: %s", body)
	}
	if strings.Contains(string(body), "nama") || strings.Contains(string(body), "email") {
		t.Fatalf("audit DTO leaked profile fields: %s", body)
	}
}

func TestAuditExportScopeFollowsOperatorHomebase(t *testing.T) {
	store := &fakeDirectoryStore{scopeIDs: []string{unitA, unitAChild}}
	mux := scopeMux(store, auth.Principal{UserID: atasanID, RoleName: "operator", HomebaseID: unitA},
		map[string]map[auth.Permission]bool{"operator": {"audit:read:homebase": true}})

	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/audit-logs/export", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
	if len(store.streamOptions.ScopeHomebaseIDs) != 2 {
		t.Fatalf("export scope units = %v, want unit + descendant", store.streamOptions.ScopeHomebaseIDs)
	}
}

func TestAuditExportAdminIsNotScoped(t *testing.T) {
	store := &fakeDirectoryStore{}
	mux := scopeMux(store, auth.Principal{UserID: atasanID, RoleName: "admin"},
		map[string]map[auth.Permission]bool{"admin": {"audit:read": true}})

	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/audit-logs/export", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
	if len(store.streamOptions.ScopeHomebaseIDs) != 0 {
		t.Fatalf("global role must not be scoped, got %v", store.streamOptions.ScopeHomebaseIDs)
	}
}
