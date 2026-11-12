package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"simpeg/backend/internal/database"
	"simpeg/backend/internal/directory/domain"
	"simpeg/backend/internal/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// countingMatcher menghitung statement yang berhasil dicocokkan. Penghitung
// ini dipakai sebagai penjaga jumlah query pada jalur volume tinggi.
type countingMatcher struct {
	inner sqlmock.QueryMatcher
	count int
}

func (m *countingMatcher) Match(expectedSQL, actualSQL string) error {
	if err := m.inner.Match(expectedSQL, actualSQL); err != nil {
		return err
	}
	m.count++
	return nil
}

// newCountingStore membuat store pengujian yang menghitung statement SQL.
func newCountingStore(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, *countingMatcher) {
	t.Helper()
	matcher := &countingMatcher{inner: sqlmock.QueryMatcherRegexp}
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(matcher))
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
	return db, mock, matcher
}

func TestCreateEmployeesRollsBackWhenAuditWriteFails(t *testing.T) {
	db, mock := newListStore(t)

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT "email" FROM "user"`).
		WillReturnRows(sqlmock.NewRows([]string{"email"}))
	mock.ExpectExec(`INSERT INTO "user"`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO "user"`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO "auditLog"`).
		WillReturnError(errors.New("synthetic audit failure"))
	mock.ExpectRollback()

	created, err := (GormStore{DB: db}).CreateEmployees(context.Background(), "actor-1", []models.User{
		{Nama: "Pegawai Satu", Email: "one@example.test", Aktif: true},
		{Nama: "Pegawai Dua", Email: "two@example.test", Aktif: true},
	})
	if err == nil || created != nil {
		t.Fatalf("created=%#v err=%v, expected transaction failure", created, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCreateEmployeesReportsDuplicateEmailWithRowIndex(t *testing.T) {
	db, mock := newListStore(t)

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT "email" FROM "user"`).
		WillReturnRows(sqlmock.NewRows([]string{"email"}).AddRow("kedua@example.test"))
	mock.ExpectRollback()

	_, err := (GormStore{DB: db}).CreateEmployees(context.Background(), "actor-1", []models.User{
		{Nama: "Pegawai Satu", Email: "pertama@example.test"},
		{Nama: "Pegawai Dua", Email: "kedua@example.test"},
	})
	var conflict *domain.EmployeeConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("err = %v, want employee conflict", err)
	}
	if conflict.Index != 1 || conflict.Field != "email" {
		t.Fatalf("conflict = %+v, want index 1 field email", conflict)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCreateEmployeesReportsDuplicateNipWithRowIndex(t *testing.T) {
	db, mock := newListStore(t)

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT "email" FROM "user"`).
		WillReturnRows(sqlmock.NewRows([]string{"email"}))
	mock.ExpectQuery(`SELECT "nip" FROM "user"`).
		WillReturnRows(sqlmock.NewRows([]string{"nip"}).AddRow("NIP-2"))
	mock.ExpectRollback()

	_, err := (GormStore{DB: db}).CreateEmployees(context.Background(), "actor-1", []models.User{
		{Nama: "Pegawai Satu", Email: "pertama@example.test", Nip: pointerTo("NIP-1")},
		{Nama: "Pegawai Dua", Email: "kedua@example.test", Nip: pointerTo("NIP-2")},
	})
	var conflict *domain.EmployeeConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("err = %v, want employee conflict", err)
	}
	if conflict.Index != 1 || conflict.Field != "nip" {
		t.Fatalf("conflict = %+v, want index 1 field nip", conflict)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCreateEmployeesReportsDuplicateInsideBatchAsIdentityConflict(t *testing.T) {
	db, mock := newListStore(t)

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT "email" FROM "user"`).
		WillReturnRows(sqlmock.NewRows([]string{"email"}))
	mock.ExpectExec(`INSERT INTO "user"`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO "user"`).
		WillReturnError(&pgconn.PgError{Code: "23505"})
	mock.ExpectRollback()

	_, err := (GormStore{DB: db}).CreateEmployees(context.Background(), "actor-1", []models.User{
		{Nama: "Pegawai Satu", Email: "sama@example.test"},
		{Nama: "Pegawai Dua", Email: "sama@example.test"},
	})
	var conflict *domain.EmployeeConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("err = %v, want employee conflict", err)
	}
	// Duplikasi di dalam batch tertangkap indeks unik saat penyimpanan, bukan
	// pemeriksaan identitas terhadap data yang sudah ada.
	if conflict.Index != 1 || conflict.Field != "identity" {
		t.Fatalf("conflict = %+v, want index 1 field identity", conflict)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCreateEmployeesUsesBoundedStatementCount(t *testing.T) {
	db, mock, matcher := newCountingStore(t)
	const rows = 50

	employees := make([]models.User, 0, rows)
	for index := 0; index < rows; index++ {
		employees = append(employees, models.User{
			Nama:  fmt.Sprintf("Pegawai %d", index),
			Email: fmt.Sprintf("pegawai-%d@example.test", index),
		})
	}

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT "email" FROM "user"`).
		WillReturnRows(sqlmock.NewRows([]string{"email"}))
	for index := 0; index < rows; index++ {
		mock.ExpectExec(`INSERT INTO "user"`).
			WillReturnResult(sqlmock.NewResult(0, 1))
	}
	mock.ExpectExec(`INSERT INTO "auditLog"`).
		WillReturnResult(sqlmock.NewResult(0, rows+1))
	mock.ExpectCommit()

	created, err := (GormStore{DB: db}).CreateEmployees(context.Background(), "actor-1", employees)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(created) != rows {
		t.Fatalf("created = %d, want %d", len(created), rows)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
	// Implementasi lama memakai satu query pemeriksaan per baris per kolom,
	// sehingga jumlah statement tumbuh lebih cepat daripada jumlah baris.
	t.Logf("statements for %d imported rows: %d", rows, matcher.count)
	if matcher.count > rows+2 {
		t.Fatalf("statements = %d, want at most %d", matcher.count, rows+2)
	}
}

func TestCreateEmployeesScansIdentityOncePerChunk(t *testing.T) {
	db, mock, matcher := newCountingStore(t)
	const rows = 300

	employees := make([]models.User, 0, rows)
	for index := 0; index < rows; index++ {
		employees = append(employees, models.User{
			Nama:  fmt.Sprintf("Pegawai %d", index),
			Email: fmt.Sprintf("pegawai-%d@example.test", index),
			Nip:   pointerTo(fmt.Sprintf("NIP-%d", index)),
			Nidn:  pointerTo(fmt.Sprintf("NIDN-%d", index)),
			Nidk:  pointerTo(fmt.Sprintf("NIDK-%d", index)),
		})
	}

	mock.ExpectBegin()
	// Dua chunk berarti dua query per kolom identitas.
	for chunk := 0; chunk < 2; chunk++ {
		mock.ExpectQuery(`SELECT "email" FROM "user"`).
			WillReturnRows(sqlmock.NewRows([]string{"email"}))
		mock.ExpectQuery(`SELECT "nip" FROM "user"`).
			WillReturnRows(sqlmock.NewRows([]string{"nip"}))
		mock.ExpectQuery(`SELECT "nidn" FROM "user"`).
			WillReturnRows(sqlmock.NewRows([]string{"nidn"}))
		mock.ExpectQuery(`SELECT "nidk" FROM "user"`).
			WillReturnRows(sqlmock.NewRows([]string{"nidk"}))
	}
	for index := 0; index < rows; index++ {
		mock.ExpectExec(`INSERT INTO "user"`).
			WillReturnResult(sqlmock.NewResult(0, 1))
	}
	mock.ExpectExec(`INSERT INTO "auditLog"`).
		WillReturnResult(sqlmock.NewResult(0, rows+1))
	mock.ExpectCommit()

	if _, err := (GormStore{DB: db}).CreateEmployees(context.Background(), "actor-1", employees); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
	t.Logf("statements for %d imported rows: %d", rows, matcher.count)
	if matcher.count > rows+9 {
		t.Fatalf("statements = %d, want at most %d", matcher.count, rows+9)
	}
}

func pointerTo(value string) *string {
	return &value
}
