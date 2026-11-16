package attendance

import (
	"strings"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
)

// newDryRunDB membuka GORM tanpa koneksi nyata. Dialek PostgreSQL tetap dipakai
// supaya bentuk SQL yang diperiksa sama dengan yang dikirim ke database target.
func newDryRunDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN: "host=127.0.0.1 user=uji dbname=uji sslmode=disable",
	}), &gorm.Config{
		DryRun:               true,
		DisableAutomaticPing: true,
		Logger:               logger.Discard,
	})
	if err != nil {
		t.Fatalf("membuka GORM dry-run: %v", err)
	}
	return db
}

func testRecordFilter() RecordFilter {
	from := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	return RecordFilter{
		From:     from,
		Until:    from.AddDate(0, 0, 14),
		Page:     1,
		PageSize: 20,
		Sort:     "tanggal",
		Order:    "desc",
	}
}

// Regresi: Select("p.*") sebelum Count membuat GORM menyusun COUNT("p"."*"),
// yang ditolak PostgreSQL dengan SQLSTATE 42703 sehingga endpoint menjawab 503.
func TestMyRecordsCountTidakMemakaiKolomBintang(t *testing.T) {
	db := newDryRunDB(t)
	filter := testRecordFilter()

	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		var total int64
		return myRecordsScope(tx, "user-uji", filter).Count(&total)
	})

	if strings.Contains(sql, `"p"."*"`) {
		t.Fatalf("query hitung masih memakai kolom bintang: %s", sql)
	}
	if !strings.Contains(strings.ToUpper(sql), "COUNT(*)") {
		t.Fatalf("query hitung tidak memakai COUNT(*): %s", sql)
	}
}

func TestMyRecordsQueryBarisMemilihKolomPresensi(t *testing.T) {
	db := newDryRunDB(t)
	filter := testRecordFilter()
	order := listing.OrderClause(RekamSortFields, filter.Sort, filter.Order, "tanggal")

	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		var records []models.Presensi
		return myRecordsScope(tx, "user-uji", filter).Select(`p.*`).Order(order).
			Offset((filter.Page - 1) * filter.PageSize).
			Limit(filter.PageSize).
			Find(&records)
	})

	if !strings.Contains(sql, "SELECT p.*") {
		t.Fatalf("query baris tidak memilih kolom presensi: %s", sql)
	}
	if !strings.Contains(sql, `ORDER BY "tanggal" DESC`) {
		t.Fatalf("query baris tidak mengurutkan menurut tanggal: %s", sql)
	}
	if !strings.Contains(sql, `p."userId" = `) {
		t.Fatalf("query baris tidak membatasi pemilik rekam: %s", sql)
	}
}

// Regresi (2026-11-14): JOIN memakai "homebase" huruf kecil ditolak PostgreSQL
// (SQLSTATE 42P01) sehingga endpoint unit-records menjawab 503. Tabel aplikasi
// bernama "Homebase".
func TestListUnitDayJoinMemakaiTabelHomebaseBertandaPetik(t *testing.T) {
	db := newDryRunDB(t)
	day := time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC)

	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		var rows []unitDayRow
		return unitDayScope(tx, []string{"20000000-0000-0000-0000-000000000002"}, day).
			Order(`u."nama" ASC`).Find(&rows)
	})

	if !strings.Contains(sql, `LEFT JOIN "Homebase"`) {
		t.Fatalf("query unit-records tidak memakai tabel \"Homebase\": %s", sql)
	}
	if strings.Contains(sql, `LEFT JOIN "homebase"`) {
		t.Fatalf("query unit-records memakai nama tabel huruf kecil: %s", sql)
	}
}

func TestMyRecordsQueryMenyaringStatusDanNama(t *testing.T) {
	db := newDryRunDB(t)
	filter := testRecordFilter()
	filter.Status = "H"
	filter.Search = "siti"

	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		var total int64
		return myRecordsScope(tx, "user-uji", filter).Count(&total)
	})

	if !strings.Contains(sql, `p."status" = `) {
		t.Fatalf("filter status tidak diterapkan: %s", sql)
	}
	if !strings.Contains(strings.ToUpper(sql), "ILIKE") {
		t.Fatalf("filter nama tidak diterapkan: %s", sql)
	}
}
