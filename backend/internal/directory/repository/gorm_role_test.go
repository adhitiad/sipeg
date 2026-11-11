package repository

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAssignEmployeeRoleRollsBackWhenAuditWriteFails(t *testing.T) {
	db, mock := newListStore(t)
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT \* FROM "user"`).
		WithArgs("employee-1", true, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "homebaseId"}).AddRow("employee-1", "unit-1"))
	mock.ExpectQuery(`SELECT \* FROM "Role"`).
		WithArgs("atasan", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow("role-atasan", "atasan"))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "user"`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO "auditLog"`).
		WillReturnError(errors.New("synthetic audit failure"))
	mock.ExpectRollback()

	updated, err := (GormStore{DB: db}).AssignEmployeeRole(context.Background(), "admin-1", "employee-1", "atasan")
	if err == nil || updated.RoleId != nil {
		t.Fatalf("updated=%#v err=%v, expected transaction rollback", updated, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAssignEmployeeRoleCommitsAuditAndReturnsNewRole(t *testing.T) {
	db, mock := newListStore(t)
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT \* FROM "user"`).
		WithArgs("employee-1", true, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "homebaseId"}).AddRow("employee-1", "unit-1"))
	mock.ExpectQuery(`SELECT \* FROM "Role"`).
		WithArgs("pegawai", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow("role-pegawai", "pegawai"))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "user"`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO "auditLog"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	updated, err := (GormStore{DB: db}).AssignEmployeeRole(context.Background(), "admin-1", "employee-1", "pegawai")
	if err != nil {
		t.Fatalf("AssignEmployeeRole error: %v", err)
	}
	if updated.RoleId == nil || *updated.RoleId != "role-pegawai" {
		t.Fatalf("roleId = %v, want role-pegawai", updated.RoleId)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
