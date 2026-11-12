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

// ArsipDTO adalah bentuk publik arsip dokumen pegawai.
type ArsipDTO struct {
	ID                 string                   `json:"id"`
	UserID             string                   `json:"userId"`
	JenisDokumen       models.JenisDokumenArsip `json:"jenisDokumen"`
	Judul              string                   `json:"judul"`
	FileUrl            string                   `json:"fileUrl"`
	FileSize           int64                    `json:"fileSize"`
	MimeType           string                   `json:"mimeType"`
	Keterangan         *string                  `json:"keterangan"`
	BerlakuSampai      *time.Time               `json:"berlakuSampai"`
	CloudinaryPublicId *string                  `json:"cloudinaryPublicId"`
	CreatedAt          time.Time                `json:"createdAt"`
	UpdatedAt          time.Time                `json:"updatedAt"`
}

type arsipCreateRequest struct {
	UserID             string  `json:"userId"`
	JenisDokumen       string  `json:"jenisDokumen"`
	Judul              string  `json:"judul"`
	FileUrl            string  `json:"fileUrl"`
	FileSize           int64   `json:"fileSize"`
	MimeType           string  `json:"mimeType"`
	Keterangan         *string `json:"keterangan"`
	BerlakuSampai      *string `json:"berlakuSampai"`
	CloudinaryPublicId *string `json:"cloudinaryPublicId"`
}

func arsipDTO(item models.ArsipDokumenPegawai) ArsipDTO {
	return ArsipDTO{
		ID:                 item.ID,
		UserID:             item.UserId,
		JenisDokumen:       item.JenisDokumen,
		Judul:              item.Judul,
		FileUrl:            item.FileUrl,
		FileSize:           item.FileSize,
		MimeType:           item.MimeType,
		Keterangan:         item.Keterangan,
		BerlakuSampai:      item.BerlakuSampai,
		CloudinaryPublicId: item.CloudinaryPublicId,
		CreatedAt:          timeutil.InLocation(item.CreatedAt),
		UpdatedAt:          timeutil.InLocation(item.UpdatedAt),
	}
}

func (api directoryAPI) listArsip(w http.ResponseWriter, r *http.Request) {
	query, err := parseListQuery(r, domain.ArsipSortFields(), "createdAt")
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	userID := r.URL.Query().Get("userId")
	if err != nil || textLength(search) > listing.MaxSearchLength || (userID != "" && !validUUID(userID)) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	options := listOptionsFromQuery(query, listing.Options{Search: search, UserID: userID})
	page, err := api.arsip.List(r.Context(), principal, options)
	if err != nil {
		writeAppError(w, err)
		return
	}
	items := make([]ArsipDTO, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, arsipDTO(item))
	}
	writeJSON(w, http.StatusOK, PageDTO[ArsipDTO]{Items: items, Page: page.Page, PageSize: page.PageSize, TotalItems: page.TotalItems, TotalPages: page.TotalPages})
}

func (api directoryAPI) getArsip(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("arsipId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	item, err := api.arsip.Get(r.Context(), principal, id)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, arsipDTO(item))
}

func (api directoryAPI) createArsip(w http.ResponseWriter, r *http.Request) {
	var request arsipCreateRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}
	var berlakuSampai *string
	if request.BerlakuSampai != nil {
		val := *request.BerlakuSampai
		berlakuSampai = &val
	}
	input := domain.CreateArsipDokumenPegawai{
		UserID:             request.UserID,
		JenisDokumen:       models.JenisDokumenArsip(request.JenisDokumen),
		Judul:              strings.TrimSpace(request.Judul),
		FileURL:            request.FileUrl,
		FileSize:           request.FileSize,
		MimeType:           request.MimeType,
		Keterangan:         request.Keterangan,
		BerlakuSampai:      berlakuSampai,
		CloudinaryPublicId: request.CloudinaryPublicId,
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	created, err := api.arsip.Create(r.Context(), principal, input)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, arsipDTO(created))
}

func (api directoryAPI) updateArsip(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("arsipId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	fields, err := decodePatchFields(w, r, domain.ArsipPatchFields)
	if err != nil {
		writeValidationError(w)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	updated, err := api.arsip.Update(r.Context(), principal, id, domain.NewArsipDokumenPatch(fields))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, arsipDTO(updated))
}

func (api directoryAPI) archiveArsip(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("arsipId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	if err := api.arsip.Archive(r.Context(), principal, id); err != nil {
		writeAppError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
