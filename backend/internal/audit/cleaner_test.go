package audit

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestCleanerDeletesExpiredAuditLogs(t *testing.T) {
	db, mock := newAuditDB(t)

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "auditLog" WHERE "createdAt" < $1`)).
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 3))

	cleaner := NewCleaner(db, 30)
	deleted, err := cleaner.Cleanup(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deleted != 3 {
		t.Fatalf("deleted = %d, want 3", deleted)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCleanerNoRetentionMeansNoDeletion(t *testing.T) {
	db, mock := newAuditDB(t)

	// Retention 0 berarti tidak ada pemotongan, tidak ada query DELETE.
	cleaner := NewCleaner(db, 0)
	deleted, err := cleaner.Cleanup(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deleted != 0 {
		t.Fatalf("deleted = %d, want 0", deleted)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestExpiredSinceReturnsZeroWhenNoRetention(t *testing.T) {
	cleaner := NewCleaner(nil, 0)
	if !cleaner.ExpiredSince().IsZero() {
		t.Fatal("expected zero time for no retention")
	}
}

func TestExpiredSinceReturnsCutoff(t *testing.T) {
	cleaner := NewCleaner(nil, 30)
	cutoff := cleaner.ExpiredSince()
	expected := time.Now().Add(-30 * 24 * time.Hour)
	if cutoff.Before(expected.Add(-time.Minute)) || cutoff.After(expected.Add(time.Minute)) {
		t.Fatalf("cutoff = %v, expected around %v", cutoff, expected)
	}
}
