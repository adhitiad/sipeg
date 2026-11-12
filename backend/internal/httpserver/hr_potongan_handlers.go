package httpserver

import (
	"net/http"
	"strings"
	"time"

	"simpeg/backend/internal/apierror"
	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
	payrolldomain "simpeg/backend/internal/payroll/domain"
	"simpeg/backend/internal/timeutil"
)

// PotonganDTO adalah bentuk publik komponen potongan pegawai.
type PotonganDTO struct {
	ID             string               `json:"id"`
	UserID         string               `json:"userId"`
	JenisPotongan  models.JenisPotongan `json:"jenisPotongan"`
	Persentase     *float64             `json:"persentase"`
	Nominal        float64              `json:"nominal"`
	PorsiInstitusi *float64             `json:"porsiInstitusi"`
	PorsiPegawai   *float64             `json:"porsiPegawai"`
	Aktif          bool                 `json:"aktif"`
	CreatedAt      time.Time            `json:"createdAt"`
	UpdatedAt      time.Time            `json:"updatedAt"`
}

type potonganCreateRequest struct {
	UserID         string   `json:"userId"`
	JenisPotongan  string   `json:"jenisPotongan"`
	Persentase     *float64 `json:"persentase"`
	Nominal        float64  `json:"nominal"`
	PorsiInstitusi *float64 `json:"porsiInstitusi"`
	PorsiPegawai   *float64 `json:"porsiPegawai"`
	Aktif          bool     `json:"aktif"`
}

func potonganDTO(item models.KomponenPotongan) PotonganDTO {
	return PotonganDTO{
		ID:             item.ID,
		UserID:         item.UserId,
		JenisPotongan:  item.JenisPotongan,
		Persentase:     item.Persentase,
		Nominal:        item.Nominal,
		PorsiInstitusi: item.PorsiInstitusi,
		PorsiPegawai:   item.PorsiPegawai,
		Aktif:          item.Aktif,
		CreatedAt:      timeutil.InLocation(item.CreatedAt),
		UpdatedAt:      timeutil.InLocation(item.UpdatedAt),
	}
}

func (api directoryAPI) listPotongan(w http.ResponseWriter, r *http.Request) {
	query, err := parseListQuery(r, payrolldomain.PotonganSortFields(), "createdAt")
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	userID := r.URL.Query().Get("userId")
	if err != nil || textLength(search) > listing.MaxSearchLength || (userID != "" && !validUUID(userID)) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	options := listOptionsFromQuery(query, listing.Options{Search: search, UserID: userID})
	page, err := api.potongan.List(r.Context(), principal, options)
	if err != nil {
		writeAppError(w, err)
		return
	}
	items := make([]PotonganDTO, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, potonganDTO(item))
	}
	writeJSON(w, http.StatusOK, PageDTO[PotonganDTO]{Items: items, Page: page.Page, PageSize: page.PageSize, TotalItems: page.TotalItems, TotalPages: page.TotalPages})
}

func (api directoryAPI) getPotongan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("potonganId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	item, err := api.potongan.Get(r.Context(), principal, id)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, potonganDTO(item))
}

func (api directoryAPI) createPotongan(w http.ResponseWriter, r *http.Request) {
	var request potonganCreateRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}
	input := payrolldomain.CreateKomponenPotongan{
		UserID:         request.UserID,
		JenisPotongan:  strings.TrimSpace(request.JenisPotongan),
		Persentase:     request.Persentase,
		Nominal:        request.Nominal,
		PorsiInstitusi: request.PorsiInstitusi,
		PorsiPegawai:   request.PorsiPegawai,
		Aktif:          request.Aktif,
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	created, err := api.potongan.Create(r.Context(), principal, input)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, potonganDTO(created))
}

func (api directoryAPI) updatePotongan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("potonganId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	fields, err := decodePatchFields(w, r, payrolldomain.KomponenPotonganPatchFields)
	if err != nil {
		writeValidationError(w)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	updated, err := api.potongan.Update(r.Context(), principal, id, payrolldomain.NewKomponenPotonganPatch(fields))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, potonganDTO(updated))
}

func (api directoryAPI) archivePotongan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("potonganId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	if err := api.potongan.Archive(r.Context(), principal, id); err != nil {
		writeAppError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
