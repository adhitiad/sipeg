// Transport HTTP untuk area kontrak kerja, perpanjangan pasca-BUP, dan master
// pesangonAturan (DEC-083).
package httpserver

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"simpeg/backend/internal/apierror"
	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/models"

	"simpeg/backend/internal/kontrak/app"
	"simpeg/backend/internal/kontrak/domain"
	"simpeg/backend/internal/kontrak/repository"
)

// kontrakAPI memegang use case kontrak kerja.
type kontrakAPI struct {
	service *app.Service
}

// pesangonAPI memegang use case master pesangonAturan.
type pesangonAPI struct {
	service *app.PesangonService
}

// KontrakDTO adalah representasi satu kontrak kerja.
type KontrakDTO struct {
	ID        string  `json:"id"`
	UserID    string  `json:"userId"`
	Jenis     string  `json:"jenis"`
	NomorSk   string  `json:"nomorSk"`
	Mulai     string  `json:"mulai"`
	Selesai   *string `json:"selesai,omitempty"`
	Status    string  `json:"status"`
	SkId      *string `json:"skId,omitempty"`
	CreatedAt string  `json:"createdAt"`
}

// KontrakDTOFromModel memetakan model ke DTO.
func KontrakDTOFromModel(item models.KontrakKerja) KontrakDTO {
	var selesai *string
	if item.Selesai != nil {
		s := item.Selesai.Format("2006-01-02")
		selesai = &s
	}
	return KontrakDTO{
		ID:        item.ID,
		UserID:    item.UserId,
		Jenis:     string(item.Jenis),
		NomorSk:   item.NomorSk,
		Mulai:     item.Mulai.Format("2006-01-02"),
		Selesai:   selesai,
		Status:    item.Status,
		SkId:      item.SkId,
		CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339),
	}
}

// RegisterKontrakRoutes mendaftarkan endpoint kontrak, perpanjangan, dan
// pesangonAturan.
func RegisterKontrakRoutes(mux *http.ServeMux, resolver auth.SessionResolver, policy auth.Policy, kontrak *app.Service, pesangon *app.PesangonService) {
	protected := func(permission auth.Permission, handler http.Handler) http.Handler {
		return auth.Authenticate(resolver)(auth.RequirePermission(policy, permission, handler))
	}
	protectedAny := func(permissions []auth.Permission, handler http.Handler) http.Handler {
		return auth.Authenticate(resolver)(auth.RequireAnyPermission(policy, permissions, handler))
	}
	kapi := kontrakAPI{service: kontrak}
	papi := pesangonAPI{service: pesangon}
	mux.Handle("POST "+APIPrefix+"/perpanjangan", protected("perpanjangan:manage", http.HandlerFunc(kapi.extend)))
	mux.Handle("GET "+APIPrefix+"/kontrak", protectedAny([]auth.Permission{"kontrak:read:any", "kontrak:read:homebase", "kontrak:read:self"}, http.HandlerFunc(kapi.list)))
	mux.Handle("GET "+APIPrefix+"/pesangon-aturan", protected("pesangon:manage", http.HandlerFunc(papi.list)))
	mux.Handle("POST "+APIPrefix+"/pesangon-aturan", protected("pesangon:manage", http.HandlerFunc(papi.upsert)))
	mux.Handle("PATCH "+APIPrefix+"/pesangon-aturan/{id}", protected("pesangon:manage", http.HandlerFunc(papi.update)))
	mux.Handle("DELETE "+APIPrefix+"/pesangon-aturan/{id}", protected("pesangon:manage", http.HandlerFunc(papi.delete)))
}

type perpanjanganRequest struct {
	UserID       string  `json:"userId"`
	Mulai        *string `json:"mulai"`
	DurasiBulan  *int    `json:"durasiBulan"`
	AllowOverlap bool    `json:"allowOverlap"`
}

func (api kontrakAPI) extend(w http.ResponseWriter, r *http.Request) {
	var request perpanjanganRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}
	var mulai *time.Time
	if request.Mulai != nil {
		parsed, err := time.Parse("2006-01-02", strings.TrimSpace(*request.Mulai))
		if err != nil {
			writeErrorResponse(w, http.StatusUnprocessableEntity, apierror.CodeInvalidQuery)
			return
		}
		mulai = &parsed
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	kontrak, err := api.service.Extend(r.Context(), principal, request.UserID, mulai, derefInt(request.DurasiBulan), request.AllowOverlap)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrForbidden):
			writeErrorResponse(w, http.StatusForbidden, apierror.CodeForbidden)
		case errors.Is(err, domain.ErrNotFound):
			writeErrorResponse(w, http.StatusNotFound, apierror.CodeNotFound)
		case errors.Is(err, domain.ErrInvalidInput):
			writeErrorResponse(w, http.StatusUnprocessableEntity, apierror.CodeInvalidQuery)
		case errors.Is(err, domain.ErrOverlap):
			writeErrorResponse(w, http.StatusConflict, apierror.CodeConflict)
		default:
			writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
		}
		return
	}
	writeJSON(w, http.StatusCreated, KontrakDTOFromModel(kontrak))
}

func (api kontrakAPI) list(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFromContext(r.Context())
	userID := r.URL.Query().Get("userId")
	jenis := r.URL.Query().Get("jenis")
	selesaiBefore := r.URL.Query().Get("selesaiBefore")
	filter := repository.KontrakFilter{}
	if userID != "" {
		if !validUUID(userID) {
			writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
			return
		}
		filter.UserID = userID
	}
	if jenis != "" {
		filter.Jenis = jenis
	}
	if selesaiBefore != "" {
		parsed, err := time.Parse("2006-01-02", selesaiBefore)
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
			return
		}
		filter.SelesaiSebelum = &parsed
	}
	items, total, err := api.service.List(r.Context(), principal, filter)
	if err != nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
		return
	}
	out := make([]KontrakDTO, 0, len(items))
	for _, item := range items {
		out = append(out, KontrakDTOFromModel(item))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items": out,
		"total": total,
	})
}

// PesangonAturanDTO adalah representasi baris aturan pesangon.
type PesangonAturanDTO struct {
	ID                 string  `json:"id"`
	KategoriStatusID   string  `json:"kategoriStatusId"`
	MasaKerjaTahun     int     `json:"masaKerjaTahun"`
	PesangonBulan      float64 `json:"pesangonBulan"`
	PenghargaanBulan   float64 `json:"penghargaanBulan"`
	GantiKerugianBulan float64 `json:"gantiKerugianBulan"`
	SumberRegulasi     string  `json:"sumberRegulasi"`
}

// PesangonAturanDTOFromModel memetakan model ke DTO.
func PesangonAturanDTOFromModel(item models.PesangonAturan) PesangonAturanDTO {
	return PesangonAturanDTO{
		ID:                 item.ID,
		KategoriStatusID:   item.KategoriStatusId,
		MasaKerjaTahun:     item.MasaKerjaTahun,
		PesangonBulan:      item.PesangonBulan,
		PenghargaanBulan:   item.PenghargaanBulan,
		GantiKerugianBulan: item.GantiKerugianBulan,
		SumberRegulasi:     item.SumberRegulasi,
	}
}

func (api pesangonAPI) list(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFromContext(r.Context())
	kat := r.URL.Query().Get("kategoriStatusId")
	items, err := api.service.List(r.Context(), principal, kat)
	if err != nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
		return
	}
	out := make([]PesangonAturanDTO, 0, len(items))
	for _, item := range items {
		out = append(out, PesangonAturanDTOFromModel(item))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

type pesangonSaveRequest struct {
	KategoriStatusID   string  `json:"kategoriStatusId"`
	MasaKerjaTahun     int     `json:"masaKerjaTahun"`
	PesangonBulan      float64 `json:"pesangonBulan"`
	PenghargaanBulan   float64 `json:"penghargaanBulan"`
	GantiKerugianBulan float64 `json:"gantiKerugianBulan"`
	SumberRegulasi     string  `json:"sumberRegulasi"`
}

func (api pesangonAPI) upsert(w http.ResponseWriter, r *http.Request) {
	var request pesangonSaveRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	item, err := api.service.Upsert(r.Context(), principal, domain.PesangonAturanInput{
		KategoriStatusID:   request.KategoriStatusID,
		MasaKerjaTahun:     request.MasaKerjaTahun,
		PesangonBulan:      request.PesangonBulan,
		PenghargaanBulan:   request.PenghargaanBulan,
		GantiKerugianBulan: request.GantiKerugianBulan,
		SumberRegulasi:     request.SumberRegulasi,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrForbidden):
			writeErrorResponse(w, http.StatusForbidden, apierror.CodeForbidden)
		case errors.Is(err, domain.ErrValidation):
			writeErrorResponse(w, http.StatusUnprocessableEntity, apierror.CodeInvalidQuery)
		default:
			writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
		}
		return
	}
	writeJSON(w, http.StatusCreated, PesangonAturanDTOFromModel(item))
}

func (api pesangonAPI) update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	var request pesangonSaveRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	item, err := api.service.Update(r.Context(), principal, id, domain.PesangonAturanInput{
		KategoriStatusID:   request.KategoriStatusID,
		MasaKerjaTahun:     request.MasaKerjaTahun,
		PesangonBulan:      request.PesangonBulan,
		PenghargaanBulan:   request.PenghargaanBulan,
		GantiKerugianBulan: request.GantiKerugianBulan,
		SumberRegulasi:     request.SumberRegulasi,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrForbidden):
			writeErrorResponse(w, http.StatusForbidden, apierror.CodeForbidden)
		case errors.Is(err, domain.ErrValidation):
			writeErrorResponse(w, http.StatusUnprocessableEntity, apierror.CodeInvalidQuery)
		case errors.Is(err, repository.ErrNotFoundPesangon):
			writeErrorResponse(w, http.StatusNotFound, apierror.CodeNotFound)
		default:
			writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
		}
		return
	}
	writeJSON(w, http.StatusOK, PesangonAturanDTOFromModel(item))
}

func (api pesangonAPI) delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	if err := api.service.Delete(r.Context(), principal, id); err != nil {
		switch {
		case errors.Is(err, domain.ErrForbidden):
			writeErrorResponse(w, http.StatusForbidden, apierror.CodeForbidden)
		case errors.Is(err, repository.ErrNotFoundPesangon):
			writeErrorResponse(w, http.StatusNotFound, apierror.CodeNotFound)
		default:
			writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// derefInt mengembalikan nilai int atau 0 bila nil.
func derefInt(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}
