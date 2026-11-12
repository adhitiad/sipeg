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

// SertifikasiDTO adalah bentuk publik riwayat sertifikasi pegawai.
type SertifikasiDTO struct {
	ID               string                   `json:"id"`
	UserID           string                   `json:"userId"`
	JenisSertifikasi models.JenisSertifikasi  `json:"jenisSertifikasi"`
	NamaSertifikat   string                   `json:"namaSertifikat"`
	NomorRegistrasi  *string                  `json:"nomorRegistrasi"`
	Penyelenggara    string                   `json:"penyelenggara"`
	TanggalTerbit    time.Time                `json:"tanggalTerbit"`
	BerlakuSampai    *time.Time               `json:"berlakuSampai"`
	Status           models.StatusSertifikasi `json:"status"`
	CreatedAt        time.Time                `json:"createdAt"`
	UpdatedAt        time.Time                `json:"updatedAt"`
}

type sertifikasiCreateRequest struct {
	UserID           string  `json:"userId"`
	JenisSertifikasi string  `json:"jenisSertifikasi"`
	NamaSertifikat   string  `json:"namaSertifikat"`
	NomorRegistrasi  *string `json:"nomorRegistrasi"`
	Penyelenggara    string  `json:"penyelenggara"`
	TanggalTerbit    string  `json:"tanggalTerbit"`
	BerlakuSampai    *string `json:"berlakuSampai"`
	Status           string  `json:"status"`
}

func sertifikasiDTO(item models.RiwayatSertifikasi) SertifikasiDTO {
	return SertifikasiDTO{
		ID:               item.ID,
		UserID:           item.UserId,
		JenisSertifikasi: item.JenisSertifikasi,
		NamaSertifikat:   item.NamaSertifikat,
		NomorRegistrasi:  item.NomorRegistrasi,
		Penyelenggara:    item.Penyelenggara,
		TanggalTerbit:    timeutil.InLocation(item.TanggalTerbit),
		BerlakuSampai:    item.BerlakuSampai,
		Status:           item.Status,
		CreatedAt:        timeutil.InLocation(item.CreatedAt),
		UpdatedAt:        timeutil.InLocation(item.UpdatedAt),
	}
}

func (api directoryAPI) listSertifikasi(w http.ResponseWriter, r *http.Request) {
	query, err := parseListQuery(r, domain.SertifikasiSortFields(), "tanggalTerbit")
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	userID := r.URL.Query().Get("userId")
	if err != nil || textLength(search) > listing.MaxSearchLength || (userID != "" && !validUUID(userID)) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	options := listOptionsFromQuery(query, listing.Options{Search: search, UserID: userID})
	page, err := api.sertifikasi.List(r.Context(), principal, options)
	if err != nil {
		writeAppError(w, err)
		return
	}
	items := make([]SertifikasiDTO, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, sertifikasiDTO(item))
	}
	writeJSON(w, http.StatusOK, PageDTO[SertifikasiDTO]{Items: items, Page: page.Page, PageSize: page.PageSize, TotalItems: page.TotalItems, TotalPages: page.TotalPages})
}

func (api directoryAPI) getSertifikasi(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("sertifikasiId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	item, err := api.sertifikasi.Get(r.Context(), principal, id)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sertifikasiDTO(item))
}

func (api directoryAPI) createSertifikasi(w http.ResponseWriter, r *http.Request) {
	var request sertifikasiCreateRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}
	var berlakuSampai *string
	if request.BerlakuSampai != nil {
		val := *request.BerlakuSampai
		berlakuSampai = &val
	}
	input := domain.CreateRiwayatSertifikasi{
		UserID:           request.UserID,
		JenisSertifikasi: models.JenisSertifikasi(request.JenisSertifikasi),
		NamaSertifikat:   strings.TrimSpace(request.NamaSertifikat),
		NomorRegistrasi:  request.NomorRegistrasi,
		Penyelenggara:    strings.TrimSpace(request.Penyelenggara),
		TanggalTerbit:    request.TanggalTerbit,
		BerlakuSampai:    berlakuSampai,
		Status:           models.StatusSertifikasi(request.Status),
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	created, err := api.sertifikasi.Create(r.Context(), principal, input)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, sertifikasiDTO(created))
}

func (api directoryAPI) updateSertifikasi(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("sertifikasiId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	fields, err := decodePatchFields(w, r, domain.SertifikasiPatchFields)
	if err != nil {
		writeValidationError(w)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	updated, err := api.sertifikasi.Update(r.Context(), principal, id, domain.NewRiwayatSertifikasiPatch(fields))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sertifikasiDTO(updated))
}

func (api directoryAPI) archiveSertifikasi(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("sertifikasiId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	if err := api.sertifikasi.Archive(r.Context(), principal, id); err != nil {
		writeAppError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
