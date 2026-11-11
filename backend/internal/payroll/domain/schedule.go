package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"simpeg/backend/internal/timeutil"
)

// Batas tanggal gajian. Nilai 1-28 menghindari pergeseran bulan: setiap
// bulan memiliki tanggal 28, sehingga jadwal tetap tidak pernah melompat ke
// bulan berikutnya.
const (
	MinPaydayDay = 1
	MaxPaydayDay = 28
)

// ErrScheduleValidation menandai nilai jadwal gajian yang tidak sah.
var ErrScheduleValidation = errors.New("payroll schedule value is not valid")

// ScheduleInput adalah tanggal gajian tetap tiap bulan untuk satu unit
// organisasi.
//
// Jadwal bersifat tetap, bukan per periode: satu nilai per unit berlaku untuk
// setiap bulan. Pemberitahuan slip gaji dikirim pada hari kalender setelah
// tanggal gajian tanpa penyesuaian akhir pekan maupun hari libur nasional.
type ScheduleInput struct {
	// UnitID adalah unit organisasi pemilik jadwal.
	UnitID string
	// TanggalGajian adalah tanggal tetap tiap bulan, antara 1 sampai 28.
	TanggalGajian int
}

// Validate memeriksa aturan jadwal gajian.
func (input ScheduleInput) Validate() error {
	if strings.TrimSpace(input.UnitID) == "" {
		return fmt.Errorf("%w: unit wajib diisi", ErrScheduleValidation)
	}
	if input.TanggalGajian < MinPaydayDay || input.TanggalGajian > MaxPaydayDay {
		return fmt.Errorf("%w: tanggal gajian harus %d-%d", ErrScheduleValidation, MinPaydayDay, MaxPaydayDay)
	}
	return nil
}

// PaydayDate mengembalikan tanggal gajian sebuah periode.
//
// Nilai disimpan sebagai tanggal kalender (tengah malam UTC) agar perbandingan
// dengan tanggal lain tidak dipengaruhi zona waktu.
func PaydayDate(period Period, day int) time.Time {
	return timeutil.DateOnly(time.Date(period.Year, period.Month, day, 0, 0, 0, 0, timeutil.Location))
}

// NotifyDate mengembalikan tanggal pemberitahuan H+1 untuk sebuah periode.
//
// H+1 memakai tanggal kalender tetap: hari setelah tanggal gajian, tanpa
// digeser ke hari kerja berikutnya walau jatuh pada Sabtu, Minggu, atau hari
// libur nasional (keputusan 11 November 2026).
func NotifyDate(period Period, day int) time.Time {
	return PaydayDate(period, day).AddDate(0, 0, 1)
}

// IsReminderDue memberitahu apakah pemberitahuan H+1 sebuah periode sudah
// jatuh tempo pada hari `today`.
//
// Perbandingan memakai tanggal kalender, sehingga hasil tidak berubah karena
// jam pengiriman.
func IsReminderDue(period Period, day int, today time.Time) bool {
	return !NotifyDate(period, day).After(timeutil.DateOnly(today))
}
