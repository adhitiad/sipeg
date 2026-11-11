// Package listing memuat kontrak parameter daftar yang dipakai seluruh modul.
//
// Satu bentuk parameter dipakai untuk pagination, pencarian, penyaring
// cakupan unit, dan pengurutan. Nilai sort selalu berupa kunci allowlist
// milik resource; kolom SQL hanya boleh ditentukan di sisi repository.
package listing

import (
	"sort"
	"strings"
)

// MaxSearchLength adalah batas panjang teks pencarian yang diterima.
const MaxSearchLength = 100

// Options memuat parameter daftar yang seragam untuk seluruh resource.
type Options struct {
	// Search adalah teks pencarian bebas yang disaring per resource.
	Search string
	// HomebaseID menyaring satu unit tertentu.
	HomebaseID string
	// ScopeHomebaseIDs membatasi hasil ke unit-unit dalam cakupan principal.
	// Kosong berarti tidak ada pembatasan cakupan (hanya untuk role global).
	ScopeHomebaseIDs []string
	// UserID membatasi hasil ke satu pegawai (cakupan "self").
	UserID string
	// Jenis menyaring jenis unit organisasi.
	Jenis string
	// ParentID menyaring unit anak dari sebuah induk.
	ParentID string
	// Aksi menyaring jenis aksi pada jejak audit.
	Aksi string
	// Periode menyaring periode bulanan penggajian dalam bentuk "YYYY-MM".
	Periode string
	// Status menyaring status slip gaji.
	Status string
	// Sort adalah kunci sort dari allowlist resource.
	Sort string
	// Order adalah arah pengurutan: "asc" atau "desc".
	Order string
	// Page adalah nomor halaman, dimulai dari 1.
	Page int
	// PageSize adalah jumlah item per halaman.
	PageSize int
}

// Page memuat hasil daftar beserta metadata pagination.
type Page[T any] struct {
	Items      []T
	Page       int
	PageSize   int
	TotalItems int64
	TotalPages int
}

// EmptyPage menyusun halaman kosong dengan metadata pagination yang diminta.
func EmptyPage[T any](page, pageSize int) Page[T] {
	return Page[T]{Items: []T{}, Page: page, PageSize: pageSize}
}

// SortKeys mengembalikan kunci allowlist dalam urutan alfabetis.
func SortKeys(columns map[string]string) []string {
	keys := make([]string, 0, len(columns))
	for key := range columns {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// OrderClause menyusun klausa ORDER dari allowlist kolom dan arah.
// Nilai di luar allowlist jatuh ke kolom bawaan.
func OrderClause(columns map[string]string, requestedSort, requestedOrder, fallback string) string {
	column, ok := columns[requestedSort]
	if !ok {
		column = columns[fallback]
	}
	direction := "ASC"
	if strings.EqualFold(requestedOrder, "desc") {
		direction = "DESC"
	}
	return column + " " + direction
}

// PageCount menghitung jumlah halaman untuk sebuah total baris.
func PageCount(total int64, pageSize int) int {
	if total == 0 || pageSize <= 0 {
		return 0
	}
	return int((total + int64(pageSize) - 1) / int64(pageSize))
}
