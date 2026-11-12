package repository

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"testing"
	"time"

	"simpeg/backend/internal/payroll/domain"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestUpsertScheduleRejectsUnknownUnit(t *testing.T) {
	db, mock := newPayrollDB(t)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "Homebase" WHERE id = $1`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectRollback()

	store := GormStore{DB: db}
	_, err := store.UpsertSchedule(context.Background(), "admin-1", domain.ScheduleInput{UnitID: "unit-x", TanggalGajian: 25})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err = %v, want not found", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUpsertScheduleWritesScheduleAndAuditInOneTransaction(t *testing.T) {
	db, mock := newPayrollDB(t)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "Homebase" WHERE id = $1`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "jadwalGaji"`)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(`(?s)SELECT \* FROM "jadwalGaji"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "unitId", "tanggalGajian", "createdAt", "updatedAt"}).
			AddRow("jadwal-1", "11111111-1111-4111-8111-111111111111", 27, time.Now().UTC(), time.Now().UTC()))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "auditLog"`)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	store := GormStore{DB: db}
	schedule, err := store.UpsertSchedule(context.Background(), "admin-1", domain.ScheduleInput{
		UnitID:        "11111111-1111-4111-8111-111111111111",
		TanggalGajian: 27,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if schedule.TanggalGajian != 27 || schedule.UnitId != "11111111-1111-4111-8111-111111111111" {
		t.Fatalf("schedule = %+v", schedule)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestListSchedulesWithEmptyScopeDoesNotQuery(t *testing.T) {
	db, _ := newPayrollDB(t)

	store := GormStore{DB: db}
	schedules, err := store.ListSchedules(context.Background(), []string{})
	if err != nil {
		t.Fatal(err)
	}
	if len(schedules) != 0 {
		t.Fatalf("schedules = %+v, want empty", schedules)
	}
}

func TestListSchedulesFiltersByUnitScope(t *testing.T) {
	db, mock := newPayrollDB(t)

	mock.ExpectQuery(`(?s)SELECT \* FROM "jadwalGaji"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "unitId", "tanggalGajian", "createdAt", "updatedAt"}).
			AddRow("jadwal-1", "unit-a", 25, time.Now().UTC(), time.Now().UTC()))
	mock.ExpectQuery(`(?s)SELECT \* FROM "Homebase"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "parentId", "createdAt", "updatedAt"}).
			AddRow("unit-a", "Unit A", nil, time.Now().UTC(), time.Now().UTC()))

	store := GormStore{DB: db}
	schedules, err := store.ListSchedules(context.Background(), []string{"unit-a"})
	if err != nil {
		t.Fatal(err)
	}
	if len(schedules) != 1 || schedules[0].UnitId != "unit-a" {
		t.Fatalf("schedules = %+v", schedules)
	}
	if schedules[0].Unit == nil || schedules[0].Unit.Name != "Unit A" {
		t.Fatalf("unit preload = %+v", schedules[0].Unit)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestFindDueRemindersReadsEligibleSlips(t *testing.T) {
	db, mock := newPayrollDB(t)

	mock.ExpectQuery(`(?s)FROM "gaji" g`).
		WillReturnRows(sqlmock.NewRows([]string{"slipId", "userId", "email", "nama", "periode", "tanggalGajian"}).
			AddRow("slip-1", "user-1", "budi@example.test", "Budi", time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), 27))

	store := GormStore{DB: db}
	reminders, err := store.FindDueReminders(context.Background(), time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(reminders) != 1 {
		t.Fatalf("reminders = %+v", reminders)
	}
	if reminders[0].SlipID != "slip-1" || reminders[0].TanggalGajian != 27 || reminders[0].Email != "budi@example.test" {
		t.Fatalf("reminder = %+v", reminders[0])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestEnqueueNotificationsQueuesOnlyNonEmptyKinds(t *testing.T) {
	db, mock := newPayrollDB(t)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "outboxMessage"`)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	store := GormStore{DB: db}
	queued, err := store.EnqueueNotifications(context.Background(), []domain.Notification{
		{Kind: "", DedupeKey: "skip-me", Payload: "{}"},
		{Kind: "email.slip-tersedia", DedupeKey: "slip-available:slip-1", Payload: "{}"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if queued != 1 {
		t.Fatalf("queued = %d, want 1", queued)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestEnqueueNotificationsStopsOnNonDuplicateError(t *testing.T) {
	db, mock := newPayrollDB(t)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "outboxMessage"`)).
		WillReturnError(errors.New("connection reset"))
	mock.ExpectRollback()

	store := GormStore{DB: db}
	queued, err := store.EnqueueNotifications(context.Background(), []domain.Notification{
		{Kind: "email.slip-tersedia", DedupeKey: "slip-available:slip-1", Payload: "{}"},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if queued != 0 {
		t.Fatalf("queued = %d, want 0", queued)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestEnqueueNotificationsWritesBatchInOneStatement(t *testing.T) {
	db, mock, matcher := newCountingPayrollDB(t)
	const notices = 50

	notifications := make([]domain.Notification, 0, notices)
	for index := 0; index < notices; index++ {
		notifications = append(notifications, domain.Notification{
			Kind:      "email.slip-tersedia",
			DedupeKey: fmt.Sprintf("slip-available:slip-%d", index),
			Payload:   "{}",
		})
	}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "outboxMessage"`)).
		WillReturnResult(sqlmock.NewResult(0, notices))
	mock.ExpectCommit()

	store := GormStore{DB: db}
	queued, err := store.EnqueueNotifications(context.Background(), notifications)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if queued != notices {
		t.Fatalf("queued = %d, want %d", queued, notices)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
	// Implementasi lama menulis satu statement per pemberitahuan.
	t.Logf("statements for %d notifications: %d", notices, matcher.count)
	if matcher.count > 1 {
		t.Fatalf("statements = %d, want at most 1", matcher.count)
	}
}
