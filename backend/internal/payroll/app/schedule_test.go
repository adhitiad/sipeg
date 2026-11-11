package app

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/models"
	"simpeg/backend/internal/notify"
	"simpeg/backend/internal/payroll/domain"
	"simpeg/backend/internal/payroll/repository"
)

func schedulePolicy() auth.Policy {
	return auth.StaticPolicy{
		"admin":    {"payroll:schedule": true, "payroll:read:any": true, "payroll:manage": true},
		"operator": {"payroll:schedule:homebase": true, "payroll:read:homebase": true},
		"pegawai":  {"payroll:read:self": true},
	}
}

func sampleSchedule(unitID string, day int) models.JadwalGaji {
	schedule := models.JadwalGaji{UnitId: unitID, TanggalGajian: day}
	schedule.ID = "jadwal-" + unitID
	unit := models.Homebase{Name: "Unit " + unitID}
	schedule.Unit = &unit
	return schedule
}

func TestUpsertScheduleAdminCanSetAnyUnit(t *testing.T) {
	repo := &fakePayrollRepository{}
	service := NewService(repo, schedulePolicy(), &fakeScopeRepository{})

	principal := auth.Principal{UserID: "admin-1", RoleName: "admin"}
	schedule, err := service.UpsertSchedule(context.Background(), principal, domain.ScheduleInput{UnitID: "unit-a", TanggalGajian: 25})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if schedule.TanggalGajian != 25 || schedule.UnitId != "unit-a" {
		t.Fatalf("schedule = %+v", schedule)
	}
}

func TestUpsertScheduleOperatorInsideScopeAllowed(t *testing.T) {
	repo := &fakePayrollRepository{}
	service := NewService(repo, schedulePolicy(), &fakeScopeRepository{unitIDs: []string{"unit-a", "unit-b"}})

	principal := auth.Principal{UserID: "operator-1", RoleName: "operator", HomebaseID: "unit-a"}
	if _, err := service.UpsertSchedule(context.Background(), principal, domain.ScheduleInput{UnitID: "unit-b", TanggalGajian: 25}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpsertScheduleOperatorOutsideScopeRejected(t *testing.T) {
	repo := &fakePayrollRepository{}
	service := NewService(repo, schedulePolicy(), &fakeScopeRepository{unitIDs: []string{"unit-a"}})

	principal := auth.Principal{UserID: "operator-1", RoleName: "operator", HomebaseID: "unit-a"}
	if _, err := service.UpsertSchedule(context.Background(), principal, domain.ScheduleInput{UnitID: "unit-c", TanggalGajian: 25}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("err = %v, want forbidden", err)
	}
}

func TestUpsertScheduleRejectsRoleWithoutPermission(t *testing.T) {
	repo := &fakePayrollRepository{}
	service := NewService(repo, schedulePolicy(), &fakeScopeRepository{})

	principal := auth.Principal{UserID: "pegawai-1", RoleName: "pegawai"}
	if _, err := service.UpsertSchedule(context.Background(), principal, domain.ScheduleInput{UnitID: "unit-a", TanggalGajian: 25}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("err = %v, want forbidden", err)
	}
}

func TestUpsertScheduleValidatesInput(t *testing.T) {
	repo := &fakePayrollRepository{}
	service := NewService(repo, schedulePolicy(), &fakeScopeRepository{})

	principal := auth.Principal{UserID: "admin-1", RoleName: "admin"}
	for _, day := range []int{0, 29} {
		if _, err := service.UpsertSchedule(context.Background(), principal, domain.ScheduleInput{UnitID: "unit-a", TanggalGajian: day}); !errors.Is(err, domain.ErrScheduleValidation) {
			t.Fatalf("day %d: err = %v, want schedule validation error", day, err)
		}
	}
}

func TestListSchedulesAdminSeesAllUnits(t *testing.T) {
	repo := &fakePayrollRepository{schedules: []models.JadwalGaji{sampleSchedule("unit-a", 25), sampleSchedule("unit-b", 28)}}
	service := NewService(repo, schedulePolicy(), &fakeScopeRepository{})

	principal := auth.Principal{UserID: "admin-1", RoleName: "admin"}
	schedules, err := service.ListSchedules(context.Background(), principal)
	if err != nil {
		t.Fatal(err)
	}
	if len(schedules) != 2 {
		t.Fatalf("schedules = %d, want 2", len(schedules))
	}
}

func TestListSchedulesOperatorSeesOnlyScope(t *testing.T) {
	repo := &fakePayrollRepository{schedules: []models.JadwalGaji{sampleSchedule("unit-a", 25), sampleSchedule("unit-b", 28)}}
	service := NewService(repo, schedulePolicy(), &fakeScopeRepository{unitIDs: []string{"unit-a"}})

	principal := auth.Principal{UserID: "operator-1", RoleName: "operator", HomebaseID: "unit-a"}
	schedules, err := service.ListSchedules(context.Background(), principal)
	if err != nil {
		t.Fatal(err)
	}
	if len(schedules) != 1 || schedules[0].UnitId != "unit-a" {
		t.Fatalf("schedules = %+v", schedules)
	}
}

func TestListSchedulesOperatorWithEmptyScopeSeesNothing(t *testing.T) {
	repo := &fakePayrollRepository{schedules: []models.JadwalGaji{sampleSchedule("unit-a", 25)}}
	service := NewService(repo, schedulePolicy(), &fakeScopeRepository{unitIDs: []string{}})

	principal := auth.Principal{UserID: "operator-1", RoleName: "operator", HomebaseID: ""}
	schedules, err := service.ListSchedules(context.Background(), principal)
	if err != nil {
		t.Fatal(err)
	}
	if len(schedules) != 0 {
		t.Fatalf("schedules = %+v, want empty", schedules)
	}
}

func TestProcessRemindersEnqueuesOnlyDueSlips(t *testing.T) {
	now := time.Date(2026, 11, 11, 3, 0, 0, 0, time.UTC) // 10:00 WIB
	repo := &fakePayrollRepository{dueReminders: []repository.Reminder{
		{SlipID: "slip-1", UserID: "user-1", Email: "budi@example.test", Nama: "Budi", Periode: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), TanggalGajian: 27},
		{SlipID: "slip-2", UserID: "user-2", Email: "andi@example.test", Nama: "Andi", Periode: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), TanggalGajian: 28},
	}}
	service := NewService(repo, schedulePolicy(), &fakeScopeRepository{})

	queued, err := service.ProcessReminders(context.Background(), now)
	if err != nil {
		t.Fatal(err)
	}
	if queued != 1 {
		t.Fatalf("queued = %d, want 1", queued)
	}
	if len(repo.enqueued) != 1 {
		t.Fatalf("enqueued = %d, want 1", len(repo.enqueued))
	}
	notification := repo.enqueued[0]
	if notification.Kind != notify.KindSlipAvailable {
		t.Fatalf("kind = %q", notification.Kind)
	}
	if notification.DedupeKey != "slip-available:slip-1" {
		t.Fatalf("dedupeKey = %q", notification.DedupeKey)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(notification.Payload), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["recipient"] != "budi@example.test" || payload["period"] != "2026-10" {
		t.Fatalf("payload = %v", payload)
	}
	for _, forbidden := range []string{"amount", "gajiPokok", "total", "nominal"} {
		if _, exists := payload[forbidden]; exists {
			t.Fatalf("payload leaked field %q", forbidden)
		}
	}
	if body := notification.Payload; strings.Contains(body, "4.250.000") || strings.Contains(body, "dibayar") {
		t.Fatalf("payload mengandung klaim pembayaran atau nominal: %s", body)
	}
}

func TestProcessRemindersSkipsEverythingWhenNoneDue(t *testing.T) {
	now := time.Date(2026, 11, 11, 3, 0, 0, 0, time.UTC)
	repo := &fakePayrollRepository{dueReminders: []repository.Reminder{
		{SlipID: "slip-2", UserID: "user-2", Email: "andi@example.test", Nama: "Andi", Periode: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), TanggalGajian: 27},
	}}
	service := NewService(repo, schedulePolicy(), &fakeScopeRepository{})

	queued, err := service.ProcessReminders(context.Background(), now)
	if err != nil {
		t.Fatal(err)
	}
	if queued != 0 || len(repo.enqueued) != 0 {
		t.Fatalf("queued = %d, enqueued = %d, want 0/0", queued, len(repo.enqueued))
	}
}
