package attendance

import (
	"context"
	"errors"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/models"
)

var ErrForbidden = errors.New("attendance access is not permitted")

func (s *ReadService) MyRecords(ctx context.Context, principal auth.Principal, filter RecordFilter) (RecordPage, error) {
	if s.policy == nil || !s.policy.Allows(principal.RoleName, "attendance:read:self") || principal.UserID == "" {
		return RecordPage{}, ErrForbidden
	}
	items, total, err := s.reader.ListMyRecords(ctx, principal.UserID, filter)
	if err != nil {
		return RecordPage{}, err
	}
	return newRecordPage(items, total, filter), nil
}

func (s *ReadService) UnitSummary(ctx context.Context, principal auth.Principal, filter RecordFilter) ([]DailySummary, error) {
	var homebaseIDs []string
	switch {
	case s.policy != nil && s.policy.Allows(principal.RoleName, "attendance:read:any"):
	case s.policy != nil && s.policy.Allows(principal.RoleName, "attendance:read:homebase"):
		if principal.HomebaseID == "" || s.scope == nil {
			return []DailySummary{}, nil
		}
		var err error
		homebaseIDs, err = s.scope.ListUnitScopeIDs(ctx, principal.HomebaseID)
		if err != nil {
			return nil, err
		}
		if len(homebaseIDs) == 0 {
			return []DailySummary{}, nil
		}
	default:
		return nil, ErrForbidden
	}
	counts, err := s.reader.ListUnitSummary(ctx, homebaseIDs, filter)
	if err != nil {
		return nil, err
	}
	return aggregateStatusCounts(counts), nil
}

type DailySummary struct {
	Date   string                          `json:"date"`
	Counts map[models.StatusPresensi]int64 `json:"counts"`
}

func newRecordPage(items []models.Presensi, total int64, filter RecordFilter) RecordPage {
	pages := 0
	if total > 0 {
		pages = int((total + int64(filter.PageSize) - 1) / int64(filter.PageSize))
	}
	return RecordPage{Items: items, Page: filter.Page, PageSize: filter.PageSize, TotalItems: total, TotalPages: pages}
}

func aggregateStatusCounts(counts []StatusCount) []DailySummary {
	byDate := make(map[string]map[models.StatusPresensi]int64)
	var dates []string
	for _, count := range counts {
		date := count.Date.Format("2006-01-02")
		if _, ok := byDate[date]; !ok {
			byDate[date] = map[models.StatusPresensi]int64{
				models.StatusPresensiHadir: 0, models.StatusPresensiIzin: 0,
				models.StatusPresensiSakit: 0, models.StatusPresensiAlpa: 0,
				models.StatusPresensiCuti: 0, models.StatusPresensiLibur: 0,
			}
			dates = append(dates, date)
		}
		byDate[date][count.Status] = count.Count
	}
	result := make([]DailySummary, 0, len(dates))
	for _, date := range dates {
		result = append(result, DailySummary{Date: date, Counts: byDate[date]})
	}
	return result
}
