package auth

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

func newIssuerDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
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

func expectUserForIssuance(mock sqlmock.Sqlmock, userID string, roleID interface{}, aktif bool) {
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "user" WHERE id = $1`)).
		WithArgs(userID, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "nama", "roleId", "homebaseId", "aktif"}).
			AddRow(userID, "person@example.test", "Pegawai Uji", roleID, "unit-a", aktif))
}

func expectRoleForIssuance(mock sqlmock.Sqlmock, roleID, roleName string) {
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "Role" WHERE id = $1`)).
		WithArgs(roleID, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(roleID, roleName))
}

func expectInsertSessionExec(mock sqlmock.Sqlmock) {
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "session"`)).
		WillReturnResult(sqlmock.NewResult(1, 1))
}

func expectInsertAuditLog(mock sqlmock.Sqlmock) {
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "auditLog"`)).
		WillReturnResult(sqlmock.NewResult(1, 1))
}

func TestIssuerCreatesSessionAndReturnsToken(t *testing.T) {
	db, mock := newIssuerDB(t)
	roleID := "role-1"
	expectUserForIssuance(mock, "user-1", roleID, true)
	expectRoleForIssuance(mock, roleID, "operator")
	mock.ExpectBegin()
	expectInsertSessionExec(mock)
	mock.ExpectCommit()
	mock.ExpectBegin()
	expectInsertAuditLog(mock)
	mock.ExpectCommit()

	issuer := NewGormSessionIssuer(db, time.Hour)
	token, expiresAt, err := issuer.IssueSession(context.Background(), "user-1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	if expiresAt.IsZero() {
		t.Fatal("expected non-zero expiresAt")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestIssuerSkipsRoleLookupWhenRoleIdEmpty(t *testing.T) {
	db, mock := newIssuerDB(t)
	expectUserForIssuance(mock, "user-2", nil, true)
	mock.ExpectBegin()
	expectInsertSessionExec(mock)
	mock.ExpectCommit()
	mock.ExpectBegin()
	expectInsertAuditLog(mock)
	mock.ExpectCommit()

	issuer := NewGormSessionIssuer(db, time.Hour)
	token, _, err := issuer.IssueSession(context.Background(), "user-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestIssuerRejectsUnknownUser(t *testing.T) {
	db, mock := newIssuerDB(t)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "user" WHERE id = $1`)).
		WithArgs("user-missing", sqlmock.AnyArg()).
		WillReturnError(gorm.ErrRecordNotFound)

	issuer := NewGormSessionIssuer(db, time.Hour)
	_, _, err := issuer.IssueSession(context.Background(), "user-missing")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestIssuerRejectsInactiveUser(t *testing.T) {
	db, mock := newIssuerDB(t)
	expectUserForIssuance(mock, "user-inactive", nil, false)

	issuer := NewGormSessionIssuer(db, time.Hour)
	_, _, err := issuer.IssueSession(context.Background(), "user-inactive")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestIssuerRejectsDanglingRole(t *testing.T) {
	db, mock := newIssuerDB(t)
	roleID := "role-dangling"
	expectUserForIssuance(mock, "user-3", roleID, true)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "Role" WHERE id = $1`)).
		WithArgs(roleID, sqlmock.AnyArg()).
		WillReturnError(gorm.ErrRecordNotFound)

	issuer := NewGormSessionIssuer(db, time.Hour)
	_, _, err := issuer.IssueSession(context.Background(), "user-3")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}
