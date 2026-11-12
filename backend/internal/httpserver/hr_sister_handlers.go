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

// SisterDTO adalah bentuk publik pemetaan ID eksternal SISTER/PDDIKTI.
type SisterDTO struct {
	ID                   string                    `json:"id"`
	UserID               string                    `json:"userId"`
	SistemEksternal      models.SistemEksternal    `json:"sistemEksternal"`
	IDEksternal          string                    `json:"idEksternal"`
	SinkronisasiTerakhir *time.Time                `json:"sinkronisasiTerakhir"`
	Status               models.StatusSinkronisasi `json:"status"`
	CreatedAt            time.Time                 `json:"createdAt"`
	UpdatedAt            time.Time                 `json:"updatedAt"`
}

type sisterCreateRequest struct {
	UserID          string `json:"userId"`
	SistemEksternal string `json:"sistemEksternal"`
	IDEksternal     string `json:"idEksternal"`
	Status          string `json:"status"`
}

func sisterDTO(item models.SisterSyncMap) SisterDTO {
	return SisterDTO{
		ID:                   item.ID,
		UserID:               item.UserId,
		SistemEksternal:      item.SistemEksternal,
		IDEksternal:          item.IDEksternal,
		SinkronisasiTerakhir: item.SinkronisasiTerakhir,
		Status:               item.Status,
		CreatedAt:            timeutil.InLocation(item.CreatedAt),
		UpdatedAt:            timeutil.InLocation(item.UpdatedAt),
	}
}

func (api directoryAPI) listSister(w http.ResponseWriter, r *http.Request) {
	query, err := parseListQuery(r, domain.SisterSortFields(), "createdAt")
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	userID := r.URL.Query().Get("userId")
	if err != nil || textLength(search) > listing.MaxSearchLength || (userID != "" && !validUUID(userID)) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	options := listOptionsFromQuery(query, listing.Options{Search: search, UserID: userID})
	page, err := api.sister.List(r.Context(), principal, options)
	if err != nil {
		writeAppError(w, err)
		return
	}
	items := make([]SisterDTO, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, sisterDTO(item))
	}
	writeJSON(w, http.StatusOK, PageDTO[SisterDTO]{Items: items, Page: page.Page, PageSize: page.PageSize, TotalItems: page.TotalItems, TotalPages: page.TotalPages})
}

func (api directoryAPI) getSister(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("sisterId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	item, err := api.sister.Get(r.Context(), principal, id)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sisterDTO(item))
}

func (api directoryAPI) createSister(w http.ResponseWriter, r *http.Request) {
	var request sisterCreateRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}
	input := domain.CreateSisterSyncMap{
		UserID:          request.UserID,
		SistemEksternal: models.SistemEksternal(request.SistemEksternal),
		IDEksternal:     strings.TrimSpace(request.IDEksternal),
		Status:          models.StatusSinkronisasi(request.Status),
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	created, err := api.sister.Create(r.Context(), principal, input)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, sisterDTO(created))
}

func (api directoryAPI) updateSister(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("sisterId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	fields, err := decodePatchFields(w, r, domain.SisterSyncMapPatchFields)
	if err != nil {
		writeValidationError(w)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	updated, err := api.sister.Update(r.Context(), principal, id, domain.NewSisterSyncMapPatch(fields))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sisterDTO(updated))
}

func (api directoryAPI) archiveSister(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("sisterId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	if err := api.sister.Archive(r.Context(), principal, id); err != nil {
		writeAppError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (api directoryAPI) syncSister(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("sisterId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	updated, err := api.sister.SyncNow(r.Context(), principal, id)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sisterDTO(updated))
}
