package httpserver

import (
	"net/http"
	"strings"
	"time"

	"simpeg/backend/internal/apierror"
	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
	"simpeg/backend/internal/timeutil"
	tridharmadomain "simpeg/backend/internal/tridharma/domain"
)

// tridharmaItemResponse adalah bentuk publik item tridharma (pendidikan, pengabdian, penunjang).
type tridharmaItemResponse struct {
	ID                string    `json:"id"`
	UserID            string    `json:"userId"`
	PeriodeAkademikID string    `json:"periodeAkademikId"`
	Uraian            string    `json:"uraian"`
	Target            float64   `json:"target"`
	Realisasi         float64   `json:"realisasi"`
	Bobot             float64   `json:"bobot"`
	Status            string    `json:"status"`
	DinilaiOleh       *string   `json:"dinilaiOleh"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

// tridharmaItemCreateRequest adalah bentuk permintaan pembuatan item tridharma.
type tridharmaItemCreateRequest struct {
	UserID            string  `json:"userId"`
	PeriodeAkademikID string  `json:"periodeAkademikId"`
	Uraian            string  `json:"uraian"`
	Target            float64 `json:"target"`
	Bobot             float64 `json:"bobot"`
}

// tridharmaItemPatchRequest adalah bentuk permintaan perubahan item tridharma.
type tridharmaItemPatchRequest struct {
	Uraian      *string  `json:"uraian"`
	Target      *float64 `json:"target"`
	Realisasi   *float64 `json:"realisasi"`
	Bobot       *float64 `json:"bobot"`
	Status      *string  `json:"status"`
	DinilaiOleh *string  `json:"dinilaiOleh"`
}

func (api directoryAPI) listTridharmaPendidikan(w http.ResponseWriter, r *http.Request) {
	query, err := parseListQuery(r, tridharmadomain.TridharmaPendidikanSortFields(), "createdAt")
	userID := strings.TrimSpace(r.URL.Query().Get("userId"))
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if err != nil || textLength(status) > listing.MaxSearchLength {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	options := listOptionsFromQuery(query, listing.Options{UserID: userID, Status: status})

	page, err := api.tridharma.ListTridharmaPendidikan(r.Context(), principal, options)
	if err != nil {
		writeAppError(w, err)
		return
	}

	items := make([]tridharmaItemResponse, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, tridharmaItemResponse{
			ID:                item.ID,
			UserID:            item.UserId,
			PeriodeAkademikID: item.PeriodeAkademikId,
			Uraian:            item.Uraian,
			Target:            item.Target,
			Realisasi:         item.Realisasi,
			Bobot:             item.Bobot,
			Status:            string(item.Status),
			DinilaiOleh:       item.DinilaiOleh,
			CreatedAt:         timeutil.InLocation(item.CreatedAt),
			UpdatedAt:         timeutil.InLocation(item.UpdatedAt),
		})
	}
	writeJSON(w, http.StatusOK, PageDTO[tridharmaItemResponse]{
		Items: items, Page: page.Page, PageSize: page.PageSize,
		TotalItems: page.TotalItems, TotalPages: page.TotalPages,
	})
}

func (api directoryAPI) getTridharmaPendidikan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("pendidikanId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	item, err := api.tridharma.GetTridharmaPendidikan(r.Context(), principal, id)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tridharmaItemResponse{
		ID:                item.ID,
		UserID:            item.UserId,
		PeriodeAkademikID: item.PeriodeAkademikId,
		Uraian:            item.Uraian,
		Target:            item.Target,
		Realisasi:         item.Realisasi,
		Bobot:             item.Bobot,
		Status:            string(item.Status),
		DinilaiOleh:       item.DinilaiOleh,
		CreatedAt:         timeutil.InLocation(item.CreatedAt),
		UpdatedAt:         timeutil.InLocation(item.UpdatedAt),
	})
}

func (api directoryAPI) createTridharmaPendidikan(w http.ResponseWriter, r *http.Request) {
	var request tridharmaItemCreateRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	item := models.TridharmaPendidikan{
		UserId:            request.UserID,
		PeriodeAkademikId: request.PeriodeAkademikID,
		Uraian:            request.Uraian,
		Target:            request.Target,
		Bobot:             request.Bobot,
	}
	item, err := api.tridharma.CreateTridharmaPendidikan(r.Context(), principal, item)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, tridharmaItemResponse{
		ID:                item.ID,
		UserID:            item.UserId,
		PeriodeAkademikID: item.PeriodeAkademikId,
		Uraian:            item.Uraian,
		Target:            item.Target,
		Realisasi:         item.Realisasi,
		Bobot:             item.Bobot,
		Status:            string(item.Status),
		DinilaiOleh:       item.DinilaiOleh,
		CreatedAt:         timeutil.InLocation(item.CreatedAt),
		UpdatedAt:         timeutil.InLocation(item.UpdatedAt),
	})
}

func (api directoryAPI) updateTridharmaPendidikan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("pendidikanId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}

	var request tridharmaItemPatchRequest
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
		updates["status"] = models.StatusTridharmaItem(*request.Status)
	}
	if request.DinilaiOleh != nil {
		updates["dinilaiOleh"] = request.DinilaiOleh
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	item, err := api.tridharma.UpdateTridharmaPendidikan(r.Context(), principal, id, updates)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tridharmaItemResponse{
		ID:                item.ID,
		UserID:            item.UserId,
		PeriodeAkademikID: item.PeriodeAkademikId,
		Uraian:            item.Uraian,
		Target:            item.Target,
		Realisasi:         item.Realisasi,
		Bobot:             item.Bobot,
		Status:            string(item.Status),
		DinilaiOleh:       item.DinilaiOleh,
		CreatedAt:         timeutil.InLocation(item.CreatedAt),
		UpdatedAt:         timeutil.InLocation(item.UpdatedAt),
	})
}

func (api directoryAPI) deleteTridharmaPendidikan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("pendidikanId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	if err := api.tridharma.DeleteTridharmaPendidikan(r.Context(), principal, id); err != nil {
		writeAppError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
