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

// PelatihanDTO adalah bentuk publik riwayat pelatihan pegawai.
type PelatihanDTO struct {
	ID                 string                `json:"id"`
	UserID             string                `json:"userId"`
	NamaPelatihan      string                `json:"namaPelatihan"`
	JenisPelatihan     models.JenisPelatihan `json:"jenisPelatihan"`
	Penyelenggara      string                `json:"penyelenggara"`
	TanggalMulai       time.Time             `json:"tanggalMulai"`
	TanggalSelesai     time.Time             `json:"tanggalSelesai"`
	DurasiJam          int                   `json:"durasiJam"`
	MemilikiSertifikat bool                  `json:"memilikiSertifikat"`
	CreatedAt          time.Time             `json:"createdAt"`
	UpdatedAt          time.Time             `json:"updatedAt"`
}

type pelatihanCreateRequest struct {
	UserID             string `json:"userId"`
	NamaPelatihan      string `json:"namaPelatihan"`
	JenisPelatihan     string `json:"jenisPelatihan"`
	Penyelenggara      string `json:"penyelenggara"`
	TanggalMulai       string `json:"tanggalMulai"`
	TanggalSelesai     string `json:"tanggalSelesai"`
	DurasiJam          int    `json:"durasiJam"`
	MemilikiSertifikat bool   `json:"memilikiSertifikat"`
}

func pelatihanDTO(item models.RiwayatPelatihan) PelatihanDTO {
	return PelatihanDTO{
		ID:                 item.ID,
		UserID:             item.UserId,
		NamaPelatihan:      item.NamaPelatihan,
		JenisPelatihan:     item.JenisPelatihan,
		Penyelenggara:      item.Penyelenggara,
		TanggalMulai:       timeutil.InLocation(item.TanggalMulai),
		TanggalSelesai:     timeutil.InLocation(item.TanggalSelesai),
		DurasiJam:          item.DurasiJam,
		MemilikiSertifikat: item.MemilikiSertifikat,
		CreatedAt:          timeutil.InLocation(item.CreatedAt),
		UpdatedAt:          timeutil.InLocation(item.UpdatedAt),
	}
}

func (api directoryAPI) listPelatihan(w http.ResponseWriter, r *http.Request) {
	query, err := parseListQuery(r, domain.PelatihanSortFields(), "tanggalMulai")
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	userID := r.URL.Query().Get("userId")
	if err != nil || textLength(search) > listing.MaxSearchLength || (userID != "" && !validUUID(userID)) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	options := listOptionsFromQuery(query, listing.Options{Search: search, UserID: userID})
	page, err := api.pelatihan.List(r.Context(), principal, options)
	if err != nil {
		writeAppError(w, err)
		return
	}
	items := make([]PelatihanDTO, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, pelatihanDTO(item))
	}
	writeJSON(w, http.StatusOK, PageDTO[PelatihanDTO]{Items: items, Page: page.Page, PageSize: page.PageSize, TotalItems: page.TotalItems, TotalPages: page.TotalPages})
}

func (api directoryAPI) getPelatihan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("pelatihanId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	item, err := api.pelatihan.Get(r.Context(), principal, id)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, pelatihanDTO(item))
}

func (api directoryAPI) createPelatihan(w http.ResponseWriter, r *http.Request) {
	var request pelatihanCreateRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}
	input := domain.CreateRiwayatPelatihan{
		UserID:             request.UserID,
		NamaPelatihan:      strings.TrimSpace(request.NamaPelatihan),
		JenisPelatihan:     models.JenisPelatihan(request.JenisPelatihan),
		Penyelenggara:      strings.TrimSpace(request.Penyelenggara),
		TanggalMulai:       request.TanggalMulai,
		TanggalSelesai:     request.TanggalSelesai,
		DurasiJam:          request.DurasiJam,
		MemilikiSertifikat: request.MemilikiSertifikat,
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	created, err := api.pelatihan.Create(r.Context(), principal, input)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, pelatihanDTO(created))
}

func (api directoryAPI) updatePelatihan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("pelatihanId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	fields, err := decodePatchFields(w, r, domain.PelatihanPatchFields)
	if err != nil {
		writeValidationError(w)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	updated, err := api.pelatihan.Update(r.Context(), principal, id, domain.NewRiwayatPelatihanPatch(fields))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, pelatihanDTO(updated))
}

func (api directoryAPI) archivePelatihan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("pelatihanId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	if err := api.pelatihan.Archive(r.Context(), principal, id); err != nil {
		writeAppError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
