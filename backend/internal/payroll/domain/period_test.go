package domain

import (
	"errors"
	"testing"
	"time"

	"simpeg/backend/internal/models"
)

func TestParsePeriodAcceptsValidValue(t *testing.T) {
	period, err := ParsePeriod("2026-11")
	if err != nil {
		t.Fatal(err)
	}
	if period.Year != 2026 || period.Month != time.November {
		t.Fatalf("period = %+v", period)
	}
	if period.Key() != "2026-11" {
		t.Fatalf("key = %q", period.Key())
	}
	if period.Label() != "November 2026" {
		t.Fatalf("label = %q", period.Label())
	}
}

func TestParsePeriodRejectsInvalidValue(t *testing.T) {
	if _, err := ParsePeriod("2026-13"); !errors.Is(err, ErrInvalidPeriod) {
		t.Fatalf("err = %v, want invalid period", err)
	}
	if _, err := ParsePeriod("bukan periode"); !errors.Is(err, ErrInvalidPeriod) {
		t.Fatalf("err = %v, want invalid period", err)
	}
}

func TestPeriodUsesOperationalTimezone(t *testing.T) {
	period, err := ParsePeriod("2026-11")
	if err != nil {
		t.Fatal(err)
	}
	start := period.Start()
	if start.Hour() != 0 || start.Day() != 1 || start.Month() != time.November {
		t.Fatalf("start = %v", start)
	}
	end := period.End()
	if end.Day() != 30 || end.Hour() != 23 {
		t.Fatalf("end = %v", end)
	}
}

func TestPeriodContainsUsesCalendarDay(t *testing.T) {
	period, err := ParsePeriod("2026-11")
	if err != nil {
		t.Fatal(err)
	}
	inside := time.Date(2026, 11, 15, 9, 0, 0, 0, time.UTC)
	outside := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	if !period.Contains(inside) {
		t.Fatal("expected the timestamp to be inside the period")
	}
	if period.Contains(outside) {
		t.Fatal("expected the timestamp to be outside the period")
	}
}

func TestCanMarkPaidOnlyAcceptsPublishedSlip(t *testing.T) {
	if CanMarkPaid(models.StatusSlipDraf) {
		t.Fatal("a draft slip must not be marked as paid")
	}
	if !CanMarkPaid(models.StatusSlipTerbit) {
		t.Fatal("a published slip must be markable as paid")
	}
	if CanMarkPaid(models.StatusSlipTerbayar) {
		t.Fatal("a paid slip must not be paid twice")
	}
}

func TestPaymentInputValidate(t *testing.T) {
	valid := PaymentInput{
		PelakuPengesahan: "Bendahara",
		TanggalBayar:     time.Date(2026, 11, 28, 0, 0, 0, 0, time.UTC),
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	missingActor := valid
	missingActor.PelakuPengesahan = "   "
	if err := missingActor.Validate(); !errors.Is(err, ErrValidation) {
		t.Fatalf("err = %v, want validation error", err)
	}

	missingDate := valid
	missingDate.TanggalBayar = time.Time{}
	if err := missingDate.Validate(); !errors.Is(err, ErrValidation) {
		t.Fatalf("err = %v, want validation error", err)
	}
}

func TestStatusLabel(t *testing.T) {
	if StatusLabel(models.StatusSlipTerbayar) != "Terbayar" {
		t.Fatalf("label = %q", StatusLabel(models.StatusSlipTerbayar))
	}
}
