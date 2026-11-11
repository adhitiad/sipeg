package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/models"
)

// principalResolver mengembalikan principal yang dapat dikonfigurasi per tes.
type principalResolver struct {
	principal auth.Principal
}

func (r principalResolver) ResolvePrincipal(context.Context, string) (auth.Principal, error) {
	return r.principal, nil
}

const (
	atasanID   = "11111111-1111-4111-8111-111111111111"
	pegawaiID  = "22222222-2222-4222-8222-222222222222"
	luarID     = "33333333-3333-4333-8333-333333333333"
	childID    = "44444444-4444-4444-8444-444444444444"
	unitA      = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	unitAChild = "cccccccc-cccc-4ccc-8ccc-cccccccccccc"
	unitB      = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
)

func pointer(value string) *string {
	return &value
}

func scopeMux(store *fakeDirectoryStore, principal auth.Principal, grants map[string]map[auth.Permission]bool) *http.ServeMux {
	return testDirectoryMuxWithResolver(principalResolver{principal: principal}, store, grants)
}

func TestAtasanListEmployeesIsScopedToOwnHomebase(t *testing.T) {
	store := &fakeDirectoryStore{
		scopeIDs: []string{unitA, unitAChild},
		employees: []models.User{
			{BaseModel: models.BaseModel{IDModel: models.IDModel{ID: atasanID}}, Nama: "Dalam Unit", HomebaseId: pointer(unitA), Aktif: true},
			{BaseModel: models.BaseModel{IDModel: models.IDModel{ID: childID}}, Nama: "Unit Anak", HomebaseId: pointer(unitAChild), Aktif: true},
			{BaseModel: models.BaseModel{IDModel: models.IDModel{ID: luarID}}, Nama: "Luar Unit", HomebaseId: pointer(unitB), Aktif: true},
		},
	}
	mux := scopeMux(store, auth.Principal{UserID: atasanID, RoleName: "atasan", HomebaseID: unitA},
		map[string]map[auth.Permission]bool{"atasan": {"employee:read:homebase": true}})

	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/employees", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
	// Cakupan memuat unit penugasan beserta turunannya, bukan filter unit tunggal.
	if len(store.listOptions.ScopeHomebaseIDs) != 2 {
		t.Fatalf("scope units = %v, want unit + descendant", store.listOptions.ScopeHomebaseIDs)
	}
	var body PageDTO[EmployeeDTO]
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 2 {
		t.Fatalf("unexpected scoped items: %#v", body.Items)
	}
	for _, item := range body.Items {
		if item.ID == luarID {
			t.Fatalf("employee outside scope must not be listed: %#v", body.Items)
		}
	}
}

func TestAtasanWithoutUnitAssignmentSeesNoEmployees(t *testing.T) {
	store := &fakeDirectoryStore{
		scopeIDs: []string{},
		employees: []models.User{
			{BaseModel: models.BaseModel{IDModel: models.IDModel{ID: luarID}}, Nama: "Luar Unit", HomebaseId: pointer(unitB), Aktif: true},
		},
	}
	mux := scopeMux(store, auth.Principal{UserID: atasanID, RoleName: "atasan"},
		map[string]map[auth.Permission]bool{"atasan": {"employee:read:homebase": true}})

	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/employees", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
	var body PageDTO[EmployeeDTO]
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 0 {
		t.Fatalf("expected no visible employees, got %#v", body.Items)
	}
}

func TestOperatorCannotReadEmployeeOutsideAssignedUnit(t *testing.T) {
	store := &fakeDirectoryStore{
		scopeIDs: []string{unitA, unitAChild},
		byID: map[string]models.User{
			luarID: {BaseModel: models.BaseModel{IDModel: models.IDModel{ID: luarID}}, Nama: "Luar Unit", HomebaseId: pointer(unitB), Aktif: true},
		},
	}
	mux := scopeMux(store, auth.Principal{UserID: atasanID, RoleName: "operator", HomebaseID: unitA},
		map[string]map[auth.Permission]bool{"operator": {"employee:read:homebase": true}})

	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/employees/"+luarID, nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403, body = %s", response.Code, response.Body)
	}
}

func TestOperatorCanReadEmployeeInDescendantUnit(t *testing.T) {
	store := &fakeDirectoryStore{
		scopeIDs: []string{unitA, unitAChild},
		byID: map[string]models.User{
			childID: {BaseModel: models.BaseModel{IDModel: models.IDModel{ID: childID}}, Nama: "Unit Anak", HomebaseId: pointer(unitAChild), Aktif: true},
		},
	}
	mux := scopeMux(store, auth.Principal{UserID: atasanID, RoleName: "operator", HomebaseID: unitA},
		map[string]map[auth.Permission]bool{"operator": {"employee:read:homebase": true}})

	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/employees/"+childID, nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body = %s", response.Code, response.Body)
	}
}

func TestOperatorCannotUpdateEmployeeOutsideAssignedUnit(t *testing.T) {
	store := &fakeDirectoryStore{
		scopeIDs: []string{unitA},
		byID: map[string]models.User{
			luarID: {BaseModel: models.BaseModel{IDModel: models.IDModel{ID: luarID}}, Nama: "Luar Unit", HomebaseId: pointer(unitB), Aktif: true},
		},
	}
	mux := scopeMux(store, auth.Principal{UserID: atasanID, RoleName: "operator", HomebaseID: unitA},
		map[string]map[auth.Permission]bool{"operator": {"employee:update:homebase": true}})

	request := httptest.NewRequest(http.MethodPatch, APIPrefix+"/employees/"+luarID,
		strings.NewReader(`{"nama":"Nama Baru"}`))
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403, body = %s", response.Code, response.Body)
	}
	if store.employeePatch.Nama.Present {
		t.Fatal("forbidden update must not reach the store")
	}
}

func TestOperatorCannotCreateEmployeeOutsideAssignedUnit(t *testing.T) {
	store := &fakeDirectoryStore{scopeIDs: []string{unitA}}
	mux := scopeMux(store, auth.Principal{UserID: atasanID, RoleName: "operator", HomebaseID: unitA},
		map[string]map[auth.Permission]bool{"operator": {"employee:create:homebase": true}})

	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/employees", strings.NewReader(
		`{"nama":"Pegawai Baru","email":"baru@example.test","homebaseId":"`+unitB+`"}`))
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403, body = %s", response.Code, response.Body)
	}
	if store.employee.ID != "" {
		t.Fatal("forbidden create must not reach the store")
	}
}

func TestAtasanCannotReadEmployeeOutsideHomebase(t *testing.T) {
	store := &fakeDirectoryStore{byID: map[string]models.User{
		luarID: {BaseModel: models.BaseModel{IDModel: models.IDModel{ID: luarID}}, Nama: "Luar Unit", HomebaseId: pointer(unitB), Aktif: true},
	}}
	mux := scopeMux(store, auth.Principal{UserID: atasanID, RoleName: "atasan", HomebaseID: unitA},
		map[string]map[auth.Permission]bool{"atasan": {"employee:read:homebase": true}})

	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/employees/"+luarID, nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403, body = %s", response.Code, response.Body)
	}
}

func TestAtasanCanReadEmployeeInsideHomebase(t *testing.T) {
	store := &fakeDirectoryStore{byID: map[string]models.User{
		atasanID: {BaseModel: models.BaseModel{IDModel: models.IDModel{ID: atasanID}}, Nama: "Dalam Unit", HomebaseId: pointer(unitA), Aktif: true},
	}}
	mux := scopeMux(store, auth.Principal{UserID: atasanID, RoleName: "atasan", HomebaseID: unitA},
		map[string]map[auth.Permission]bool{"atasan": {"employee:read:homebase": true}})

	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/employees/"+atasanID, nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body = %s", response.Code, response.Body)
	}
}

func TestPegawaiOnlySeesOwnRecord(t *testing.T) {
	store := &fakeDirectoryStore{employees: []models.User{
		{BaseModel: models.BaseModel{IDModel: models.IDModel{ID: pegawaiID}}, Nama: "Saya", HomebaseId: pointer(unitA), Aktif: true},
		{BaseModel: models.BaseModel{IDModel: models.IDModel{ID: luarID}}, Nama: "Rekan", HomebaseId: pointer(unitA), Aktif: true},
	}}
	mux := scopeMux(store, auth.Principal{UserID: pegawaiID, RoleName: "pegawai", HomebaseID: unitA},
		map[string]map[auth.Permission]bool{"pegawai": {"employee:read:self": true}})

	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/employees", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
	var body PageDTO[EmployeeDTO]
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 1 || body.Items[0].ID != pegawaiID {
		t.Fatalf("unexpected items for pegawai scope: %#v", body.Items)
	}
}

func TestPegawaiCannotReadOtherEmployeeRecord(t *testing.T) {
	store := &fakeDirectoryStore{byID: map[string]models.User{
		luarID: {BaseModel: models.BaseModel{IDModel: models.IDModel{ID: luarID}}, Nama: "Rekan", HomebaseId: pointer(unitA), Aktif: true},
	}}
	mux := scopeMux(store, auth.Principal{UserID: pegawaiID, RoleName: "pegawai", HomebaseID: unitA},
		map[string]map[auth.Permission]bool{"pegawai": {"employee:read:self": true}})

	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/employees/"+luarID, nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403, body = %s", response.Code, response.Body)
	}
}

func TestReadAnyRoleIsNotScopedToHomebase(t *testing.T) {
	store := &fakeDirectoryStore{employees: []models.User{
		{BaseModel: models.BaseModel{IDModel: models.IDModel{ID: atasanID}}, Nama: "Dalam Unit", HomebaseId: pointer(unitA), Aktif: true},
		{BaseModel: models.BaseModel{IDModel: models.IDModel{ID: luarID}}, Nama: "Luar Unit", HomebaseId: pointer(unitB), Aktif: true},
	}}
	mux := scopeMux(store, auth.Principal{UserID: atasanID, RoleName: "operator", HomebaseID: unitA},
		map[string]map[auth.Permission]bool{"operator": {"employee:read:any": true}})

	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/employees", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
	if store.listHomebaseID != "" {
		t.Fatalf("employee:read:any must not be narrowed to homebase, got filter %q", store.listHomebaseID)
	}
}

func TestRoleComesFromServerSideSessionNotRequestHeader(t *testing.T) {
	store := &fakeDirectoryStore{employees: []models.User{
		{BaseModel: models.BaseModel{IDModel: models.IDModel{ID: pegawaiID}}, Nama: "Saya", HomebaseId: pointer(unitA), Aktif: true},
	}}
	mux := scopeMux(store, auth.Principal{UserID: pegawaiID, RoleName: "pegawai", HomebaseID: unitA},
		map[string]map[auth.Permission]bool{
			"pegawai":  {"employee:read:self": true},
			"operator": {"employee:read:any": true},
		})

	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/employees", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	request.Header.Set("X-Role", "operator")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
	var body PageDTO[EmployeeDTO]
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 1 || body.Items[0].ID != pegawaiID {
		t.Fatalf("request header must not elevate scope: %#v", body.Items)
	}
}
