package attendance

import (
	"context"
	"fmt"
	"time"

	"simpeg/backend/internal/audit"
	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/models"
)

// attendanceManualSources memuat nilai `sumber` yang dianggap manual dan
// tidak boleh ditimpa derivasi. Baris tanpa sumber dianggap manual (aman:
// derivasi tidak menimpa data lama yang tidak jelas asalnya).
var attendanceManualSources = []string{"manual", "import_manual", "rekap_manual", "", "seed", "presensi_manual"}

func windowContains(start, end, day time.Time) bool {
	return !day.Before(start) && !day.After(end)
}

// Derive menurunkan status harian `Presensi` dari `AttendanceEvent` untuk
// pegawai yang terpetakan, dalam rentang [from, to] (maksimal 93 hari).
// Baris manual (sumber ≠ 'derivasi_mesin') tidak pernah ditimpa.
func (s *AdminService) Derive(ctx context.Context, principal auth.Principal, from, to time.Time) (DeriveSummary, error) {
	scopeIDs, err := s.scopeIDs(ctx, principal)
	if err != nil {
		return DeriveSummary{}, err
	}
	if from.IsZero() || to.IsZero() || to.Before(from) {
		return DeriveSummary{}, ErrAlpaSuspectRange
	}
	from = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	to = time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, time.UTC)
	if to.Sub(from) > maxDeriveWindowDays*24*time.Hour {
		return DeriveSummary{}, ErrAlpaSuspectRange
	}

	employees, err := s.store.MappedEmployees(ctx, scopeIDs)
	if err != nil {
		return DeriveSummary{}, fmt.Errorf("list mapped employees: %w", err)
	}

	events, err := s.store.EventsPerPinPerDay(ctx, from, to)
	if err != nil {
		return DeriveSummary{}, fmt.Errorf("read attendance events: %w", err)
	}
	pinToUser := make(map[string]string, len(employees))
	for _, employee := range employees {
		pinToUser[employee.MachinePin] = employee.UserID
	}
	tapPerUserPerDay := map[string]map[string]bool{}
	for day, dayEvents := range events {
		for pin := range dayEvents {
			userID, ok := pinToUser[pin]
			if !ok {
				continue
			}
			set := tapPerUserPerDay[userID]
			if set == nil {
				set = map[string]bool{}
				tapPerUserPerDay[userID] = set
			}
			set[day] = true
		}
	}

	cuti, err := s.store.ApprovedCutiPerUser(ctx, scopeIDs, from, to)
	if err != nil {
		return DeriveSummary{}, fmt.Errorf("read approved cuti: %w", err)
	}
	libur, err := s.store.HariLiburDates(ctx, from, to)
	if err != nil {
		return DeriveSummary{}, fmt.Errorf("read hari libur: %w", err)
	}

	summary := DeriveSummary{
		From:     from.Format("2006-01-02"),
		To:       to.Format("2006-01-02"),
		ByStatus: map[models.StatusPresensi]int64{},
	}
	for day := from; !day.After(to); day = day.AddDate(0, 0, 1) {
		dayKey := day.Format("2006-01-02")
		isSunday := day.Weekday() == time.Sunday
		isLibur := libur[dayKey]
		for _, employee := range employees {
			var status models.StatusPresensi
			switch {
			case tapPerUserPerDay[employee.UserID][dayKey]:
				status = models.StatusPresensiHadir
			default:
				if window, ok := cutiWindowFor(cuti[employee.UserID], day); ok {
					status = window.Status
				} else if isSunday || isLibur {
					status = models.StatusPresensiLibur
				} else {
					status = models.StatusPresensiAlpa
				}
			}

			existing, found, err := s.store.PresensiExists(ctx, employee.UserID, day)
			if err != nil {
				return DeriveSummary{}, err
			}
			row := models.Presensi{
				UserId:     employee.UserID,
				Tanggal:    day,
				Status:     status,
				Keterangan: keteranganFor(status, isSunday, isLibur),
				Sumber:     ptr("derivasi_mesin"),
			}
			login := principal.UserID
			row.UpdatedBy = &login
			if found {
				if isManualSource(existing) {
					summary.SkippedManual++
					continue
				}
				if err := s.store.UpdatePresensiDerived(ctx, row, principal.UserID); err != nil {
					return DeriveSummary{}, err
				}
				summary.Updated++
				summary.ByStatus[status]++
				continue
			}
			if err := s.store.InsertPresensi(ctx, row); err != nil {
				return DeriveSummary{}, err
			}
			summary.Created++
			summary.ByStatus[status]++
		}
	}

	note := fmt.Sprintf("rentang=%s..%s; dibuat=%d; diperbarui=%d; manual_dilewati=%d", summary.From, summary.To, summary.Created, summary.Updated, summary.SkippedManual)
	entry := audit.Entry{
		ActorID: principal.UserID,
		Action:  audit.ActionPresensiDerived,
		Note:    &note,
	}
	if principal.HomebaseID != "" {
		unitID := principal.HomebaseID
		entry.UnitID = &unitID
	}
	// Best-effort: data derivasi sudah tersimpan; kegagalan audit tidak
	// menggagalkan rangkaian ini (kegagalan dicatat pada respons).
	if err := s.store.WriteAudit(ctx, entry); err != nil {
		summary.AuditFailed = true
	}
	return summary, nil
}

func cutiWindowFor(windows []CutiWindow, day time.Time) (CutiWindow, bool) {
	for _, w := range windows {
		if windowContains(w.Start, w.End, day) {
			return w, true
		}
	}
	return CutiWindow{}, false
}

func keteranganFor(status models.StatusPresensi, isSunday, isLibur bool) *string {
	switch status {
	case models.StatusPresensiAlpa:
		return ptr("derivasi mesin: tidak ada tap pada hari kerja")
	case models.StatusPresensiLibur:
		if isLibur {
			return ptr("derivasi mesin: hari libur nasional")
		}
		return ptr("derivasi mesin: libur mingguan")
	default:
		return nil
	}
}

func isManualSource(source string) bool {
	for _, manual := range attendanceManualSources {
		if source == manual {
			return true
		}
	}
	return false
}
