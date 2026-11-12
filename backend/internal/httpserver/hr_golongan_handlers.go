package httpserver

import (
	"net/http"
	"strings"
	"time"

	"simpeg/backend/internal/apierror"
	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/directory/domain"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
	"simpeg/backend/internal/timeutil"
)

// GolonganDTO adalah bentuk publik riwayat golongan pegawai.
type GolonganDTO struct {
	ID          string    `json:"id"`
	UserID      string    `json:"userId"`
	Golongan    string    `json:"golongan"`
	Ruang       string    `json:"ruang"`
	TMT         time.Time `json:"tmt"`
	AngkaKredit int       `json:"angkaKredit"`
	NomorSk     *string   `json:"nomorSk"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type golonganCreateRequest struct {
	UserID      string  `json:"userId"`
	Golongan    string  `json:"golongan"`
	Ruang       string  `json:"ruang"`
	TMT         string  `json:"tmt"`
	AngkaKredit int     `json:"angkaKredit"`
	NomorSk     *string `json:"nomorSk"`
}

func golonganDTO(item models.RiwayatGolongan) GolonganDTO {
	return GolonganDTO{
		ID:          item.ID,
		UserID:      item.UserId,
		Golongan:    item.Golongan,
		Ruang:       item.Ruang,
		TMT:         timeutil.InLocation(item.TMT),
		AngkaKredit: item.AngkaKredit,
		NomorSk:     item.NomorSk,
		CreatedAt:   timeutil.InLocation(item.CreatedAt),
		UpdatedAt:   timeutil.InLocation(item.UpdatedAt),
	}
}

func (api directoryAPI) listGolongan(w http.ResponseWriter, r *http.Request) {
	query, err := parseListQuery(r, domain.GolonganSortFields(), "tmt")
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	userID := r.URL.Query().Get("userId")
	if err != nil || textLength(search) > listing.MaxSearchLength || (userID != "" && !validUUID(userID)) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	options := listOptionsFromQuery(query, listing.Options{Search: search, UserID: userID})
	page, err := api.golongan.List(r.Context(), principal, options)
	if err != nil {
		writeAppError(w, err)
		return
	}
	items := make([]GolonganDTO, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, golonganDTO(item))
	}
	writeJSON(w, http.StatusOK, PageDTO[GolonganDTO]{Items: items, Page: page.Page, PageSize: page.PageSize, TotalItems: page.TotalItems, TotalPages: page.TotalPages})
}

func (api directoryAPI) getGolongan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("golonganId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	item, err := api.golongan.Get(r.Context(), principal, id)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, golonganDTO(item))
}

func (api directoryAPI) createGolongan(w http.ResponseWriter, r *http.Request) {
	var request golonganCreateRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}
	input := domain.CreateRiwayatGolongan{
		UserID:      request.UserID,
		Golongan:    strings.TrimSpace(request.Golongan),
		Ruang:       strings.TrimSpace(request.Ruang),
		TMT:         request.TMT,
		AngkaKredit: request.AngkaKredit,
		NomorSk:     request.NomorSk,
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	created, err := api.golongan.Create(r.Context(), principal, input)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, golonganDTO(created))
}

func (api directoryAPI) updateGolongan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("golonganId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	fields, err := decodePatchFields(w, r, domain.GolonganPatchFields)
	if err != nil {
		writeValidationError(w)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	updated, err := api.golongan.Update(r.Context(), principal, id, domain.NewRiwayatGolonganPatch(fields))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, golonganDTO(updated))
}

func (api directoryAPI) archiveGolongan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("golonganId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	if err := api.golongan.Archive(r.Context(), principal, id); err != nil {
		writeAppError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
