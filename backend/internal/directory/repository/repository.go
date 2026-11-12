// Package repository memuat port persistensi dan adapter GORM untuk area
// direktori (pegawai, unit organisasi, kategori pegawai, dan jejak audit).
//
// Lapisan ini hanya concerned dengan penyimpanan: ia menerjemahkan parameter
// daftar dan perubahan domain menjadi query berparameter, menjaga transaksi
// tetap utuh, dan menulis jejak audit di dalam transaksi yang sama.
package repository

import (
	"context"
	"time"

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
	AssignEmployeeRole(context.Context, string, string, string) (models.User, error)
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

// LeaveRepository adalah port persistensi pengajuan cuti pegawai.
type LeaveRepository interface {
	ListCuti(context.Context, listing.Options) ([]models.Cuti, int64, error)
	GetCuti(context.Context, string) (models.Cuti, error)
	CreateCuti(context.Context, string, models.Cuti) (models.Cuti, error)
	UpdateCuti(context.Context, string, string, models.Cuti) (models.Cuti, error)
	ApproveCuti(context.Context, string, string, *string) (models.Cuti, error)
	RejectCuti(context.Context, string, string, *string) (models.Cuti, error)
	UserHomebaseID(context.Context, string) (string, error)
}

// AuditRepository adalah port persistensi jejak audit.
type AuditRepository interface {
	ListAuditLogs(context.Context, listing.Options) ([]models.AuditLog, int64, error)
	StreamAuditLogs(context.Context, listing.Options, func(models.AuditLog) error) error
	RecordAudit(context.Context, audit.Entry) error
}

// ScopeRepository adalah port resolusi cakupan unit pada pohon organisasi.
type ScopeRepository interface {
	ListUnitScopeIDs(context.Context, string) ([]string, error)
}

// KeluargaRepository adalah port persistensi data keluarga pegawai.
type KeluargaRepository interface {
	ListKeluarga(context.Context, listing.Options) ([]models.KeluargaPegawai, int64, error)
	GetKeluarga(context.Context, string) (models.KeluargaPegawai, error)
	CreateKeluarga(context.Context, string, models.KeluargaPegawai) (models.KeluargaPegawai, error)
	UpdateKeluarga(context.Context, string, string, domain.KeluargaPatch) (models.KeluargaPegawai, error)
	ArchiveKeluarga(context.Context, string, string) error
}

// GolonganRepository adalah port persistensi riwayat golongan pegawai.
type GolonganRepository interface {
	ListGolongan(context.Context, listing.Options) ([]models.RiwayatGolongan, int64, error)
	GetGolongan(context.Context, string) (models.RiwayatGolongan, error)
	CreateGolongan(context.Context, string, models.RiwayatGolongan) (models.RiwayatGolongan, error)
	UpdateGolongan(context.Context, string, string, domain.RiwayatGolonganPatch) (models.RiwayatGolongan, error)
	ArchiveGolongan(context.Context, string, string) error
}

// ArsipRepository adalah port persistensi arsip dokumen pegawai.
type ArsipRepository interface {
	ListArsip(context.Context, listing.Options) ([]models.ArsipDokumenPegawai, int64, error)
	GetArsip(context.Context, string) (models.ArsipDokumenPegawai, error)
	CreateArsip(context.Context, string, models.ArsipDokumenPegawai) (models.ArsipDokumenPegawai, error)
	UpdateArsip(context.Context, string, string, domain.ArsipDokumenPatch) (models.ArsipDokumenPegawai, error)
	ArchiveArsip(context.Context, string, string) error
}

// SertifikasiRepository adalah port persistensi riwayat sertifikasi pegawai.
type SertifikasiRepository interface {
	ListSertifikasi(context.Context, listing.Options) ([]models.RiwayatSertifikasi, int64, error)
	GetSertifikasi(context.Context, string) (models.RiwayatSertifikasi, error)
	CreateSertifikasi(context.Context, string, models.RiwayatSertifikasi) (models.RiwayatSertifikasi, error)
	UpdateSertifikasi(context.Context, string, string, domain.RiwayatSertifikasiPatch) (models.RiwayatSertifikasi, error)
	ArchiveSertifikasi(context.Context, string, string) error
}

// PelatihanRepository adalah port persistensi riwayat pelatihan pegawai.
type PelatihanRepository interface {
	ListPelatihan(context.Context, listing.Options) ([]models.RiwayatPelatihan, int64, error)
	GetPelatihan(context.Context, string) (models.RiwayatPelatihan, error)
	CreatePelatihan(context.Context, string, models.RiwayatPelatihan) (models.RiwayatPelatihan, error)
	UpdatePelatihan(context.Context, string, string, domain.RiwayatPelatihanPatch) (models.RiwayatPelatihan, error)
	ArchivePelatihan(context.Context, string, string) error
}

// SisterRepository adalah port persistensi pemetaan ID eksternal SISTER/PDDIKTI.
type SisterRepository interface {
	ListSister(context.Context, listing.Options) ([]models.SisterSyncMap, int64, error)
	GetSister(context.Context, string) (models.SisterSyncMap, error)
	CreateSister(context.Context, string, models.SisterSyncMap) (models.SisterSyncMap, error)
	UpdateSister(context.Context, string, string, domain.SisterSyncMapPatch) (models.SisterSyncMap, error)
	ArchiveSister(context.Context, string, string) error
	UpdateSisterSyncTime(context.Context, string, string, time.Time) (models.SisterSyncMap, error)
}

// PenghargaanRepository adalah port persistensi riwayat penghargaan pegawai.
type PenghargaanRepository interface {
	ListPenghargaan(context.Context, listing.Options) ([]models.RiwayatPenghargaan, int64, error)
	GetPenghargaan(context.Context, string) (models.RiwayatPenghargaan, error)
	CreatePenghargaan(context.Context, string, models.RiwayatPenghargaan) (models.RiwayatPenghargaan, error)
	UpdatePenghargaan(context.Context, string, string, domain.RiwayatPenghargaanPatch) (models.RiwayatPenghargaan, error)
	ArchivePenghargaan(context.Context, string, string) error
}
