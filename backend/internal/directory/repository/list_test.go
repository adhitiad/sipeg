package repository

import (
	"context"
	"regexp"
	"testing"

	"simpeg/backend/internal/database"
	"simpeg/backend/internal/listing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newListStore(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
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

func TestEmployeeListUsesAllowlistedSortColumn(t *testing.T) {
	db, mock := newListStore(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "user"`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "user"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "nama"}).AddRow("user-1", "Pegawai"))

	_, _, err := (GormStore{DB: db}).ListEmployees(context.Background(), listing.Options{
		Sort: "nip", Order: "desc", Page: 1, PageSize: 20,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestEmployeeListFallsBackToDefaultSortForUnknownColumn(t *testing.T) {
	db, mock := newListStore(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "user"`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(regexp.QuoteMeta(`ORDER BY "nama" ASC`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "nama"}).AddRow("user-1", "Pegawai"))

	// Nilai sort di luar allowlist harus jatuh ke kolom bawaan, bukan disisipkan apa adanya.
	_, _, err := (GormStore{DB: db}).ListEmployees(context.Background(), listing.Options{
		Sort: `nama; DROP TABLE "user"`, Order: "asc", Page: 1, PageSize: 20,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAuditLogListUsesAllowlistedSortColumn(t *testing.T) {
	db, mock := newListStore(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "auditLog"`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(regexp.QuoteMeta(`ORDER BY "createdAt" DESC`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "aksi"}).AddRow("log-1", "employee.updated"))

	_, _, err := (GormStore{DB: db}).ListAuditLogs(context.Background(), listing.Options{
		Sort: "createdAt", Order: "desc", Page: 1, PageSize: 20,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCategoryListUsesAllowlistedSortColumn(t *testing.T) {
	db, mock := newListStore(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "kategoriPegawai"`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(regexp.QuoteMeta(`ORDER BY "kode" ASC`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kode"}).AddRow("kat-1", "DOSEN"))

	_, _, err := (GormStore{DB: db}).ListCategories(context.Background(), listing.Options{
		Sort: "kode", Order: "asc", Page: 1, PageSize: 20,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestListUnitScopeIDsReturnsRootAndDescendants(t *testing.T) {
	db, mock := newListStore(t)

	mock.ExpectQuery(regexp.QuoteMeta(`WITH RECURSIVE subtree AS`)).
		WithArgs("unit-a").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("unit-a").AddRow("unit-a-child").AddRow("unit-a-grandchild"))

	unitIDs, err := (GormStore{DB: db}).ListUnitScopeIDs(context.Background(), "unit-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(unitIDs) != 3 {
		t.Fatalf("scope = %v, want 3 units", unitIDs)
	}
	if unitIDs[0] != "unit-a" {
		t.Fatalf("root unit must be included, got %v", unitIDs)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestListUnitScopeIDsWithoutRootReturnsEmpty(t *testing.T) {
	db, _ := newListStore(t)

	unitIDs, err := (GormStore{DB: db}).ListUnitScopeIDs(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(unitIDs) != 0 {
		t.Fatalf("scope = %v, want empty", unitIDs)
	}
}

func TestListUnitScopeIDsQuotesParentIdColumn(t *testing.T) {
	db, mock := newListStore(t)

	// Kolom "parentId" harus dikutip; tanpa kutip PostgreSQL/folders ke huruf kecil
	// dan query tidak akan menemukan turunan.
	mock.ExpectQuery(regexp.QuoteMeta(`child."parentId" = subtree.id`)).
		WithArgs("unit-a").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("unit-a"))

	_, err := (GormStore{DB: db}).ListUnitScopeIDs(context.Background(), "unit-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
