package attendance

import (
	"context"

	"simpeg/backend/internal/models"
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
