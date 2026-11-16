package httpserver

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"simpeg/backend/internal/apierror"
	"simpeg/backend/internal/audit"
	"simpeg/backend/internal/auth"
	calcasidomain "simpeg/backend/internal/calculasi/domain"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
	"simpeg/backend/internal/money"
	"simpeg/backend/internal/payroll/app"
	payrolldomain "simpeg/backend/internal/payroll/domain"
	"simpeg/backend/internal/pdf"
	"simpeg/backend/internal/timeutil"
)

// payrollAPI memegang use case penggajian.
type payrollAPI struct {
	payroll       *app.Service
	generate      *app.GenerateSlipService
	orgName       string
	verifyBaseURL string
	// recorder mencatat pengunduhan PDF slip ke jejak audit (best-effort); boleh nil.
	recorder AuditRecorder
}

// TunjanganItemDTO adalah satu item tunjangan dalam rincian slip gaji.
type TunjanganItemDTO struct {
	Jenis string `json:"jenis"`
	Nilai string `json:"nilai"`
}

// PotonganItemDTO adalah satu item potongan dalam rincian slip gaji.
type PotonganItemDTO struct {
	Jenis   string `json:"jenis"`
	Nominal string `json:"nominal"`
}

// PayrollDTO adalah bentuk publik slip gaji.
//
// Seluruh nominal dinyatakan dalam Rupiah Indonesia dan sudah diformat
// dengan pemisah ribuan versi Indonesia. Pendapatan dan potongan disajikan
// sebagai array rincian per komponen.
type PayrollDTO struct {
	ID             string             `json:"id"`
	NomorSlip      string             `json:"nomorSlip"`
	UserID         string             `json:"userId"`
	Nama           string             `json:"nama"`
	Periode        string             `json:"periode"`
	PeriodeLabel   string             `json:"periodeLabel"`
	Status         string             `json:"status"`
	StatusLabel    string             `json:"statusLabel"`
	TanggalGaji    string             `json:"tanggalGaji"`
	GajiPokok      string             `json:"gajiPokok"`
	Pendapatan     []TunjanganItemDTO `json:"pendapatan"`
	Potongan       []PotonganItemDTO  `json:"potongan"`
	TotalGaji      string             `json:"totalGaji"`
	TanggalDibayar *string            `json:"tanggalDibayar"`
}

// generateSlipRequest adalah bentuk permintaan pembuatan slip otomatis.
// KategoriStatusID dan Tingkat diisi admin untuk menentukan gaji pokok dari
// skala efektif; keduanya opsional bila pegawai tidak tercakup skala.
type generateSlipRequest struct {
	UserID           string `json:"userId"`
	Periode          string `json:"periode"`
	KategoriStatusID string `json:"kategoriStatusId"`
	Tingkat          *int   `json:"tingkat"`
}

func (api payrollAPI) generateSlip(w http.ResponseWriter, r *http.Request) {
	if api.generate == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
		return
	}

	var request generateSlipRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}
	userID := strings.TrimSpace(request.UserID)
	if !validUUID(userID) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	if _, err := payrolldomain.ParsePeriod(strings.TrimSpace(request.Periode)); err != nil {
		writeValidationError(w)
		return
	}
	kategoriStatusID := strings.TrimSpace(request.KategoriStatusID)
	if kategoriStatusID != "" && !validUUID(kategoriStatusID) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	tingkat := 0
	if request.Tingkat != nil {
		if *request.Tingkat < 1 {
			writeValidationError(w)
			return
		}
		tingkat = *request.Tingkat
	}
	if kategoriStatusID != "" && tingkat == 0 {
		writeValidationError(w)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	slip, err := api.generate.GenerateSlip(r.Context(), principal, app.GenerateSlipInput{
		UserID:           userID,
		Periode:          strings.TrimSpace(request.Periode),
		KategoriStatusID: kategoriStatusID,
		Tingkat:          tingkat,
	})
	if err != nil {
		writePayrollError(w, err)
		return
	}
	// Muat ulang slip beserta asosiasi (pegawai, tunjangan, potongan) agar
	// DTO respons lengkap; slip hasil upsert tidak memuat relasi.
	if full, getErr := api.payroll.Get(r.Context(), principal, slip.ID); getErr == nil {
		slip = full
	}
	writeJSON(w, http.StatusOK, payrollDTO(slip))
}

func (api payrollAPI) slipPDF(w http.ResponseWriter, r *http.Request) {
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
	if slip.Status == models.StatusSlipBatal {
		writeErrorResponse(w, http.StatusConflict, apierror.CodeConflict)
		return
	}

	period := timeutil.InLocation(slip.Periode)
	parsed, parseErr := payrolldomain.ParsePeriod(period.Format("2006-01"))
	periodeLabel := ""
	if parseErr == nil {
		periodeLabel = parsed.Label()
	}
	nama := ""
	if slip.Pegawai != nil {
		nama = slip.Pegawai.Nama
	}

	pendapatan := make([]pdf.SlipRow, 0, len(slip.GajiTunjangans))
	for _, item := range slip.GajiTunjangans {
		pendapatan = append(pendapatan, pdf.SlipRow{Jenis: item.Jenis, Nominal: money.Format(item.Nominal)})
	}
	potongan := make([]pdf.SlipRow, 0, len(slip.GajiPotongans))
	totalPotongan := 0.0
	for _, item := range slip.GajiPotongans {
		potongan = append(potongan, pdf.SlipRow{Jenis: item.Jenis, Nominal: money.Format(item.Nominal)})
		totalPotongan += item.Nominal
	}
	gajiBersih := slip.TotalGaji - totalPotongan

	kodeVerifikasi := ""
	verifyURL := ""
	if slip.KodeVerifikasi != nil {
		kodeVerifikasi = *slip.KodeVerifikasi
		if base := strings.TrimRight(api.verifyBaseURL, "/"); base != "" {
			verifyURL = base + "/verify/slip/" + kodeVerifikasi
		}
	}

	document, err := pdf.RenderSlip(pdf.SlipData{
		OrgName:        api.orgName,
		NomorSlip:      slip.NomorSlip,
		Nama:           nama,
		PeriodeLabel:   periodeLabel,
		TanggalGaji:    timeutil.FormatDate(slip.TanggalGaji),
		StatusLabel:    payrolldomain.StatusLabel(slip.Status),
		GajiPokok:      money.Format(slip.GajiPokok),
		Pendapatan:     pendapatan,
		Potongan:       potongan,
		TotalGaji:      money.Format(slip.TotalGaji),
		TotalPotongan:  money.Format(totalPotongan),
		GajiBersih:     money.Format(gajiBersih),
		KodeVerifikasi: kodeVerifikasi,
		VerifyURL:      verifyURL,
	})
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, apierror.CodeInternal)
		return
	}

	// Jejak audit pengunduhan slip (forensik/observabilitas) — best-effort;
	// bukan pembatas jumlah unduhan. ActorID = pengunduh; UnitID = unit
	// pemilik slip.
	if api.recorder != nil {
		targetID := slip.ID
		var unitID *string
		if slip.Pegawai != nil {
			unitID = slip.Pegawai.HomebaseId
		}
		_ = api.recorder.Record(r.Context(), audit.Entry{
			ActorID:  principal.UserID,
			Action:   audit.ActionSlipPDFDownloaded,
			TargetID: &targetID,
			UnitID:   unitID,
		})
	}

	filename := strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, "slip-"+slip.NomorSlip+".pdf")
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", filename))
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(document)
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

func (api payrollAPI) terbitPayroll(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("payrollId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	slip, err := api.payroll.Publish(r.Context(), principal, id)
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
//
// Tunjangan diambil dari slip.GajiTunjangans dan potongan dari
// slip.GajiPotongans. Seluruh nominal diformat dengan money.Format agar
// mengikuti konvensi Rupiah Indonesia.
func payrollDTO(slip models.Gaji) PayrollDTO {
	period := timeutil.InLocation(slip.Periode)
	parsed, _ := payrolldomain.ParsePeriod(period.Format("2006-01"))

	pendapatan := make([]TunjanganItemDTO, 0, len(slip.GajiTunjangans))
	for _, t := range slip.GajiTunjangans {
		pendapatan = append(pendapatan, TunjanganItemDTO{
			Jenis: t.Jenis,
			Nilai: money.Format(t.Nominal),
		})
	}

	potongan := make([]PotonganItemDTO, 0, len(slip.GajiPotongans))
	for _, p := range slip.GajiPotongans {
		potongan = append(potongan, PotonganItemDTO{
			Jenis:   p.Jenis,
			Nominal: money.Format(p.Nominal),
		})
	}

	dto := PayrollDTO{
		ID:           slip.ID,
		NomorSlip:    slip.NomorSlip,
		UserID:       slip.UserId,
		Periode:      parsed.Key(),
		PeriodeLabel: parsed.Label(),
		Status:       string(slip.Status),
		StatusLabel:  payrolldomain.StatusLabel(slip.Status),
		TanggalGaji:  timeutil.FormatDate(slip.TanggalGaji),
		GajiPokok:    money.Format(slip.GajiPokok),
		Pendapatan:   pendapatan,
		Potongan:     potongan,
		TotalGaji:    money.Format(slip.TotalGaji),
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
	case errors.Is(err, payrolldomain.ErrPegawaiTidakAktif):
		writeErrorResponse(w, http.StatusConflict, apierror.CodeConflict)
	case errors.Is(err, payrolldomain.ErrForbidden):
		writeErrorResponse(w, http.StatusForbidden, apierror.CodeForbidden)
	case errors.Is(err, payrolldomain.ErrInvalidPeriod), errors.Is(err, payrolldomain.ErrValidation),
		errors.Is(err, calcasidomain.ErrValidation):
		writeValidationError(w)
	case errors.Is(err, calcasidomain.ErrNotFound):
		// Data referensi (skala/tarif) tidak tersedia untuk periode yang
		// diminta; klien perlu diarahkan ke layar konfigurasi.
		writeErrorResponse(w, http.StatusUnprocessableEntity, apierror.CodeValidation)
	default:
		writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
	}
}

// PayrollBranding memuat identitas instansi dan basis URL verifikasi untuk
// slip PDF.
type PayrollBranding struct {
	// Organization adalah nama instansi pada kop slip (env PAYROLL_SLIP_ORG_NAME).
	Organization string
	// VerifyBaseURL adalah origin aplikasi untuk tautan verifikasi publik
	// (env APP_BASE_URL). Kosong berarti slip tidak memuat QR.
	VerifyBaseURL string
}

// RegisterPayrollRoutes mendaftarkan endpoint penggajian.
// recorder mencatat pengunduhan PDF slip ke jejak audit (best-effort); boleh nil.
func RegisterPayrollRoutes(mux *http.ServeMux, resolver auth.SessionResolver, policy auth.Policy, payroll *app.Service, payrollGenerate *app.GenerateSlipService, branding PayrollBranding, recorder AuditRecorder) {
	api := payrollAPI{payroll: payroll, generate: payrollGenerate, orgName: branding.Organization, verifyBaseURL: branding.VerifyBaseURL, recorder: recorder}
	schedules := payrollScheduleAPI{payroll: payroll}
	protected := func(permission auth.Permission, handler http.Handler) http.Handler {
		return auth.Authenticate(resolver)(auth.RequirePermission(policy, permission, handler))
	}
	protectedAny := func(permissions []auth.Permission, handler http.Handler) http.Handler {
		return auth.Authenticate(resolver)(auth.RequireAnyPermission(policy, permissions, handler))
	}

	mux.Handle("GET "+APIPrefix+"/payroll/slips", protectedAny([]auth.Permission{"payroll:read:any", "payroll:read:homebase", "payroll:read:self"}, http.HandlerFunc(api.listPayroll)))
	mux.Handle("POST "+APIPrefix+"/payroll/slips/generate", protected("payroll:manage", http.HandlerFunc(api.generateSlip)))
	mux.Handle("GET "+APIPrefix+"/payroll/slips/{payrollId}/pdf", protectedAny([]auth.Permission{"payroll:read:any", "payroll:read:homebase", "payroll:read:self"}, http.HandlerFunc(api.slipPDF)))
	mux.Handle("GET "+APIPrefix+"/payroll/slips/{payrollId}", protectedAny([]auth.Permission{"payroll:read:any", "payroll:read:homebase", "payroll:read:self"}, http.HandlerFunc(api.getPayroll)))
	mux.Handle("POST "+APIPrefix+"/payroll/slips/{payrollId}/terbit", protected("payroll:manage", http.HandlerFunc(api.terbitPayroll)))
	mux.Handle("POST "+APIPrefix+"/payroll/slips/{payrollId}/payment", protected("payroll:manage", http.HandlerFunc(api.confirmPayment)))
	mux.Handle("GET "+APIPrefix+"/payroll/schedules", protectedAny([]auth.Permission{"payroll:schedule", "payroll:schedule:homebase"}, http.HandlerFunc(schedules.listPayrollSchedules)))
	mux.Handle("PUT "+APIPrefix+"/payroll/schedules/{unitId}", protectedAny([]auth.Permission{"payroll:schedule", "payroll:schedule:homebase"}, http.HandlerFunc(schedules.upsertPayrollSchedule)))
}
