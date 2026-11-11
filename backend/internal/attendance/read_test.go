package attendance

import (
	"context"
	"testing"
	"time"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/models"
)

type fakeRecordReader struct {
	userID      string
	homebaseIDs []string
	records     []models.Presensi
	counts      []StatusCount
}

func (r *fakeRecordReader) ListMyRecords(_ context.Context, userID string, _ RecordFilter) ([]models.Presensi, int64, error) {
	r.userID = userID
	return r.records, int64(len(r.records)), nil
}

func (r *fakeRecordReader) ListUnitSummary(_ context.Context, homebaseIDs []string, _ RecordFilter) ([]StatusCount, error) {
	r.homebaseIDs = homebaseIDs
	return r.counts, nil
}

type fakeUnitScope struct {
	ids []string
}

func (s fakeUnitScope) ListUnitScopeIDs(context.Context, string) ([]string, error) {
	return s.ids, nil
}

func TestMyRecordsOnlyUsesAuthenticatedPrincipal(t *testing.T) {
	reader := &fakeRecordReader{}
	service := NewReadService(reader, nil, auth.StaticPolicy{
		"pegawai": {"attendance:read:self": true},
	})
	page, err := service.MyRecords(context.Background(), auth.Principal{
		UserID: "user-self", RoleName: "pegawai",
	}, RecordFilter{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("MyRecords error: %v", err)
	}
	if reader.userID != "user-self" || page.Page != 1 || page.PageSize != 20 {
		t.Fatalf("user=%q page=%#v", reader.userID, page)
	}
}

func TestUnitSummaryAppliesResolvedHierarchyAndGroupsStatuses(t *testing.T) {
	day := time.Date(2026, 11, 11, 0, 0, 0, 0, time.UTC)
	reader := &fakeRecordReader{counts: []StatusCount{
		{Date: day, Status: models.StatusPresensiHadir, Count: 3},
		{Date: day, Status: models.StatusPresensiIzin, Count: 1},
	}}
	service := NewReadService(reader, fakeUnitScope{ids: []string{"unit-a", "unit-child"}}, auth.StaticPolicy{
		"atasan": {"attendance:read:homebase": true},
	})
	summary, err := service.UnitSummary(context.Background(), auth.Principal{
		RoleName: "atasan", HomebaseID: "unit-a",
	}, RecordFilter{})
	if err != nil {
		t.Fatalf("UnitSummary error: %v", err)
	}
	if len(reader.homebaseIDs) != 2 || reader.homebaseIDs[1] != "unit-child" {
		t.Fatalf("homebase scope = %v", reader.homebaseIDs)
	}
	if len(summary) != 1 || summary[0].Counts[models.StatusPresensiHadir] != 3 || summary[0].Counts[models.StatusPresensiIzin] != 1 {
		t.Fatalf("summary = %#v", summary)
	}
	if summary[0].Counts[models.StatusPresensiAlpa] != 0 {
		t.Fatalf("missing status should be zero: %#v", summary[0].Counts)
	}
}

func TestUnitSummaryRejectsRoleWithoutGrant(t *testing.T) {
	service := NewReadService(&fakeRecordReader{}, fakeUnitScope{}, auth.StaticPolicy{})
	_, err := service.UnitSummary(context.Background(), auth.Principal{RoleName: "pegawai"}, RecordFilter{})
	if err != ErrForbidden {
		t.Fatalf("err = %v, want forbidden", err)
	}
}
