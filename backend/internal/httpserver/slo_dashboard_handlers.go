package httpserver

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"simpeg/backend/internal/apierror"
	"simpeg/backend/internal/auth"
	sloapp "simpeg/backend/internal/slo/app"
	"simpeg/backend/internal/timeutil"
)

// SLODashboardReader adalah port pembacaan ringkasan metrik SLO untuk
// halaman admin internal.
type SLODashboardReader interface {
	Snapshot(ctx context.Context, lookback time.Duration) (sloapp.Snapshot, error)
}

// SLORouteDTO adalah bentuk publik ringkasan metrik satu rute.
type SLORouteDTO struct {
	Route     string  `json:"route"`
	Method    string  `json:"method"`
	Requests  int64   `json:"requests"`
	Errors    int64   `json:"errors"`
	ErrorRate float64 `json:"errorRate"`
	AvgMs     float64 `json:"avgMs"`
	P95Ms     int64   `json:"p95Ms"`
	P99Ms     int64   `json:"p99Ms"`
	MaxMs     int64   `json:"maxMs"`
}

// SLOMetricsDTO adalah bentuk publik ringkasan metrik SLO pada satu rentang
// waktu. Seluruh angka latensi dalam milidetik.
type SLOMetricsDTO struct {
	WindowStart time.Time     `json:"windowStart"`
	WindowEnd   time.Time     `json:"windowEnd"`
	Requests    int64         `json:"requests"`
	Errors      int64         `json:"errors"`
	ErrorRate   float64       `json:"errorRate"`
	AvgMs       float64       `json:"avgMs"`
	P95Ms       int64         `json:"p95Ms"`
	P99Ms       int64         `json:"p99Ms"`
	MaxMs       int64         `json:"maxMs"`
	Routes      []SLORouteDTO `json:"routes"`
}

const (
	sloDefaultLookbackHours = 24
	sloMaxLookbackHours     = 168
)

// RegisterSLOAdminRoutes mendaftarkan endpoint admin pembacaan metrik SLO.
//
// Endpoint hanya membuka ringkasan teragregasi; metrik mentah per jendela
// tidak pernah keluar dari server. Akses dibatasi permission slo:read
// (admin) dan deny-by-default untuk role lain.
func RegisterSLOAdminRoutes(mux *http.ServeMux, resolver auth.SessionResolver, policy auth.Policy, dashboard SLODashboardReader) {
	if dashboard == nil {
		return
	}
	api := sloAdminAPI{dashboard: dashboard}
	protected := func(permission auth.Permission, handler http.Handler) http.Handler {
		return auth.Authenticate(resolver)(auth.RequirePermission(policy, permission, handler))
	}
	mux.Handle("GET "+APIPrefix+"/admin/slo/metrics", protected("slo:read", http.HandlerFunc(api.metrics)))
}

// sloAdminAPI menangani pembacaan ringkasan metrik SLO.
type sloAdminAPI struct {
	dashboard SLODashboardReader
}

// metrics mengembalikan ringkasan metrik SLO pada rentang jam yang diminta.
// Parameter `hours` bawaan 24 dan dibatasi maksimum 168 (7 hari).
func (api sloAdminAPI) metrics(w http.ResponseWriter, r *http.Request) {
	raw := strings.TrimSpace(r.URL.Query().Get("hours"))
	hours := sloDefaultLookbackHours
	if raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > sloMaxLookbackHours {
			writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
			return
		}
		hours = value
	}

	snapshot, err := api.dashboard.Snapshot(r.Context(), time.Duration(hours)*time.Hour)
	if err != nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
		return
	}
	writeJSON(w, http.StatusOK, sloMetricsDTO(snapshot))
}

// sloMetricsDTO mengubah ringkasan domain menjadi bentuk publik.
func sloMetricsDTO(snapshot sloapp.Snapshot) SLOMetricsDTO {
	routes := make([]SLORouteDTO, 0, len(snapshot.Routes))
	for _, route := range snapshot.Routes {
		routes = append(routes, SLORouteDTO{
			Route:     route.Route,
			Method:    route.Method,
			Requests:  route.Requests,
			Errors:    route.Errors,
			ErrorRate: route.ErrorRate,
			AvgMs:     route.AvgMs,
			P95Ms:     route.P95Ms,
			P99Ms:     route.P99Ms,
			MaxMs:     route.MaxMs,
		})
	}
	return SLOMetricsDTO{
		WindowStart: timeutil.InLocation(snapshot.WindowStart),
		WindowEnd:   timeutil.InLocation(snapshot.WindowEnd),
		Requests:    snapshot.Requests,
		Errors:      snapshot.Errors,
		ErrorRate:   snapshot.ErrorRate,
		AvgMs:       snapshot.AvgMs,
		P95Ms:       snapshot.P95Ms,
		P99Ms:       snapshot.P99Ms,
		MaxMs:       snapshot.MaxMs,
		Routes:      routes,
	}
}
