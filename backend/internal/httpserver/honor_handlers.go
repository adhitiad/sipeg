package httpserver

import (
	"net/http"
	"strings"
	"time"

	"simpeg/backend/internal/apierror"
	"simpeg/backend/internal/auth"
	honorapp "simpeg/backend/internal/honorsks/app"
	"simpeg/backend/internal/honorsks/domain"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
	"simpeg/backend/internal/timeutil"
)

// HonorSksRateDTO adalah bentuk publik tarif honor per SKS.
type HonorSksRateDTO struct {
	ID           string    `json:"id"`
	BerlakuMulai time.Time `json:"berlakuMulai"`
	PerSks       float64   `json:"perSks"`
	Aktif        bool      `json:"aktif"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// HonorSksPeriodeDTO adalah bentuk publik baris beban mengajar.
type HonorSksPeriodeDTO struct {
	ID                string    `json:"id"`
	UserId            string    `json:"userId"`
	PeriodeAkademikId string    `json:"periodeAkademikId"`
	Sks               float64   `json:"sks"`
	RatePerSks        float64   `json:"ratePerSks"`
	TotalHonor        float64   `json:"totalHonor"`
	SkId              *string   `json:"skId,omitempty"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

func honorSksRateDTO(row models.HonorSksRate) HonorSksRateDTO {
	return HonorSksRateDTO{
		ID:           row.ID,
		BerlakuMulai: timeutil.InLocation(row.BerlakuMulai),
		PerSks:       row.PerSks,
		Aktif:        row.Aktif,
		CreatedAt:    timeutil.InLocation(row.CreatedAt),
		UpdatedAt:    timeutil.InLocation(row.UpdatedAt),
	}
}

func honorSksPeriodeDTO(row models.HonorSksPeriode) HonorSksPeriodeDTO {
	return HonorSksPeriodeDTO{
		ID:                row.ID,
		UserId:            row.UserId,
		PeriodeAkademikId: row.PeriodeAkademikId,
		Sks:               row.Sks,
		RatePerSks:        row.RatePerSks,
		TotalHonor:        row.TotalHonor,
		SkId:              row.SkId,
		CreatedAt:         timeutil.InLocation(row.CreatedAt),
		UpdatedAt:         timeutil.InLocation(row.UpdatedAt),
	}
}

func writeHonorError(w http.ResponseWriter, err error) {
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

type honorSksAPI struct {
	service *honorapp.Service
}

func (api honorSksAPI) listRates(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFromContext(r.Context())
	rows, err := api.service.ListRates(r.Context(), principal)
	if err != nil {
		writeHonorError(w, err)
		return
	}
	items := make([]HonorSksRateDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, honorSksRateDTO(row))
	}
	writeJSON(w, http.StatusOK, items)
}

func (api honorSksAPI) currentRate(w http.ResponseWriter, r *http.Request) {
	asOf := strings.TrimSpace(r.URL.Query().Get("asOf"))
	if asOf == "" {
		asOf = time.Now().UTC().Format("2006-01-02")
	}
	if _, err := time.Parse("2006-01-02", asOf); err != nil {
		writeValidationError(w)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	row, err := api.service.CurrentRate(r.Context(), principal, asOf)
	if err != nil {
		writeHonorError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, honorSksRateDTO(row))
}

func (api honorSksAPI) createRate(w http.ResponseWriter, r *http.Request) {
	var request struct {
		BerlakuMulai string  `json:"berlakuMulai"`
		PerSks       float64 `json:"perSks"`
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
	row, err := api.service.CreateRate(r.Context(), principal, domain.CreateHonorRate{
		BerlakuMulai: start,
		PerSks:       request.PerSks,
	})
	if err != nil {
		writeHonorError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, honorSksRateDTO(row))
}

func (api honorSksAPI) updateRate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("rateID")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	fields, err := decodePatchFields(w, r, domain.HonorRatePatchFields)
	if err != nil {
		writeValidationError(w)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	row, err := api.service.UpdateRate(r.Context(), principal, id, domain.NewHonorRatePatch(fields))
	if err != nil {
		writeHonorError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, honorSksRateDTO(row))
}

func (api honorSksAPI) listPeriode(w http.ResponseWriter, r *http.Request) {
	query, err := parseListQuery(r, []string{"userId"}, "userId")
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
		return
	}
	periode := strings.TrimSpace(r.URL.Query().Get("periodeAkademikId"))
	options := listOptionsFromQuery(query, listing.Options{Periode: periode})

	principal, _ := auth.PrincipalFromContext(r.Context())
	page, err := api.service.ListPeriode(r.Context(), principal, options)
	if err != nil {
		writeHonorError(w, err)
		return
	}
	items := make([]HonorSksPeriodeDTO, 0, len(page.Items))
	for _, row := range page.Items {
		items = append(items, honorSksPeriodeDTO(row))
	}
	writeJSON(w, http.StatusOK, PageDTO[HonorSksPeriodeDTO]{
		Items: items, Page: page.Page, PageSize: page.PageSize,
		TotalItems: page.TotalItems, TotalPages: page.TotalPages,
	})
}

func (api honorSksAPI) createPeriode(w http.ResponseWriter, r *http.Request) {
	var request struct {
		UserId            string  `json:"userId"`
		PeriodeAkademikId string  `json:"periodeAkademikId"`
		Sks               float64 `json:"sks"`
		RatePerSks        float64 `json:"ratePerSks"`
		SkId              *string `json:"skId,omitempty"`
	}
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	row, err := api.service.CreatePeriode(r.Context(), principal, domain.CreateHonorPeriode{
		UserId:            request.UserId,
		PeriodeAkademikId: request.PeriodeAkademikId,
		Sks:               request.Sks,
		RatePerSks:        request.RatePerSks,
		SkId:              request.SkId,
	})
	if err != nil {
		writeHonorError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, honorSksPeriodeDTO(row))
}

func (api honorSksAPI) getPeriode(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("periodeID")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	row, err := api.service.GetPeriode(r.Context(), principal, id)
	if err != nil {
		writeHonorError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, honorSksPeriodeDTO(row))
}

func (api honorSksAPI) updatePeriode(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("periodeID")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	fields, err := decodePatchFields(w, r, domain.HonorPeriodePatchFields)
	if err != nil {
		writeValidationError(w)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	row, err := api.service.UpdatePeriode(r.Context(), principal, id, domain.NewHonorPeriodePatch(fields))
	if err != nil {
		writeHonorError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, honorSksPeriodeDTO(row))
}

func (api honorSksAPI) deletePeriode(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("periodeID")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	if err := api.service.DeletePeriode(r.Context(), principal, id); err != nil {
		writeHonorError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// RegisterHonorSksRoutes mendaftarkan endpoint honor mengajar per SKS.
func RegisterHonorSksRoutes(mux *http.ServeMux, resolver auth.SessionResolver, policy auth.Policy, service *honorapp.Service) {
	if service == nil {
		return
	}
	api := honorSksAPI{service: service}
	protected := func(permission auth.Permission, handler http.Handler) http.Handler {
		return auth.Authenticate(resolver)(auth.RequirePermission(policy, permission, handler))
	}
	protectedAny := func(permissions []auth.Permission, handler http.Handler) http.Handler {
		return auth.Authenticate(resolver)(auth.RequireAnyPermission(policy, permissions, handler))
	}
	mux.Handle("GET "+APIPrefix+"/payroll/honor-rate", protectedAny([]auth.Permission{"payroll:read:any", "payroll:read:homebase"}, http.HandlerFunc(api.listRates)))
	mux.Handle("POST "+APIPrefix+"/payroll/honor-rate", protected("payroll:manage", http.HandlerFunc(api.createRate)))
	mux.Handle("GET "+APIPrefix+"/payroll/honor-rate/current", protectedAny([]auth.Permission{"payroll:read:any", "payroll:read:homebase"}, http.HandlerFunc(api.currentRate)))
	mux.Handle("PATCH "+APIPrefix+"/payroll/honor-rate/{rateID}", protected("payroll:manage", http.HandlerFunc(api.updateRate)))
	mux.Handle("GET "+APIPrefix+"/payroll/honor-periode", protectedAny([]auth.Permission{"payroll:read:any", "payroll:read:homebase", "payroll:read:self"}, http.HandlerFunc(api.listPeriode)))
	mux.Handle("POST "+APIPrefix+"/payroll/honor-periode", protected("payroll:manage", http.HandlerFunc(api.createPeriode)))
	mux.Handle("GET "+APIPrefix+"/payroll/honor-periode/{periodeID}", protectedAny([]auth.Permission{"payroll:read:any", "payroll:read:homebase", "payroll:read:self"}, http.HandlerFunc(api.getPeriode)))
	mux.Handle("PATCH "+APIPrefix+"/payroll/honor-periode/{periodeID}", protected("payroll:manage", http.HandlerFunc(api.updatePeriode)))
	mux.Handle("DELETE "+APIPrefix+"/payroll/honor-periode/{periodeID}", protected("payroll:manage", http.HandlerFunc(api.deletePeriode)))
}
