package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"simpeg/backend/internal/audit"
	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/directory/app"
	"simpeg/backend/internal/directory/domain"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
)

type fakeDirectoryStore struct {
	employee       models.User
	employees      []models.User
	byID           map[string]models.User
	auditLogs      []models.AuditLog
	categories     []models.Kategori
	scopeIDs       []string
	recordedAudits []audit.Entry
	listHomebaseID string
	listOptions    listing.Options
	batchCalls     int
	organization   models.Homebase
	employeePatch  domain.EmployeePatch
	assignedRole   string
	orgPatch       domain.OrganizationPatch
	actorID        string
	archived       bool
	err            error
}

func (s *fakeDirectoryStore) RecordAudit(_ context.Context, entry audit.Entry) error {
	s.recordedAudits = append(s.recordedAudits, entry)
	return s.err
}

func (s *fakeDirectoryStore) ListUnitScopeIDs(context.Context, string) ([]string, error) {
	if s.scopeIDs != nil {
		return s.scopeIDs, s.err
	}
	return []string{"unit-a"}, s.err
}

func (s *fakeDirectoryStore) ListEmployees(_ context.Context, options listing.Options) ([]models.User, int64, error) {
	s.listHomebaseID = options.HomebaseID
	s.listOptions = options
	items := s.employees
	if len(items) == 0 {
		items = []models.User{s.employee}
	}
	if options.HomebaseID != "" {
		items = filterByHomebase(items, options.HomebaseID)
	}
	if len(options.ScopeHomebaseIDs) > 0 {
		items = filterByScope(items, options.ScopeHomebaseIDs)
	}
	if options.UserID != "" {
		items = filterByUserID(items, options.UserID)
	}
	return items, int64(len(items)), s.err
}

func filterByHomebase(items []models.User, homebaseID string) []models.User {
	filtered := make([]models.User, 0, len(items))
	for _, user := range items {
		if user.HomebaseId != nil && *user.HomebaseId == homebaseID {
			filtered = append(filtered, user)
		}
	}
	return filtered
}

func filterByScope(items []models.User, unitIDs []string) []models.User {
	allowed := make(map[string]bool, len(unitIDs))
	for _, id := range unitIDs {
		allowed[id] = true
	}
	filtered := make([]models.User, 0, len(items))
	for _, user := range items {
		if user.HomebaseId != nil && allowed[*user.HomebaseId] {
			filtered = append(filtered, user)
		}
	}
	return filtered
}

func filterByUserID(items []models.User, userID string) []models.User {
	filtered := make([]models.User, 0, 1)
	for _, user := range items {
		if user.ID == userID {
			filtered = append(filtered, user)
		}
	}
	return filtered
}

func (s *fakeDirectoryStore) ListOrganizations(_ context.Context, options listing.Options) ([]models.Homebase, int64, error) {
	s.listOptions = options
	return []models.Homebase{s.organization}, 1, s.err
}

func (s *fakeDirectoryStore) ListAuditLogs(_ context.Context, options listing.Options) ([]models.AuditLog, int64, error) {
	s.listOptions = options
	return s.auditLogs, int64(len(s.auditLogs)), s.err
}

func (s *fakeDirectoryStore) ListCategories(_ context.Context, options listing.Options) ([]models.Kategori, int64, error) {
	s.listOptions = options
	return s.categories, int64(len(s.categories)), s.err
}

func (s *fakeDirectoryStore) GetEmployee(_ context.Context, id string) (models.User, error) {
	if s.byID != nil {
		user, ok := s.byID[id]
		if !ok {
			return models.User{}, domain.ErrNotFound
		}
		return user, s.err
	}
	return s.employee, s.err
}

func (s *fakeDirectoryStore) CreateEmployee(_ context.Context, actor string, user models.User) (models.User, error) {
	s.actorID, s.employee = actor, user
	s.employee.ID = "11111111-1111-4111-8111-111111111111"
	return s.employee, s.err
}

func (s *fakeDirectoryStore) CreateEmployees(_ context.Context, actor string, users []models.User) ([]models.User, error) {
	s.actorID = actor
	s.batchCalls++
	if s.err != nil {
		return nil, s.err
	}
	s.employees = append([]models.User(nil), users...)
	return users, nil
}

func (s *fakeDirectoryStore) UpdateEmployee(_ context.Context, actor, _ string, patch domain.EmployeePatch) (models.User, error) {
	s.actorID, s.employeePatch = actor, patch
	return s.employee, s.err
}

func (s *fakeDirectoryStore) AssignEmployeeRole(_ context.Context, actor, id, roleName string) (models.User, error) {
	s.actorID = actor
	s.employee.ID = id
	s.assignedRole = roleName
	return s.employee, s.err
}

func (s *fakeDirectoryStore) ArchiveEmployee(_ context.Context, actor, _ string) error {
	s.actorID, s.archived = actor, true
	return s.err
}

func (s *fakeDirectoryStore) GetOrganization(context.Context, string) (models.Homebase, error) {
	return s.organization, s.err
}
func (s *fakeDirectoryStore) CreateOrganization(_ context.Context, actor string, organization models.Homebase) (models.Homebase, error) {
	s.actorID, s.organization = actor, organization
	s.organization.ID = "22222222-2222-4222-8222-222222222222"
	return s.organization, s.err
}

func (s *fakeDirectoryStore) UpdateOrganization(_ context.Context, actor, _ string, patch domain.OrganizationPatch) (models.Homebase, error) {
	s.actorID, s.orgPatch = actor, patch
	return s.organization, s.err
}

func (s *fakeDirectoryStore) ArchiveOrganization(_ context.Context, actor, _ string) error {
	s.actorID, s.archived = actor, true
	return s.err
}

// testDirectoryMux menyusun mux direktori dengan use case sungguhan dan
// repository palsu.
func testDirectoryMux(store *fakeDirectoryStore, grants map[string]map[auth.Permission]bool) *http.ServeMux {
	return testDirectoryMuxWithResolver(testSessionResolver{}, store, grants)
}

func testDirectoryMuxWithResolver(resolver auth.SessionResolver, store *fakeDirectoryStore, grants map[string]map[auth.Permission]bool) *http.ServeMux {
	policy := auth.StaticPolicy(grants)
	return NewMuxWithOptions(MuxOptions{
		Resolver: resolver,
		Policy:   policy,
		Directory: DirectoryServices{
			Employees:     app.NewEmployeeService(store, policy, store),
			Organizations: app.NewOrganizationService(store, policy, store),
			Categories:    app.NewCategoryService(store, nil),
			Audits:        app.NewAuditService(store, policy, store),
		},
	})
}

func TestDirectoryRoutesRequireAuthenticationAndPermission(t *testing.T) {
	store := &fakeDirectoryStore{}
	mux := testDirectoryMux(store, nil)
	for _, path := range []string{
		APIPrefix + "/employees",
		APIPrefix + "/organizations",
	} {
		for _, authorization := range []string{"", "Bearer valid-session"} {
			request := httptest.NewRequest(http.MethodGet, path, nil)
			if authorization != "" {
				request.Header.Set("Authorization", authorization)
			}
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			expected := http.StatusUnauthorized
			if authorization != "" {
				expected = http.StatusForbidden
			}
			if response.Code != expected {
				t.Errorf("GET %s authorization %q: got %d, want %d", path, authorization, response.Code, expected)
			}
		}
	}
}

func TestCreateEmployeePersistsOnlyAllowedFields(t *testing.T) {
	store := &fakeDirectoryStore{}
	mux := testDirectoryMux(store, map[string]map[auth.Permission]bool{
		"operator": {"employee:create": true},
	})
	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/employees", strings.NewReader(
		`{"nama":"Pegawai Uji","email":"TEST@EXAMPLE.TEST","nip":"NIP-1"}`,
	))
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
	if store.employee.Email != "test@example.test" || store.employee.RoleId != nil || store.employee.EmailVerified {
		t.Fatalf("unexpected employee provisioning fields: %#v", store.employee)
	}
	if store.actorID != "user-1" {
		t.Fatalf("audit actor = %q, want user-1", store.actorID)
	}
	var responseBody map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &responseBody); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"noHp", "bankRekening", "gajiPokok", "roleId", "nik"} {
		if _, exists := responseBody[field]; exists {
			t.Errorf("response leaked field %q", field)
		}
	}
}

func TestCreateOrganizationUsesExplicitTypeAndParent(t *testing.T) {
	store := &fakeDirectoryStore{}
	mux := testDirectoryMux(store, map[string]map[auth.Permission]bool{
		"operator": {"organization:manage": true},
	})
	parentID := "33333333-3333-4333-8333-333333333333"
	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/organizations", strings.NewReader(
		`{"name":"Fakultas Contoh","jenis":"FAKULTAS","parentId":"`+parentID+`"}`,
	))
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
	if store.organization.Jenis != models.JenisOrganisasiFakultas || store.organization.ParentId == nil || *store.organization.ParentId != parentID {
		t.Fatalf("unexpected organization: %#v", store.organization)
	}
	if store.actorID != "user-1" {
		t.Fatalf("audit actor = %q, want user-1", store.actorID)
	}
}

func TestListEmployeesReturnsPaginatedDTO(t *testing.T) {
	store := &fakeDirectoryStore{employee: models.User{
		BaseModel: models.BaseModel{IDModel: models.IDModel{ID: "11111111-1111-4111-8111-111111111111"}},
		Nama:      "Pegawai Uji", Email: "person@example.test", Aktif: true,
	}}
	mux := testDirectoryMux(store, map[string]map[auth.Permission]bool{
		"operator": {"employee:read:any": true},
	})
	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/employees?page=2&pageSize=10", nil)
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
	if body.Page != 2 || body.PageSize != 10 || body.TotalItems != 1 || body.Items[0].Nama != "Pegawai Uji" {
		t.Fatalf("unexpected page response: %#v", body)
	}
}

func TestEmployeeUpdateAcceptsEditableProfileFields(t *testing.T) {
	store := &fakeDirectoryStore{}
	mux := testDirectoryMux(store, map[string]map[auth.Permission]bool{
		"operator": {"employee:update:any": true},
	})
	request := httptest.NewRequest(http.MethodPatch, APIPrefix+"/employees/11111111-1111-4111-8111-111111111111",
		strings.NewReader(`{"nama":"Nama Baru","nip":null}`))
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
	if store.employeePatch.Nama.Value != "Nama Baru" || !store.employeePatch.NIP.Null {
		t.Fatalf("unexpected profile updates: %#v", store.employeePatch)
	}
}

func TestArchiveEmployeeUsesResolvedActor(t *testing.T) {
	store := &fakeDirectoryStore{}
	mux := testDirectoryMux(store, map[string]map[auth.Permission]bool{
		"operator": {"employee:archive": true},
	})
	request := httptest.NewRequest(http.MethodDelete, APIPrefix+"/employees/11111111-1111-4111-8111-111111111111", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent || !store.archived || store.actorID != "user-1" {
		t.Fatalf("status=%d archived=%t actor=%q", response.Code, store.archived, store.actorID)
	}
}

func TestOrganizationArchiveConflictIsReported(t *testing.T) {
	store := &fakeDirectoryStore{err: domain.ErrHasActiveMember}
	mux := testDirectoryMux(store, map[string]map[auth.Permission]bool{
		"operator": {"organization:manage": true},
	})
	request := httptest.NewRequest(http.MethodDelete, APIPrefix+"/organizations/22222222-2222-4222-8222-222222222222", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", response.Code)
	}
	if !strings.Contains(response.Body.String(), "conflict") {
		t.Fatalf("expected conflict code, body = %s", response.Body)
	}
}

func TestStoreErrorsDoNotLeakDatabaseDetails(t *testing.T) {
	store := &fakeDirectoryStore{err: errors.New("db password or connection string")}
	mux := testDirectoryMux(store, map[string]map[auth.Permission]bool{
		"operator": {"employee:read:any": true},
	})
	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/employees", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", response.Code)
	}
	if strings.Contains(response.Body.String(), "password") {
		t.Fatalf("internal error leaked: %s", response.Body)
	}
}
