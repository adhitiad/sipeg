package attendance

import (
	"context"
	"time"

	"gorm.io/gorm"

	"simpeg/backend/internal/crypto"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
	"simpeg/backend/internal/timeutil"
)

// RekamSortFields memetakan kunci sort ke kolom SQL untuk rekam presensi.
// Nilai di luar allowlist jatuh ke kolom bawaan sehingga query tidak pernah
// memakai kolom dari input mentah. Kunci "date" dipertahankan sebagai alias
// lama untuk "tanggal" agar klien yang sudah berjalan tidak berubah perilaku.
var RekamSortFields = map[string]string{
	"tanggal": `"tanggal"`,
	"date":    `"tanggal"`,
	"status":  `"status"`,
}

// myRecordsScope membangun query dasar rekam pribadi tanpa klausa SELECT.
// Pemanggil menambahkan Select("p.*") hanya pada query baris; klausa itu tidak
// boleh dipakai saat menghitung total karena GORM akan membentuk
// COUNT("p"."*") yang ditolak PostgreSQL (SQLSTATE 42703).
func myRecordsScope(db *gorm.DB, userID string, filter RecordFilter) *gorm.DB {
	query := db.Table(`"Presensi" AS p`).
		Joins(`JOIN "user" AS u ON u."id" = p."userId"`).
		Where(`p."userId" = ? AND p."tanggal" >= ? AND p."tanggal" < ?`, userID, filter.From, filter.Until)
	if filter.Status != "" {
		query = query.Where(`p."status" = ?`, filter.Status)
	}
	if filter.Search != "" {
		query = query.Where(`u."nama" ILIKE ?`, "%"+filter.Search+"%")
	}
	return query
}

func (s GormStore) ListMyRecords(ctx context.Context, userID string, filter RecordFilter) ([]models.Presensi, int64, error) {
	db := s.DB.WithContext(ctx)
	var total int64
	if err := myRecordsScope(db, userID, filter).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var records []models.Presensi
	order := listing.OrderClause(RekamSortFields, filter.Sort, filter.Order, "tanggal")
	err := myRecordsScope(db, userID, filter).Select(`p.*`).Order(order).
		Offset((filter.Page - 1) * filter.PageSize).
		Limit(filter.PageSize).
		Scan(&records).Error
	return records, total, err
}

type unitDayRow struct {
	UserID   string                 `gorm:"column:id"`
	Nama     string                 `gorm:"column:nama"`
	Nip      *crypto.Encrypted      `gorm:"column:nip"`
	UnitID   *string                `gorm:"column:unitId"`
	UnitName *string                `gorm:"column:unitName"`
	Status   *models.StatusPresensi `gorm:"column:status"`
	Note     *string                `gorm:"column:note"`
}

// unitDayScope menyusun query pegawai aktif dalam cakupan beserta catatan
// Presensi untuk satu tanggal. Dipisah dari ListUnitDay agar bentuk SQL dapat
// diperiksa tanpa database nyata.
func unitDayScope(tx *gorm.DB, homebaseIDs []string, day time.Time) *gorm.DB {
	start := timeutil.DateOnly(day)
	end := start.AddDate(0, 0, 1)
	query := tx.Table(`"user" AS u`).
		Select(`u."id" AS "id", u."nama" AS "nama", u."nip" AS "nip", u."homebaseId" AS "unitId",
			h."name" AS "unitName", p."status" AS "status", p."keterangan" AS "note"`).
		Joins(`LEFT JOIN "Homebase" AS h ON h."id" = u."homebaseId"`).
		Joins(`LEFT JOIN "Presensi" AS p ON p."userId" = u."id" AND p."tanggal" >= ? AND p."tanggal" < ?`,
			start, end).
		Where(`u."aktif" = ?`, true)
	if homebaseIDs != nil {
		query = query.Where(`u."homebaseId" IN ?`, homebaseIDs)
	}
	return query
}

// ListUnitDay mengembalikan seluruh pegawai aktif dalam cakupan beserta
// catatan Presensi untuk tanggal `day` (LEFT JOIN; pegawai tanpa catatan
// tampil dengan status NULL). `homebaseIDs` nil berarti lintas organisasi.
func (s GormStore) ListUnitDay(ctx context.Context, homebaseIDs []string, day time.Time) ([]UnitDayRow, error) {
	var rows []unitDayRow
	if err := unitDayScope(s.DB.WithContext(ctx), homebaseIDs, day).
		Order(`u."nama" ASC`).Scan(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]UnitDayRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, UnitDayRow(row))
	}
	return result, nil
}

func (s GormStore) ListUnitSummary(ctx context.Context, homebaseIDs []string, filter RecordFilter) ([]StatusCount, error) {
	query := s.DB.WithContext(ctx).Table(`"Presensi" AS p`).
		Select(`DATE(p."tanggal") AS date, p."status" AS status, COUNT(*) AS count`).
		Joins(`JOIN "user" AS u ON u."id" = p."userId"`).
		Where(`p."tanggal" >= ? AND p."tanggal" < ?`, filter.From, filter.Until)
	if homebaseIDs != nil {
		query = query.Where(`u."homebaseId" IN ?`, homebaseIDs)
	}
	var counts []StatusCount
	err := query.Group(`DATE(p."tanggal"), p."status"`).Order(`DATE(p."tanggal") ASC, p."status" ASC`).Scan(&counts).Error
	return counts, err
}

var _ RecordReader = GormStore{}
