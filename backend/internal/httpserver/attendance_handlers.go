package httpserver

import (
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"simpeg/backend/internal/apierror"
	"simpeg/backend/internal/attendance"
	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/models"

	"github.com/google/uuid"
)

const attendanceIngestPermission auth.Permission = "attendance:event:ingest"

type attendanceAPI struct {
	store attendance.Store
}

type attendanceEventRequest struct {
	MachineID      string     `json:"machineId"`
	MachinePIN     string     `json:"machinePin"`
	OccurredAt     *time.Time `json:"occurredAt"`
	IdempotencyKey string     `json:"idempotencyKey"`
}

type attendanceEventResponse struct {
	EventID   string `json:"eventId"`
	Duplicate bool   `json:"duplicate"`
}

func RegisterAttendanceRoutes(mux *http.ServeMux, resolver auth.SessionResolver, policy auth.Policy, store attendance.Store) {
	if store == nil {
		return
	}
	api := attendanceAPI{store: store}
	handler := auth.Authenticate(resolver)(
		auth.RequirePermission(policy, attendanceIngestPermission, http.HandlerFunc(api.createEvent)),
	)
	mux.Handle("POST "+APIPrefix+"/attendance/events", handler)
}

func (api attendanceAPI) createEvent(w http.ResponseWriter, r *http.Request) {
	var request attendanceEventRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeBadRequest)
		return
	}

	machineID, err := uuid.Parse(request.MachineID)
	if err != nil {
		writeValidationError(w)
		return
	}
	if strings.TrimSpace(request.MachinePIN) == "" || utf8.RuneCountInString(request.MachinePIN) > 100 ||
		strings.TrimSpace(request.IdempotencyKey) == "" || utf8.RuneCountInString(request.IdempotencyKey) > 200 ||
		request.OccurredAt == nil {
		writeValidationError(w)
		return
	}

	event, duplicate, err := api.store.SaveEvent(r.Context(), currentActor(r), models.AttendanceEvent{
		MachineId:      machineID.String(),
		MachinePin:     request.MachinePIN,
		OccurredAt:     request.OccurredAt.UTC(),
		IdempotencyKey: request.IdempotencyKey,
	})
	switch {
	case errors.Is(err, attendance.ErrMachineNotFound):
		writeErrorResponse(w, http.StatusNotFound, "machine_not_found")
	case errors.Is(err, attendance.ErrEventConflict):
		writeErrorResponse(w, http.StatusConflict, "idempotency_key_conflict")
	case err != nil:
		writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
	case duplicate:
		writeJSON(w, http.StatusOK, attendanceEventResponse{EventID: event.ID, Duplicate: true})
	default:
		writeJSON(w, http.StatusCreated, attendanceEventResponse{EventID: event.ID})
	}
}
