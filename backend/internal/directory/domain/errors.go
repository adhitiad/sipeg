// Package domain memuat aturan bisnis direktori: parameter daftar yang
// seragam, allowlist kolom sortable, validasi field, dan pemetaan error.
//
// Lapisan ini tidak melakukan akses database maupun HTTP. Aturan di sini
// menjadi satu-satunya sumber kebenaran atas validitas data yang dipakai
// oleh lapisan aplikasi (logika use case) dan lapisan transport.
package domain

import "errors"

var (
	// ErrNotFound menandai record yang diminta tidak ada.
	ErrNotFound = errors.New("directory record not found")
	// ErrConflict menandai record yang bertentangan dengan data yang ada.
	ErrConflict = errors.New("directory record conflicts with existing data")
	// ErrInvalidRef menandai referensi yang tidak menunjuk record valid.
	ErrInvalidRef = errors.New("directory reference is invalid")
	// ErrInvalidParent menandai induk unit yang tidak valid.
	ErrInvalidParent = errors.New("organization parent is invalid")
	// ErrHasActiveMember menandai unit yang masih memiliki anggota atau anak aktif.
	ErrHasActiveMember = errors.New("organization has active members or children")
	// ErrForbidden menandai aksi yang di luar cakupan atau permission principal.
	ErrForbidden = errors.New("action is not permitted for this principal")
	// ErrValidation menandai nilai yang melanggar aturan validasi.
	ErrValidation = errors.New("value is not valid")
)

// EmployeeConflictError menandai benturan identitas pegawai pada satu baris
// impor. Index menunjuk posisi baris yang bentrok.
type EmployeeConflictError struct {
	Index int
	Field string
}

func (e *EmployeeConflictError) Error() string { return "employee identity conflict" }
func (e *EmployeeConflictError) Unwrap() error { return ErrConflict }
