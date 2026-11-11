package holiday

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

func newHolidayDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
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

func TestReplaceYearSwapsCalendarInOneTransaction(t *testing.T) {
	db, mock := newHolidayDB(t)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "HariLibur"`)).WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "HariLibur"`)).WillReturnResult(sqlmock.NewResult(1, 2))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "auditLog"`)).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	store := GormStore{DB: db}
	stored, err := store.ReplaceYear(context.Background(), "admin-1", 2026, []Entry{
		{Tanggal: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Nama: "Tahun Baru 2026"},
		{Tanggal: time.Date(2026, 8, 17, 0, 0, 0, 0, time.UTC), Nama: "Hari Kemerdekaan RI"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stored != 2 {
		t.Fatalf("stored = %d, want 2", stored)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestReplaceYearRollsBackWhenWriteFails(t *testing.T) {
	db, mock := newHolidayDB(t)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "HariLibur"`)).WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "HariLibur"`)).WillReturnError(errors.New("database unavailable"))
	mock.ExpectRollback()

	store := GormStore{DB: db}
	if _, err := store.ReplaceYear(context.Background(), "admin-1", 2026, []Entry{
		{Tanggal: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Nama: "Tahun Baru 2026"},
	}); err == nil {
		t.Fatal("expected error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestReplaceYearWithEmptyEntriesStillAudits(t *testing.T) {
	db, mock := newHolidayDB(t)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "HariLibur"`)).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "auditLog"`)).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	store := GormStore{DB: db}
	stored, err := store.ReplaceYear(context.Background(), "admin-1", 2026, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stored != 0 {
		t.Fatalf("stored = %d, want 0", stored)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestListYearMapsRowsToEntries(t *testing.T) {
	db, mock := newHolidayDB(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "HariLibur"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "tanggal", "nama", "createdAt", "updatedAt"}).
			AddRow("id-1", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), "Tahun Baru 2026", time.Now().UTC(), time.Now().UTC()))

	store := GormStore{DB: db}
	entries, err := store.ListYear(context.Background(), 2026)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Nama != "Tahun Baru 2026" {
		t.Fatalf("entries = %+v", entries)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
