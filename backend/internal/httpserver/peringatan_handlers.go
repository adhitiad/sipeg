// Transport HTTP untuk area peringatan (SP1) pegawai (DEC-077).
package httpserver

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"simpeg/backend/internal/apierror"
	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/models"

	"simpeg/backend/internal/peringatan/app"
	"simpeg/backend/internal/peringatan/domain"
	"simpeg/backend/internal/peringatan/repository"
)

// PeringatanDTO adalah representasi peringatan (SP1).
type PeringatanDTO struct {
	ID            string  `json:"id"`
	UserID        string  `json:"userId"`
	Tingkat       string  `json:"tingkat"`
	Alasan        string  `json:"alasan"`
	NomorSurat    *string `json:"nomorSurat,omitempty"`
	Tanggal       string  `json:"tanggal"`
	Penerbit      string  `json:"penerbit"`
	Sanksi        *string `json:"sanksi,omitempty"`
	SanksiCatatan *string `json:"sanksiCatatan,omitempty"`
	CreatedAt     string  `json:"createdAt"`
}

type peringatanAPI struct {
	service *app.Service
}

// RegisterPeringatanRoutes mendaftarkan endpoint peringatan (SP1).
func RegisterPeringatanRoutes(mux *http.ServeMux, resolver auth.SessionResolver, policy auth.Policy, service *app.Service) {
	if service == nil {
		return
	}
	protectedAny := func(permissions []auth.Permission, handler http.Handler) http.Handler {
		return auth.Authenticate(resolver)(auth.RequireAnyPermission(policy, permissions, handler))
	}
	protectedOne := func(permission auth.Permission, handler http.Handler) http.Handler {
		return auth.Authenticate(resolver)(auth.RequirePermission(policy, permission, handler))
	}
	mux.Handle("POST "+APIPrefix+"/peringatan", protectedOne("peringatan:sp1:manage", http.HandlerFunc(peringatanAPI{service}.create)))
	mux.Handle("GET "+APIPrefix+"/peringatan", protectedAny([]auth.Permission{"peringatan:read:any", "peringatan:read:homebase"}, http.HandlerFunc(peringatanAPI{service}.list)))
	mux.Handle("GET "+APIPrefix+"/peringatan/{id}", protectedAny([]auth.Permission{"peringatan:read:any", "peringatan:read:homebase"}, http.HandlerFunc(peringatanAPI{service}.get)))
}

func peringatanDTO(item models.Peringatan) PeringatanDTO {
	sanksi := (*string)(nil)
	if item.Sanksi != nil {
		v := string(*item.Sanksi)
		sanksi = &v
	}
	return PeringatanDTO{
		ID:            item.ID,
		UserID:        item.UserId,
		Tingkat:       string(item.Tingkat),
		Alasan:        item.Alasan,
		NomorSurat:    item.NomorSurat,
		Tanggal:       item.Tanggal.Format("2006-01-02"),
		Penerbit:      item.Penerbit,
		Sanksi:        sanksi,
		SanksiCatatan: item.SanksiCatatan,
		CreatedAt:     item.CreatedAt.UTC().Format(time.RFC3339),
	}
}

type peringatanCreateRequest struct {
	UserID        string  `json:"userId"`
	Tingkat       string  `json:"tingkat"`
	Alasan        string  `json:"alasan"`
	NomorSurat    *string `json:"nomorSurat,omitempty"`
	Tanggal       string  `json:"tanggal"`
	Penerbit      string  `json:"penerbit"`
	Sanksi        *string `json:"sanksi,omitempty"`
	SanksiCatatan *string `json:"sanksiCatatan,omitempty"`
}

func (api peringatanAPI) create(w http.ResponseWriter, r *http.Request) {
	var request peringatanCreateRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}
	tanggal, err := time.Parse("2006-01-02", strings.TrimSpace(request.Tanggal))
	if err != nil {
		writeErrorResponse(w, http.StatusUnprocessableEntity, apierror.CodeInvalidQuery)
		return
	}
	var sanksi *models.JenisSanksi
	if request.Sanksi != nil && strings.TrimSpace(*request.Sanksi) != "" {
		v := models.JenisSanksi(strings.ToUpper(strings.TrimSpace(*request.Sanksi)))
		sanksi = &v
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	item, err := api.service.Create(r.Context(), principal, repository.CreateInput{
		UserID:        request.UserID,
		Tingkat:       domain.NormalizeTingkat(request.Tingkat),
		Alasan:        request.Alasan,
		NomorSurat:    request.NomorSurat,
		Tanggal:       tanggal,
		Penerbit:      request.Penerbit,
		Sanksi:        sanksi,
		SanksiCatatan: request.SanksiCatatan,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidUserID), errors.Is(err, domain.ErrInvalidAlasan),
			errors.Is(err, domain.ErrInvalidTingkat), errors.Is(err, domain.ErrInvalidDate),
			errors.Is(err, domain.ErrInvalidPenerbit), errors.Is(err, domain.ErrInvalidSanksi):
			writeErrorResponse(w, http.StatusUnprocessableEntity, apierror.CodeInvalidQuery)
		case errors.Is(err, domain.ErrNotFound):
			writeErrorResponse(w, http.StatusNotFound, apierror.CodeNotFound)
		case errors.Is(err, domain.ErrForbidden):
			writeErrorResponse(w, http.StatusForbidden, apierror.CodeForbidden)
		default:
			writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
		}
		return
	}
	writeJSON(w, http.StatusCreated, peringatanDTO(item))
}

func (api peringatanAPI) list(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFromContext(r.Context())
	userID := r.URL.Query().Get("userId")
	options := repository.ListOptions{}
	if userID != "" {
		if !validUUID(userID) {
			writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
			return
		}
		options.UserID = userID
	}
	if raw := r.URL.Query().Get("tingkat"); raw != "" {
		options.Tingkat = domain.NormalizeTingkat(raw)
	}
	items, total, err := api.service.List(r.Context(), principal, options)
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			writeErrorResponse(w, http.StatusForbidden, apierror.CodeForbidden)
			return
		}
		writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
		return
	}
	out := make([]PeringatanDTO, 0, len(items))
	for _, item := range items {
		out = append(out, peringatanDTO(item))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items": out,
		"total": total,
	})
}

func (api peringatanAPI) get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	item, err := api.service.Get(r.Context(), principal, id)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			writeErrorResponse(w, http.StatusNotFound, apierror.CodeNotFound)
		case errors.Is(err, domain.ErrForbidden):
			writeErrorResponse(w, http.StatusForbidden, apierror.CodeForbidden)
		default:
			writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
		}
		return
	}
	writeJSON(w, http.StatusOK, peringatanDTO(item))
}
