package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
	payrollapp "simpeg/backend/internal/payroll/app"
	payrolldomain "simpeg/backend/internal/payroll/domain"
	"simpeg/backend/internal/payroll/repository"
)

// fakePayrollRepository adalah repository penggajian sintetis untuk handler.
type fakePayrollRepository struct {
	slips     []models.Gaji
	recorded  []payrolldomain.Notification
	schedules []models.JadwalGaji
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
	return models.Gaji{}, payrolldomain.ErrNotFound
}

func (r *fakePayrollRepository) RecordPayment(_ context.Context, _ string, slipID string, input payrolldomain.PaymentInput, notification payrolldomain.Notification) (models.Gaji, error) {
	r.recorded = append(r.recorded, notification)
	for _, slip := range r.slips {
		if slip.ID == slipID {
			slip.Status = models.StatusSlipTerbayar
			slip.TanggalDibayar = &input.TanggalBayar
			return slip, nil
		}
	}
	return models.Gaji{}, payrolldomain.ErrNotFound
}

func (r *fakePayrollRepository) ConfirmedPayment(context.Context, string, string) (bool, error) {
	return false, nil
}

func (r *fakePayrollRepository) UpsertSchedule(_ context.Context, _ string, input payrolldomain.ScheduleInput) (models.JadwalGaji, error) {
	for index := range r.schedules {
		if r.schedules[index].UnitId == input.UnitID {
			r.schedules[index].TanggalGajian = input.TanggalGajian
			return r.schedules[index], nil
		}
	}
	created := models.JadwalGaji{
		BaseModel:     models.BaseModel{IDModel: models.IDModel{ID: "jadwal-1"}},
		UnitId:        input.UnitID,
		TanggalGajian: input.TanggalGajian,
	}
	r.schedules = append(r.schedules, created)
	return created, nil
}

func (r *fakePayrollRepository) ListSchedules(_ context.Context, unitIDs []string) ([]models.JadwalGaji, error) {
	if unitIDs == nil {
		return r.schedules, nil
	}
	items := make([]models.JadwalGaji, 0, len(r.schedules))
	for _, schedule := range r.schedules {
		for _, unitID := range unitIDs {
			if schedule.UnitId == unitID {
				items = append(items, schedule)
			}
		}
	}
	return items, nil
}

func (r *fakePayrollRepository) FindDueReminders(context.Context, time.Time) ([]repository.Reminder, error) {
	return nil, nil
}

func (r *fakePayrollRepository) EnqueueNotifications(_ context.Context, notifications []payrolldomain.Notification) (int, error) {
	return len(notifications), nil
}

func payrollTestMux(repo *fakePayrollRepository, role string, grants map[string]map[auth.Permission]bool) *http.ServeMux {
	policy := auth.StaticPolicy(grants)
	return NewMuxWithOptions(MuxOptions{
		Resolver: principalResolver{principal: auth.Principal{UserID: "user-1", RoleName: role}},
		Policy:   policy,
		Payroll:  payrollapp.NewService(repo, policy, nil),
	})
}

func payrollTestSlip() models.Gaji {
	return models.Gaji{
		BaseModel:   models.BaseModel{IDModel: models.IDModel{ID: "11111111-1111-4111-8111-111111111111"}},
		UserId:      "user-1",
		NomorSlip:   "SLIP-2026-11-USER1",
		Periode:     time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
		TanggalGaji: time.Date(2026, 11, 28, 0, 0, 0, 0, time.UTC),
		GajiPokok:   3_500_000,
		TotalGaji:   4_250_000,
		Status:      models.StatusSlipTerbit,
		Pegawai:     &models.User{Nama: "Budi", Email: "budi@example.test"},
	}
}

func TestPayrollRoutesRequireAuthenticationAndPermission(t *testing.T) {
	mux := payrollTestMux(&fakePayrollRepository{}, "operator", nil)
	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/payroll/slips", nil)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", response.Code)
	}
}

func TestPayrollListFormatsAmountsAsRupiah(t *testing.T) {
	repo := &fakePayrollRepository{slips: []models.Gaji{payrollTestSlip()}}
	mux := payrollTestMux(repo, "pegawai", map[string]map[auth.Permission]bool{
		"pegawai": {"payroll:read:self": true},
	})
	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/payroll/slips", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
	var body PageDTO[PayrollDTO]
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 1 {
		t.Fatalf("items = %d, want 1", len(body.Items))
	}
	item := body.Items[0]
	if item.TotalGaji != "Rp 4.250.000" {
		t.Fatalf("totalGaji = %q", item.TotalGaji)
	}
	if item.GajiPokok != "Rp 3.500.000" {
		t.Fatalf("gajiPokok = %q", item.GajiPokok)
	}
	if item.Periode != "2026-11" {
		t.Fatalf("periode = %q", item.Periode)
	}
	if item.PeriodeLabel != "November 2026" {
		t.Fatalf("periodeLabel = %q", item.PeriodeLabel)
	}
}

func TestPayrollListRejectsInvalidPeriod(t *testing.T) {
	mux := payrollTestMux(&fakePayrollRepository{}, "pegawai", map[string]map[auth.Permission]bool{
		"pegawai": {"payroll:read:self": true},
	})
	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/payroll/slips?periode=bukan-periode", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", response.Code)
	}
}

func TestPayrollPaymentRequiresManagePermission(t *testing.T) {
	repo := &fakePayrollRepository{slips: []models.Gaji{payrollTestSlip()}}
	mux := payrollTestMux(repo, "pegawai", map[string]map[auth.Permission]bool{
		"pegawai": {"payroll:read:self": true},
	})
	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/payroll/slips/11111111-1111-4111-8111-111111111111/payment",
		strings.NewReader(`{"pelakuPengesahan":"Bendahara","tanggalBayar":"2026-11-28"}`))
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403, body = %s", response.Code, response.Body)
	}
	if len(repo.recorded) != 0 {
		t.Fatal("no payment may be recorded without permission")
	}
}

func TestPayrollPaymentRecordsConfirmation(t *testing.T) {
	repo := &fakePayrollRepository{slips: []models.Gaji{payrollTestSlip()}}
	mux := payrollTestMux(repo, "admin", map[string]map[auth.Permission]bool{
		"admin": {"payroll:read:any": true, "payroll:manage": true},
	})
	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/payroll/slips/11111111-1111-4111-8111-111111111111/payment",
		strings.NewReader(`{"pelakuPengesahan":"Bendahara","tanggalBayar":"2026-11-28"}`))
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
	if len(repo.recorded) != 1 {
		t.Fatalf("notifications = %d, want 1", len(repo.recorded))
	}
	var body PayrollDTO
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Status != string(models.StatusSlipTerbayar) {
		t.Fatalf("status = %q", body.Status)
	}
	if body.TanggalDibayar == nil || *body.TanggalDibayar != "28-11-2026" {
		t.Fatalf("tanggalDibayar = %v", body.TanggalDibayar)
	}
}

func TestPayrollScheduleRoutesRequirePermission(t *testing.T) {
	mux := payrollTestMux(&fakePayrollRepository{}, "operator", nil)
	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/payroll/schedules", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", response.Code)
	}
}

func TestListPayrollSchedulesAsAdmin(t *testing.T) {
	repo := &fakePayrollRepository{schedules: []models.JadwalGaji{{
		UnitId:        "11111111-1111-4111-8111-111111111111",
		TanggalGajian: 25,
		Unit:          &models.Homebase{Name: "Kantor Pusat"},
	}}}
	mux := payrollTestMux(repo, "admin", map[string]map[auth.Permission]bool{
		"admin": {"payroll:schedule": true},
	})
	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/payroll/schedules", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
	if !strings.Contains(response.Body.String(), `"tanggalGajian":25`) || !strings.Contains(response.Body.String(), "Kantor Pusat") {
		t.Fatalf("body = %s", response.Body)
	}
}

func TestUpsertPayrollScheduleRejectsInvalidDay(t *testing.T) {
	repo := &fakePayrollRepository{}
	mux := payrollTestMux(repo, "admin", map[string]map[auth.Permission]bool{
		"admin": {"payroll:schedule": true},
	})
	request := httptest.NewRequest(http.MethodPut, APIPrefix+"/payroll/schedules/11111111-1111-4111-8111-111111111111",
		strings.NewReader(`{"tanggalGajian":29}`))
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", response.Code)
	}
}

func TestUpsertPayrollScheduleStoresValue(t *testing.T) {
	repo := &fakePayrollRepository{}
	mux := payrollTestMux(repo, "admin", map[string]map[auth.Permission]bool{
		"admin": {"payroll:schedule": true},
	})
	request := httptest.NewRequest(http.MethodPut, APIPrefix+"/payroll/schedules/11111111-1111-4111-8111-111111111111",
		strings.NewReader(`{"tanggalGajian":27}`))
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
	if len(repo.schedules) != 1 || repo.schedules[0].TanggalGajian != 27 {
		t.Fatalf("schedules = %+v", repo.schedules)
	}
}

func TestUpsertPayrollScheduleOperatorWithEmptyScopeIsForbidden(t *testing.T) {
	repo := &fakePayrollRepository{}
	mux := payrollTestMux(repo, "operator", map[string]map[auth.Permission]bool{
		"operator": {"payroll:schedule:homebase": true},
	})
	request := httptest.NewRequest(http.MethodPut, APIPrefix+"/payroll/schedules/11111111-1111-4111-8111-111111111111",
		strings.NewReader(`{"tanggalGajian":27}`))
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", response.Code)
	}
	if len(repo.schedules) != 0 {
		t.Fatal("no schedule may be stored outside the operator scope")
	}
}
