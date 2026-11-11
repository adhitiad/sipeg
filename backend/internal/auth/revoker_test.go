package auth

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestGormSessionRevokerDeletesSessionRow(t *testing.T) {
	db, mock := newResolverDB(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "session" WHERE token = $1`)).
		WithArgs("valid-session", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "userId", "token"}).
			AddRow("session-1", "user-1", "valid-session"))
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "session" WHERE token = $1`)).
		WithArgs("valid-session").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "auditLog"`)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	actor := Principal{UserID: "user-1", HomebaseID: "unit-a"}
	if err := (GormSessionRevoker{DB: db}).RevokeSession(context.Background(), "valid-session", actor); err != nil {
		t.Fatalf("expected revoke success, got: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGormSessionRevokerReportsMissingSession(t *testing.T) {
	db, mock := newResolverDB(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "session" WHERE token = $1`)).
		WithArgs("unknown-session", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "userId", "token"}))

	err := (GormSessionRevoker{DB: db}).RevokeSession(context.Background(), "unknown-session", Principal{})
	if err != ErrSessionNotFound {
		t.Fatalf("expected ErrSessionNotFound, got: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGormSessionRevokerRequiresDatabaseAndToken(t *testing.T) {
	if err := (GormSessionRevoker{}).RevokeSession(context.Background(), "token", Principal{}); err != ErrSessionNotFound {
		t.Fatalf("expected ErrSessionNotFound without database, got: %v", err)
	}
	db, _ := newResolverDB(t)
	if err := (GormSessionRevoker{DB: db}).RevokeSession(context.Background(), "  ", Principal{}); err != ErrSessionNotFound {
		t.Fatalf("expected ErrSessionNotFound for blank token, got: %v", err)
	}
}
