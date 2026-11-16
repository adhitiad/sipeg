// Transport HTTP untuk area THR (`DEC-083`): konfigurasi per tahun
// (CRUD) dan generate slip THR.
package httpserver

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"simpeg/backend/internal/apierror"
	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/models"

	"simpeg/backend/internal/thr/app"
	"simpeg/backend/internal/thr/domain"
)

// ThrKonfigurasiDTO adalah representasi konfigurasi THR.
type ThrKonfigurasiDTO struct {
	Tahun                int      `json:"tahun"`
	BesaranTipe          string   `json:"besaranTipe"`
	Persen               *float64 `json:"persen,omitempty"`
	SyaratMasaKerjaBulan int      `json:"syaratMasaKerjaBulan"`
	BerlakuTanggal       string   `json:"berlakuTanggal"`
	UpdatedAt            string   `json:"updatedAt"`
}

func thrKonfigurasiDTO(item models.ThrKonfigurasi) ThrKonfigurasiDTO {
	return ThrKonfigurasiDTO{
		Tahun:                item.Tahun,
		BesaranTipe:          item.BesaranTipe,
		Persen:               item.Persen,
		SyaratMasaKerjaBulan: item.SyaratMasaKerjaBulan,
		BerlakuTanggal:       item.BerlakuTanggal.Format("2006-01-02"),
		UpdatedAt:            item.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

// ThrKonfigurasiRequest memuat isian membuat/mengubah konfigurasi.
type ThrKonfigurasiRequest struct {
	Tahun                int      `json:"tahun"`
	BesaranTipe          string   `json:"besaranTipe"`
	Persen               *float64 `json:"persen,omitempty"`
	SyaratMasaKerjaBulan int      `json:"syaratMasaKerjaBulan"`
	BerlakuTanggal       string   `json:"berlakuTanggal"`
}

// THRGenerateRequest memuat isian generate THR tahun tertentu beserta nominal
// yang diinput admin per pegawai (prorata).
type THRGenerateRequest struct {
	Tahun     int               `json:"tahun"`
	Overrides []THROverrideItem `json:"overrides"`
}

// THROverrideItem adalah nominal THR yang diinput admin untuk satu pegawai.
type THROverrideItem struct {
	UserID  string  `json:"userId"`
	Nominal float64 `json:"nominal"`
}

// RegisterThrRoutes mendaftarkan endpoint THR.
func RegisterThrRoutes(mux *http.ServeMux, resolver auth.SessionResolver, policy auth.Policy, service *app.Service) {
	if service == nil {
		return
	}
	protected := func(permission auth.Permission, handler http.Handler) http.Handler {
		return auth.Authenticate(resolver)(auth.RequirePermission(policy, permission, handler))
	}
	mux.Handle("GET "+APIPrefix+"/payroll/thr/config", protected("thr:manage", http.HandlerFunc(thrAPI{service}.list)))
	mux.Handle("POST "+APIPrefix+"/payroll/thr/config", protected("thr:manage", http.HandlerFunc(thrAPI{service}.upsert)))
	mux.Handle("DELETE "+APIPrefix+"/payroll/thr/config/{tahun}", protected("thr:manage", http.HandlerFunc(thrAPI{service}.delete)))
	mux.Handle("GET "+APIPrefix+"/payroll/thr/plan", protected("thr:manage", http.HandlerFunc(thrAPI{service}.plan)))
	mux.Handle("POST "+APIPrefix+"/payroll/thr/generate", protected("thr:manage", http.HandlerFunc(thrAPI{service}.generate)))
}

type thrAPI struct {
	service *app.Service
}

func (api thrAPI) list(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFromContext(r.Context())
	items, err := api.service.List(r.Context(), principal)
	if err != nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
		return
	}
	out := make([]ThrKonfigurasiDTO, 0, len(items))
	for _, item := range items {
		out = append(out, thrKonfigurasiDTO(item))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (api thrAPI) upsert(w http.ResponseWriter, r *http.Request) {
	var request ThrKonfigurasiRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}
	berlaku, err := time.Parse("2006-01-02", strings.TrimSpace(request.BerlakuTanggal))
	if err != nil {
		writeErrorResponse(w, http.StatusUnprocessableEntity, apierror.CodeInvalidQuery)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	item, err := api.service.Upsert(r.Context(), principal, app.ConfigInput{
		Tahun:                request.Tahun,
		BesaranTipe:          request.BesaranTipe,
		Persen:               request.Persen,
		SyaratMasaKerjaBulan: request.SyaratMasaKerjaBulan,
		BerlakuTanggal:       berlaku,
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
	writeJSON(w, http.StatusCreated, thrKonfigurasiDTO(item))
}

func (api thrAPI) delete(w http.ResponseWriter, r *http.Request) {
	tahun, err := strconv.Atoi(r.PathValue("tahun"))
	if err != nil || tahun < 2026 {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	if err := api.service.Delete(r.Context(), principal, tahun); err != nil {
		switch {
		case errors.Is(err, domain.ErrForbidden):
			writeErrorResponse(w, http.StatusForbidden, apierror.CodeForbidden)
		case errors.Is(err, domain.ErrConfigNotFound):
			writeErrorResponse(w, http.StatusNotFound, apierror.CodeNotFound)
		default:
			writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (api thrAPI) plan(w http.ResponseWriter, r *http.Request) {
	tahun, err := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("tahun")))
	if err != nil || tahun < 2026 {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	plan, err := api.service.Plan(r.Context(), principal, tahun)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrForbidden):
			writeErrorResponse(w, http.StatusForbidden, apierror.CodeForbidden)
		case errors.Is(err, domain.ErrConfigNotFound):
			writeErrorResponse(w, http.StatusNotFound, apierror.CodeNotFound)
		default:
			writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
		}
		return
	}
	writeJSON(w, http.StatusOK, plan)
}

func (api thrAPI) generate(w http.ResponseWriter, r *http.Request) {
	var request THRGenerateRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}
	if request.Tahun < 2026 {
		writeErrorResponse(w, http.StatusUnprocessableEntity, apierror.CodeInvalidQuery)
		return
	}
	overrides := make(map[string]float64, len(request.Overrides))
	for _, item := range request.Overrides {
		if strings.TrimSpace(item.UserID) == "" || item.Nominal <= 0 {
			continue
		}
		overrides[strings.TrimSpace(item.UserID)] = item.Nominal
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	report, err := api.service.Generate(r.Context(), principal, app.GenerateInput{
		Tahun:     request.Tahun,
		Overrides: overrides,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrForbidden):
			writeErrorResponse(w, http.StatusForbidden, apierror.CodeForbidden)
		case errors.Is(err, domain.ErrConfigNotFound):
			writeErrorResponse(w, http.StatusNotFound, apierror.CodeNotFound)
		case errors.Is(err, domain.ErrSlipConflict):
			writeErrorResponse(w, http.StatusConflict, apierror.CodeConflict)
		default:
			writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
		}
		return
	}
	writeJSON(w, http.StatusOK, report)
}
