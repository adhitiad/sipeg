package httpserver

import (
	"context"
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

// deviceIDHeader adalah header yang mengirimkan ID mesin saat autentikasi perangkat.
const deviceIDHeader = "X-Device-ID"

type deviceActorKey struct{}

type attendanceAPI struct {
	store      attendance.Store
	deviceAuth attendance.DeviceAuthenticator
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

func RegisterAttendanceRoutes(mux *http.ServeMux, resolver auth.SessionResolver, policy auth.Policy, store attendance.Store, deviceAuth attendance.DeviceAuthenticator) {
	if store == nil {
		return
	}
	api := attendanceAPI{store: store, deviceAuth: deviceAuth}
	var handler http.Handler = http.HandlerFunc(api.createEvent)

	if deviceAuth != nil {
		// Device auth tersedia: coba autentikasi perangkat terlebih dahulu.
		// Bila tidak ada kredensial perangkat, fallback ke sesi pengguna + permission.
		handler = requireDeviceOrSession(deviceAuth, resolver, policy, attendanceIngestPermission)(handler)
	} else {
		handler = auth.Authenticate(resolver)(
			auth.RequirePermission(policy, attendanceIngestPermission, handler),
		)
	}
	mux.Handle("POST "+APIPrefix+"/attendance/events", handler)
}

// requireDeviceOrSession mengizinkan akses lewat salah satu:
// 1. Autentikasi perangkat (X-Device-ID + Authorization: Bearer <device-token>).
// 2. Sesi pengguna dengan permission yang diberikan.
// Bila kredensial perangkat disertakan tetapi tidak valid, permintaan ditolak 401
// tanpa fallback ke sesi.
func requireDeviceOrSession(deviceAuth attendance.DeviceAuthenticator, resolver auth.SessionResolver, policy auth.Policy, permission auth.Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			deviceID := strings.TrimSpace(r.Header.Get(deviceIDHeader))
			deviceToken := BearerToken(r.Header.Get("Authorization"))

			if deviceID != "" && deviceToken != "" {
				if deviceAuth == nil {
					writeErrorResponse(w, http.StatusUnauthorized, apierror.CodeUnauthorized)
					return
				}
				machine, err := deviceAuth.AuthenticateDevice(r.Context(), deviceID, deviceToken)
				if err != nil {
					writeErrorResponse(w, http.StatusUnauthorized, apierror.CodeUnauthorized)
					return
				}
				ctx := context.WithValue(r.Context(), deviceActorKey{}, machine.ID)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Tidak ada kredensial perangkat: gunakan sesi pengguna + permission.
			auth.Authenticate(resolver)(
				auth.RequirePermission(policy, permission, next),
			).ServeHTTP(w, r)
		})
	}
}

// BearerToken mengekstrak token dari header Authorization.
func BearerToken(header string) string {
	parts := strings.Fields(header)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1]
	}
	return ""
}

// actorFromContext mengembalikan ID aktor dari konteks: machine ID bila
// diautentikasi sebagai perangkat, atau user ID bila diautentikasi sebagai pengguna.
func actorFromContext(ctx context.Context) (string, bool) {
	if actor, ok := ctx.Value(deviceActorKey{}).(string); ok && actor != "" {
		return actor, true
	}
	principal, ok := auth.PrincipalFromContext(ctx)
	if !ok || principal.UserID == "" {
		return "", false
	}
	return principal.UserID, true
}

func (api attendanceAPI) createEvent(w http.ResponseWriter, r *http.Request) {
	actorID, ok := actorFromContext(r.Context())
	if !ok {
		writeErrorResponse(w, http.StatusUnauthorized, apierror.CodeUnauthorized)
		return
	}

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

	event := models.AttendanceEvent{
		MachineId:      machineID.String(),
		MachinePin:     request.MachinePIN,
		OccurredAt:     request.OccurredAt.UTC(),
		IdempotencyKey: request.IdempotencyKey,
	}

	stored, duplicate, err := api.store.SaveEvent(r.Context(), actorID, event)
	switch {
	case errors.Is(err, attendance.ErrMachineNotFound):
		writeErrorResponse(w, http.StatusNotFound, "machine_not_found")
	case errors.Is(err, attendance.ErrEventConflict):
		writeErrorResponse(w, http.StatusConflict, "idempotency_key_conflict")
	case err != nil:
		writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
	case duplicate:
		writeJSON(w, http.StatusOK, attendanceEventResponse{EventID: stored.ID, Duplicate: true})
	default:
		writeJSON(w, http.StatusCreated, attendanceEventResponse{EventID: stored.ID})
	}
}
