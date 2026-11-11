// Package timeutil Concentrates the application's timezone policy.
//
// Storage policy: timestamps are persisted in UTC. The PostgreSQL columns are
// `timestamp without time zone`, so every write goes through `NowUTC` and the
// value read back is a UTC wall clock.
//
// Presentation policy: every date and time shown to a user, and every
// business rule that depends on the calendar day (payroll period, attendance
// date, leave cut-off), uses Asia/Jakarta (UTC+7, WIB). `InLocation` converts
// a stored timestamp into the operational timezone for display.
package timeutil

import (
	"fmt"
	"time"
)

// LocationName adalah nama zona waktu operasional aplikasi.
const LocationName = "Asia/Jakarta"

// Location adalah zona waktu operasional aplikasi: Asia/Jakarta (UTC+7, WIB).
//
// Zona ini dimuat sekali saat paket diinisialisasi. `time.LoadLocation` dapat
// gagal bila basis data zona waktu sistem tidak tersedia; pada kondisi itu
// aplikasi memakai offset tetap WIB (+07:00) agar operasi tanggal tidak
// berhenti total.
var Location = loadLocation()

// loadLocation memuat Asia/Jakarta dan jatuh ke offset WIB bila basis data
// zona waktu sistem tidak tersedia.
func loadLocation() *time.Location {
	location, err := time.LoadLocation(LocationName)
	if err == nil {
		return location
	}
	return time.FixedZone("WIB", 7*60*60)
}

// Now mengembalikan waktu sekarang pada zona waktu operasional.
func Now() time.Time {
	return time.Now().In(Location)
}

// NowUTC mengembalikan waktu sekarang dalam UTC untuk disimpan ke database.
func NowUTC() time.Time {
	return time.Now().UTC()
}

// InLocation mengubah timestamp tersimpan menjadi zona waktu operasional.
// Timestamp yang dibaca dari database adalah jam dinding UTC, sehingga
// konversi ini menghasilkan waktu lokal yang benar.
func InLocation(value time.Time) time.Time {
	if value.IsZero() {
		return value
	}
	return value.In(Location)
}

// FormatDate memformat tanggal pada zona waktu operasional.
func FormatDate(value time.Time) string {
	return InLocation(value).Format("02-01-2006")
}

// FormatDateTime memformat tanggal dan waktu pada zona waktu operasional.
func FormatDateTime(value time.Time) string {
	return InLocation(value).Format("02-01-2006 15:04") + " WIB"
}

// FormatPeriod memformat periode bulanan sebagai "YYYY-MM".
func FormatPeriod(value time.Time) string {
	return InLocation(value).Format("2006-01")
}

// ParsePeriod mengubah teks periode "YYYY-MM" menjadi tanggal awal periode
// pada zona waktu operasional.
func ParsePeriod(period string) (time.Time, error) {
	parsed, err := time.ParseInLocation("2006-01", period, Location)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse period %q: %w", period, err)
	}
	return parsed, nil
}

// PeriodKey mengembalikan kunci periode bulanan untuk sebuah waktu.
func PeriodKey(value time.Time) string {
	return FormatPeriod(value)
}

// MonthStart mengembalikan awal bulan pada zona waktu operasional.
func MonthStart(value time.Time) time.Time {
	local := InLocation(value)
	return time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, Location)
}

// AddMonths menambahkan sejumlah bulan pada sebuah waktu.
func AddMonths(value time.Time, months int) time.Time {
	return InLocation(value).AddDate(0, months, 0)
}

// DaysBetween menghitung selisih hari kalender antara dua waktu pada zona
// waktu operasional.
func DaysBetween(from, to time.Time) int {
	localFrom := time.Date(InLocation(from).Year(), InLocation(from).Month(), InLocation(from).Day(), 0, 0, 0, 0, Location)
	localTo := time.Date(InLocation(to).Year(), InLocation(to).Month(), InLocation(to).Day(), 0, 0, 0, 0, Location)
	return int(localTo.Sub(localFrom).Hours() / 24)
}

// DateOnly membalikkan sebuah waktu menjadi tanggal kalender pada zona waktu
// operasional, disimpan sebagai tengah malam UTC.
//
// Kolom tanggal pada skema acuan (`date`) menyimpan kalender tanpa jam. Nilai
// tengah malam UTC dipakai supaya tanggal yang dibaca kembali tetap sama
// setelah dikonversi ke zona waktu operasional.
func DateOnly(value time.Time) time.Time {
	local := InLocation(value)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)
}

// CalendarDate mengembalikan tahun, bulan, dan hari pada zona waktu
// operasional.
func CalendarDate(value time.Time) (int, time.Month, int) {
	local := InLocation(value)
	return local.Year(), local.Month(), local.Day()
}
