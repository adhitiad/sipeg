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

// ═══════════════════════════ DTO Penilaian Kinerja Akhir ═══════════════════════════

// PenilaianDTO adalah bentuk publik penilaian kinerja akhir.
type PenilaianDTO struct {
	ID               string    `json:"id"`
	UserID           string    `json:"userId"`
	SKPPeriodeID     string    `json:"skpPeriodeId"`
	NilaiAkhir       float64   `json:"nilaiAkhir"`
	Predikat         string    `json:"predikat"`
	Catatan          *string   `json:"catatan"`
	DinilaiOleh      *string   `json:"dinilaiOleh"`
	TanggalPenilaian string    `json:"tanggalPenilaian"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

// penilaianCreateRequest adalah bentuk permintaan pembuatan penilaian kinerja akhir.
type penilaianCreateRequest struct {
	UserID           string  `json:"userId"`
	SKPPeriodeID     string  `json:"skpPeriodeId"`
	NilaiAkhir       float64 `json:"nilaiAkhir"`
	Predikat         string  `json:"predikat"`
	Catatan          *string `json:"catatan"`
	TanggalPenilaian string  `json:"tanggalPenilaian"`
}

// penilaianPatchRequest adalah bentuk permintaan perubahan penilaian kinerja akhir.
type penilaianPatchRequest struct {
	NilaiAkhir       *float64 `json:"nilaiAkhir"`
	Predikat         *string  `json:"predikat"`
	Catatan          *string  `json:"catatan"`
	TanggalPenilaian *string  `json:"tanggalPenilaian"`
}

func penilaianDTO(penilaian models.PenilaianKinerjaAkhir) PenilaianDTO {
	return PenilaianDTO{
		ID:               penilaian.ID,
		UserID:           penilaian.UserId,
		SKPPeriodeID:     penilaian.SKPPeriodeId,
		NilaiAkhir:       penilaian.NilaiAkhir,
		Predikat:         penilaian.Predikat,
		Catatan:          penilaian.Catatan,
		DinilaiOleh:      penilaian.DinilaiOleh,
		TanggalPenilaian: timeutil.FormatDate(penilaian.TanggalPenilaian),
		CreatedAt:        timeutil.InLocation(penilaian.CreatedAt),
		UpdatedAt:        timeutil.InLocation(penilaian.UpdatedAt),
	}
}

// ═══════════════════════════ handlers Penilaian Kinerja Akhir ═══════════════════════════

func (api directoryAPI) listPenilaian(w http.ResponseWriter, r *http.Request) {
	query, err := parseListQuery(r, skpdomain.PenilaianSortFields(), "tanggalPenilaian")
	userID := strings.TrimSpace(r.URL.Query().Get("userId"))
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	options := listOptionsFromQuery(query, listing.Options{UserID: userID})

	page, err := api.skp.ListPenilaian(r.Context(), principal, options)
	if err != nil {
		writeAppError(w, err)
		return
	}

	items := make([]PenilaianDTO, 0, len(page.Items))
	for _, penilaian := range page.Items {
		items = append(items, penilaianDTO(penilaian))
	}
	writeJSON(w, http.StatusOK, PageDTO[PenilaianDTO]{
		Items: items, Page: page.Page, PageSize: page.PageSize,
		TotalItems: page.TotalItems, TotalPages: page.TotalPages,
	})
}

func (api directoryAPI) getPenilaian(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("penilaianId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	penilaian, err := api.skp.GetPenilaian(r.Context(), principal, id)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, penilaianDTO(penilaian))
}

func (api directoryAPI) createPenilaian(w http.ResponseWriter, r *http.Request) {
	var request penilaianCreateRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}

	tanggalPenilaian, err := time.Parse("2006-01-02", strings.TrimSpace(request.TanggalPenilaian))
	if err != nil {
		writeValidationError(w)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	penilaian := models.PenilaianKinerjaAkhir{
		UserId:           request.UserID,
		SKPPeriodeId:     request.SKPPeriodeID,
		NilaiAkhir:       request.NilaiAkhir,
		Predikat:         request.Predikat,
		Catatan:          request.Catatan,
		TanggalPenilaian: tanggalPenilaian,
	}
	created, err := api.skp.CreatePenilaian(r.Context(), principal, penilaian)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, penilaianDTO(created))
}

func (api directoryAPI) updatePenilaian(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("penilaianId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}

	var request penilaianPatchRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}

	updates := make(map[string]any)
	if request.NilaiAkhir != nil {
		updates["nilaiAkhir"] = *request.NilaiAkhir
	}
	if request.Predikat != nil {
		updates["predikat"] = *request.Predikat
	}
	if request.Catatan != nil {
		updates["catatan"] = request.Catatan
	}
	if request.TanggalPenilaian != nil {
		tanggal, err := time.Parse("2006-01-02", strings.TrimSpace(*request.TanggalPenilaian))
		if err != nil {
			writeValidationError(w)
			return
		}
		updates["tanggalPenilaian"] = tanggal
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	penilaian, err := api.skp.UpdatePenilaian(r.Context(), principal, id, updates)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, penilaianDTO(penilaian))
}

func (api directoryAPI) deletePenilaian(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("penilaianId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	if err := api.skp.DeletePenilaian(r.Context(), principal, id); err != nil {
		writeAppError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
