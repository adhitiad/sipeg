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

// KeluargaDTO adalah bentuk publik data keluarga pegawai.
type KeluargaDTO struct {
	ID               string                       `json:"id"`
	UserID           string                       `json:"userId"`
	Hubungan         models.JenisHubunganKeluarga `json:"hubungan"`
	Nama             string                       `json:"nama"`
	NIK              string                       `json:"nik"`
	TanggalLahir     time.Time                    `json:"tanggalLahir"`
	Pekerjaan        *string                      `json:"pekerjaan"`
	StatusTanggungan models.StatusTanggungan      `json:"statusTanggungan"`
	CreatedAt        time.Time                    `json:"createdAt"`
	UpdatedAt        time.Time                    `json:"updatedAt"`
}

type keluargaCreateRequest struct {
	UserID           string  `json:"userId"`
	Hubungan         string  `json:"hubungan"`
	Nama             string  `json:"nama"`
	NIK              string  `json:"nik"`
	TanggalLahir     string  `json:"tanggalLahir"`
	Pekerjaan        *string `json:"pekerjaan"`
	StatusTanggungan string  `json:"statusTanggungan"`
}

func keluargaDTO(item models.KeluargaPegawai) KeluargaDTO {
	return KeluargaDTO{
		ID:               item.ID,
		UserID:           item.UserId,
		Hubungan:         item.Hubungan,
		Nama:             item.Nama,
		NIK:              item.Nik,
		TanggalLahir:     timeutil.InLocation(item.TanggalLahir),
		Pekerjaan:        item.Pekerjaan,
		StatusTanggungan: item.StatusTanggungan,
		CreatedAt:        timeutil.InLocation(item.CreatedAt),
		UpdatedAt:        timeutil.InLocation(item.UpdatedAt),
	}
}

func (api directoryAPI) listKeluarga(w http.ResponseWriter, r *http.Request) {
	query, err := parseListQuery(r, domain.KeluargaSortFields(), "createdAt")
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	userID := r.URL.Query().Get("userId")
	if err != nil || textLength(search) > listing.MaxSearchLength || (userID != "" && !validUUID(userID)) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	options := listOptionsFromQuery(query, listing.Options{Search: search, UserID: userID})
	page, err := api.keluarga.List(r.Context(), principal, options)
	if err != nil {
		writeAppError(w, err)
		return
	}
	items := make([]KeluargaDTO, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, keluargaDTO(item))
	}
	writeJSON(w, http.StatusOK, PageDTO[KeluargaDTO]{Items: items, Page: page.Page, PageSize: page.PageSize, TotalItems: page.TotalItems, TotalPages: page.TotalPages})
}

func (api directoryAPI) getKeluarga(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("keluargaId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	item, err := api.keluarga.Get(r.Context(), principal, id)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, keluargaDTO(item))
}

func (api directoryAPI) createKeluarga(w http.ResponseWriter, r *http.Request) {
	var request keluargaCreateRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}
	input := domain.CreateKeluargaPegawai{
		UserID:           request.UserID,
		Hubungan:         models.JenisHubunganKeluarga(request.Hubungan),
		Nama:             strings.TrimSpace(request.Nama),
		NIK:              strings.TrimSpace(request.NIK),
		TanggalLahir:     request.TanggalLahir,
		Pekerjaan:        request.Pekerjaan,
		StatusTanggungan: models.StatusTanggungan(request.StatusTanggungan),
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	created, err := api.keluarga.Create(r.Context(), principal, input)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, keluargaDTO(created))
}

func (api directoryAPI) updateKeluarga(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("keluargaId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	fields, err := decodePatchFields(w, r, domain.KeluargaPatchFields)
	if err != nil {
		writeValidationError(w)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	updated, err := api.keluarga.Update(r.Context(), principal, id, domain.NewKeluargaPatch(fields))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, keluargaDTO(updated))
}

func (api directoryAPI) archiveKeluarga(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("keluargaId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	if err := api.keluarga.Archive(r.Context(), principal, id); err != nil {
		writeAppError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
