package repository

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"simpeg/backend/internal/models"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestCreateEmployeesRollsBackWhenAuditWriteFails(t *testing.T) {
	db, mock := newListStore(t)

	mock.ExpectBegin()
	for _, email := range []string{"one@example.test", "two@example.test"} {
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "user" WHERE LOWER(email) = $1`)).
			WithArgs(email).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	}
	mock.ExpectExec(`INSERT INTO "user"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
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
