package httpserver

import (
	"errors"
	"net/http"
	"time"

	"simpeg/backend/internal/apierror"
	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/models"
	"simpeg/backend/internal/payroll/app"
	payrolldomain "simpeg/backend/internal/payroll/domain"
)

// PayrollScheduleDTO adalah bentuk publik jadwal tanggal gajian sebuah unit.
type PayrollScheduleDTO struct {
	UnitID        string    `json:"unitId"`
	UnitNama      string    `json:"unitNama"`
	TanggalGajian int       `json:"tanggalGajian"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// payrollScheduleAPI memegang use case jadwal tanggal gajian.
type payrollScheduleAPI struct {
	payroll *app.Service
}

// upsertScheduleRequest adalah bentuk body PUT jadwal tanggal gajian.
type upsertScheduleRequest struct {
	TanggalGajian int `json:"tanggalGajian"`
}

// listPayrollSchedules mengembalikan daftar jadwal tanggal gajian.
//
// Admin melihat seluruh unit. Operator hanya melihat unit dalam cakupan
// penugasannya beserta seluruh turunannya.
func (api payrollScheduleAPI) listPayrollSchedules(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		writeErrorResponse(w, http.StatusUnauthorized, apierror.CodeUnauthorized)
		return
	}
	schedules, err := api.payroll.ListSchedules(r.Context(), principal)
	if err != nil {
		writeScheduleError(w, err)
		return
	}
	items := make([]PayrollScheduleDTO, 0, len(schedules))
	for _, schedule := range schedules {
		items = append(items, payrollScheduleDTO(schedule))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

// upsertPayrollSchedule menyimpan jadwal tanggal gajian sebuah unit.
func (api payrollScheduleAPI) upsertPayrollSchedule(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		writeErrorResponse(w, http.StatusUnauthorized, apierror.CodeUnauthorized)
		return
	}
	unitID := r.PathValue("unitId")
	if !validUUID(unitID) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	var body upsertScheduleRequest
	if err := decodeJSON(w, r, &body); err != nil {
		return
	}
	schedule, err := api.payroll.UpsertSchedule(r.Context(), principal, payrolldomain.ScheduleInput{
		UnitID:        unitID,
		TanggalGajian: body.TanggalGajian,
	})
	if err != nil {
		writeScheduleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, payrollScheduleDTO(schedule))
}

func payrollScheduleDTO(schedule models.JadwalGaji) PayrollScheduleDTO {
	unitNama := ""
	if schedule.Unit != nil {
		unitNama = schedule.Unit.Name
	}
	return PayrollScheduleDTO{
		UnitID:        schedule.UnitId,
		UnitNama:      unitNama,
		TanggalGajian: schedule.TanggalGajian,
		UpdatedAt:     schedule.UpdatedAt,
	}
}

func writeScheduleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, payrolldomain.ErrScheduleValidation):
		writeValidationError(w)
	case errors.Is(err, payrolldomain.ErrNotFound):
		writeErrorResponse(w, http.StatusNotFound, apierror.CodeNotFound)
	case errors.Is(err, payrolldomain.ErrForbidden):
		writeErrorResponse(w, http.StatusForbidden, apierror.CodeForbidden)
	default:
		writeErrorResponse(w, http.StatusInternalServerError, apierror.CodeInternal)
	}
}
