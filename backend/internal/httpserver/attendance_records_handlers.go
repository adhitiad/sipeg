package httpserver

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"simpeg/backend/internal/apierror"
	"simpeg/backend/internal/attendance"
	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/crypto"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
	"simpeg/backend/internal/timeutil"
)

type attendanceReadAPI struct {
	service *attendance.ReadService
}

type attendanceRecordDTO struct {
	ID     string                `json:"id"`
	Date   string                `json:"date"`
	Status models.StatusPresensi `json:"status"`
	Note   *string               `json:"note,omitempty"`
	Source *string               `json:"source,omitempty"`
}

type attendanceRecordsResponse struct {
	Items      []attendanceRecordDTO `json:"items"`
	Page       int                   `json:"page"`
	PageSize   int                   `json:"pageSize"`
	TotalItems int64                 `json:"totalItems"`
	TotalPages int                   `json:"totalPages"`
}

type unitDayRecordDTO struct {
	ID       string                 `json:"id"`
	Nama     string                 `json:"nama"`
	Nip      *string                `json:"nip"`
	UnitID   *string                `json:"unitId"`
	UnitName *string                `json:"unitName"`
	Status   *models.StatusPresensi `json:"status"`
	Note     *string                `json:"note"`
}

func RegisterAttendanceReadRoutes(mux *http.ServeMux, resolver auth.SessionResolver, policy auth.Policy, service *attendance.ReadService) {
	if service == nil {
		return
	}
	api := attendanceReadAPI{service: service}
	mux.Handle("GET "+APIPrefix+"/attendance/my-records", auth.Authenticate(resolver)(auth.RequirePermission(policy, "attendance:read:self", http.HandlerFunc(api.myRecords))))
	mux.Handle("GET "+APIPrefix+"/attendance/unit-summary", auth.Authenticate(resolver)(auth.RequireAnyPermission(policy, []auth.Permission{"attendance:read:any", "attendance:read:homebase"}, http.HandlerFunc(api.unitSummary))))
	mux.Handle("GET "+APIPrefix+"/attendance/unit-records", auth.Authenticate(resolver)(auth.RequireAnyPermission(policy, []auth.Permission{"attendance:read:any", "attendance:read:homebase"}, http.HandlerFunc(api.unitRecords))))
}

func (api attendanceReadAPI) myRecords(w http.ResponseWriter, r *http.Request) {
	filter, err := attendanceRecordFilter(r)
	if err != nil {
		writeAttendanceFilterError(w, err)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	page, err := api.service.MyRecords(r.Context(), principal, filter)
	if err != nil {
		writeAttendanceReadError(w, err)
		return
	}
	items := make([]attendanceRecordDTO, 0, len(page.Items))
	for _, record := range page.Items {
		items = append(items, attendanceRecordDTO{
			ID: record.ID, Date: timeutil.InLocation(record.Tanggal).Format("2006-01-02"),
			Status: record.Status, Note: record.Keterangan, Source: record.Sumber,
		})
	}
	writeJSON(w, http.StatusOK, attendanceRecordsResponse{
		Items: items, Page: page.Page, PageSize: page.PageSize,
		TotalItems: page.TotalItems, TotalPages: page.TotalPages,
	})
}

func (api attendanceReadAPI) unitRecords(w http.ResponseWriter, r *http.Request) {
	date, err := time.Parse("2006-01-02", r.URL.Query().Get("date"))
	if err != nil {
		writeValidationError(w)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	rows, err := api.service.UnitDay(r.Context(), principal, timeutil.DateOnly(date))
	if err != nil {
		writeAttendanceReadError(w, err)
		return
	}
	items := make([]unitDayRecordDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, unitDayRecordDTO{
			ID: row.UserID, Nama: row.Nama, Nip: crypto.PtrString(row.Nip),
			UnitID: row.UnitID, UnitName: row.UnitName,
			Status: row.Status, Note: row.Note,
		})
	}
	writeJSON(w, http.StatusOK, items)
}

func (api attendanceReadAPI) unitSummary(w http.ResponseWriter, r *http.Request) {
	filter, err := attendanceRecordFilter(r)
	if err != nil {
		writeAttendanceFilterError(w, err)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	summary, err := api.service.UnitSummary(r.Context(), principal, filter)
	if err != nil {
		writeAttendanceReadError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

// attendanceRecordFilter menyusun parameter daftar rekam presensi. Sort hanya
// diterima bila berada dalam allowlist attendance.RekamSortFields; kolom SQL
// ditentukan di sisi repository. Urutan bawaan tetap tanggal terbaru lebih
// dulu supaya klien lama tidak berubah perilakunya.
func attendanceRecordFilter(r *http.Request) (attendance.RecordFilter, error) {
	from, err := time.Parse("2006-01-02", r.URL.Query().Get("from"))
	if err != nil {
		return attendance.RecordFilter{}, err
	}
	to, err := time.Parse("2006-01-02", r.URL.Query().Get("to"))
	if err != nil {
		return attendance.RecordFilter{}, err
	}
	from = timeutil.DateOnly(from)
	to = timeutil.DateOnly(to)
	if to.Before(from) || to.Sub(from) > 365*24*time.Hour {
		return attendance.RecordFilter{}, errors.New("attendance date range is invalid")
	}
	query, err := parseListQuery(r, listing.SortKeys(attendance.RekamSortFields), "tanggal")
	if err != nil {
		return attendance.RecordFilter{}, err
	}
	order := query.Order
	if strings.TrimSpace(r.URL.Query().Get("order")) == "" {
		order = "desc"
	}
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status != "" && !validAttendanceStatus(status) {
		return attendance.RecordFilter{}, errInvalidAttendanceStatus
	}
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	if len(search) > listing.MaxSearchLength {
		return attendance.RecordFilter{}, errInvalidAttendanceSearch
	}
	return attendance.RecordFilter{
		From:     from,
		Until:    to.AddDate(0, 0, 1),
		Page:     query.Page,
		PageSize: query.PageSize,
		Sort:     query.Sort,
		Order:    order,
		Status:   status,
		Search:   search,
	}, nil
}

// validAttendanceStatus memeriksa status presensi yang dikenal backend.
func validAttendanceStatus(value string) bool {
	switch models.StatusPresensi(value) {
	case models.StatusPresensiHadir, models.StatusPresensiIzin, models.StatusPresensiSakit,
		models.StatusPresensiAlpa, models.StatusPresensiCuti, models.StatusPresensiLibur:
		return true
	default:
		return false
	}
}

// errInvalidAttendanceStatus dan errInvalidAttendanceSearch menandai filter
// yang tidak dikenal.
var (
	errInvalidAttendanceStatus = &listParamError{param: "status"}
	errInvalidAttendanceSearch = &listParamError{param: "search"}
)

// writeAttendanceFilterError memisahkan parameter daftar yang tidak dikenal
// (400 invalid_query) dari kesalahan nilai lain (422 validation_error).
func writeAttendanceFilterError(w http.ResponseWriter, err error) {
	var paramErr *listParamError
	if errors.As(err, &paramErr) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
		return
	}
	writeValidationError(w)
}

func writeAttendanceReadError(w http.ResponseWriter, err error) {
	if errors.Is(err, attendance.ErrForbidden) {
		writeErrorResponse(w, http.StatusForbidden, apierror.CodeForbidden)
		return
	}
	writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
}
