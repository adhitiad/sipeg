package repository

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"simpeg/backend/internal/database"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/payroll/domain"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newPayrollDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{
		NamingStrategy:       database.PostgresNamingStrategy{},
		TranslateError:       true,
		DisableAutomaticPing: true,
		Logger:               logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	return db, mock
}

func TestRecordPaymentWritesPaymentSlipAndNotificationInOneTransaction(t *testing.T) {
	db, mock := newPayrollDB(t)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "gaji" WHERE id = $1`)).WillReturnRows(
		sqlmock.NewRows([]string{"id", "nomorSlip", "status", "periode"}).
			AddRow("slip-1", "SLIP-2026-11", "TERBIT", time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)),
	)
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "pembayaran"`)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "gaji" SET`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "outboxMessage"`)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "auditLog"`)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	store := GormStore{DB: db}
	notification := domain.Notification{
		Kind:      "email.penggajian",
		DedupeKey: "salary-paid:2026-11:user-1",
		Payload:   `{"kind":"email.penggajian"}`,
	}
	_, err := store.RecordPayment(context.Background(), "admin-1", "slip-1", domain.PaymentInput{
		PelakuPengesahan: "Bendahara",
		TanggalBayar:     time.Date(2026, 11, 28, 0, 0, 0, 0, time.UTC),
	}, notification)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRecordPaymentRejectsSlipThatIsNotPublished(t *testing.T) {
	db, mock := newPayrollDB(t)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "gaji" WHERE id = $1`)).WillReturnRows(
		sqlmock.NewRows([]string{"id", "nomorSlip", "status"}).
			AddRow("slip-1", "SLIP-2026-11", "DRAF"),
	)
	mock.ExpectRollback()

	store := GormStore{DB: db}
	_, err := store.RecordPayment(context.Background(), "admin-1", "slip-1", domain.PaymentInput{
		PelakuPengesahan: "Bendahara",
		TanggalBayar:     time.Date(2026, 11, 28, 0, 0, 0, 0, time.UTC),
	}, domain.Notification{})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("err = %v, want conflict", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRecordPaymentRollsBackWhenNotificationCannotBeQueued(t *testing.T) {
	db, mock := newPayrollDB(t)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "gaji" WHERE id = $1`)).WillReturnRows(
		sqlmock.NewRows([]string{"id", "nomorSlip", "status"}).
			AddRow("slip-1", "SLIP-2026-11", "TERBIT"),
	)
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "pembayaran"`)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "gaji" SET`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "outboxMessage"`)).
		WillReturnError(errors.New("outbox unavailable"))
	mock.ExpectRollback()

	store := GormStore{DB: db}
	_, err := store.RecordPayment(context.Background(), "admin-1", "slip-1", domain.PaymentInput{
		PelakuPengesahan: "Bendahara",
		TanggalBayar:     time.Date(2026, 11, 28, 0, 0, 0, 0, time.UTC),
	}, domain.Notification{Kind: "email.penggajian", DedupeKey: "k", Payload: "{}"})
	if err == nil {
		t.Fatal("expected the transaction to fail when the notification cannot be queued")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestConfirmedPaymentOnlyCountsRecordedPayments(t *testing.T) {
	db, mock := newPayrollDB(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "pembayaran"`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	store := GormStore{DB: db}
	confirmed, err := store.ConfirmedPayment(context.Background(), "user-1", "2026-11")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !confirmed {
		t.Fatal("expected the payment to be confirmed")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestConfirmedPaymentRejectsInvalidPeriod(t *testing.T) {
	store := GormStore{}
	confirmed, err := store.ConfirmedPayment(context.Background(), "user-1", "bukan-periode")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if confirmed {
		t.Fatal("an invalid period must never be treated as confirmed")
	}
}

func TestListSlipsFiltersByScopeAndPeriod(t *testing.T) {
	db, mock := newPayrollDB(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "gaji"`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(regexp.QuoteMeta(`FROM "gaji" WHERE`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	store := GormStore{DB: db}
	_, _, err := store.ListSlips(context.Background(), listing.Options{
		ScopeHomebaseIDs: []string{"unit-a"},
		Periode:          "2026-11",
		Page:             1,
		PageSize:         20,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
