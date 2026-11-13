package repository

import (
	"context"
	"regexp"
	. "simpeg/backend/internal/tridharma/repository"
	"testing"

	"simpeg/backend/internal/database"
	"simpeg/backend/internal/listing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newTridharmaDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
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

// TestListEvaluasiBKDOrdersByDefaultColumn memastikan kolom bawaan pengurutan
// evaluasi BKD ada pada allowlist. Tanpa entri itu klausa ORDER hanya berisi
// arah (" ASC") dan PostgreSQL menolaknya sebagai kesalahan sintaks.
func TestListEvaluasiBKDOrdersByDefaultColumn(t *testing.T) {
	db, mock := newTridharmaDB(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "evaluasiBkd"`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(regexp.QuoteMeta(`ORDER BY "tanggalEvaluasi" ASC`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "tanggalEvaluasi"}))

	_, _, err := (GormStore{DB: db}).ListEvaluasiBKD(context.Background(), listing.Options{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// TestListTridharmaPenelitianSortsByResearchColumns memastikan allowlist
// penelitian tidak tertukar dengan allowlist pendidikan.
func TestListTridharmaPenelitianSortsByResearchColumns(t *testing.T) {
	db, mock := newTridharmaDB(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "tridharmaPenelitian"`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(regexp.QuoteMeta(`ORDER BY "judul" DESC`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "judul"}))

	_, _, err := (GormStore{DB: db}).ListTridharmaPenelitian(context.Background(), listing.Options{
		Sort: "judul", Order: "desc", Page: 1, PageSize: 20,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
