package app

import (
	"context"
	"errors"
	"testing"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/directory/domain"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
)

// fakeEmployeeRepository adalah repository pegawai sintetis untuk menguji
// aturan cakupan pada lapisan aplikasi.
type fakeEmployeeRepository struct {
	employees   []models.User
	lastOptions listing.Options
	created     []models.User
	archived    []string
}

func (r *fakeEmployeeRepository) ListEmployees(_ context.Context, options listing.Options) ([]models.User, int64, error) {
	r.lastOptions = options
	return r.employees, int64(len(r.employees)), nil
}

func (r *fakeEmployeeRepository) GetEmployee(_ context.Context, id string) (models.User, error) {
	for _, employee := range r.employees {
		if employee.ID == id {
			return employee, nil
		}
	}
	return models.User{}, domain.ErrNotFound
}

func (r *fakeEmployeeRepository) CreateEmployee(_ context.Context, _ string, employee models.User) (models.User, error) {
	r.created = append(r.created, employee)
	return employee, nil
}

func (r *fakeEmployeeRepository) CreateEmployees(_ context.Context, _ string, employees []models.User) ([]models.User, error) {
	r.created = append(r.created, employees...)
	return employees, nil
}

func (r *fakeEmployeeRepository) UpdateEmployee(_ context.Context, _ string, _ string, _ domain.EmployeePatch) (models.User, error) {
	return models.User{}, nil
}

func (r *fakeEmployeeRepository) AssignEmployeeRole(_ context.Context, _, _, _ string) (models.User, error) {
	return models.User{}, nil
}

func (r *fakeEmployeeRepository) ArchiveEmployee(_ context.Context, _ string, id string) error {
	r.archived = append(r.archived, id)
	return nil
}

// fakeScopeRepository mengembalikan daftar unit turunan yang statis.
type fakeScopeRepository struct {
	unitIDs []string
}

func (r *fakeScopeRepository) ListUnitScopeIDs(_ context.Context, _ string) ([]string, error) {
	return r.unitIDs, nil
}

func operatorPolicy() auth.Policy {
	return auth.StaticPolicy{
		"operator": {
			"employee:read:homebase":    true,
			"employee:create:homebase":  true,
			"employee:archive:homebase": true,
			"employee:import:homebase":  true,
		},
		"pegawai": {
			"employee:read:self": true,
		},
	}
}

func operatorPrincipal() auth.Principal {
	return auth.Principal{UserID: "operator-1", RoleName: "operator", HomebaseID: "unit-a"}
}

func TestEmployeeListLimitsOperatorToUnitScope(t *testing.T) {
	repo := &fakeEmployeeRepository{}
	scope := &fakeScopeRepository{unitIDs: []string{"unit-a", "unit-a-child"}}
	service := NewEmployeeService(repo, operatorPolicy(), scope)

	if _, err := service.List(context.Background(), operatorPrincipal(), listing.Options{Page: 1, PageSize: 20}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.lastOptions.ScopeHomebaseIDs) != 2 {
		t.Fatalf("scope = %v, want 2 units", repo.lastOptions.ScopeHomebaseIDs)
	}
}

func TestEmployeeListWithoutUnitAssignmentReturnsEmptyPage(t *testing.T) {
	repo := &fakeEmployeeRepository{}
	scope := &fakeScopeRepository{}
	service := NewEmployeeService(repo, operatorPolicy(), scope)

	principal := operatorPrincipal()
	principal.HomebaseID = ""
	page, err := service.List(context.Background(), principal, listing.Options{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page.Items) != 0 {
		t.Fatalf("items = %v, want empty", page.Items)
	}
	if repo.lastOptions.ScopeHomebaseIDs != nil {
		t.Fatal("repository must not be queried outside the assigned scope")
	}
}

func TestEmployeeListRejectsRoleWithoutReadPermission(t *testing.T) {
	repo := &fakeEmployeeRepository{}
	service := NewEmployeeService(repo, operatorPolicy(), &fakeScopeRepository{})

	principal := auth.Principal{UserID: "guest-1", RoleName: "tamu"}
	if _, err := service.List(context.Background(), principal, listing.Options{Page: 1, PageSize: 20}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("err = %v, want forbidden", err)
	}
}

func TestEmployeeGetRejectsEmployeeOutsideScope(t *testing.T) {
	repo := &fakeEmployeeRepository{employees: []models.User{
		{BaseModel: models.BaseModel{IDModel: models.IDModel{ID: "user-2"}}, HomebaseId: strPtr("unit-b")},
	}}
	service := NewEmployeeService(repo, operatorPolicy(), &fakeScopeRepository{unitIDs: []string{"unit-a"}})

	if _, err := service.Get(context.Background(), operatorPrincipal(), "user-2"); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("err = %v, want forbidden", err)
	}
}

func TestEmployeeCreateRejectsUnitOutsideScope(t *testing.T) {
	repo := &fakeEmployeeRepository{}
	service := NewEmployeeService(repo, operatorPolicy(), &fakeScopeRepository{unitIDs: []string{"unit-a"}})

	outsideUnit := "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
	input := domain.CreateEmployee{Nama: "Pegawai Baru", Email: "baru@example.test", HomebaseID: &outsideUnit}
	if _, err := service.Create(context.Background(), operatorPrincipal(), input); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("err = %v, want forbidden", err)
	}
	if len(repo.created) != 0 {
		t.Fatal("employee must not be created outside the assigned scope")
	}
}

func TestEmployeeImportAssignsOperatorUnit(t *testing.T) {
	repo := &fakeEmployeeRepository{}
	service := NewEmployeeService(repo, operatorPolicy(), &fakeScopeRepository{unitIDs: []string{"unit-a"}})

	employees := []models.User{{Nama: "Pegawai Impor", Email: "impor@example.test"}}
	if _, err := service.Import(context.Background(), operatorPrincipal(), employees); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if employees[0].HomebaseId == nil || *employees[0].HomebaseId != "unit-a" {
		t.Fatalf("homebase = %v, want unit-a", employees[0].HomebaseId)
	}
}

func TestEmployeeImportRejectsRoleWithoutImportPermission(t *testing.T) {
	repo := &fakeEmployeeRepository{}
	service := NewEmployeeService(repo, operatorPolicy(), &fakeScopeRepository{unitIDs: []string{"unit-a"}})

	employees := []models.User{{Nama: "Pegawai Impor", Email: "impor@example.test"}}
	principal := auth.Principal{UserID: "atasan-1", RoleName: "atasan"}
	if _, err := service.Import(context.Background(), principal, employees); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("err = %v, want forbidden", err)
	}
}

func strPtr(value string) *string {
	return &value
}
