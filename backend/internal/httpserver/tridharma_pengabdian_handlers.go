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

// tridharmaPengabdianResponse adalah bentuk publik item tridharma pengabdian.
type tridharmaPengabdianResponse struct {
	ID                string    `json:"id"`
	UserID            string    `json:"userId"`
	PeriodeAkademikID string    `json:"periodeAkademikId"`
	Judul             string    `json:"judul"`
	JenisPengabdian   string    `json:"jenisPengabdian"`
	Target            float64   `json:"target"`
	Realisasi         float64   `json:"realisasi"`
	Bobot             float64   `json:"bobot"`
	Status            string    `json:"status"`
	DinilaiOleh       *string   `json:"dinilaiOleh"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

// tridharmaPengabdianCreateRequest adalah bentuk permintaan pembuatan item pengabdian.
type tridharmaPengabdianCreateRequest struct {
	UserID            string  `json:"userId"`
	PeriodeAkademikID string  `json:"periodeAkademikId"`
	Judul             string  `json:"judul"`
	JenisPengabdian   string  `json:"jenisPengabdian"`
	Target            float64 `json:"target"`
	Bobot             float64 `json:"bobot"`
}

// tridharmaPengabdianPatchRequest adalah bentuk permintaan perubahan item pengabdian.
type tridharmaPengabdianPatchRequest struct {
	Judul           *string  `json:"judul"`
	JenisPengabdian *string  `json:"jenisPengabdian"`
	Target          *float64 `json:"target"`
	Realisasi       *float64 `json:"realisasi"`
	Bobot           *float64 `json:"bobot"`
	Status          *string  `json:"status"`
	DinilaiOleh     *string  `json:"dinilaiOleh"`
}

func (api directoryAPI) listTridharmaPengabdian(w http.ResponseWriter, r *http.Request) {
	query, err := parseListQuery(r, tridharmadomain.TridharmaPengabdianSortFields(), "createdAt")
	userID := strings.TrimSpace(r.URL.Query().Get("userId"))
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if err != nil || textLength(status) > listing.MaxSearchLength {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	options := listOptionsFromQuery(query, listing.Options{UserID: userID, Status: status})

	page, err := api.tridharma.ListTridharmaPengabdian(r.Context(), principal, options)
	if err != nil {
		writeAppError(w, err)
		return
	}

	items := make([]tridharmaPengabdianResponse, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, tridharmaPengabdianResponse{
			ID:                item.ID,
			UserID:            item.UserId,
			PeriodeAkademikID: item.PeriodeAkademikId,
			Judul:             item.Judul,
			JenisPengabdian:   item.JenisPengabdian,
			Target:            item.Target,
			Realisasi:         item.Realisasi,
			Bobot:             item.Bobot,
			Status:            string(item.Status),
			DinilaiOleh:       item.DinilaiOleh,
			CreatedAt:         timeutil.InLocation(item.CreatedAt),
			UpdatedAt:         timeutil.InLocation(item.UpdatedAt),
		})
	}
	writeJSON(w, http.StatusOK, PageDTO[tridharmaPengabdianResponse]{
		Items: items, Page: page.Page, PageSize: page.PageSize,
		TotalItems: page.TotalItems, TotalPages: page.TotalPages,
	})
}

func (api directoryAPI) getTridharmaPengabdian(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("pengabdianId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	item, err := api.tridharma.GetTridharmaPengabdian(r.Context(), principal, id)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tridharmaPengabdianResponse{
		ID:                item.ID,
		UserID:            item.UserId,
		PeriodeAkademikID: item.PeriodeAkademikId,
		Judul:             item.Judul,
		JenisPengabdian:   item.JenisPengabdian,
		Target:            item.Target,
		Realisasi:         item.Realisasi,
		Bobot:             item.Bobot,
		Status:            string(item.Status),
		DinilaiOleh:       item.DinilaiOleh,
		CreatedAt:         timeutil.InLocation(item.CreatedAt),
		UpdatedAt:         timeutil.InLocation(item.UpdatedAt),
	})
}

func (api directoryAPI) createTridharmaPengabdian(w http.ResponseWriter, r *http.Request) {
	var request tridharmaPengabdianCreateRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	item := models.TridharmaPengabdian{
		UserId:            request.UserID,
		PeriodeAkademikId: request.PeriodeAkademikID,
		Judul:             request.Judul,
		JenisPengabdian:   request.JenisPengabdian,
		Target:            request.Target,
		Bobot:             request.Bobot,
	}
	item, err := api.tridharma.CreateTridharmaPengabdian(r.Context(), principal, item)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, tridharmaPengabdianResponse{
		ID:                item.ID,
		UserID:            item.UserId,
		PeriodeAkademikID: item.PeriodeAkademikId,
		Judul:             item.Judul,
		JenisPengabdian:   item.JenisPengabdian,
		Target:            item.Target,
		Realisasi:         item.Realisasi,
		Bobot:             item.Bobot,
		Status:            string(item.Status),
		DinilaiOleh:       item.DinilaiOleh,
		CreatedAt:         timeutil.InLocation(item.CreatedAt),
		UpdatedAt:         timeutil.InLocation(item.UpdatedAt),
	})
}

func (api directoryAPI) updateTridharmaPengabdian(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("pengabdianId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}

	var request tridharmaPengabdianPatchRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}

	updates := make(map[string]any)
	if request.Judul != nil {
		updates["judul"] = *request.Judul
	}
	if request.JenisPengabdian != nil {
		updates["jenisPengabdian"] = *request.JenisPengabdian
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
	item, err := api.tridharma.UpdateTridharmaPengabdian(r.Context(), principal, id, updates)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tridharmaPengabdianResponse{
		ID:                item.ID,
		UserID:            item.UserId,
		PeriodeAkademikID: item.PeriodeAkademikId,
		Judul:             item.Judul,
		JenisPengabdian:   item.JenisPengabdian,
		Target:            item.Target,
		Realisasi:         item.Realisasi,
		Bobot:             item.Bobot,
		Status:            string(item.Status),
		DinilaiOleh:       item.DinilaiOleh,
		CreatedAt:         timeutil.InLocation(item.CreatedAt),
		UpdatedAt:         timeutil.InLocation(item.UpdatedAt),
	})
}

func (api directoryAPI) deleteTridharmaPengabdian(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("pengabdianId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	if err := api.tridharma.DeleteTridharmaPengabdian(r.Context(), principal, id); err != nil {
		writeAppError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
