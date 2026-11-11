// Package repository memuat port persistensi dan adapter GORM untuk area
// direktori (pegawai, unit organisasi, kategori pegawai, dan jejak audit).
//
// Lapisan ini hanya concerned dengan penyimpanan: ia menerjemahkan parameter
// daftar dan perubahan domain menjadi query berparameter, menjaga transaksi
// tetap utuh, dan menulis jejak audit di dalam transaksi yang sama.
package repository

import (
	"context"

	"simpeg/backend/internal/audit"
	"simpeg/backend/internal/directory/domain"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
)

// EmployeeRepository adalah port persistensi data pegawai.
type EmployeeRepository interface {
	ListEmployees(context.Context, listing.Options) ([]models.User, int64, error)
	GetEmployee(context.Context, string) (models.User, error)
	CreateEmployee(context.Context, string, models.User) (models.User, error)
	CreateEmployees(context.Context, string, []models.User) ([]models.User, error)
	UpdateEmployee(context.Context, string, string, domain.EmployeePatch) (models.User, error)
	ArchiveEmployee(context.Context, string, string) error
}

// OrganizationRepository adalah port persistensi unit organisasi.
type OrganizationRepository interface {
	ListOrganizations(context.Context, listing.Options) ([]models.Homebase, int64, error)
	GetOrganization(context.Context, string) (models.Homebase, error)
	CreateOrganization(context.Context, string, models.Homebase) (models.Homebase, error)
	UpdateOrganization(context.Context, string, string, domain.OrganizationPatch) (models.Homebase, error)
	ArchiveOrganization(context.Context, string, string) error
}

// CategoryRepository adalah port persistensi kategori pegawai.
type CategoryRepository interface {
	ListCategories(context.Context, listing.Options) ([]models.Kategori, int64, error)
}

// AuditRepository adalah port persistensi jejak audit.
type AuditRepository interface {
	ListAuditLogs(context.Context, listing.Options) ([]models.AuditLog, int64, error)
	RecordAudit(context.Context, audit.Entry) error
}

// ScopeRepository adalah port resolusi cakupan unit pada pohon organisasi.
type ScopeRepository interface {
	ListUnitScopeIDs(context.Context, string) ([]string, error)
}
