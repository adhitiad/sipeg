package repository

import (
	"context"
	"regexp"
	. "simpeg/backend/internal/directory/repository"
	"testing"

	"simpeg/backend/internal/database"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"

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
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT "user".*, "Role"."name" AS role_name FROM "user" LEFT JOIN "Role"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "nama"}).AddRow("user-1", "Pegawai"))

	_, _, err := (GormStore{DB: db}).ListEmployees(context.Background(), listing.Options{
		Sort: "nama", Order: "desc", Page: 1, PageSize: 20,
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
	mock.ExpectQuery(regexp.QuoteMeta(`ORDER BY "user"."nama" ASC`)).
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

func TestAuditLogStreamStreamsAllBatches(t *testing.T) {
	db, mock := newListStore(t)

	mock.ExpectQuery(regexp.QuoteMeta(`ORDER BY "createdAt" ASC, "id" ASC`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "aksi"}).
			AddRow("log-1", "employee.updated").
			AddRow("log-2", "session.issued"))
	// Batch berikutnya kosong menandai akhir stream.
	mock.ExpectQuery(regexp.QuoteMeta(`ORDER BY "createdAt" ASC, "id" ASC`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "aksi"}))

	var streamed []string
	err := (GormStore{DB: db}).StreamAuditLogs(context.Background(), listing.Options{}, func(entry models.AuditLog) error {
		streamed = append(streamed, entry.Aksi)
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(streamed) != 2 || streamed[0] != "employee.updated" || streamed[1] != "session.issued" {
		t.Fatalf("unexpected streamed actions: %v", streamed)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// TestListCutiQuotesCamelCaseColumns memastikan predikat cakupan memakai
// identifier berkutip ganda. Tanpa kutip PostgreSQL melipat nama menjadi
// huruf kecil dan query gagal dengan "column does not exist".
func TestListCutiQuotesCamelCaseColumns(t *testing.T) {
	db, mock := newListStore(t)

	mock.ExpectQuery(`count\(\*\) FROM "Cuti" WHERE "userId" IN \(SELECT "id" FROM "user" WHERE "homebaseId" IN`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`SELECT \* FROM "Cuti" WHERE "userId" IN \(SELECT "id" FROM "user" WHERE "homebaseId" IN`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "userId"}))

	_, _, err := (GormStore{DB: db}).ListCuti(context.Background(), listing.Options{
		ScopeHomebaseIDs: []string{"unit-a"}, Page: 1, PageSize: 20,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// TestAuditLogStreamOmitsKeysetOnFirstBatch memastikan batch pertama tidak
// memakai predikat keyset. Nilai awal keyset adalah zero value, dan string
// kosong ditolak PostgreSQL sebagai uuid.
func TestAuditLogStreamOmitsKeysetOnFirstBatch(t *testing.T) {
	db, mock := newListStore(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "auditLog" ORDER BY "createdAt" ASC, "id" ASC LIMIT $1`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "aksi"}).AddRow("log-1", "employee.updated"))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "auditLog" WHERE ("createdAt", "id") > ($1, $2) ORDER BY "createdAt" ASC, "id" ASC LIMIT $3`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "aksi"}))

	err := (GormStore{DB: db}).StreamAuditLogs(context.Background(), listing.Options{}, func(models.AuditLog) error {
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestEmployeeStreamStreamsAllBatches(t *testing.T) {
	db, mock := newListStore(t)

	mock.ExpectQuery(regexp.QuoteMeta(`ORDER BY "user"."nama" ASC, "user"."id" ASC`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "nama"}).
			AddRow("user-1", "Agus").
			AddRow("user-2", "Budi"))
	// Batch berikutnya kosong menandai akhir stream.
	mock.ExpectQuery(regexp.QuoteMeta(`ORDER BY "user"."nama" ASC, "user"."id" ASC`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "nama"}))

	var streamed []string
	err := (GormStore{DB: db}).StreamEmployees(context.Background(), listing.Options{}, func(user models.User) error {
		streamed = append(streamed, user.Nama)
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(streamed) != 2 || streamed[0] != "Agus" || streamed[1] != "Budi" {
		t.Fatalf("unexpected streamed names: %v", streamed)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// TestEmployeeStreamOmitsKeysetOnFirstBatch memastikan batch pertama tidak
// memakai predikat keyset. Nilai awal keyset adalah zero value, dan string
// kosong ditolak PostgreSQL sebagai uuid.
func TestEmployeeStreamOmitsKeysetOnFirstBatch(t *testing.T) {
	db, mock := newListStore(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "user" WHERE "user"."aktif" = $1 ORDER BY "user"."nama" ASC, "user"."id" ASC LIMIT $2`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "nama"}).AddRow("user-1", "Agus"))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "user" WHERE "user"."aktif" = $1 AND ("user"."nama", "user"."id") > ($2, $3) ORDER BY "user"."nama" ASC, "user"."id" ASC LIMIT $4`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "nama"}))

	err := (GormStore{DB: db}).StreamEmployees(context.Background(), listing.Options{}, func(models.User) error {
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// TestEmployeeStreamIgnoresSortAndPaging memastikan opsi urutan dan halaman
// tidak menembus ekspor: urutan selalu nama naik lalu id naik, tanpa OFFSET.
func TestEmployeeStreamIgnoresSortAndPaging(t *testing.T) {
	db, mock := newListStore(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "user" WHERE "user"."aktif" = $1 ORDER BY "user"."nama" ASC, "user"."id" ASC LIMIT $2`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "nama"}))

	err := (GormStore{DB: db}).StreamEmployees(context.Background(), listing.Options{
		Sort: `nama; DROP TABLE "user"`, Order: "desc", Page: 9, PageSize: 5,
	}, func(models.User) error {
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// TestEmployeeStreamAppliesScopeFilter memastikan cakupan unit diteruskan ke
// predikat IN dengan kolom berkutip ganda.
func TestEmployeeStreamAppliesScopeFilter(t *testing.T) {
	db, mock := newListStore(t)

	mock.ExpectQuery(regexp.QuoteMeta(`"user"."homebaseId" IN`)).
		WithArgs(true, "unit-a", "unit-a-child", 1000).
		WillReturnRows(sqlmock.NewRows([]string{"id", "nama"}))

	err := (GormStore{DB: db}).StreamEmployees(context.Background(), listing.Options{
		ScopeHomebaseIDs: []string{"unit-a", "unit-a-child"},
	}, func(models.User) error {
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
