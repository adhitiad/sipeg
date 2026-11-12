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

// tridharmaPenelitianResponse adalah bentuk publik item tridharma penelitian.
type tridharmaPenelitianResponse struct {
	ID                string    `json:"id"`
	UserID            string    `json:"userId"`
	PeriodeAkademikID string    `json:"periodeAkademikId"`
	Judul             string    `json:"judul"`
	JenisPenelitian   string    `json:"jenisPenelitian"`
	Target            float64   `json:"target"`
	Realisasi         float64   `json:"realisasi"`
	Bobot             float64   `json:"bobot"`
	Status            string    `json:"status"`
	DinilaiOleh       *string   `json:"dinilaiOleh"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

// tridharmaPenelitianCreateRequest adalah bentuk permintaan pembuatan item penelitian.
type tridharmaPenelitianCreateRequest struct {
	UserID            string  `json:"userId"`
	PeriodeAkademikID string  `json:"periodeAkademikId"`
	Judul             string  `json:"judul"`
	JenisPenelitian   string  `json:"jenisPenelitian"`
	Target            float64 `json:"target"`
	Bobot             float64 `json:"bobot"`
}

// tridharmaPenelitianPatchRequest adalah bentuk permintaan perubahan item penelitian.
type tridharmaPenelitianPatchRequest struct {
	Judul           *string  `json:"judul"`
	JenisPenelitian *string  `json:"jenisPenelitian"`
	Target          *float64 `json:"target"`
	Realisasi       *float64 `json:"realisasi"`
	Bobot           *float64 `json:"bobot"`
	Status          *string  `json:"status"`
	DinilaiOleh     *string  `json:"dinilaiOleh"`
}

func (api directoryAPI) listTridharmaPenelitian(w http.ResponseWriter, r *http.Request) {
	query, err := parseListQuery(r, tridharmadomain.TridharmaPenelitianSortFields(), "createdAt")
	userID := strings.TrimSpace(r.URL.Query().Get("userId"))
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if err != nil || textLength(status) > listing.MaxSearchLength {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	options := listOptionsFromQuery(query, listing.Options{UserID: userID, Status: status})

	page, err := api.tridharma.ListTridharmaPenelitian(r.Context(), principal, options)
	if err != nil {
		writeAppError(w, err)
		return
	}

	items := make([]tridharmaPenelitianResponse, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, tridharmaPenelitianResponse{
			ID:                item.ID,
			UserID:            item.UserId,
			PeriodeAkademikID: item.PeriodeAkademikId,
			Judul:             item.Judul,
			JenisPenelitian:   item.JenisPenelitian,
			Target:            item.Target,
			Realisasi:         item.Realisasi,
			Bobot:             item.Bobot,
			Status:            string(item.Status),
			DinilaiOleh:       item.DinilaiOleh,
			CreatedAt:         timeutil.InLocation(item.CreatedAt),
			UpdatedAt:         timeutil.InLocation(item.UpdatedAt),
		})
	}
	writeJSON(w, http.StatusOK, PageDTO[tridharmaPenelitianResponse]{
		Items: items, Page: page.Page, PageSize: page.PageSize,
		TotalItems: page.TotalItems, TotalPages: page.TotalPages,
	})
}

func (api directoryAPI) getTridharmaPenelitian(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("penelitianId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	item, err := api.tridharma.GetTridharmaPenelitian(r.Context(), principal, id)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tridharmaPenelitianResponse{
		ID:                item.ID,
		UserID:            item.UserId,
		PeriodeAkademikID: item.PeriodeAkademikId,
		Judul:             item.Judul,
		JenisPenelitian:   item.JenisPenelitian,
		Target:            item.Target,
		Realisasi:         item.Realisasi,
		Bobot:             item.Bobot,
		Status:            string(item.Status),
		DinilaiOleh:       item.DinilaiOleh,
		CreatedAt:         timeutil.InLocation(item.CreatedAt),
		UpdatedAt:         timeutil.InLocation(item.UpdatedAt),
	})
}

func (api directoryAPI) createTridharmaPenelitian(w http.ResponseWriter, r *http.Request) {
	var request tridharmaPenelitianCreateRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	item := models.TridharmaPenelitian{
		UserId:            request.UserID,
		PeriodeAkademikId: request.PeriodeAkademikID,
		Judul:             request.Judul,
		JenisPenelitian:   request.JenisPenelitian,
		Target:            request.Target,
		Bobot:             request.Bobot,
	}
	item, err := api.tridharma.CreateTridharmaPenelitian(r.Context(), principal, item)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, tridharmaPenelitianResponse{
		ID:                item.ID,
		UserID:            item.UserId,
		PeriodeAkademikID: item.PeriodeAkademikId,
		Judul:             item.Judul,
		JenisPenelitian:   item.JenisPenelitian,
		Target:            item.Target,
		Realisasi:         item.Realisasi,
		Bobot:             item.Bobot,
		Status:            string(item.Status),
		DinilaiOleh:       item.DinilaiOleh,
		CreatedAt:         timeutil.InLocation(item.CreatedAt),
		UpdatedAt:         timeutil.InLocation(item.UpdatedAt),
	})
}

func (api directoryAPI) updateTridharmaPenelitian(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("penelitianId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}

	var request tridharmaPenelitianPatchRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}

	updates := make(map[string]any)
	if request.Judul != nil {
		updates["judul"] = *request.Judul
	}
	if request.JenisPenelitian != nil {
		updates["jenisPenelitian"] = *request.JenisPenelitian
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
	item, err := api.tridharma.UpdateTridharmaPenelitian(r.Context(), principal, id, updates)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tridharmaPenelitianResponse{
		ID:                item.ID,
		UserID:            item.UserId,
		PeriodeAkademikID: item.PeriodeAkademikId,
		Judul:             item.Judul,
		JenisPenelitian:   item.JenisPenelitian,
		Target:            item.Target,
		Realisasi:         item.Realisasi,
		Bobot:             item.Bobot,
		Status:            string(item.Status),
		DinilaiOleh:       item.DinilaiOleh,
		CreatedAt:         timeutil.InLocation(item.CreatedAt),
		UpdatedAt:         timeutil.InLocation(item.UpdatedAt),
	})
}

func (api directoryAPI) deleteTridharmaPenelitian(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("penelitianId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	if err := api.tridharma.DeleteTridharmaPenelitian(r.Context(), principal, id); err != nil {
		writeAppError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
