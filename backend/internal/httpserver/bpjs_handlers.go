package httpserver

import (
	"net/http"
	"strings"
	"time"

	"simpeg/backend/internal/apierror"
	"simpeg/backend/internal/auth"
	bpjsapp "simpeg/backend/internal/bpjs/app"
	"simpeg/backend/internal/bpjs/domain"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
	"simpeg/backend/internal/timeutil"
)

// BpjsKonfigurasiDTO adalah bentuk publik tarif iuran BPJS.
type BpjsKonfigurasiDTO struct {
	ID                   string    `json:"id"`
	Jenis                string    `json:"jenis"`
	IuranPegawaiPct      float64   `json:"iuranPegawaiPct"`
	IuranPemberiKerjaPct float64   `json:"iuranPemberiKerjaPct"`
	Dasar                string    `json:"dasar"`
	BatasAtas            *float64  `json:"batasAtas,omitempty"`
	BerlakuMulai         time.Time `json:"berlakuMulai"`
	Aktif                bool      `json:"aktif"`
	CreatedAt            time.Time `json:"createdAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
}

// BpjsPesertaDTO adalah bentuk publik kepesertaan BPJS.
type BpjsPesertaDTO struct {
	ID               string     `json:"id"`
	UserId           string     `json:"userId"`
	Jenis            string     `json:"jenis"`
	NomorKepesertaan *string    `json:"nomorKepesertaan,omitempty"`
	Mulai            time.Time  `json:"mulai"`
	Selesai          *time.Time `json:"selesai,omitempty"`
	Aktif            bool       `json:"aktif"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

func bpjsKonfigurasiDTO(row models.BpjsKonfigurasi) BpjsKonfigurasiDTO {
	return BpjsKonfigurasiDTO{
		ID:                   row.ID,
		Jenis:                string(row.Jenis),
		IuranPegawaiPct:      row.IuranPegawaiPct,
		IuranPemberiKerjaPct: row.IuranPemberiKerjaPct,
		Dasar:                row.Dasar,
		BatasAtas:            row.BatasAtas,
		BerlakuMulai:         timeutil.InLocation(row.BerlakuMulai),
		Aktif:                row.Aktif,
		CreatedAt:            timeutil.InLocation(row.CreatedAt),
		UpdatedAt:            timeutil.InLocation(row.UpdatedAt),
	}
}

func bpjsPesertaDTO(row models.BpjsPeserta) BpjsPesertaDTO {
	return BpjsPesertaDTO{
		ID:               row.ID,
		UserId:           row.UserId,
		Jenis:            string(row.Jenis),
		NomorKepesertaan: row.NomorKepesertaan,
		Mulai:            timeutil.InLocation(row.Mulai),
		Selesai:          waktuOpsional(row.Selesai),
		Aktif:            row.Aktif,
		CreatedAt:        timeutil.InLocation(row.CreatedAt),
		UpdatedAt:        timeutil.InLocation(row.UpdatedAt),
	}
}

func waktuOpsional(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	converted := timeutil.InLocation(*value)
	return &converted
}

func writeBpjsError(w http.ResponseWriter, err error) {
	switch err {
	case domain.ErrNotFound:
		writeErrorResponse(w, http.StatusNotFound, apierror.CodeNotFound)
	case domain.ErrConflict:
		writeErrorResponse(w, http.StatusConflict, apierror.CodeConflict)
	case domain.ErrForbidden:
		writeErrorResponse(w, http.StatusForbidden, apierror.CodeForbidden)
	case domain.ErrValidation:
		writeValidationError(w)
	default:
		writeErrorResponse(w, http.StatusInternalServerError, apierror.CodeInternal)
	}
}

type bpjsAPI struct {
	service *bpjsapp.Service
}

func (api bpjsAPI) listKonfigurasi(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFromContext(r.Context())
	rows, err := api.service.ListKonfigurasi(r.Context(), principal)
	if err != nil {
		writeBpjsError(w, err)
		return
	}
	items := make([]BpjsKonfigurasiDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, bpjsKonfigurasiDTO(row))
	}
	writeJSON(w, http.StatusOK, items)
}

func (api bpjsAPI) createKonfigurasi(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Jenis                string   `json:"jenis"`
		IuranPegawaiPct      float64  `json:"iuranPegawaiPct"`
		IuranPemberiKerjaPct float64  `json:"iuranPemberiKerjaPct"`
		Dasar                string   `json:"dasar"`
		BatasAtas            *float64 `json:"batasAtas,omitempty"`
		BerlakuMulai         string   `json:"berlakuMulai"`
	}
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}
	start, err := time.Parse("2006-01-02", strings.TrimSpace(request.BerlakuMulai))
	if err != nil {
		writeValidationError(w)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	row, err := api.service.CreateKonfigurasi(r.Context(), principal, domain.CreateKonfigurasi{
		Jenis:                strings.TrimSpace(request.Jenis),
		IuranPegawaiPct:      request.IuranPegawaiPct,
		IuranPemberiKerjaPct: request.IuranPemberiKerjaPct,
		Dasar:                strings.TrimSpace(request.Dasar),
		BatasAtas:            request.BatasAtas,
		BerlakuMulai:         start,
	})
	if err != nil {
		writeBpjsError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, bpjsKonfigurasiDTO(row))
}

func (api bpjsAPI) updateKonfigurasi(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("bpjsID")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	fields, err := decodePatchFields(w, r, domain.KonfigurasiPatchFields)
	if err != nil {
		writeValidationError(w)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	row, err := api.service.UpdateKonfigurasi(r.Context(), principal, id, domain.NewKonfigurasiPatch(fields))
	if err != nil {
		writeBpjsError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, bpjsKonfigurasiDTO(row))
}

func (api bpjsAPI) listPeserta(w http.ResponseWriter, r *http.Request) {
	query, err := parseListQuery(r, []string{"userId", "jenis", "mulai"}, "userId")
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
		return
	}
	options := listOptionsFromQuery(query, listing.Options{})
	principal, _ := auth.PrincipalFromContext(r.Context())
	page, err := api.service.ListPeserta(r.Context(), principal, options)
	if err != nil {
		writeBpjsError(w, err)
		return
	}
	items := make([]BpjsPesertaDTO, 0, len(page.Items))
	for _, row := range page.Items {
		items = append(items, bpjsPesertaDTO(row))
	}
	writeJSON(w, http.StatusOK, PageDTO[BpjsPesertaDTO]{
		Items: items, Page: page.Page, PageSize: page.PageSize,
		TotalItems: page.TotalItems, TotalPages: page.TotalPages,
	})
}

func (api bpjsAPI) createPeserta(w http.ResponseWriter, r *http.Request) {
	var request struct {
		UserId           string  `json:"userId"`
		Jenis            string  `json:"jenis"`
		NomorKepesertaan *string `json:"nomorKepesertaan,omitempty"`
		Mulai            string  `json:"mulai"`
		Selesai          *string `json:"selesai,omitempty"`
	}
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}
	start, err := time.Parse("2006-01-02", strings.TrimSpace(request.Mulai))
	if err != nil {
		writeValidationError(w)
		return
	}
	var end *time.Time
	if request.Selesai != nil && strings.TrimSpace(*request.Selesai) != "" {
		parsed, err := time.Parse("2006-01-02", strings.TrimSpace(*request.Selesai))
		if err != nil {
			writeValidationError(w)
			return
		}
		end = &parsed
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	row, err := api.service.CreatePeserta(r.Context(), principal, domain.CreatePeserta{
		UserId:           request.UserId,
		Jenis:            strings.TrimSpace(request.Jenis),
		NomorKepesertaan: request.NomorKepesertaan,
		Mulai:            start,
		Selesai:          end,
	})
	if err != nil {
		writeBpjsError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, bpjsPesertaDTO(row))
}

func (api bpjsAPI) getPeserta(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("pesertaID")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	row, err := api.service.GetPeserta(r.Context(), principal, id)
	if err != nil {
		writeBpjsError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, bpjsPesertaDTO(row))
}

func (api bpjsAPI) updatePeserta(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("pesertaID")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	fields, err := decodePatchFields(w, r, domain.PesertaPatchFields)
	if err != nil {
		writeValidationError(w)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	row, err := api.service.UpdatePeserta(r.Context(), principal, id, domain.NewPesertaPatch(fields))
	if err != nil {
		writeBpjsError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, bpjsPesertaDTO(row))
}

// RegisterBpjsRoutes mendaftarkan endpoint BPJS Kesehatan/Ketenagakerjaan.
func RegisterBpjsRoutes(mux *http.ServeMux, resolver auth.SessionResolver, policy auth.Policy, service *bpjsapp.Service) {
	if service == nil {
		return
	}
	api := bpjsAPI{service: service}
	protected := func(permission auth.Permission, handler http.Handler) http.Handler {
		return auth.Authenticate(resolver)(auth.RequirePermission(policy, permission, handler))
	}
	protectedAny := func(permissions []auth.Permission, handler http.Handler) http.Handler {
		return auth.Authenticate(resolver)(auth.RequireAnyPermission(policy, permissions, handler))
	}
	mux.Handle("GET "+APIPrefix+"/payroll/bpjs/config", protectedAny([]auth.Permission{"payroll:read:any", "payroll:read:homebase"}, http.HandlerFunc(api.listKonfigurasi)))
	mux.Handle("POST "+APIPrefix+"/payroll/bpjs/config", protected("payroll:manage", http.HandlerFunc(api.createKonfigurasi)))
	mux.Handle("PATCH "+APIPrefix+"/payroll/bpjs/config/{bpjsID}", protected("payroll:manage", http.HandlerFunc(api.updateKonfigurasi)))
	mux.Handle("GET "+APIPrefix+"/payroll/bpjs/peserta", protectedAny([]auth.Permission{"payroll:read:any", "payroll:read:homebase", "payroll:read:self"}, http.HandlerFunc(api.listPeserta)))
	mux.Handle("POST "+APIPrefix+"/payroll/bpjs/peserta", protected("payroll:manage", http.HandlerFunc(api.createPeserta)))
	mux.Handle("GET "+APIPrefix+"/payroll/bpjs/peserta/{pesertaID}", protectedAny([]auth.Permission{"payroll:read:any", "payroll:read:homebase", "payroll:read:self"}, http.HandlerFunc(api.getPeserta)))
	mux.Handle("PATCH "+APIPrefix+"/payroll/bpjs/peserta/{pesertaID}", protected("payroll:manage", http.HandlerFunc(api.updatePeserta)))
}
