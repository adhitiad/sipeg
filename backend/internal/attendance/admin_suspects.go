package attendance

import (
	"context"
	"sort"
	"time"

	"simpeg/backend/internal/auth"
)

// AlpaSuspects mengembalikan pegawai yang alpa pada SEMUA bulan berturut-turut
// dalam rentang `months` bulan terakhir yang berakhir pada `until` (inklusif).
// Laporan ini untuk ditindaklanjuti manual oleh Kabiro/yayasan (SP1 tidak
// diterbitkan otomatis oleh sistem).
func (s *AdminService) AlpaSuspects(ctx context.Context, principal auth.Principal, until time.Time, months int) ([]AlpaSuspect, error) {
	scopeIDs, err := s.scopeIDs(ctx, principal)
	if err != nil {
		return nil, err
	}
	if until.IsZero() {
		now := time.Now()
		until = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	} else {
		until = time.Date(until.Year(), until.Month(), 1, 0, 0, 0, 0, time.UTC)
	}
	if months < 2 {
		months = 2
	}
	if months > 12 {
		return nil, ErrAlpaSuspectRange
	}
	monthKeys := make([]string, 0, months)
	for i := months - 1; i >= 0; i-- {
		m := until.AddDate(0, -i, 0)
		monthKeys = append(monthKeys, m.Format("2006-01"))
	}
	from := until.AddDate(0, -(months - 1), 0)
	to := until.AddDate(0, 1, 0)

	rows, err := s.store.CountAlpaPerMonth(ctx, scopeIDs, from, to)
	if err != nil {
		return nil, err
	}
	byUser := map[string]map[string]int64{}
	for _, row := range rows {
		m := byUser[row.UserID]
		if m == nil {
			m = map[string]int64{}
			byUser[row.UserID] = m
		}
		m[row.Month] = row.Count
	}

	suspects := []AlpaSuspect{}
	for userID, counts := range byUser {
		report := AlpaSuspect{UserID: userID, Alpa: map[string]int64{}}
		allMonths := true
		for _, month := range monthKeys {
			count := counts[month]
			report.Alpa[month] = count
			report.TotalAlpa += count
			if count == 0 {
				allMonths = false
			}
		}
		if !allMonths || report.TotalAlpa == 0 {
			continue
		}
		suspects = append(suspects, report)
	}
	if len(suspects) == 0 {
		return suspects, nil
	}

	users, err := s.store.EmployeesInScope(ctx, scopeIDs, userIDs(suspects))
	if err != nil {
		return nil, err
	}
	byID := map[string]struct {
		nama     string
		homebase *string
	}{}
	for _, user := range users {
		byID[user.ID] = struct {
			nama     string
			homebase *string
		}{user.Nama, user.HomebaseId}
	}
	final := suspects[:0]
	for _, suspect := range suspects {
		id, ok := byID[suspect.UserID]
		if !ok {
			continue
		}
		suspect.Nama = id.nama
		suspect.HomebaseID = id.homebase
		final = append(final, suspect)
	}
	sort.Slice(final, func(i, j int) bool {
		if final[i].TotalAlpa == final[j].TotalAlpa {
			return final[i].UserID < final[j].UserID
		}
		return final[i].TotalAlpa > final[j].TotalAlpa
	})
	return final, nil
}

func userIDs(items []AlpaSuspect) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.UserID)
	}
	return out
}
