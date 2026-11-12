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

// ═══════════════════════════ DTO Periode Akademik ═══════════════════════════

// PeriodeAkademikDTO adalah bentuk publik periode akademik.
type PeriodeAkademikDTO struct {
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

// periodeAkademikCreateRequest adalah bentuk permintaan pembuatan periode akademik.
type periodeAkademikCreateRequest struct {
	Tahun          int    `json:"tahun"`
	Semester       int    `json:"semester"`
	TanggalMulai   string `json:"tanggalMulai"`
	TanggalSelesai string `json:"tanggalSelesai"`
	Status         string `json:"status"`
	Aktif          bool   `json:"aktif"`
}

// periodeAkademikPatchRequest adalah bentuk permintaan perubahan periode akademik.
type periodeAkademikPatchRequest struct {
	TanggalMulai   *string `json:"tanggalMulai"`
	TanggalSelesai *string `json:"tanggalSelesai"`
	Status         *string `json:"status"`
	Aktif          *bool   `json:"aktif"`
}

func periodeAkademikDTO(periode models.PeriodeAkademik) PeriodeAkademikDTO {
	return PeriodeAkademikDTO{
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

// ═══════════════════════════ handlers Periode Akademik ══════════════════════════

func (api directoryAPI) listPeriodeAkademik(w http.ResponseWriter, r *http.Request) {
	query, err := parseListQuery(r, tridharmadomain.PeriodeAkademikSortFields(), "tahun")
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if err != nil || textLength(status) > listing.MaxSearchLength {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	options := listOptionsFromQuery(query, listing.Options{Status: status})

	page, err := api.tridharma.ListPeriodeAkademik(r.Context(), principal, options)
	if err != nil {
		writeAppError(w, err)
		return
	}

	items := make([]PeriodeAkademikDTO, 0, len(page.Items))
	for _, periode := range page.Items {
		items = append(items, periodeAkademikDTO(periode))
	}
	writeJSON(w, http.StatusOK, PageDTO[PeriodeAkademikDTO]{
		Items: items, Page: page.Page, PageSize: page.PageSize,
		TotalItems: page.TotalItems, TotalPages: page.TotalPages,
	})
}

func (api directoryAPI) getPeriodeAkademik(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("periodeId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	periode, err := api.tridharma.GetPeriodeAkademik(r.Context(), principal, id)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, periodeAkademikDTO(periode))
}

func (api directoryAPI) createPeriodeAkademik(w http.ResponseWriter, r *http.Request) {
	var request periodeAkademikCreateRequest
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
	status := models.StatusPeriodeAkademikDraft
	if request.Status != "" {
		status = models.StatusPeriodeAkademik(request.Status)
	}
	periode := models.PeriodeAkademik{
		Tahun:          request.Tahun,
		Semester:       request.Semester,
		TanggalMulai:   tanggalMulai,
		TanggalSelesai: tanggalSelesai,
		Status:         status,
		Aktif:          request.Aktif,
	}
	periode, err = api.tridharma.CreatePeriodeAkademik(r.Context(), principal, periode)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, periodeAkademikDTO(periode))
}

func (api directoryAPI) updatePeriodeAkademik(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("periodeId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}

	var request periodeAkademikPatchRequest
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
		updates["status"] = models.StatusPeriodeAkademik(*request.Status)
	}
	if request.Aktif != nil {
		updates["aktif"] = *request.Aktif
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	periode, err := api.tridharma.UpdatePeriodeAkademik(r.Context(), principal, id, updates)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, periodeAkademikDTO(periode))
}

func (api directoryAPI) deletePeriodeAkademik(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("periodeId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	if err := api.tridharma.DeletePeriodeAkademik(r.Context(), principal, id); err != nil {
		writeAppError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
