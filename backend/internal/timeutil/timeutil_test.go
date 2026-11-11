package timeutil

import (
	"testing"
	"time"
)

func TestLocationIsAsiaJakarta(t *testing.T) {
	_, offset := time.Now().In(Location).Zone()
	if offset != 7*60*60 {
		t.Fatalf("offset = %d, want %d", offset, 7*60*60)
	}
}

func TestInLocationConvertsStoredUTCTimestamp(t *testing.T) {
	stored := time.Date(2026, 11, 11, 2, 0, 0, 0, time.UTC)

	local := InLocation(stored)
	if local.Hour() != 9 {
		t.Fatalf("hour = %d, want 9", local.Hour())
	}
	if local.Location().String() != LocationName && local.Location().String() != "WIB" {
		t.Fatalf("location = %q", local.Location().String())
	}
}

func TestInLocationKeepsZeroValue(t *testing.T) {
	if !InLocation(time.Time{}).IsZero() {
		t.Fatal("expected zero timestamp to stay zero")
	}
}

func TestFormatDateUsesOperationalTimezone(t *testing.T) {
	stored := time.Date(2026, 11, 11, 20, 30, 0, 0, time.UTC)

	if got := FormatDate(stored); got != "12-11-2026" {
		t.Fatalf("FormatDate = %q", got)
	}
	if got := FormatDateTime(stored); got != "12-11-2026 03:30 WIB" {
		t.Fatalf("FormatDateTime = %q", got)
	}
}

func TestParsePeriodUsesOperationalTimezone(t *testing.T) {
	parsed, err := ParsePeriod("2026-11")
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Year() != 2026 || parsed.Month() != time.November || parsed.Day() != 1 {
		t.Fatalf("parsed = %v", parsed)
	}
	if parsed.Location().String() != LocationName && parsed.Location().String() != "WIB" {
		t.Fatalf("location = %q", parsed.Location().String())
	}
}

func TestParsePeriodRejectsInvalidInput(t *testing.T) {
	if _, err := ParsePeriod("2026-13"); err == nil {
		t.Fatal("expected error for invalid period")
	}
}

func TestPeriodKeyRoundTrip(t *testing.T) {
	value := time.Date(2026, 3, 15, 8, 0, 0, 0, time.UTC)

	period := PeriodKey(value)
	if period != "2026-03" {
		t.Fatalf("PeriodKey = %q", period)
	}
	parsed, err := ParsePeriod(period)
	if err != nil {
		t.Fatal(err)
	}
	if MonthStart(parsed).Day() != 1 {
		t.Fatal("expected month start")
	}
}

func TestDaysBetweenCountsCalendarDays(t *testing.T) {
	from := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	to := time.Date(2026, 11, 3, 5, 0, 0, 0, time.UTC)

	if got := DaysBetween(from, to); got != 2 {
		t.Fatalf("DaysBetween = %d, want 2", got)
	}
}
