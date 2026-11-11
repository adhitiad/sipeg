package app

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
	"simpeg/backend/internal/notify"
	"simpeg/backend/internal/payroll/domain"
	"simpeg/backend/internal/payroll/repository"
)

// fakePayrollRepository adalah repository penggajian sintetis.
type fakePayrollRepository struct {
	slips          []models.Gaji
	recorded       []domain.Notification
	recordedSlipID string
	confirmed      bool

	schedules    []models.JadwalGaji
	dueReminders []repository.Reminder
	enqueued     []domain.Notification
}

func (r *fakePayrollRepository) UpsertSchedule(_ context.Context, _ string, input domain.ScheduleInput) (models.JadwalGaji, error) {
	for index := range r.schedules {
		if r.schedules[index].UnitId == input.UnitID {
			r.schedules[index].TanggalGajian = input.TanggalGajian
			return r.schedules[index], nil
		}
	}
	created := models.JadwalGaji{UnitId: input.UnitID, TanggalGajian: input.TanggalGajian}
	created.ID = "jadwal-1"
	r.schedules = append(r.schedules, created)
	return created, nil
}

func (r *fakePayrollRepository) ListSchedules(_ context.Context, unitIDs []string) ([]models.JadwalGaji, error) {
	if unitIDs == nil {
		return r.schedules, nil
	}
	if len(unitIDs) == 0 {
		return []models.JadwalGaji{}, nil
	}
	items := make([]models.JadwalGaji, 0, len(r.schedules))
	for _, unitID := range unitIDs {
		for _, schedule := range r.schedules {
			if schedule.UnitId == unitID {
				items = append(items, schedule)
			}
		}
	}
	return items, nil
}

func (r *fakePayrollRepository) FindDueReminders(_ context.Context, _ time.Time) ([]repository.Reminder, error) {
	return r.dueReminders, nil
}

func (r *fakePayrollRepository) EnqueueNotifications(_ context.Context, notifications []domain.Notification) (int, error) {
	r.enqueued = append(r.enqueued, notifications...)
	return len(notifications), nil
}

func (r *fakePayrollRepository) ListSlips(_ context.Context, options listing.Options) ([]models.Gaji, int64, error) {
	items := r.slips
	if options.UserID != "" {
		filtered := make([]models.Gaji, 0, len(items))
		for _, slip := range items {
			if slip.UserId == options.UserID {
				filtered = append(filtered, slip)
			}
		}
		items = filtered
	}
	return items, int64(len(items)), nil
}

func (r *fakePayrollRepository) GetSlip(_ context.Context, id string) (models.Gaji, error) {
	for _, slip := range r.slips {
		if slip.ID == id {
			return slip, nil
		}
	}
	return models.Gaji{}, domain.ErrNotFound
}

func (r *fakePayrollRepository) RecordPayment(_ context.Context, _ string, slipID string, _ domain.PaymentInput, notification domain.Notification) (models.Gaji, error) {
	r.recorded = append(r.recorded, notification)
	r.recordedSlipID = slipID
	for _, slip := range r.slips {
		if slip.ID == slipID {
			slip.Status = models.StatusSlipTerbayar
			return slip, nil
		}
	}
	return models.Gaji{}, domain.ErrNotFound
}

func (r *fakePayrollRepository) ConfirmedPayment(_ context.Context, _ string, _ string) (bool, error) {
	return r.confirmed, nil
}

// fakeScopeRepository mengembalikan unit turunan yang statis.
type fakeScopeRepository struct {
	unitIDs []string
}

func (r *fakeScopeRepository) ListUnitScopeIDs(_ context.Context, _ string) ([]string, error) {
	return r.unitIDs, nil
}

func adminPolicy() auth.Policy {
	return auth.StaticPolicy{
		"admin":    {"payroll:read:any": true, "payroll:manage": true, "payroll:schedule": true},
		"pegawai":  {"payroll:read:self": true},
		"operator": {"payroll:read:homebase": true, "payroll:schedule:homebase": true},
	}
}

func sampleSlip() models.Gaji {
	period := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	return models.Gaji{
		BaseModel: models.BaseModel{IDModel: models.IDModel{ID: "slip-1"}},
		UserId:    "user-1",
		NomorSlip: "SLIP-2026-11-user-1",
		Periode:   period,
		GajiPokok: 3_500_000,
		TotalGaji: 4_250_000,
		Status:    models.StatusSlipTerbit,
		Pegawai:   &models.User{Nama: "Budi", Email: "budi@example.test"},
	}
}

func TestPayrollListLimitsEmployeeToOwnSlip(t *testing.T) {
	repo := &fakePayrollRepository{slips: []models.Gaji{sampleSlip()}}
	service := NewService(repo, adminPolicy(), &fakeScopeRepository{})

	principal := auth.Principal{UserID: "user-1", RoleName: "pegawai"}
	page, err := service.List(context.Background(), principal, listing.Options{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("items = %d, want 1", len(page.Items))
	}
}

func TestPayrollListRejectsRoleWithoutReadPermission(t *testing.T) {
	repo := &fakePayrollRepository{}
	service := NewService(repo, adminPolicy(), &fakeScopeRepository{})

	principal := auth.Principal{UserID: "guest-1", RoleName: "tamu"}
	if _, err := service.List(context.Background(), principal, listing.Options{Page: 1, PageSize: 20}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("err = %v, want forbidden", err)
	}
}

func TestPayrollGetRejectsEmployeeOutsideOwnSlip(t *testing.T) {
	repo := &fakePayrollRepository{slips: []models.Gaji{sampleSlip()}}
	service := NewService(repo, adminPolicy(), &fakeScopeRepository{})

	principal := auth.Principal{UserID: "user-2", RoleName: "pegawai"}
	if _, err := service.Get(context.Background(), principal, "slip-1"); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("err = %v, want forbidden", err)
	}
}

func TestRecordPaymentEnqueuesSalaryNoticeWithoutAmount(t *testing.T) {
	repo := &fakePayrollRepository{slips: []models.Gaji{sampleSlip()}}
	service := NewService(repo, adminPolicy(), &fakeScopeRepository{})

	principal := auth.Principal{UserID: "admin-1", RoleName: "admin"}
	input := domain.PaymentInput{
		PelakuPengesahan: "Bendahara",
		TanggalBayar:     time.Date(2026, 11, 28, 0, 0, 0, 0, time.UTC),
	}
	if _, err := service.RecordPayment(context.Background(), principal, "slip-1", input); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.recorded) != 1 {
		t.Fatalf("notifications = %d, want 1", len(repo.recorded))
	}
	notification := repo.recorded[0]
	if notification.Kind != notify.KindSalaryPaid {
		t.Fatalf("kind = %q", notification.Kind)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(notification.Payload), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["recipient"] != "budi@example.test" {
		t.Fatalf("recipient = %v", payload["recipient"])
	}
	if payload["period"] != "2026-11" {
		t.Fatalf("period = %v", payload["period"])
	}
	for _, forbidden := range []string{"amount", "total", "gajiPokok", "nominal"} {
		if _, exists := payload[forbidden]; exists {
			t.Fatalf("payload leaked field %q", forbidden)
		}
	}
}

func TestRecordPaymentRejectsRoleWithoutManagePermission(t *testing.T) {
	repo := &fakePayrollRepository{slips: []models.Gaji{sampleSlip()}}
	service := NewService(repo, adminPolicy(), &fakeScopeRepository{})

	principal := auth.Principal{UserID: "user-1", RoleName: "pegawai"}
	input := domain.PaymentInput{
		PelakuPengesahan: "Bendahara",
		TanggalBayar:     time.Date(2026, 11, 28, 0, 0, 0, 0, time.UTC),
	}
	if _, err := service.RecordPayment(context.Background(), principal, "slip-1", input); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("err = %v, want forbidden", err)
	}
	if len(repo.recorded) != 0 {
		t.Fatal("no notification may be queued without permission")
	}
}

func TestRecordPaymentRejectsDraftSlip(t *testing.T) {
	slip := sampleSlip()
	slip.Status = models.StatusSlipDraf
	repo := &fakePayrollRepository{slips: []models.Gaji{slip}}
	service := NewService(repo, adminPolicy(), &fakeScopeRepository{})

	principal := auth.Principal{UserID: "admin-1", RoleName: "admin"}
	input := domain.PaymentInput{
		PelakuPengesahan: "Bendahara",
		TanggalBayar:     time.Date(2026, 11, 28, 0, 0, 0, 0, time.UTC),
	}
	if _, err := service.RecordPayment(context.Background(), principal, "slip-1", input); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("err = %v, want conflict", err)
	}
}

func TestConfirmedPaymentDelegatesToRepository(t *testing.T) {
	repo := &fakePayrollRepository{confirmed: true}
	service := NewService(repo, adminPolicy(), &fakeScopeRepository{})

	confirmed, err := service.ConfirmedPayment(context.Background(), "user-1", "2026-11")
	if err != nil {
		t.Fatal(err)
	}
	if !confirmed {
		t.Fatal("expected the payment to be confirmed")
	}
}
