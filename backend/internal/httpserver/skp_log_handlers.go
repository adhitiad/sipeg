package httpserver

import (
	"net/http"
	"strings"
	"time"

	"simpeg/backend/internal/apierror"
	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
	skpdomain "simpeg/backend/internal/skp/domain"
	"simpeg/backend/internal/timeutil"
)

// ═══════════════════════════ DTO Log Aktivitas Harian ═══════════════════════════

// LogAktivitasDTO adalah bentuk publik log aktivitas harian.
type LogAktivitasDTO struct {
	ID          string    `json:"id"`
	UserID      string    `json:"userId"`
	Tanggal     string    `json:"tanggal"`
	Aktivitas   string    `json:"aktivitas"`
	DurasiMenit int       `json:"durasiMenit"`
	Keterangan  *string   `json:"keterangan"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// logAktivitasCreateRequest adalah bentuk permintaan pembuatan log aktivitas.
type logAktivitasCreateRequest struct {
	UserID      string  `json:"userId"`
	Tanggal     string  `json:"tanggal"`
	Aktivitas   string  `json:"aktivitas"`
	DurasiMenit int     `json:"durasiMenit"`
	Keterangan  *string `json:"keterangan"`
}

// logAktivitasPatchRequest adalah bentuk permintaan perubahan log aktivitas.
type logAktivitasPatchRequest struct {
	Tanggal     *string `json:"tanggal"`
	Aktivitas   *string `json:"aktivitas"`
	DurasiMenit *int    `json:"durasiMenit"`
	Keterangan  *string `json:"keterangan"`
}

func logAktivitasDTO(log models.LogAktivitasHarian) LogAktivitasDTO {
	return LogAktivitasDTO{
		ID:          log.ID,
		UserID:      log.UserId,
		Tanggal:     timeutil.FormatDate(log.Tanggal),
		Aktivitas:   log.Aktivitas,
		DurasiMenit: log.DurasiMenit,
		Keterangan:  log.Keterangan,
		CreatedAt:   timeutil.InLocation(log.CreatedAt),
		UpdatedAt:   timeutil.InLocation(log.UpdatedAt),
	}
}

// ═══════════════════════════ handlers Log Aktivitas Harian ═══════════════════════════

func (api directoryAPI) listLogAktivitas(w http.ResponseWriter, r *http.Request) {
	query, err := parseListQuery(r, skpdomain.LogAktivitasSortFields(), "tanggal")
	userID := strings.TrimSpace(r.URL.Query().Get("userId"))
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	options := listOptionsFromQuery(query, listing.Options{UserID: userID})

	page, err := api.skp.ListLogAktivitas(r.Context(), principal, options)
	if err != nil {
		writeAppError(w, err)
		return
	}

	items := make([]LogAktivitasDTO, 0, len(page.Items))
	for _, log := range page.Items {
		items = append(items, logAktivitasDTO(log))
	}
	writeJSON(w, http.StatusOK, PageDTO[LogAktivitasDTO]{
		Items: items, Page: page.Page, PageSize: page.PageSize,
		TotalItems: page.TotalItems, TotalPages: page.TotalPages,
	})
}

func (api directoryAPI) getLogAktivitas(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("logId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	log, err := api.skp.GetLogAktivitas(r.Context(), principal, id)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, logAktivitasDTO(log))
}

func (api directoryAPI) createLogAktivitas(w http.ResponseWriter, r *http.Request) {
	var request logAktivitasCreateRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}

	tanggal, err := time.Parse("2006-01-02", strings.TrimSpace(request.Tanggal))
	if err != nil {
		writeValidationError(w)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	log := models.LogAktivitasHarian{
		UserId:      request.UserID,
		Tanggal:     tanggal,
		Aktivitas:   request.Aktivitas,
		DurasiMenit: request.DurasiMenit,
		Keterangan:  request.Keterangan,
	}
	log, err = api.skp.CreateLogAktivitas(r.Context(), principal, log)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, logAktivitasDTO(log))
}

func (api directoryAPI) updateLogAktivitas(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("logId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}

	var request logAktivitasPatchRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}

	updates := make(map[string]any)
	if request.Tanggal != nil {
		tanggal, err := time.Parse("2006-01-02", strings.TrimSpace(*request.Tanggal))
		if err != nil {
			writeValidationError(w)
			return
		}
		updates["tanggal"] = tanggal
	}
	if request.Aktivitas != nil {
		updates["aktivitas"] = *request.Aktivitas
	}
	if request.DurasiMenit != nil {
		updates["durasiMenit"] = *request.DurasiMenit
	}
	if request.Keterangan != nil {
		updates["keterangan"] = request.Keterangan
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	log, err := api.skp.UpdateLogAktivitas(r.Context(), principal, id, updates)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, logAktivitasDTO(log))
}

func (api directoryAPI) deleteLogAktivitas(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("logId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	if err := api.skp.DeleteLogAktivitas(r.Context(), principal, id); err != nil {
		writeAppError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
