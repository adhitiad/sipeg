package auth

import (
	"context"
	"regexp"
	"sync"
	"testing"
	"time"

	"simpeg/backend/internal/database"
	"simpeg/backend/internal/models"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func newResolverDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{
		NamingStrategy:       database.PrismaNamingStrategy{},
		TranslateError:       true,
		DisableAutomaticPing: true,
		Logger:               logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	return db, mock
}

func expectSession(mock sqlmock.Sqlmock, token string, userID string, expiresAt time.Time) {
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "session" WHERE token = $1`)).
		WithArgs(token, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "userId", "expiresAt", "token"}).
			AddRow("session-1", userID, expiresAt, token))
}

func expectUser(mock sqlmock.Sqlmock, userID, roleID string, aktif bool) {
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "user" WHERE id = $1`)).
		WithArgs(userID, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "nama", "roleId", "homebaseId", "aktif"}).
			AddRow(userID, "person@example.test", "Pegawai Uji", roleID, "unit-a", aktif))
}

func expectRole(mock sqlmock.Sqlmock, roleID, roleName string) {
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "Role" WHERE id = $1`)).
		WithArgs(roleID, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(roleID, roleName))
}

func TestResolverReadsRoleFromServerSideData(t *testing.T) {
	db, mock := newResolverDB(t)
	expiresAt := time.Now().Add(time.Hour)

	expectSession(mock, "valid-session", "user-1", expiresAt)
	expectUser(mock, "user-1", "role-1", true)
	expectRole(mock, "role-1", "operator")

	principal, err := (GormSessionResolver{DB: db}).ResolvePrincipal(context.Background(), "valid-session")
	if err != nil {
		t.Fatalf("expected principal, got error: %v", err)
	}
	if principal.RoleName != "operator" || principal.UserID != "user-1" || principal.HomebaseID != "unit-a" {
		t.Fatalf("unexpected principal: %+v", principal)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestResolverPicksUpRoleChangeOnNextRequest(t *testing.T) {
	db, mock := newResolverDB(t)
	expiresAt := time.Now().Add(time.Hour)
	resolver := GormSessionResolver{DB: db}

	// Permintaan pertama: role operator.
	expectSession(mock, "valid-session", "user-1", expiresAt)
	expectUser(mock, "user-1", "role-1", true)
	expectRole(mock, "role-1", "operator")

	first, err := resolver.ResolvePrincipal(context.Background(), "valid-session")
	if err != nil {
		t.Fatalf("expected principal, got error: %v", err)
	}
	if first.RoleName != "operator" {
		t.Fatalf("first role = %q, want operator", first.RoleName)
	}

	// Role diubah di server: permintaan berikutnya harus memakai role baru tanpa sesi baru.
	expectSession(mock, "valid-session", "user-1", expiresAt)
	expectUser(mock, "user-1", "role-2", true)
	expectRole(mock, "role-2", "atasan")

	second, err := resolver.ResolvePrincipal(context.Background(), "valid-session")
	if err != nil {
		t.Fatalf("expected principal after role change, got error: %v", err)
	}
	if second.RoleName != "atasan" {
		t.Fatalf("second role = %q, want atasan", second.RoleName)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestResolverRejectsDisabledAccount(t *testing.T) {
	db, mock := newResolverDB(t)

	expectSession(mock, "valid-session", "user-1", time.Now().Add(time.Hour))
	expectUser(mock, "user-1", "role-1", false)

	_, err := (GormSessionResolver{DB: db}).ResolvePrincipal(context.Background(), "valid-session")
	if err != ErrInvalidSession {
		t.Fatalf("expected ErrInvalidSession for disabled account, got: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestResolverRejectsExpiredSession(t *testing.T) {
	db, mock := newResolverDB(t)

	expectSession(mock, "expired-session", "user-1", time.Now().Add(-time.Minute))

	_, err := (GormSessionResolver{DB: db}).ResolvePrincipal(context.Background(), "expired-session")
	if err != ErrInvalidSession {
		t.Fatalf("expected ErrInvalidSession for expired session, got: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestResolverRejectsRevokedSession(t *testing.T) {
	db, mock := newResolverDB(t)

	// Baris sesi dihapus (setelah logout atau pencabutan): query tidak menemukan baris.
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "session" WHERE token = $1`)).
		WithArgs("revoked-session", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "userId", "expiresAt", "token"}))

	_, err := (GormSessionResolver{DB: db}).ResolvePrincipal(context.Background(), "revoked-session")
	if err != ErrInvalidSession {
		t.Fatalf("expected ErrInvalidSession for revoked session, got: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestResolverDeniesRoleWithoutRoleRecord(t *testing.T) {
	db, mock := newResolverDB(t)

	// Pengguna tanpa role: principal tetap terbentuk tetapi tanpa RoleName,
	// sehingga policy menolak semua permission.
	expectSession(mock, "valid-session", "user-1", time.Now().Add(time.Hour))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "user" WHERE id = $1`)).
		WithArgs("user-1", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "nama", "roleId", "homebaseId", "aktif"}).
			AddRow("user-1", "person@example.test", "Pegawai Uji", nil, "unit-a", true))

	principal, err := (GormSessionResolver{DB: db}).ResolvePrincipal(context.Background(), "valid-session")
	if err != nil {
		t.Fatalf("expected principal, got error: %v", err)
	}
	if principal.RoleName != "" {
		t.Fatalf("expected empty role name, got %q", principal.RoleName)
	}
	if (StaticPolicy{"operator": {"employee:read:any": true}}).Allows(principal.RoleName, "employee:read:any") {
		t.Fatal("principal without role must not receive any grant")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestResolverRejectsSessionWithoutUser(t *testing.T) {
	db, mock := newResolverDB(t)

	expectSession(mock, "orphan-session", "user-1", time.Now().Add(time.Hour))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "user" WHERE id = $1`)).
		WithArgs("user-1", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "nama"}))

	_, err := (GormSessionResolver{DB: db}).ResolvePrincipal(context.Background(), "orphan-session")
	if err != ErrInvalidSession {
		t.Fatalf("expected ErrInvalidSession for session without user, got: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSessionModelMatchesReferenceSchema(t *testing.T) {
	parsed, err := schema.Parse(&models.Session{}, &sync.Map{}, database.PrismaNamingStrategy{})
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Table != "session" {
		t.Fatalf("table = %q, want session", parsed.Table)
	}
	for field, expected := range map[string]string{
		"UserId":    "userId",
		"ExpiresAt": "expiresAt",
		"Token":     "token",
		"IpAddress": "ipAddress",
		"UserAgent": "userAgent",
	} {
		found := parsed.LookUpField(field)
		if found == nil || found.DBName != expected {
			t.Errorf("column for %s = %v, want %q", field, found, expected)
		}
	}
}
