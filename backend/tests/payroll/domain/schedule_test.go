package domain

import (
	"errors"
	. "simpeg/backend/internal/payroll/domain"
	"testing"
	"time"

	"simpeg/backend/internal/timeutil"
)

func TestScheduleInputValidate(t *testing.T) {
	testCases := []struct {
		name    string
		input   ScheduleInput
		wantErr bool
	}{
		{name: "tanggal 1 sah", input: ScheduleInput{UnitID: "unit-a", TanggalGajian: 1}},
		{name: "tanggal 28 sah", input: ScheduleInput{UnitID: "unit-a", TanggalGajian: 28}, wantErr: false},
		{name: "tanggal 0 ditolak", input: ScheduleInput{UnitID: "unit-a", TanggalGajian: 0}, wantErr: true},
		{name: "tanggal 29 ditolak", input: ScheduleInput{UnitID: "unit-a", TanggalGajian: 29}, wantErr: true},
		{name: "tanggal negatif ditolak", input: ScheduleInput{UnitID: "unit-a", TanggalGajian: -1}, wantErr: true},
		{name: "unit kosong ditolak", input: ScheduleInput{UnitID: "   ", TanggalGajian: 25}, wantErr: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := testCase.input.Validate()
			if testCase.wantErr {
				if !errors.Is(err, ErrScheduleValidation) {
					t.Fatalf("err = %v, want schedule validation error", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestPaydayDateIsCalendarDate(t *testing.T) {
	period := Period{Year: 2026, Month: time.November}
	payday := PaydayDate(period, 25)
	if payday.Day() != 25 || payday.Month() != time.December || payday.Year() != 2026 {
		t.Fatalf("payday = %v, want 2026-12-25 (slip periode November dibayar 25 Desember)", payday)
	}
	if payday.Location() != time.UTC {
		t.Fatalf("payday location = %v, want UTC", payday.Location())
	}
}

func TestNotifyDateIsCalendarNextDay(t *testing.T) {
	// 25 Desember 2026 adalah Jumat: pemberitahuan tetap berjalan pada
	// tanggal kalender H-1 dan tidak digeser ke hari kerja.
	period := Period{Year: 2026, Month: time.November}
	notify := NotifyDate(period, 25)
	if notify.Day() != 24 || notify.Month() != time.December {
		t.Fatalf("notify = %v, want 2026-12-24", notify)
	}
}

func TestNotifyDateRollsIntoNextMonthFromPayday28(t *testing.T) {
	// Slip periode Februari dibayar 28 Maret (bukan 28 Februari) — periode
	// bulan sebelumnya sesuai keputusan pemilik 2026-11-14.
	period := Period{Year: 2026, Month: time.February}
	notify := NotifyDate(period, 28)
	if notify.Month() != time.March || notify.Day() != 27 {
		t.Fatalf("notify = %v, want 2026-03-27", notify)
	}
}

func TestIsReminderDueAroundNotifyDate(t *testing.T) {
	period := Period{Year: 2026, Month: time.October}
	payday := 27

	before := timeutil.DateOnly(time.Date(2026, 11, 25, 12, 0, 0, 0, timeutil.Location))
	if IsReminderDue(period, payday, before) {
		t.Fatal("pemberitahuan belum jatuh tempo sebelum H-1")
	}

	sameDay := timeutil.DateOnly(time.Date(2026, 11, 26, 0, 30, 0, 0, timeutil.Location))
	if !IsReminderDue(period, payday, sameDay) {
		t.Fatal("pemberitahuan harus jatuh tempo pada H-1")
	}

	after := timeutil.DateOnly(time.Date(2026, 12, 10, 0, 0, 0, 0, timeutil.Location))
	if !IsReminderDue(period, payday, after) {
		t.Fatal("pemberitahuan tetap jatuh tempo setelah H-1 lewat")
	}
}

func TestIsReminderDueUsesCurrentPeriodSchedule(t *testing.T) {
	// Periode berjalan dengan tanggal gajian akhir bulan belum boleh
	// diberitahukan pada tanggal 11.
	period := Period{Year: 2026, Month: time.November}
	today := timeutil.DateOnly(time.Date(2026, 11, 11, 8, 0, 0, 0, timeutil.Location))
	if IsReminderDue(period, 28, today) {
		t.Fatal("periode berjalan belum jatuh tempo")
	}
}
