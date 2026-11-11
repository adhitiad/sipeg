package domain

import (
	"errors"
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
	if payday.Day() != 25 || payday.Month() != time.November || payday.Year() != 2026 {
		t.Fatalf("payday = %v", payday)
	}
	if payday.Location() != time.UTC {
		t.Fatalf("payday location = %v, want UTC", payday.Location())
	}
}

func TestNotifyDateIsCalendarNextDay(t *testing.T) {
	// 28 November 2026 adalah Sabtu: pemberitahuan tetap berjalan pada
	// tanggal kalender berikutnya dan tidak digeser ke hari kerja.
	period := Period{Year: 2026, Month: time.November}
	notify := NotifyDate(period, 27)
	if notify.Day() != 28 || notify.Month() != time.November {
		t.Fatalf("notify = %v, want 2026-11-28", notify)
	}
}

func TestNotifyDateRollsIntoNextMonthFromPayday28(t *testing.T) {
	period := Period{Year: 2026, Month: time.February}
	notify := NotifyDate(period, 28)
	if notify.Month() != time.March || notify.Day() != 1 {
		t.Fatalf("notify = %v, want 2026-03-01", notify)
	}
}

func TestIsReminderDueAroundNotifyDate(t *testing.T) {
	period := Period{Year: 2026, Month: time.October}
	payday := 27

	before := timeutil.DateOnly(time.Date(2026, 10, 27, 12, 0, 0, 0, timeutil.Location))
	if IsReminderDue(period, payday, before) {
		t.Fatal("pemberitahuan belum jatuh tempo sebelum H+1")
	}

	sameDay := timeutil.DateOnly(time.Date(2026, 10, 28, 0, 30, 0, 0, timeutil.Location))
	if !IsReminderDue(period, payday, sameDay) {
		t.Fatal("pemberitahuan harus jatuh tempo pada H+1")
	}

	after := timeutil.DateOnly(time.Date(2026, 11, 10, 0, 0, 0, 0, timeutil.Location))
	if !IsReminderDue(period, payday, after) {
		t.Fatal("pemberitahuan tetap jatuh tempo setelah H+1 lewat")
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
