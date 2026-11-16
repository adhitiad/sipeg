package httpserver

import (
	"net/http"
	"strings"
	"time"

	"simpeg/backend/internal/apierror"
	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/models"
	pph21clapp "simpeg/backend/internal/pph21ter/app"
	"simpeg/backend/internal/pph21ter/domain"
	"simpeg/backend/internal/timeutil"
)

// Pph21TerDTO adalah bentuk publik baris tarif efektif PPh 21 TER.
type Pph21TerDTO struct {
	ID           string    `json:"id"`
	Kategori     string    `json:"kategori"`
	Lapisan      int       `json:"lapisan"`
	BatasBawah   float64   `json:"batasBawah"`
	BatasAtas    *float64  `json:"batasAtas,omitempty"`
	TarifPct     float64   `json:"tarifPct"`
	BerlakuMulai time.Time `json:"berlakuMulai"`
}

func pph21TerDTO(row models.Pph21TerTarif) Pph21TerDTO {
	return Pph21TerDTO{
		ID:           row.ID,
		Kategori:     string(row.Kategori),
		Lapisan:      row.Lapisan,
		BatasBawah:   row.BatasBawah,
		BatasAtas:    row.BatasAtas,
		TarifPct:     row.TarifPct,
		BerlakuMulai: timeutil.InLocation(row.BerlakuMulai),
	}
}

func writeTerError(w http.ResponseWriter, err error) {
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

type pph21TerAPI struct {
	service *pph21clapp.Service
}

func (api pph21TerAPI) listTer(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFromContext(r.Context())
	asOf := strings.TrimSpace(r.URL.Query().Get("asOf"))
	var rows []models.Pph21TerTarif
	var err error
	if asOf == "" {
		rows, err = api.service.ListTer(r.Context(), principal)
	} else {
		if _, parseErr := time.Parse("2006-01-02", asOf); parseErr != nil {
			writeValidationError(w)
			return
		}
		rows, err = api.service.ListTerOn(r.Context(), principal, asOf)
	}
	if err != nil {
		writeTerError(w, err)
		return
	}
	items := make([]Pph21TerDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, pph21TerDTO(row))
	}
	writeJSON(w, http.StatusOK, items)
}

func (api pph21TerAPI) createTer(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Kategori     string   `json:"kategori"`
		Lapisan      int      `json:"lapisan"`
		BatasBawah   float64  `json:"batasBawah"`
		BatasAtas    *float64 `json:"batasAtas,omitempty"`
		TarifPct     float64  `json:"tarifPct"`
		BerlakuMulai string   `json:"berlakuMulai"`
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
	row, err := api.service.CreateTer(r.Context(), principal, domain.CreateTer{
		Kategori:     strings.TrimSpace(request.Kategori),
		Lapisan:      request.Lapisan,
		BatasBawah:   request.BatasBawah,
		BatasAtas:    request.BatasAtas,
		TarifPct:     request.TarifPct,
		BerlakuMulai: start,
	})
	if err != nil {
		writeTerError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, pph21TerDTO(row))
}

func (api pph21TerAPI) updateTer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("terID")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	fields, err := decodePatchFields(w, r, domain.TerPatchFields)
	if err != nil {
		writeValidationError(w)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	row, err := api.service.UpdateTer(r.Context(), principal, id, domain.NewTerPatch(fields))
	if err != nil {
		writeTerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, pph21TerDTO(row))
}

func (api pph21TerAPI) deleteTer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("terID")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	if err := api.service.DeleteTer(r.Context(), principal, id); err != nil {
		writeTerError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// RegisterPph21TerRoutes mendaftarkan endpoint PPh 21 TER.
func RegisterPph21TerRoutes(mux *http.ServeMux, resolver auth.SessionResolver, policy auth.Policy, service *pph21clapp.Service) {
	if service == nil {
		return
	}
	api := pph21TerAPI{service: service}
	protected := func(permission auth.Permission, handler http.Handler) http.Handler {
		return auth.Authenticate(resolver)(auth.RequirePermission(policy, permission, handler))
	}
	protectedAny := func(permissions []auth.Permission, handler http.Handler) http.Handler {
		return auth.Authenticate(resolver)(auth.RequireAnyPermission(policy, permissions, handler))
	}
	mux.Handle("GET "+APIPrefix+"/payroll/pph21-ter", protectedAny([]auth.Permission{"payroll:read:any", "payroll:read:homebase"}, http.HandlerFunc(api.listTer)))
	mux.Handle("POST "+APIPrefix+"/payroll/pph21-ter", protected("payroll:manage", http.HandlerFunc(api.createTer)))
	mux.Handle("PATCH "+APIPrefix+"/payroll/pph21-ter/{terID}", protected("payroll:manage", http.HandlerFunc(api.updateTer)))
	mux.Handle("DELETE "+APIPrefix+"/payroll/pph21-ter/{terID}", protected("payroll:manage", http.HandlerFunc(api.deleteTer)))
}
