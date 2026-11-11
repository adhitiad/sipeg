package attendance

import (
	"context"
	"regexp"
	"testing"
	"time"

	"simpeg/backend/internal/database"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newDeviceAuthenticatorDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
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

var mockTime = time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)

func expectCredential(mock sqlmock.Sqlmock, machineID, credentialHash string, active bool) {
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "machineCredential"`)).
		WithArgs(machineID, active, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "machineId", "credentialType", "credentialHash", "active", "lastUsedAt", "createdAt", "updatedAt"}).
			AddRow("credential-1", machineID, "API_KEY", credentialHash, true, nil, mockTime, mockTime))
}

func expectMachine(mock sqlmock.Sqlmock, machineID string, aktif bool) {
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "MesinAbsensi"`)).
		WithArgs(machineID, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "nama", "merek", "host", "port", "comKey", "aktif", "terakhirSinkron", "createdAt", "updatedAt"}).
			AddRow(machineID, "Mesin Uji", "FINGERSPOT_SOAP", nil, nil, nil, aktif, nil, mockTime, mockTime))
}

func TestDeviceAuthenticatorValidCredentials(t *testing.T) {
	db, mock := newDeviceAuthenticatorDB(t)
	machineID := "machine-1"
	credentialHash := "stored-hash"

	expectCredential(mock, machineID, credentialHash, true)
	expectMachine(mock, machineID, true)

	authenticator := NewGormDeviceAuthenticator(db)
	machine, err := authenticator.AuthenticateDevice(context.Background(), machineID, "stored-hash")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if machine.ID != machineID {
		t.Fatalf("machine ID = %q, want %q", machine.ID, machineID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestDeviceAuthenticatorInvalidHash(t *testing.T) {
	db, mock := newDeviceAuthenticatorDB(t)
	machineID := "machine-1"

	expectCredential(mock, machineID, "stored-hash", true)

	authenticator := NewGormDeviceAuthenticator(db)
	_, err := authenticator.AuthenticateDevice(context.Background(), machineID, "wrong-hash")

	if err != ErrInvalidDeviceCredentials {
		t.Fatalf("expected ErrInvalidDeviceCredentials, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestDeviceAuthenticatorUnknownMachine(t *testing.T) {
	db, mock := newDeviceAuthenticatorDB(t)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "machineCredential"`)).
		WithArgs("machine-missing", true, sqlmock.AnyArg()).
		WillReturnError(gorm.ErrRecordNotFound)

	authenticator := NewGormDeviceAuthenticator(db)
	_, err := authenticator.AuthenticateDevice(context.Background(), "machine-missing", "token")

	if err != ErrInvalidDeviceCredentials {
		t.Fatalf("expected ErrInvalidDeviceCredentials, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestDeviceAuthenticatorInactiveMachine(t *testing.T) {
	db, mock := newDeviceAuthenticatorDB(t)
	machineID := "machine-inactive"

	expectCredential(mock, machineID, "hash", true)
	expectMachine(mock, machineID, false)

	authenticator := NewGormDeviceAuthenticator(db)
	_, err := authenticator.AuthenticateDevice(context.Background(), machineID, "hash")

	if err != ErrInvalidDeviceCredentials {
		t.Fatalf("expected ErrInvalidDeviceCredentials, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestDeviceAuthenticatorDBNil(t *testing.T) {
	authenticator := NewGormDeviceAuthenticator(nil)
	_, err := authenticator.AuthenticateDevice(context.Background(), "machine-1", "token")
	if err == nil {
		t.Fatal("expected error for nil database")
	}
}
