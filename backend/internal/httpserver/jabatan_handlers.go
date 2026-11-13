package httpserver

import (
	"net/http"
	"strings"

	"simpeg/backend/internal/apierror"
	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/directory/domain"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
	"simpeg/backend/internal/timeutil"
)

func jabatanDTO(jabatan models.Jabatan) JabatanDTO {
	return JabatanDTO{
		ID:        jabatan.ID,
		Nama:      jabatan.Nama,
		Jenis:     string(jabatan.Jenis),
		Aktif:     jabatan.Aktif,
		CreatedAt: timeutil.InLocation(jabatan.CreatedAt),
		UpdatedAt: timeutil.InLocation(jabatan.UpdatedAt),
	}
}

func (api directoryAPI) listJabatans(w http.ResponseWriter, r *http.Request) {
	query, err := parseListQuery(r, domain.JabatanSortFields(), "nama")
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	jenis := domain.NormalizeJabatanType(r.URL.Query().Get("jenis"))
	if err != nil || textLength(search) > listing.MaxSearchLength || (jenis != "" && !domain.ValidJabatanType(jenis)) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	options := listOptionsFromQuery(query, listing.Options{Search: search, Jenis: jenis})

	page, err := api.jabatan.List(r.Context(), principal, options)
	if err != nil {
		writeAppError(w, err)
		return
	}

	items := make([]JabatanDTO, 0, len(page.Items))
	for _, jabatan := range page.Items {
		items = append(items, jabatanDTO(jabatan))
	}
	writeJSON(w, http.StatusOK, PageDTO[JabatanDTO]{
		Items: items, Page: page.Page, PageSize: page.PageSize,
		TotalItems: page.TotalItems, TotalPages: page.TotalPages,
	})
}

func (api directoryAPI) getJabatan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("jabatanId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	jabatan, err := api.jabatan.Get(r.Context(), principal, id)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, jabatanDTO(jabatan))
}

type jabatanCreateRequest struct {
	Nama  string `json:"nama"`
	Jenis string `json:"jenis"`
}

func (api directoryAPI) createJabatan(w http.ResponseWriter, r *http.Request) {
	var request jabatanCreateRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	jabatan, err := api.jabatan.Create(r.Context(), principal, domain.CreateJabatan{
		Nama:  request.Nama,
		Jenis: request.Jenis,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, jabatanDTO(jabatan))
}

func (api directoryAPI) updateJabatan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("jabatanId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}

	fields, err := decodePatchFields(w, r, domain.JabatanPatchFields)
	if err != nil {
		writeValidationError(w)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	jabatan, err := api.jabatan.Update(r.Context(), principal, id, domain.NewJabatanPatch(fields))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, jabatanDTO(jabatan))
}

func (api directoryAPI) archiveJabatan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("jabatanId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	if err := api.jabatan.Archive(r.Context(), principal, id); err != nil {
		writeAppError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
