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

// ═══════════════════════════ DTO SKP Periode ═══════════════════════════

// SKPPeriodeDTO adalah bentuk publik periode SKP.
type SKPPeriodeDTO struct {
	ID             string    `json:"id"`
	Tahun          int       `json:"tahun"`
	Semester       int       `json:"semester"`
	TanggalMulai   string    `json:"tanggalMulai"`
	TanggalSelesai string    `json:"tanggalSelesai"`
	Status         string    `json:"status"`
	Aktif          bool      `json:"aktif"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// skpPeriodeCreateRequest adalah bentuk permintaan pembuatan periode SKP.
type skpPeriodeCreateRequest struct {
	Tahun          int    `json:"tahun"`
	Semester       int    `json:"semester"`
	TanggalMulai   string `json:"tanggalMulai"`
	TanggalSelesai string `json:"tanggalSelesai"`
	Status         string `json:"status"`
	Aktif          bool   `json:"aktif"`
}

// skpPeriodePatchRequest adalah bentuk permintaan perubahan periode SKP.
type skpPeriodePatchRequest struct {
	TanggalMulai   *string `json:"tanggalMulai"`
	TanggalSelesai *string `json:"tanggalSelesai"`
	Status         *string `json:"status"`
	Aktif          *bool   `json:"aktif"`
}

func skpPeriodeDTO(periode models.SKPPeriode) SKPPeriodeDTO {
	return SKPPeriodeDTO{
		ID:             periode.ID,
		Tahun:          periode.Tahun,
		Semester:       periode.Semester,
		TanggalMulai:   timeutil.FormatDate(periode.TanggalMulai),
		TanggalSelesai: timeutil.FormatDate(periode.TanggalSelesai),
		Status:         string(periode.Status),
		Aktif:          periode.Aktif,
		CreatedAt:      timeutil.InLocation(periode.CreatedAt),
		UpdatedAt:      timeutil.InLocation(periode.UpdatedAt),
	}
}

// ═══════════════════════════ handlers SKP Periode ═══════════════════════════

func (api directoryAPI) listSKPPeriode(w http.ResponseWriter, r *http.Request) {
	query, err := parseListQuery(r, skpdomain.SKPPeriodeSortFields(), "tahun")
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if err != nil || textLength(status) > listing.MaxSearchLength {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	options := listOptionsFromQuery(query, listing.Options{Status: status})

	page, err := api.skp.ListSKPPeriode(r.Context(), principal, options)
	if err != nil {
		writeAppError(w, err)
		return
	}

	items := make([]SKPPeriodeDTO, 0, len(page.Items))
	for _, periode := range page.Items {
		items = append(items, skpPeriodeDTO(periode))
	}
	writeJSON(w, http.StatusOK, PageDTO[SKPPeriodeDTO]{
		Items: items, Page: page.Page, PageSize: page.PageSize,
		TotalItems: page.TotalItems, TotalPages: page.TotalPages,
	})
}

func (api directoryAPI) getSKPPeriode(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("periodeId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	periode, err := api.skp.GetSKPPeriode(r.Context(), principal, id)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, skpPeriodeDTO(periode))
}

func (api directoryAPI) createSKPPeriode(w http.ResponseWriter, r *http.Request) {
	var request skpPeriodeCreateRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}

	tanggalMulai, err := time.Parse("2006-01-02", strings.TrimSpace(request.TanggalMulai))
	if err != nil {
		writeValidationError(w)
		return
	}
	tanggalSelesai, err := time.Parse("2006-01-02", strings.TrimSpace(request.TanggalSelesai))
	if err != nil {
		writeValidationError(w)
		return
	}
	if !tanggalSelesai.After(tanggalMulai) {
		writeValidationError(w)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	status := models.StatusSKPDraft
	if request.Status != "" {
		status = models.StatusSKP(request.Status)
	}
	periode := models.SKPPeriode{
		Tahun:          request.Tahun,
		Semester:       request.Semester,
		TanggalMulai:   tanggalMulai,
		TanggalSelesai: tanggalSelesai,
		Status:         status,
		Aktif:          request.Aktif,
	}
	periode, err = api.skp.CreateSKPPeriode(r.Context(), principal, periode)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, skpPeriodeDTO(periode))
}

func (api directoryAPI) updateSKPPeriode(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("periodeId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}

	var request skpPeriodePatchRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}

	updates := make(map[string]any)
	if request.TanggalMulai != nil {
		tanggal, err := time.Parse("2006-01-02", strings.TrimSpace(*request.TanggalMulai))
		if err != nil {
			writeValidationError(w)
			return
		}
		updates["tanggalMulai"] = tanggal
	}
	if request.TanggalSelesai != nil {
		tanggal, err := time.Parse("2006-01-02", strings.TrimSpace(*request.TanggalSelesai))
		if err != nil {
			writeValidationError(w)
			return
		}
		updates["tanggalSelesai"] = tanggal
	}
	if request.Status != nil {
		updates["status"] = models.StatusSKP(*request.Status)
	}
	if request.Aktif != nil {
		updates["aktif"] = *request.Aktif
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	periode, err := api.skp.UpdateSKPPeriode(r.Context(), principal, id, updates)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, skpPeriodeDTO(periode))
}

func (api directoryAPI) deleteSKPPeriode(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("periodeId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	if err := api.skp.DeleteSKPPeriode(r.Context(), principal, id); err != nil {
		writeAppError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
