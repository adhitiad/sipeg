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

// ═══════════════════════════ DTO SKP Target Kinerja ═══════════════════════════

// SKPTargetDTO adalah bentuk publik target kinerja SKP.
type SKPTargetDTO struct {
	ID           string    `json:"id"`
	UserID       string    `json:"userId"`
	SKPPeriodeID string    `json:"skpPeriodeId"`
	Uraian       string    `json:"uraian"`
	Target       float64   `json:"target"`
	Realisasi    float64   `json:"realisasi"`
	Bobot        float64   `json:"bobot"`
	Status       string    `json:"status"`
	DinilaiOleh  *string   `json:"dinilaiOleh"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// skpTargetCreateRequest adalah bentuk permintaan pembuatan target kinerja SKP.
type skpTargetCreateRequest struct {
	UserID       string  `json:"userId"`
	SKPPeriodeID string  `json:"skpPeriodeId"`
	Uraian       string  `json:"uraian"`
	Target       float64 `json:"target"`
	Bobot        float64 `json:"bobot"`
}

// skpTargetPatchRequest adalah bentuk permintaan perubahan target kinerja SKP.
type skpTargetPatchRequest struct {
	Uraian      *string  `json:"uraian"`
	Target      *float64 `json:"target"`
	Realisasi   *float64 `json:"realisasi"`
	Bobot       *float64 `json:"bobot"`
	Status      *string  `json:"status"`
	DinilaiOleh *string  `json:"dinilaiOleh"`
}

func skpTargetDTO(target models.SKPTargetKinerja) SKPTargetDTO {
	return SKPTargetDTO{
		ID:           target.ID,
		UserID:       target.UserId,
		SKPPeriodeID: target.SKPPeriodeId,
		Uraian:       target.Uraian,
		Target:       target.Target,
		Realisasi:    target.Realisasi,
		Bobot:        target.Bobot,
		Status:       string(target.Status),
		DinilaiOleh:  target.DinilaiOleh,
		CreatedAt:    timeutil.InLocation(target.CreatedAt),
		UpdatedAt:    timeutil.InLocation(target.UpdatedAt),
	}
}

// ═══════════════════════════ handlers SKP Target Kinerja ═══════════════════════════

func (api directoryAPI) listSKPTarget(w http.ResponseWriter, r *http.Request) {
	query, err := parseListQuery(r, skpdomain.SKPTargetSortFields(), "createdAt")
	userID := strings.TrimSpace(r.URL.Query().Get("userId"))
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if err != nil || textLength(status) > listing.MaxSearchLength {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	options := listOptionsFromQuery(query, listing.Options{UserID: userID, Status: status})

	page, err := api.skp.ListSKPTarget(r.Context(), principal, options)
	if err != nil {
		writeAppError(w, err)
		return
	}

	items := make([]SKPTargetDTO, 0, len(page.Items))
	for _, target := range page.Items {
		items = append(items, skpTargetDTO(target))
	}
	writeJSON(w, http.StatusOK, PageDTO[SKPTargetDTO]{
		Items: items, Page: page.Page, PageSize: page.PageSize,
		TotalItems: page.TotalItems, TotalPages: page.TotalPages,
	})
}

func (api directoryAPI) getSKPTarget(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("targetId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	target, err := api.skp.GetSKPTarget(r.Context(), principal, id)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, skpTargetDTO(target))
}

func (api directoryAPI) createSKPTarget(w http.ResponseWriter, r *http.Request) {
	var request skpTargetCreateRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	target := models.SKPTargetKinerja{
		UserId:       request.UserID,
		SKPPeriodeId: request.SKPPeriodeID,
		Uraian:       request.Uraian,
		Target:       request.Target,
		Bobot:        request.Bobot,
	}
	target, err := api.skp.CreateSKPTarget(r.Context(), principal, target)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, skpTargetDTO(target))
}

func (api directoryAPI) updateSKPTarget(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("targetId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}

	var request skpTargetPatchRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}

	updates := make(map[string]any)
	if request.Uraian != nil {
		updates["uraian"] = *request.Uraian
	}
	if request.Target != nil {
		updates["target"] = *request.Target
	}
	if request.Realisasi != nil {
		updates["realisasi"] = *request.Realisasi
	}
	if request.Bobot != nil {
		updates["bobot"] = *request.Bobot
	}
	if request.Status != nil {
		updates["status"] = models.StatusSKPItem(*request.Status)
	}
	if request.DinilaiOleh != nil {
		updates["dinilaiOleh"] = request.DinilaiOleh
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	target, err := api.skp.UpdateSKPTarget(r.Context(), principal, id, updates)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, skpTargetDTO(target))
}

func (api directoryAPI) deleteSKPTarget(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("targetId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	if err := api.skp.DeleteSKPTarget(r.Context(), principal, id); err != nil {
		writeAppError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
