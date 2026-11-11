package httpserver

import (
	"net/http"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/directory/app"
)

// DirectoryServices memuat use case yang dipakai oleh modul direktori.
type DirectoryServices struct {
	Employees     *app.EmployeeService
	Organizations *app.OrganizationService
	Categories    *app.CategoryService
	Audits        *app.AuditService
}

// RegisterDirectoryRoutes mendaftarkan seluruh endpoint direktori.
func RegisterDirectoryRoutes(mux *http.ServeMux, resolver auth.SessionResolver, policy auth.Policy, services DirectoryServices) {
	api := directoryAPI{
		employees:     services.Employees,
		organizations: services.Organizations,
		categories:    services.Categories,
		audits:        services.Audits,
	}
	protected := func(permission auth.Permission, handler http.Handler) http.Handler {
		return auth.Authenticate(resolver)(auth.RequirePermission(policy, permission, handler))
	}
	protectedAny := func(permissions []auth.Permission, handler http.Handler) http.Handler {
		return auth.Authenticate(resolver)(auth.RequireAnyPermission(policy, permissions, handler))
	}

	mux.Handle("GET "+APIPrefix+"/employees", protectedAny([]auth.Permission{"employee:read:any", "employee:read:homebase", "employee:read:self"}, http.HandlerFunc(api.listEmployees)))
	mux.Handle("POST "+APIPrefix+"/employees", protectedAny([]auth.Permission{"employee:create", "employee:create:homebase"}, http.HandlerFunc(api.createEmployee)))
	mux.Handle("GET "+APIPrefix+"/employees/import-template.csv", protectedAny([]auth.Permission{"employee:import", "employee:import:homebase"}, http.HandlerFunc(api.employeeImportTemplate)))
	mux.Handle("POST "+APIPrefix+"/employees/import", protectedAny([]auth.Permission{"employee:import", "employee:import:homebase"}, http.HandlerFunc(api.importEmployees)))
	mux.Handle("GET "+APIPrefix+"/employees/{employeeId}", protectedAny([]auth.Permission{"employee:read:any", "employee:read:homebase", "employee:read:self"}, http.HandlerFunc(api.getEmployee)))
	mux.Handle("PATCH "+APIPrefix+"/employees/{employeeId}", protectedAny([]auth.Permission{"employee:update:any", "employee:update:homebase"}, http.HandlerFunc(api.updateEmployee)))
	mux.Handle("DELETE "+APIPrefix+"/employees/{employeeId}", protectedAny([]auth.Permission{"employee:archive", "employee:archive:homebase"}, http.HandlerFunc(api.archiveEmployee)))

	mux.Handle("GET "+APIPrefix+"/organizations", protectedAny([]auth.Permission{"organization:read", "organization:read:homebase"}, http.HandlerFunc(api.listOrganizations)))
	mux.Handle("POST "+APIPrefix+"/organizations", protectedAny([]auth.Permission{"organization:manage", "organization:manage:homebase"}, http.HandlerFunc(api.createOrganization)))
	mux.Handle("GET "+APIPrefix+"/organizations/{organizationID}", protectedAny([]auth.Permission{"organization:read", "organization:read:homebase"}, http.HandlerFunc(api.getOrganization)))
	mux.Handle("PATCH "+APIPrefix+"/organizations/{organizationID}", protectedAny([]auth.Permission{"organization:manage", "organization:manage:homebase"}, http.HandlerFunc(api.updateOrganization)))
	mux.Handle("DELETE "+APIPrefix+"/organizations/{organizationID}", protectedAny([]auth.Permission{"organization:manage", "organization:manage:homebase"}, http.HandlerFunc(api.archiveOrganization)))

	mux.Handle("GET "+APIPrefix+"/audit-logs", protectedAny([]auth.Permission{"audit:read", "audit:read:homebase"}, http.HandlerFunc(api.listAuditLogs)))
	mux.Handle("GET "+APIPrefix+"/categories", protected("category:read", http.HandlerFunc(api.listCategories)))
}
