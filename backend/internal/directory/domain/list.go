package domain

import (
	"simpeg/backend/internal/listing"
)

// Allowlist kolom sortable per resource. Nilai allowlist adalah satu-satunya
// sumber kebenaran; kolom SQL tidak pernah berasal dari input mentah pengguna.
// Nama kolom memakai kutip ganda karena skema acuan memakai camelCase.
var (
	EmployeeSortColumns = map[string]string{
		"nama":      `"nama"`,
		"nip":       `"nip"`,
		"createdAt": `"createdAt"`,
		"updatedAt": `"updatedAt"`,
	}
	OrganizationSortColumns = map[string]string{
		"name":      `"name"`,
		"jenis":     `"jenis"`,
		"createdAt": `"createdAt"`,
		"updatedAt": `"updatedAt"`,
	}
	AuditLogSortColumns = map[string]string{
		"createdAt": `"createdAt"`,
		"aksi":      `"aksi"`,
	}
	CategorySortColumns = map[string]string{
		"kode":      `"kode"`,
		"nama":      `"nama"`,
		"createdAt": `"createdAt"`,
		"updatedAt": `"updatedAt"`,
	}
)

// EmployeeSortFields mengembalikan kunci sort yang valid untuk daftar pegawai.
func EmployeeSortFields() []string { return listing.SortKeys(EmployeeSortColumns) }

// OrganizationSortFields mengembalikan kunci sort yang valid untuk daftar organisasi.
func OrganizationSortFields() []string { return listing.SortKeys(OrganizationSortColumns) }

// AuditLogSortFields mengembalikan kunci sort yang valid untuk daftar audit.
func AuditLogSortFields() []string { return listing.SortKeys(AuditLogSortColumns) }

// CategorySortFields mengembalikan kunci sort yang valid untuk daftar kategori.
func CategorySortFields() []string { return listing.SortKeys(CategorySortColumns) }
