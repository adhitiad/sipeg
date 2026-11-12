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

// TunjanganDTO adalah bentuk publik komponen tunjangan pegawai.
type TunjanganDTO struct {
	ID             string                `json:"id"`
	UserID         string                `json:"userId"`
	JenisTunjangan models.JenisTunjangan `json:"jenisTunjangan"`
	Nilai          float64               `json:"nilai"`
	Aktif          bool                  `json:"aktif"`
	CreatedAt      time.Time             `json:"createdAt"`
	UpdatedAt      time.Time             `json:"updatedAt"`
}

type tunjanganCreateRequest struct {
	UserID         string  `json:"userId"`
	JenisTunjangan string  `json:"jenisTunjangan"`
	Nilai          float64 `json:"nilai"`
	Aktif          bool    `json:"aktif"`
}

func tunjanganDTO(item models.KomponenTunjangan) TunjanganDTO {
	return TunjanganDTO{
		ID:             item.ID,
		UserID:         item.UserId,
		JenisTunjangan: item.JenisTunjangan,
		Nilai:          item.Nilai,
		Aktif:          item.Aktif,
		CreatedAt:      timeutil.InLocation(item.CreatedAt),
		UpdatedAt:      timeutil.InLocation(item.UpdatedAt),
	}
}

func (api directoryAPI) listTunjangan(w http.ResponseWriter, r *http.Request) {
	query, err := parseListQuery(r, payrolldomain.TunjanganSortFields(), "createdAt")
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	userID := r.URL.Query().Get("userId")
	if err != nil || textLength(search) > listing.MaxSearchLength || (userID != "" && !validUUID(userID)) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	options := listOptionsFromQuery(query, listing.Options{Search: search, UserID: userID})
	page, err := api.tunjangan.List(r.Context(), principal, options)
	if err != nil {
		writeAppError(w, err)
		return
	}
	items := make([]TunjanganDTO, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, tunjanganDTO(item))
	}
	writeJSON(w, http.StatusOK, PageDTO[TunjanganDTO]{Items: items, Page: page.Page, PageSize: page.PageSize, TotalItems: page.TotalItems, TotalPages: page.TotalPages})
}

func (api directoryAPI) getTunjangan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("tunjanganId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	item, err := api.tunjangan.Get(r.Context(), principal, id)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tunjanganDTO(item))
}

func (api directoryAPI) createTunjangan(w http.ResponseWriter, r *http.Request) {
	var request tunjanganCreateRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}
	input := payrolldomain.CreateKomponenTunjangan{
		UserID:         request.UserID,
		JenisTunjangan: strings.TrimSpace(request.JenisTunjangan),
		Nilai:          request.Nilai,
		Aktif:          request.Aktif,
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	created, err := api.tunjangan.Create(r.Context(), principal, input)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, tunjanganDTO(created))
}

func (api directoryAPI) updateTunjangan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("tunjanganId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	fields, err := decodePatchFields(w, r, payrolldomain.KomponenTunjanganPatchFields)
	if err != nil {
		writeValidationError(w)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	updated, err := api.tunjangan.Update(r.Context(), principal, id, payrolldomain.NewKomponenTunjanganPatch(fields))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tunjanganDTO(updated))
}

func (api directoryAPI) archiveTunjangan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("tunjanganId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	if err := api.tunjangan.Archive(r.Context(), principal, id); err != nil {
		writeAppError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
