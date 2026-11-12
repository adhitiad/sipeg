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
	KeluargaSortColumns = map[string]string{
		"nama":             `"nama"`,
		"nik":              `"nik"`,
		"tanggalLahir":     `"tanggalLahir"`,
		"statusTanggungan": `"statusTanggungan"`,
		"createdAt":        `"createdAt"`,
		"updatedAt":        `"updatedAt"`,
	}
	GolonganSortColumns = map[string]string{
		"golongan":  `"golongan"`,
		"ruang":     `"ruang"`,
		"tmt":       `"tmt"`,
		"createdAt": `"createdAt"`,
		"updatedAt": `"updatedAt"`,
	}
	ArsipSortColumns = map[string]string{
		"jenisDokumen":  `"jenisDokumen"`,
		"judul":         `"judul"`,
		"berlakuSampai": `"berlakuSampai"`,
		"createdAt":     `"createdAt"`,
		"updatedAt":     `"updatedAt"`,
	}
	SertifikasiSortColumns = map[string]string{
		"jenisSertifikasi": `"jenisSertifikasi"`,
		"namaSertifikat":   `"namaSertifikat"`,
		"status":           `"status"`,
		"tanggalTerbit":    `"tanggalTerbit"`,
		"createdAt":        `"createdAt"`,
		"updatedAt":        `"updatedAt"`,
	}
	PelatihanSortColumns = map[string]string{
		"namaPelatihan":  `"namaPelatihan"`,
		"jenisPelatihan": `"jenisPelatihan"`,
		"tanggalMulai":   `"tanggalMulai"`,
		"createdAt":      `"createdAt"`,
		"updatedAt":      `"updatedAt"`,
	}
	SisterSortColumns = map[string]string{
		"sistemEksternal": `"sistemEksternal"`,
		"idEksternal":     `"idEksternal"`,
		"status":          `"status"`,
		"createdAt":       `"createdAt"`,
		"updatedAt":       `"updatedAt"`,
	}
	PenghargaanSortColumns = map[string]string{
		"namaPenghargaan":  `"namaPenghargaan"`,
		"jenisPenghargaan": `"jenisPenghargaan"`,
		"tanggalTerima":    `"tanggalTerima"`,
		"createdAt":        `"createdAt"`,
		"updatedAt":        `"updatedAt"`,
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

// KeluargaSortFields mengembalikan kunci sort yang valid untuk daftar keluarga pegawai.
func KeluargaSortFields() []string { return listing.SortKeys(KeluargaSortColumns) }

// GolonganSortFields mengembalikan kunci sort yang valid untuk daftar riwayat golongan.
func GolonganSortFields() []string { return listing.SortKeys(GolonganSortColumns) }

// ArsipSortFields mengembalikan kunci sort yang valid untuk daftar arsip dokumen.
func ArsipSortFields() []string { return listing.SortKeys(ArsipSortColumns) }

// SertifikasiSortFields mengembalikan kunci sort yang valid untuk daftar sertifikasi.
func SertifikasiSortFields() []string { return listing.SortKeys(SertifikasiSortColumns) }

// PelatihanSortFields mengembalikan kunci sort yang valid untuk daftar pelatihan.
func PelatihanSortFields() []string { return listing.SortKeys(PelatihanSortColumns) }

// SisterSortFields mengembalikan kunci sort yang valid untuk daftar pemetaan SISTER.
func SisterSortFields() []string { return listing.SortKeys(SisterSortColumns) }

// PenghargaanSortFields mengembalikan kunci sort yang valid untuk daftar penghargaan.
func PenghargaanSortFields() []string { return listing.SortKeys(PenghargaanSortColumns) }
