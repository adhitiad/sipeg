package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"simpeg/backend/internal/apierror"
	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/models"
)

func TestListQueryRejectsUnknownSort(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/employees?sort=unknown_field", nil)
	_, err := parseListQuery(request, []string{"nama", "nip"}, "nama")
	if err == nil {
		t.Fatal("expected error for unknown sort field")
	}
}

func TestListQueryRejectsSortInjectionAttempt(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/employees?sort=nama%20OR%201%3D1", nil)
	_, err := parseListQuery(request, []string{"nama", "nip"}, "nama")
	if err == nil {
		t.Fatal("expected error for sort value outside allowlist")
	}
}

func TestListQueryRejectsUnknownOrder(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/employees?order=sideways", nil)
	_, err := parseListQuery(request, []string{"nama"}, "nama")
	if err == nil {
		t.Fatal("expected error for unknown order")
	}
}

func TestListQueryAppliesDefaults(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/employees", nil)
	query, err := parseListQuery(request, []string{"nama", "nip"}, "nama")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if query.Sort != "nama" || query.Order != "asc" || query.Page != 1 || query.PageSize != 20 {
		t.Fatalf("unexpected defaults: %+v", query)
	}
}

func TestListQueryAcceptsAllowedSortAndOrder(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/employees?sort=nip&order=desc&page=2&pageSize=10", nil)
	query, err := parseListQuery(request, []string{"nama", "nip"}, "nama")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if query.Sort != "nip" || query.Order != "desc" || query.Page != 2 || query.PageSize != 10 {
		t.Fatalf("unexpected query: %+v", query)
	}
}

func TestEmployeeListRejectsUnknownSortField(t *testing.T) {
	store := &fakeDirectoryStore{}
	mux := testDirectoryMux(store, map[string]map[auth.Permission]bool{
		"operator": {"employee:read:any": true},
	})
	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/employees?sort=roleId", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body = %s", response.Code, response.Body)
	}
	if !strings.Contains(response.Body.String(), apierror.CodeInvalidQuery) {
		t.Fatalf("unexpected body: %s", response.Body)
	}
}

func TestEmployeeListPassesSortToStore(t *testing.T) {
	store := &fakeDirectoryStore{}
	mux := testDirectoryMux(store, map[string]map[auth.Permission]bool{
		"operator": {"employee:read:any": true},
	})
	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/employees?sort=createdAt&order=desc", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
	if store.listOptions.Sort != "createdAt" || store.listOptions.Order != "desc" {
		t.Fatalf("unexpected list options: %+v", store.listOptions)
	}
}

func TestAuditLogListReturnsDTOWithoutProfileData(t *testing.T) {
	actor := "11111111-1111-4111-8111-111111111111"
	target := "22222222-2222-4222-8222-222222222222"
	note := "profil diubah"
	store := &fakeDirectoryStore{auditLogs: []models.AuditLog{
		{
			IDModel:    models.IDModel{ID: "33333333-3333-4333-8333-333333333333"},
			AktorId:    &actor,
			Aksi:       "employee.updated",
			Sasaran:    &target,
			Keterangan: &note,
		},
	}}
	mux := testDirectoryMux(store, map[string]map[auth.Permission]bool{
		"operator": {"audit:read": true},
	})
	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/audit-logs", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
	var body PageDTO[AuditLogDTO]
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 1 || body.Items[0].Aksi != "employee.updated" {
		t.Fatalf("unexpected audit items: %#v", body.Items)
	}
	// DTO audit tidak boleh memuat isi profil pegawai.
	if strings.Contains(response.Body.String(), "nama") || strings.Contains(response.Body.String(), "email") {
		t.Fatalf("audit response leaked profile fields: %s", response.Body)
	}
}

func TestAuditLogListRequiresPermission(t *testing.T) {
	mux := testDirectoryMux(&fakeDirectoryStore{}, nil)
	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/audit-logs", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", response.Code)
	}
}

func TestCategoryListReturnsDTO(t *testing.T) {
	store := &fakeDirectoryStore{categories: []models.Kategori{
		{
			BaseModel:   models.BaseModel{IDModel: models.IDModel{ID: "44444444-4444-4444-8444-444444444444"}},
			Kode:        "DOSEN",
			Nama:        "Dosen",
			UsiaPensiun: 60,
			Aktif:       true,
		},
	}}
	mux := testDirectoryMux(store, map[string]map[auth.Permission]bool{
		"operator": {"category:read": true},
	})
	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/categories", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
	var body PageDTO[CategoryDTO]
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 1 || body.Items[0].Kode != "DOSEN" || body.Items[0].UsiaPensiun != 60 {
		t.Fatalf("unexpected category items: %#v", body.Items)
	}
}
