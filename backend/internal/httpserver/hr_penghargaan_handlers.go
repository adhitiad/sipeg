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

// PenghargaanDTO adalah bentuk publik riwayat penghargaan pegawai.
type PenghargaanDTO struct {
	ID               string                  `json:"id"`
	UserID           string                  `json:"userId"`
	NamaPenghargaan  string                  `json:"namaPenghargaan"`
	JenisPenghargaan models.JenisPenghargaan `json:"jenisPenghargaan"`
	Penyelenggara    string                  `json:"penyelenggara"`
	TanggalTerima    time.Time               `json:"tanggalTerima"`
	NomorSk          *string                 `json:"nomorSk"`
	CreatedAt        time.Time               `json:"createdAt"`
	UpdatedAt        time.Time               `json:"updatedAt"`
}

type penghargaanCreateRequest struct {
	UserID           string  `json:"userId"`
	NamaPenghargaan  string  `json:"namaPenghargaan"`
	JenisPenghargaan string  `json:"jenisPenghargaan"`
	Penyelenggara    string  `json:"penyelenggara"`
	TanggalTerima    string  `json:"tanggalTerima"`
	NomorSk          *string `json:"nomorSk"`
}

func penghargaanDTO(item models.RiwayatPenghargaan) PenghargaanDTO {
	return PenghargaanDTO{
		ID:               item.ID,
		UserID:           item.UserId,
		NamaPenghargaan:  item.NamaPenghargaan,
		JenisPenghargaan: item.JenisPenghargaan,
		Penyelenggara:    item.Penyelenggara,
		TanggalTerima:    timeutil.InLocation(item.TanggalTerima),
		NomorSk:          item.NomorSk,
		CreatedAt:        timeutil.InLocation(item.CreatedAt),
		UpdatedAt:        timeutil.InLocation(item.UpdatedAt),
	}
}

func (api directoryAPI) listPenghargaan(w http.ResponseWriter, r *http.Request) {
	query, err := parseListQuery(r, domain.PenghargaanSortFields(), "tanggalTerima")
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	userID := r.URL.Query().Get("userId")
	if err != nil || textLength(search) > listing.MaxSearchLength || (userID != "" && !validUUID(userID)) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	options := listOptionsFromQuery(query, listing.Options{Search: search, UserID: userID})
	page, err := api.penghargaan.List(r.Context(), principal, options)
	if err != nil {
		writeAppError(w, err)
		return
	}
	items := make([]PenghargaanDTO, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, penghargaanDTO(item))
	}
	writeJSON(w, http.StatusOK, PageDTO[PenghargaanDTO]{Items: items, Page: page.Page, PageSize: page.PageSize, TotalItems: page.TotalItems, TotalPages: page.TotalPages})
}

func (api directoryAPI) getPenghargaan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("penghargaanId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	item, err := api.penghargaan.Get(r.Context(), principal, id)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, penghargaanDTO(item))
}

func (api directoryAPI) createPenghargaan(w http.ResponseWriter, r *http.Request) {
	var request penghargaanCreateRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}
	input := domain.CreateRiwayatPenghargaan{
		UserID:           request.UserID,
		NamaPenghargaan:  strings.TrimSpace(request.NamaPenghargaan),
		JenisPenghargaan: models.JenisPenghargaan(request.JenisPenghargaan),
		Penyelenggara:    strings.TrimSpace(request.Penyelenggara),
		TanggalTerima:    request.TanggalTerima,
		NomorSk:          request.NomorSk,
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	created, err := api.penghargaan.Create(r.Context(), principal, input)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, penghargaanDTO(created))
}

func (api directoryAPI) updatePenghargaan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("penghargaanId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	fields, err := decodePatchFields(w, r, domain.PenghargaanPatchFields)
	if err != nil {
		writeValidationError(w)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	updated, err := api.penghargaan.Update(r.Context(), principal, id, domain.NewRiwayatPenghargaanPatch(fields))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, penghargaanDTO(updated))
}

func (api directoryAPI) archivePenghargaan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("penghargaanId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	if err := api.penghargaan.Archive(r.Context(), principal, id); err != nil {
		writeAppError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
