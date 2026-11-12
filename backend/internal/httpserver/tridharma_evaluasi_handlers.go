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

// evaluasiBKDResponse adalah bentuk publik evaluasi BKD.
type evaluasiBKDResponse struct {
	ID                string    `json:"id"`
	UserID            string    `json:"userId"`
	PeriodeAkademikID string    `json:"periodeAkademikId"`
	Nilai             float64   `json:"nilai"`
	Predikat          string    `json:"predikat"`
	Catatan           *string   `json:"catatan"`
	DinilaiOleh       *string   `json:"dinilaiOleh"`
	TanggalEvaluasi   string    `json:"tanggalEvaluasi"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

func toEvaluasiBKDResponse(evaluasi models.EvaluasiBKD) evaluasiBKDResponse {
	return evaluasiBKDResponse{
		ID:                evaluasi.ID,
		UserID:            evaluasi.UserId,
		PeriodeAkademikID: evaluasi.PeriodeAkademikId,
		Nilai:             evaluasi.Nilai,
		Predikat:          evaluasi.Predikat,
		Catatan:           evaluasi.Catatan,
		DinilaiOleh:       evaluasi.DinilaiOleh,
		TanggalEvaluasi:   timeutil.FormatDate(evaluasi.TanggalEvaluasi),
		CreatedAt:         timeutil.InLocation(evaluasi.CreatedAt),
		UpdatedAt:         timeutil.InLocation(evaluasi.UpdatedAt),
	}
}

// evaluasiBKDCreateRequest adalah bentuk permintaan pembuatan evaluasi BKD.
type evaluasiBKDCreateRequest struct {
	UserID            string  `json:"userId"`
	PeriodeAkademikID string  `json:"periodeAkademikId"`
	Nilai             float64 `json:"nilai"`
	Predikat          string  `json:"predikat"`
	Catatan           *string `json:"catatan"`
	DinilaiOleh       *string `json:"dinilaiOleh"`
	TanggalEvaluasi   string  `json:"tanggalEvaluasi"`
}

// evaluasiBKDPatchRequest adalah bentuk permintaan perubahan evaluasi BKD.
type evaluasiBKDPatchRequest struct {
	Nilai           *float64 `json:"nilai"`
	Predikat        *string  `json:"predikat"`
	Catatan         *string  `json:"catatan"`
	DinilaiOleh     *string  `json:"dinilaiOleh"`
	TanggalEvaluasi *string  `json:"tanggalEvaluasi"`
}

func (api directoryAPI) listEvaluasiBKD(w http.ResponseWriter, r *http.Request) {
	query, err := parseListQuery(r, tridharmadomain.EvaluasiBKDSortFields(), "tanggalEvaluasi")
	userID := strings.TrimSpace(r.URL.Query().Get("userId"))
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if err != nil || textLength(status) > listing.MaxSearchLength {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	options := listOptionsFromQuery(query, listing.Options{UserID: userID, Status: status})

	page, err := api.tridharma.ListEvaluasiBKD(r.Context(), principal, options)
	if err != nil {
		writeAppError(w, err)
		return
	}

	items := make([]evaluasiBKDResponse, 0, len(page.Items))
	for _, evaluasi := range page.Items {
		items = append(items, toEvaluasiBKDResponse(evaluasi))
	}
	writeJSON(w, http.StatusOK, PageDTO[evaluasiBKDResponse]{
		Items: items, Page: page.Page, PageSize: page.PageSize,
		TotalItems: page.TotalItems, TotalPages: page.TotalPages,
	})
}

func (api directoryAPI) getEvaluasiBKD(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("evaluasiId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	evaluasi, err := api.tridharma.GetEvaluasiBKD(r.Context(), principal, id)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toEvaluasiBKDResponse(evaluasi))
}

func (api directoryAPI) createEvaluasiBKD(w http.ResponseWriter, r *http.Request) {
	var request evaluasiBKDCreateRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}

	tanggalEvaluasi, err := time.Parse("2006-01-02", strings.TrimSpace(request.TanggalEvaluasi))
	if err != nil {
		writeValidationError(w)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	evaluasi := models.EvaluasiBKD{
		UserId:            request.UserID,
		PeriodeAkademikId: request.PeriodeAkademikID,
		Nilai:             request.Nilai,
		Predikat:          request.Predikat,
		Catatan:           request.Catatan,
		DinilaiOleh:       request.DinilaiOleh,
		TanggalEvaluasi:   tanggalEvaluasi,
	}
	evaluasi, err = api.tridharma.CreateEvaluasiBKD(r.Context(), principal, evaluasi)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toEvaluasiBKDResponse(evaluasi))
}

func (api directoryAPI) updateEvaluasiBKD(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("evaluasiId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}

	var request evaluasiBKDPatchRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}

	updates := make(map[string]any)
	if request.Nilai != nil {
		updates["nilai"] = *request.Nilai
	}
	if request.Predikat != nil {
		updates["predikat"] = *request.Predikat
	}
	if request.Catatan != nil {
		updates["catatan"] = request.Catatan
	}
	if request.DinilaiOleh != nil {
		updates["dinilaiOleh"] = request.DinilaiOleh
	}
	if request.TanggalEvaluasi != nil {
		tanggal, err := time.Parse("2006-01-02", strings.TrimSpace(*request.TanggalEvaluasi))
		if err != nil {
			writeValidationError(w)
			return
		}
		updates["tanggalEvaluasi"] = tanggal
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	evaluasi, err := api.tridharma.UpdateEvaluasiBKD(r.Context(), principal, id, updates)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toEvaluasiBKDResponse(evaluasi))
}

func (api directoryAPI) deleteEvaluasiBKD(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("evaluasiId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	if err := api.tridharma.DeleteEvaluasiBKD(r.Context(), principal, id); err != nil {
		writeAppError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
