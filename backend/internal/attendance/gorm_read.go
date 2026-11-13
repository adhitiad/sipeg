package attendance

import (
	"context"
	"time"

	"simpeg/backend/internal/crypto"
	"simpeg/backend/internal/models"
	"simpeg/backend/internal/timeutil"
)

func (s GormStore) ListMyRecords(ctx context.Context, userID string, filter RecordFilter) ([]models.Presensi, int64, error) {
	query := s.DB.WithContext(ctx).Model(&models.Presensi{}).
		Where(`"userId" = ? AND "tanggal" >= ? AND "tanggal" < ?`, userID, filter.From, filter.Until)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var records []models.Presensi
	err := query.Order(`"tanggal" DESC`).Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize).Find(&records).Error
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

// ListUnitDay mengembalikan seluruh pegawai aktif dalam cakupan beserta
// catatan Presensi untuk tanggal `day` (LEFT JOIN; pegawai tanpa catatan
// tampil dengan status NULL). `homebaseIDs` nil berarti lintas organisasi.
func (s GormStore) ListUnitDay(ctx context.Context, homebaseIDs []string, day time.Time) ([]UnitDayRow, error) {
	start := timeutil.DateOnly(day)
	end := start.AddDate(0, 0, 1)
	query := s.DB.WithContext(ctx).Table(`"user" AS u`).
		Select(`u."id" AS "id", u."nama" AS "nama", u."nip" AS "nip", u."homebaseId" AS "unitId",
			h."name" AS "unitName", p."status" AS "status", p."keterangan" AS "note"`).
		Joins(`LEFT JOIN "homebase" AS h ON h."id" = u."homebaseId"`).
		Joins(`LEFT JOIN "Presensi" AS p ON p."userId" = u."id" AND p."tanggal" >= ? AND p."tanggal" < ?`,
			start, end).
		Where(`u."aktif" = ?`, true)
	if homebaseIDs != nil {
		query = query.Where(`u."homebaseId" IN ?`, homebaseIDs)
	}
	var rows []unitDayRow
	if err := query.Order(`u."nama" ASC`).Scan(&rows).Error; err != nil {
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
