// Package domain memuat aturan bisnis penggajian: periode bulanan, transisi
// status slip, dan validasi konfirmasi pembayaran.
//
// Lapisan ini tidak melakukan akses database maupun HTTP. Seluruh nominal
// pada sistem dinyatakan dalam Rupiah Indonesia.
package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"simpeg/backend/internal/timeutil"
)

var (
	// ErrInvalidPeriod menandai periode bulanan yang tidak sah.
	ErrInvalidPeriod = errors.New("payroll period is not valid")
	// ErrValidation menandai nilai yang melanggar aturan penggajian.
	ErrValidation = errors.New("payroll value is not valid")
	// ErrNotFound menandai slip gaji yang diminta tidak ada.
	ErrNotFound = errors.New("payroll slip not found")
	// ErrConflict menandai slip yang sudah dibayar atau statusnya tidak
	// memungkinkan untuk dibayar.
	ErrConflict = errors.New("payroll slip conflicts with existing data")
	// ErrForbidden menandai aksi yang di luar cakupan atau permission principal.
	ErrForbidden = errors.New("action is not permitted for this principal")
)

// Period adalah periode bulanan penggajian.
type Period struct {
	Year  int
	Month time.Month
}

// monthNames memuat nama bulan dalam Bahasa Indonesia.
var monthNames = map[time.Month]string{
	time.January: "Januari", time.February: "Februari", time.March: "Maret",
	time.April: "April", time.May: "Mei", time.June: "Juni",
	time.July: "Juli", time.August: "Agustus", time.September: "September",
	time.October: "Oktober", time.November: "November", time.December: "Desember",
}

// ParsePeriod mengubah teks periode "YYYY-MM" menjadi periode penggajian.
func ParsePeriod(value string) (Period, error) {
	trimmed := strings.TrimSpace(value)
	parsed, err := timeutil.ParsePeriod(trimmed)
	if err != nil {
		return Period{}, fmt.Errorf("%w: %q", ErrInvalidPeriod, trimmed)
	}
	return Period{Year: parsed.Year(), Month: parsed.Month()}, nil
}

// Key mengembalikan kunci periode "YYYY-MM".
func (p Period) Key() string {
	return fmt.Sprintf("%04d-%02d", p.Year, int(p.Month))
}

// Label mengembalikan nama periode dalam Bahasa Indonesia, misalnya
// "November 2026".
func (p Period) Label() string {
	name, ok := monthNames[p.Month]
	if !ok {
		name = p.Month.String()
	}
	return fmt.Sprintf("%s %d", name, p.Year)
}

// Start mengembalikan awal periode pada zona waktu operasional.
func (p Period) Start() time.Time {
	return time.Date(p.Year, p.Month, 1, 0, 0, 0, 0, timeutil.Location)
}

// End mengembalikan akhir periode pada zona waktu operasional.
func (p Period) End() time.Time {
	return p.Start().AddDate(0, 1, 0).Add(-time.Second)
}

// Contains melaporkan apakah sebuah waktu berada di dalam periode.
func (p Period) Contains(value time.Time) bool {
	local := timeutil.InLocation(value)
	return !local.Before(p.Start()) && !local.After(p.End())
}
