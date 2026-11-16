// Transport HTTP untuk area pemberhentian pegawai (DEC-083).
//
// Frontend dan klien memakai prefix /api/v1; handler di sini menerjemahkan
// request ke use case, dan error domain ke status HTTP.
package httpserver

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"simpeg/backend/internal/apierror"
	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/models"

	"simpeg/backend/internal/pemberhentian/app"
	"simpeg/backend/internal/pemberhentian/domain"
	"simpeg/backend/internal/pemberhentian/repository"
)

// PemberhentianDTO adalah representasi pengajuan/keputusan pemberhentian.
type PemberhentianDTO struct {
	ID              string  `json:"id"`
	UserID          string  `json:"userId"`
	Jenis           string  `json:"jenis"`
	Alasan          string  `json:"alasan"`
	Tanggal         string  `json:"tanggal"`
	Status          string  `json:"status"`
	NomorSk         string  `json:"nomorSk"`
	HakTerakhir     *string `json:"hakTerakhir,omitempty"`
	Keterangan      *string `json:"keterangan,omitempty"`
	KompensasiBulan float64 `json:"kompensasiBulan"`
	CreatedAt       string  `json:"createdAt"`
}

type pemberhentianAPI struct {
	service *app.Service
}

// RegisterPemberhentianRoutes mendaftarkan endpoint pemberhentian pegawai.
func RegisterPemberhentianRoutes(mux *http.ServeMux, resolver auth.SessionResolver, policy auth.Policy, service *app.Service) {
	if service == nil {
		return
	}
	protectedAny := func(permissions []auth.Permission, handler http.Handler) http.Handler {
		return auth.Authenticate(resolver)(auth.RequireAnyPermission(policy, permissions, handler))
	}
	mux.Handle("POST "+APIPrefix+"/pemberhentian", protectedAny([]auth.Permission{"pemberhentian:propose:any", "pemberhentian:propose:homebase"}, http.HandlerFunc(pemberhentianAPI{service}.propose)))
	mux.Handle("GET "+APIPrefix+"/pemberhentian", protectedAny([]auth.Permission{"pemberhentian:read:any", "pemberhentian:read:homebase"}, http.HandlerFunc(pemberhentianAPI{service}.list)))
	mux.Handle("GET "+APIPrefix+"/pemberhentian/{id}", protectedAny([]auth.Permission{"pemberhentian:read:any", "pemberhentian:read:homebase"}, http.HandlerFunc(pemberhentianAPI{service}.get)))
	mux.Handle("POST "+APIPrefix+"/pemberhentian/{id}/approve", protectedAny([]auth.Permission{"pemberhentian:approve"}, http.HandlerFunc(pemberhentianAPI{service}.approve)))
	mux.Handle("GET "+APIPrefix+"/pemberhentian/{id}/kompensasi-preview", protectedAny([]auth.Permission{"pemberhentian:approve"}, http.HandlerFunc(pemberhentianAPI{service}.preview)))
	mux.Handle("POST "+APIPrefix+"/pemberhentian/{id}/reject", protectedAny([]auth.Permission{"pemberhentian:approve"}, http.HandlerFunc(pemberhentianAPI{service}.reject)))
}

func pemberhentianDTO(item models.Pemberhentian) PemberhentianDTO {
	alasan := ""
	if item.Alasan != nil {
		alasan = *item.Alasan
	}
	hakTerakhir := item.HakTerakhir
	keterangan := item.Keterangan
	return PemberhentianDTO{
		ID:              item.ID,
		UserID:          item.UserId,
		Jenis:           string(item.Jenis),
		Alasan:          alasan,
		Tanggal:         item.Tanggal.Format("2006-01-02"),
		Status:          string(item.Status),
		NomorSk:         item.NomorSk,
		HakTerakhir:     hakTerakhir,
		Keterangan:      keterangan,
		KompensasiBulan: item.KompensasiBulan,
		CreatedAt:       item.CreatedAt.UTC().Format(time.RFC3339),
	}
}

type pemberhentianProposeRequest struct {
	UserID     string  `json:"userId"`
	Alasan     string  `json:"alasan"`
	Tanggal    string  `json:"tanggal"`
	Keterangan *string `json:"keterangan"`
}

func (api pemberhentianAPI) propose(w http.ResponseWriter, r *http.Request) {
	var request pemberhentianProposeRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}
	tanggal, err := time.Parse("2006-01-02", strings.TrimSpace(request.Tanggal))
	if err != nil {
		writeErrorResponse(w, http.StatusUnprocessableEntity, apierror.CodeInvalidQuery)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	item, err := api.service.Propose(r.Context(), principal, repository.ProposeInput{
		UserID:     request.UserID,
		Alasan:     request.Alasan,
		Tanggal:    tanggal,
		Keterangan: request.Keterangan,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidAlasan), errors.Is(err, domain.ErrInvalidDate), errors.Is(err, domain.ErrInvalidUserID):
			writeErrorResponse(w, http.StatusUnprocessableEntity, apierror.CodeInvalidQuery)
		case errors.Is(err, domain.ErrNotFound):
			writeErrorResponse(w, http.StatusNotFound, apierror.CodeNotFound)
		case errors.Is(err, domain.ErrPending):
			writeErrorResponse(w, http.StatusConflict, apierror.CodeConflict)
		default:
			writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
		}
		return
	}
	writeJSON(w, http.StatusCreated, pemberhentianDTO(item))
}

func (api pemberhentianAPI) list(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFromContext(r.Context())
	userID := r.URL.Query().Get("userId")
	status := r.URL.Query().Get("status")
	options := repository.ListOptions{}
	if userID != "" {
		if !validUUID(userID) {
			writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
			return
		}
		options.UserID = userID
	}
	if status != "" {
		switch status {
		case "MENUNGGU", "DISETUJUI", "DITOLAK":
			options.Status = models.StatusPersetujuan(status)
		default:
			writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
			return
		}
	}
	items, total, err := api.service.List(r.Context(), principal, options)
	if err != nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
		return
	}
	out := make([]PemberhentianDTO, 0, len(items))
	for _, item := range items {
		out = append(out, pemberhentianDTO(item))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items": out,
		"total": total,
	})
}

func (api pemberhentianAPI) get(w http.ResponseWriter, r *http.Request) {
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
	writeJSON(w, http.StatusOK, pemberhentianDTO(item))
}

type pemberhentianActionRequest struct {
	Note *string `json:"note"`
}

func (api pemberhentianAPI) preview(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	preview, err := api.service.CompensasiPreview(r.Context(), principal, id)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound), errors.Is(err, domain.ErrNotFound):
			writeErrorResponse(w, http.StatusNotFound, apierror.CodeNotFound)
		case errors.Is(err, domain.ErrForbidden):
			writeErrorResponse(w, http.StatusForbidden, apierror.CodeForbidden)
		default:
			writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
		}
		return
	}
	writeJSON(w, http.StatusOK, preview)
}

func (api pemberhentianAPI) approve(w http.ResponseWriter, r *http.Request) {
	api.decide(w, r, true)
}

func (api pemberhentianAPI) reject(w http.ResponseWriter, r *http.Request) {
	api.decide(w, r, false)
}

func (api pemberhentianAPI) decide(w http.ResponseWriter, r *http.Request, approve bool) {
	id := r.PathValue("id")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	var request pemberhentianActionRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	var item models.Pemberhentian
	var err error
	if approve {
		item, err = api.service.Approve(r.Context(), principal, id, request.Note)
	} else {
		item, err = api.service.Reject(r.Context(), principal, id, request.Note)
	}
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			writeErrorResponse(w, http.StatusNotFound, apierror.CodeNotFound)
		case errors.Is(err, domain.ErrNotPending), errors.Is(err, domain.ErrPending):
			writeErrorResponse(w, http.StatusConflict, apierror.CodeConflict)
		case errors.Is(err, domain.ErrForbidden):
			writeErrorResponse(w, http.StatusForbidden, apierror.CodeForbidden)
		default:
			writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
		}
		return
	}
	writeJSON(w, http.StatusOK, pemberhentianDTO(item))
}
