package httpserver

import (
	"errors"
	"net/http"
	"strconv"

	"simpeg/backend/internal/apierror"
	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/holiday"

	"simpeg/backend/internal/timeutil"
)

// HolidayDTO adalah bentuk publik satu hari libur kalender.
type HolidayDTO struct {
	Tanggal string `json:"tanggal"`
	Nama    string `json:"nama"`
}

// holidayAPI memegang use case kalender hari libur.
type holidayAPI struct {
	calendar *holiday.Service
}

// listHolidays mengembalikan kalender hari libur satu tahun.
func (api holidayAPI) listHolidays(w http.ResponseWriter, r *http.Request) {
	year, err := holidayYear(r.URL.Query().Get("year"))
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
		return
	}
	entries, err := api.calendar.List(r.Context(), year)
	if err != nil {
		writeHolidayError(w, err)
		return
	}
	items := make([]HolidayDTO, 0, len(entries))
	for _, entry := range entries {
		items = append(items, HolidayDTO{Tanggal: entry.Tanggal.Format("2006-01-02"), Nama: entry.Nama})
	}
	writeJSON(w, http.StatusOK, map[string]any{"year": year, "items": items})
}

// refreshHolidays mengambil ulang kalender sebuah tahun dari penyedia.
func (api holidayAPI) refreshHolidays(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		writeErrorResponse(w, http.StatusUnauthorized, apierror.CodeUnauthorized)
		return
	}
	year, err := strconv.Atoi(r.PathValue("year"))
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	stored, err := api.calendar.Refresh(r.Context(), principal.UserID, year)
	if err != nil {
		writeHolidayError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"year": year, "refreshed": stored})
}

// holidayYear membaca tahun dari query parameter. Kosong berarti tahun berjalan.
func holidayYear(raw string) (int, error) {
	if raw == "" {
		return timeutil.Now().Year(), nil
	}
	year, err := strconv.Atoi(raw)
	if err != nil {
		return 0, err
	}
	if err := holiday.ValidateYear(year); err != nil {
		return 0, err
	}
	return year, nil
}

func writeHolidayError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, holiday.ErrInvalidYear):
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
	case errors.Is(err, holiday.ErrProviderNotConfigured):
		writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
	default:
		var providerErr holiday.ErrProvider
		if errors.As(err, &providerErr) {
			writeErrorResponse(w, http.StatusBadGateway, apierror.CodeServiceUnavailable)
			return
		}
		writeErrorResponse(w, http.StatusInternalServerError, apierror.CodeInternal)
	}
}

// RegisterHolidayRoutes mendaftarkan endpoint kalender hari libur.
func RegisterHolidayRoutes(mux *http.ServeMux, resolver auth.SessionResolver, policy auth.Policy, calendar *holiday.Service) {
	api := holidayAPI{calendar: calendar}
	protectedAny := func(permissions []auth.Permission, handler http.Handler) http.Handler {
		return auth.Authenticate(resolver)(auth.RequireAnyPermission(policy, permissions, handler))
	}
	protected := func(permission auth.Permission, handler http.Handler) http.Handler {
		return auth.Authenticate(resolver)(auth.RequirePermission(policy, permission, handler))
	}

	mux.Handle("GET "+APIPrefix+"/holidays", protectedAny([]auth.Permission{"holiday:read"}, http.HandlerFunc(api.listHolidays)))
	mux.Handle("POST "+APIPrefix+"/holidays/{year}/refresh", protected("holiday:manage", http.HandlerFunc(api.refreshHolidays)))
}
