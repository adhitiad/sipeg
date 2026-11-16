package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/models"
	"simpeg/backend/internal/notify"
	"simpeg/backend/internal/payroll/domain"

	"simpeg/backend/internal/timeutil"
)

// ListSchedules mengembalikan daftar jadwal tanggal gajian.
//
// Admin melihat seluruh unit. Operator hanya melihat unit yang berada dalam
// cakupan penugasannya beserta seluruh turunannya.
func (s *Service) ListSchedules(ctx context.Context, principal auth.Principal) ([]models.JadwalGaji, error) {
	unitIDs, err := scheduleScope(ctx, s, principal, "payroll:schedule", "payroll:schedule:homebase")
	if err != nil {
		return nil, err
	}
	return s.repo.ListSchedules(ctx, unitIDs)
}

// UpsertSchedule menyimpan atau memperbarui jadwal tanggal gajian sebuah unit.
//
// Admin boleh mengatur jadwal unit mana pun. Operator hanya boleh mengatur
// unit dalam cakupan penugasannya.
func (s *Service) UpsertSchedule(ctx context.Context, principal auth.Principal, input domain.ScheduleInput) (models.JadwalGaji, error) {
	if err := input.Validate(); err != nil {
		return models.JadwalGaji{}, err
	}
	if allows(ctx, s.policy, principal, "payroll:schedule") {
		return s.repo.UpsertSchedule(ctx, principal.UserID, input)
	}
	if !allows(ctx, s.policy, principal, "payroll:schedule:homebase") {
		return models.JadwalGaji{}, domain.ErrForbidden
	}
	scope, err := s.scope.resolve(ctx, principal)
	if err != nil {
		return models.JadwalGaji{}, err
	}
	if !scope.Contains(input.UnitID) {
		return models.JadwalGaji{}, domain.ErrForbidden
	}
	return s.repo.UpsertSchedule(ctx, principal.UserID, input)
}

// ProcessReminders mengantrekan pemberitahuan ketersediaan slip gaji H-1.
//
// Slip yang diberitahukan adalah seluruh slip milik pegawai yang unitnya
// memiliki jadwal gajian, dengan periode tidak melewati periode berjalan,
// dan yang tanggal pemberitahuan H-1-nya sudah lewat. Pemberitahuan memuat
// nama, periode, tautan, dan QR halaman verifikasi publik slip; nominal
// tidak pernah ikut. Slip yang pesannya pernah diantrekan dilewati oleh
// kunci idempotensi antrean keluar, sehingga pekerjaan ini boleh dijalankan
// berulang.
//
// Slip tanpa token verifikasi (dibuat sebelum fitur tersedia) dilewati dan
// dihitung pada hasil untuk transparansi; slip tersebut tidak mendapat
// pemberitahuan sampai ditandai ulang oleh admin.
func (s *Service) ProcessReminders(ctx context.Context, now time.Time) (int, error) {
	today := timeutil.DateOnly(now)
	rows, err := s.repo.FindDueReminders(ctx, today)
	if err != nil {
		return 0, err
	}

	notifications := make([]domain.Notification, 0, len(rows))
	for _, row := range rows {
		period := domain.Period{Year: row.Periode.Year(), Month: row.Periode.Month()}
		if !domain.IsReminderDue(period, row.TanggalGajian, today) {
			continue
		}
		if row.KodeVerifikasi == nil || strings.TrimSpace(*row.KodeVerifikasi) == "" {
			continue // slip tanpa token (data lama); tidak ada tautan verifikasi publik yang layak
		}
		payload, err := notify.SlipAvailablePayload(row.Email, row.Nama, period.Key(), row.UserID, *row.KodeVerifikasi)
		if err != nil {
			return 0, fmt.Errorf("build slip notice: %w", err)
		}
		notifications = append(notifications, domain.Notification{
			Kind:      notify.KindSlipAvailable,
			DedupeKey: "slip-available:" + row.SlipID,
			Payload:   string(payload),
		})
	}
	queued, err := s.repo.EnqueueNotifications(ctx, notifications)
	if err != nil {
		return 0, err
	}
	// Slip tanpa token dilewati tanpa pemberitahuan; jumlahnya hanya
	// dapat diamati lewat selisih antara jumlah slip yang jatuh tempo dan
	// yang diantrekan. Slip baru selalu memiliki token (dibuat oleh mesin
	// hitung), sehingga keadaan ini hanya mungkin pada data lama.
	return queued, nil
}

// scheduleScope menentukan daftar unit yang boleh dilihat principal.
//
// Hasil nil berarti seluruh unit (permission global). Hasil kosong berarti
// tidak ada unit yang boleh dijangkau.
func scheduleScope(ctx context.Context, s *Service, principal auth.Principal, globalPermission, scopedPermission auth.Permission) ([]string, error) {
	if allows(ctx, s.policy, principal, globalPermission) {
		return nil, nil
	}
	if !allows(ctx, s.policy, principal, scopedPermission) {
		return nil, domain.ErrForbidden
	}
	scope, err := s.scope.resolve(ctx, principal)
	if err != nil {
		return nil, err
	}
	if scope.Empty() {
		// Slice kosong berarti tidak ada unit yang boleh dijangkau; nilai
		// ini berbeda dari nil yang berarti seluruh unit.
		return []string{}, nil
	}
	return scope.List(), nil
}
