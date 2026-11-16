// Package domain memuat aturan perpanjangan masa kerja dosen pasca-BUP
// (DEC-083): kontrak tahunan dosen tidak tetap.
package domain

import "errors"

// KategoriDosenTidakTetapID adalah id baris `DOSEN_TIDAK_TETAP` pada tabel
// `kategoriStatusPegawai` (di-seed lewat migrasi 000029_kategori_dosen_tidak_tetap.sql).
const KategoriDosenTidakTetapID = "d0000000-0000-0000-0000-000000000006"

// DurasiBulanPerjanjian adalah panjang perjanjian kerja tahunan pasca-BUP
// (perjanjian kerja tahunan: 12 bulan).
const DurasiBulanPerjanjian = 12

// ErrInvalidInput menandai isian perpanjangan yang tidak sah.
var ErrInvalidInput = errors.New("input perpanjangan tidak sah")

// ErrNotFound menandai pegawai/kontrak yang tidak ditemukan.
var ErrNotFound = errors.New("pegawai atau kontrak tidak ditemukan")

// ErrOverlap menandai perjanjian kerja baru yang tumpang tindih dengan
// kontrak AKTIF pegawai (`DEC-083` gunakan `allowOverlap` untuk memaksa
// dan mencatat override di audit).
var ErrOverlap = errors.New("kontrak aktif tumpang tindih dengan perjanjian kerja baru")

// ErrForbidden menandai penolakan perpanjangan.
var ErrForbidden = errors.New("tidak berwenang memproses perpanjangan")
