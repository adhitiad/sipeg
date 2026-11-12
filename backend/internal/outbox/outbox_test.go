package outbox

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"simpeg/backend/internal/database"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newOutboxDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
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

func TestEnqueueInsertsPendingMessage(t *testing.T) {
	db, mock := newOutboxDB(t)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "outboxMessage"`)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	message, err := NewGormStore(db).Enqueue(context.Background(), "email.penggajian", "pembayaran-1", `{"userId":"u1"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if message.Kind != "email.penggajian" || message.DedupeKey != "pembayaran-1" {
		t.Fatalf("unexpected message: %+v", message)
	}
	if message.Status != StatusPending {
		t.Fatalf("status = %q, want %q", message.Status, StatusPending)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestEnqueueRejectsDuplicateDedupeKey(t *testing.T) {
	db, mock := newOutboxDB(t)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "outboxMessage"`)).
		WillReturnError(errors.New("duplicate key value violates unique constraint"))
	mock.ExpectRollback()

	_, err := NewGormStore(db).Enqueue(context.Background(), "email.penggajian", "pembayaran-1", "{}")
	if err == nil {
		t.Fatal("expected duplicate error")
	}
}

func TestEnqueueBatchWritesAllMessagesInOneStatement(t *testing.T) {
	db, mock := newOutboxDB(t)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "outboxMessage"`)).
		WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectCommit()

	queued, err := NewGormStore(db).EnqueueBatch(context.Background(), []Message{
		NewMessage("email.penggajian", "slip-available:slip-1", `{"slipId":"slip-1"}`),
		NewMessage("email.penggajian", "slip-available:slip-2", `{"slipId":"slip-2"}`),
		NewMessage("email.penggajian", "slip-available:slip-3", `{"slipId":"slip-3"}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if queued != 3 {
		t.Fatalf("queued = %d, want 3", queued)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestEnqueueBatchSkipsDuplicateDedupeKeys(t *testing.T) {
	db, mock := newOutboxDB(t)

	// ON CONFLICT DO NOTHING membuat baris yang sudah ada dilewati tanpa error.
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "outboxMessage"`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	queued, err := NewGormStore(db).EnqueueBatch(context.Background(), []Message{
		NewMessage("email.penggajian", "slip-available:slip-1", "{}"),
		NewMessage("email.penggajian", "slip-available:slip-1", "{}"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if queued != 1 {
		t.Fatalf("queued = %d, want 1", queued)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestEnqueueBatchSkipsMessagesWithoutKind(t *testing.T) {
	db, _ := newOutboxDB(t)

	queued, err := NewGormStore(db).EnqueueBatch(context.Background(), []Message{
		NewMessage("", "tanpa-jenis", "{}"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if queued != 0 {
		t.Fatalf("queued = %d, want 0", queued)
	}
}

func TestClaimLocksPendingMessages(t *testing.T) {
	db, mock := newOutboxDB(t)
	now := time.Now().UTC()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "outboxMessage"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kind", "dedupeKey", "payload", "status", "attempts", "availableAt"}).
			AddRow("msg-1", "email.penggajian", "key-1", "{}", StatusPending, 0, now))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "outboxMessage" SET`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	messages, err := NewGormStore(db).Claim(context.Background(), 10, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("expected 1 claimed message, got %d", len(messages))
	}
	if messages[0].Status != StatusProcessing {
		t.Fatalf("status = %q, want %q", messages[0].Status, StatusProcessing)
	}
	if messages[0].LockedAt == nil {
		t.Fatal("claimed message must be locked")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCompleteMarksMessageDone(t *testing.T) {
	db, mock := newOutboxDB(t)
	now := time.Now().UTC()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "outboxMessage" SET`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := NewGormStore(db).Complete(context.Background(), "msg-1", now); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestFailReschedulesMessageWhenAttemptsRemain(t *testing.T) {
	db, mock := newOutboxDB(t)
	now := time.Now().UTC()

	// Pembacaan attempt terjadi di luar transaksi.
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "outboxMessage"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "attempts", "status"}).
			AddRow("msg-1", 1, StatusProcessing))
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "outboxMessage" SET`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := NewGormStore(db).Fail(context.Background(), "msg-1", "kirim gagal", now.Add(time.Minute), now, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestFailMovesMessageToDeadLetterWhenAttemptsExhausted(t *testing.T) {
	db, mock := newOutboxDB(t)
	now := time.Now().UTC()

	// Pembacaan attempt terjadi di luar transaksi.
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "outboxMessage"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "attempts", "status"}).
			AddRow("msg-1", 4, StatusProcessing))
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "outboxMessage" SET`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := NewGormStore(db).Fail(context.Background(), "msg-1", "kirim gagal", now.Add(time.Minute), now, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestReleaseLockedReturnsStuckMessages(t *testing.T) {
	db, mock := newOutboxDB(t)
	now := time.Now().UTC()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "outboxMessage" SET`)).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	if err := NewGormStore(db).ReleaseLocked(context.Background(), now); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
