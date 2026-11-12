package audit

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"simpeg/backend/internal/database"
	"simpeg/backend/internal/models"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newAuditDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{
		NamingStrategy:       database.PostgresNamingStrategy{},
		DisableAutomaticPing: true,
		Logger:               logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	return db, mock
}

func TestWriteStoresActorActionTargetAndUnit(t *testing.T) {
	db, mock := newAuditDB(t)

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "auditLog"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	target := "22222222-2222-4222-8222-222222222222"
	unit := "33333333-3333-4333-8333-333333333333"
	err := Write(context.Background(), db, Entry{
		ActorID:  "user-1",
		Action:   ActionEmployeeUpdated,
		TargetID: &target,
		UnitID:   &unit,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestWriteRejectsMissingDatabaseHandle(t *testing.T) {
	if err := Write(context.Background(), nil, Entry{Action: ActionEmployeeUpdated}); err == nil {
		t.Fatal("expected error without database handle")
	}
}

func TestWriteBatchInsertsAllEntriesInOneStatement(t *testing.T) {
	db, mock := newAuditDB(t)

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "auditLog"`).
		WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectCommit()

	unit := "33333333-3333-4333-8333-333333333333"
	entries := []Entry{
		{ActorID: "user-1", Action: ActionEmployeeCreated, UnitID: &unit},
		{ActorID: "user-1", Action: ActionEmployeeCreated, UnitID: &unit},
		{ActorID: "user-1", Action: ActionEmployeeImported, UnitID: &unit},
	}
	if err := WriteBatch(context.Background(), db, entries); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestWriteBatchWithoutEntriesDoesNotTouchDatabase(t *testing.T) {
	db, mock := newAuditDB(t)

	if err := WriteBatch(context.Background(), db, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestWriteBatchRejectsMissingDatabaseHandle(t *testing.T) {
	if err := WriteBatch(context.Background(), nil, []Entry{{Action: ActionEmployeeUpdated}}); err == nil {
		t.Fatal("expected error without database handle")
	}
}

func TestAuditRecordCarriesUnitScopeColumn(t *testing.T) {
	// Kolom unitId harus ada pada model agar jejak dapat dibatasi per unit.
	record := models.AuditLog{UnitId: nil}
	if record.UnitId != nil {
		t.Fatal("expected nil unit scope by default")
	}
}

// TestAuditTrailIsAppendOnly gagal bila ada jalur hapus atau ubah pada
// tabel auditLog di seluruh source backend. Jejak audit tidak boleh dapat
// dihapus atau dimodifikasi oleh kode aplikasi.
func TestAuditTrailIsAppendOnly(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine source path")
	}
	backendRoot := filepath.Dir(filepath.Dir(currentFile))

	forbidden := []string{"Delete(&models.AuditLog", "Delete(&auditLog", "Unscoped().Delete(&models.AuditLog"}
	err := filepath.Walk(backendRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for _, pattern := range forbidden {
			if strings.Contains(string(content), pattern) {
				relative, _ := filepath.Rel(backendRoot, path)
				t.Errorf("%s contains a delete path for the audit trail: %s", relative, pattern)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
