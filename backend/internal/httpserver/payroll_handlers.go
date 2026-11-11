package httpserver

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"simpeg/backend/internal/apierror"
	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
	"simpeg/backend/internal/money"
	"simpeg/backend/internal/payroll/app"
	payrolldomain "simpeg/backend/internal/payroll/domain"
	"simpeg/backend/internal/timeutil"
)

// payrollAPI memegang use case penggajian.
type payrollAPI struct {
	payroll *app.Service
}

// PayrollDTO adalah bentuk publik slip gaji.
//
// Seluruh nominal dinyatakan dalam Rupiah Indonesia dan sudah diformat
// dengan pemisah ribuan versi Indonesia.
type PayrollDTO struct {
	ID             string  `json:"id"`
	NomorSlip      string  `json:"nomorSlip"`
	UserID         string  `json:"userId"`
	Nama           string  `json:"nama"`
	Periode        string  `json:"periode"`
	PeriodeLabel   string  `json:"periodeLabel"`
	Status         string  `json:"status"`
	StatusLabel    string  `json:"statusLabel"`
	TanggalGaji    string  `json:"tanggalGaji"`
	GajiPokok      string  `json:"gajiPokok"`
	TotalGaji      string  `json:"totalGaji"`
	PotonganPph21  string  `json:"potonganPph21"`
	TanggalDibayar *string `json:"tanggalDibayar"`
}

// paymentRequest adalah bentuk permintaan konfirmasi pembayaran.
//
// Nominal tidak pernah menjadi bagian dari permintaan ini; nominal tetap pada
// slip gaji dan hanya ditampilkan pada halaman terautentikasi.
type paymentRequest struct {
	ReferensiTransaksi *string `json:"referensiTransaksi"`
	PelakuPengesahan   string  `json:"pelakuPengesahan"`
	TanggalBayar       string  `json:"tanggalBayar"`
	Keterangan         *string `json:"keterangan"`
}

func (api payrollAPI) listPayroll(w http.ResponseWriter, r *http.Request) {
	query, err := parseListQuery(r, payrolldomain.SlipSortFields(), "periode")
	period := strings.TrimSpace(r.URL.Query().Get("periode"))
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if err != nil || textLength(period) > listing.MaxSearchLength || textLength(status) > listing.MaxSearchLength {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
		return
	}
	if period != "" {
		if _, err := payrolldomain.ParsePeriod(period); err != nil {
			writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
			return
		}
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	options := listOptionsFromQuery(query, listing.Options{Periode: period, Status: status})

	page, err := api.payroll.List(r.Context(), principal, options)
	if err != nil {
		writePayrollError(w, err)
		return
	}

	items := make([]PayrollDTO, 0, len(page.Items))
	for _, slip := range page.Items {
		items = append(items, payrollDTO(slip))
	}
	writeJSON(w, http.StatusOK, PageDTO[PayrollDTO]{
		Items: items, Page: page.Page, PageSize: page.PageSize,
		TotalItems: page.TotalItems, TotalPages: page.TotalPages,
	})
}

func (api payrollAPI) getPayroll(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("payrollId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	slip, err := api.payroll.Get(r.Context(), principal, id)
	if err != nil {
		writePayrollError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, payrollDTO(slip))
}

func (api payrollAPI) confirmPayment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("payrollId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}

	var request paymentRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}
	tanggalBayar, err := time.Parse("2006-01-02", strings.TrimSpace(request.TanggalBayar))
	if err != nil {
		writeValidationError(w)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	slip, err := api.payroll.RecordPayment(r.Context(), principal, id, payrolldomain.PaymentInput{
		ReferensiTransaksi: request.ReferensiTransaksi,
		PelakuPengesahan:   strings.TrimSpace(request.PelakuPengesahan),
		TanggalBayar:       timeutil.DateOnly(tanggalBayar),
		Keterangan:         request.Keterangan,
	})
	if err != nil {
		writePayrollError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, payrollDTO(slip))
}

// payrollDTO memetakan slip gaji ke bentuk publik.
func payrollDTO(slip models.Gaji) PayrollDTO {
	period := timeutil.InLocation(slip.Periode)
	parsed, _ := payrolldomain.ParsePeriod(period.Format("2006-01"))

	dto := PayrollDTO{
		ID:            slip.ID,
		NomorSlip:     slip.NomorSlip,
		UserID:        slip.UserId,
		Periode:       parsed.Key(),
		PeriodeLabel:  parsed.Label(),
		Status:        string(slip.Status),
		StatusLabel:   payrolldomain.StatusLabel(slip.Status),
		TanggalGaji:   timeutil.FormatDate(slip.TanggalGaji),
		GajiPokok:     money.Format(slip.GajiPokok),
		TotalGaji:     money.Format(slip.TotalGaji),
		PotonganPph21: money.Format(slip.PotonganPph21),
	}
	if slip.Pegawai != nil {
		dto.Nama = slip.Pegawai.Nama
	}
	if slip.TanggalDibayar != nil {
		paid := timeutil.FormatDate(*slip.TanggalDibayar)
		dto.TanggalDibayar = &paid
	}
	return dto
}

// writePayrollError memetakan error lapisan penggajian menjadi bentuk error
// HTTP yang seragam.
func writePayrollError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, payrolldomain.ErrNotFound):
		writeErrorResponse(w, http.StatusNotFound, apierror.CodeNotFound)
	case errors.Is(err, payrolldomain.ErrConflict):
		writeErrorResponse(w, http.StatusConflict, apierror.CodeConflict)
	case errors.Is(err, payrolldomain.ErrForbidden):
		writeErrorResponse(w, http.StatusForbidden, apierror.CodeForbidden)
	case errors.Is(err, payrolldomain.ErrInvalidPeriod), errors.Is(err, payrolldomain.ErrValidation):
		writeValidationError(w)
	default:
		writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
	}
}

// RegisterPayrollRoutes mendaftarkan endpoint penggajian.
func RegisterPayrollRoutes(mux *http.ServeMux, resolver auth.SessionResolver, policy auth.Policy, payroll *app.Service) {
	api := payrollAPI{payroll: payroll}
	schedules := payrollScheduleAPI{payroll: payroll}
	protected := func(permission auth.Permission, handler http.Handler) http.Handler {
		return auth.Authenticate(resolver)(auth.RequirePermission(policy, permission, handler))
	}
	protectedAny := func(permissions []auth.Permission, handler http.Handler) http.Handler {
		return auth.Authenticate(resolver)(auth.RequireAnyPermission(policy, permissions, handler))
	}

	mux.Handle("GET "+APIPrefix+"/payroll/slips", protectedAny([]auth.Permission{"payroll:read:any", "payroll:read:homebase", "payroll:read:self"}, http.HandlerFunc(api.listPayroll)))
	mux.Handle("GET "+APIPrefix+"/payroll/slips/{payrollId}", protectedAny([]auth.Permission{"payroll:read:any", "payroll:read:homebase", "payroll:read:self"}, http.HandlerFunc(api.getPayroll)))
	mux.Handle("POST "+APIPrefix+"/payroll/slips/{payrollId}/payment", protected("payroll:manage", http.HandlerFunc(api.confirmPayment)))
	mux.Handle("GET "+APIPrefix+"/payroll/schedules", protectedAny([]auth.Permission{"payroll:schedule", "payroll:schedule:homebase"}, http.HandlerFunc(schedules.listPayrollSchedules)))
	mux.Handle("PUT "+APIPrefix+"/payroll/schedules/{unitId}", protectedAny([]auth.Permission{"payroll:schedule", "payroll:schedule:homebase"}, http.HandlerFunc(schedules.upsertPayrollSchedule)))
}
