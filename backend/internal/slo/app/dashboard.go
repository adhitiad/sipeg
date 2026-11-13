// Dasbor internal: ringkasan metrik SLO lintas jendela untuk ditampilkan
// pada halaman admin. Paket ini tidak melakukan I/O selain membaca port
// yang diberikan pemanggil.
package app

import (
	"context"
	"fmt"
	"sort"
	"time"

	"simpeg/backend/internal/slo/domain"
)

// RangeReader adalah port pembacaan metrik SLO pada rentang waktu.
//
// Implementasi membaca seluruh bucket agregasi pada rentang yang diberikan;
// window boleh memuat banyak jendela agregasi sekaligus.
type RangeReader interface {
	ListWindow(ctx context.Context, window domain.Window) ([]domain.Metric, error)
}

// DefaultLookback dipakai bila pemanggil tidak menyebut rentang.
const DefaultLookback = 24 * time.Hour

// MaxLookback membatasi rentang maksimum yang boleh dibaca sekali panggil
// agar ringkasan tidak memindai tabel tanpa batas.
const MaxLookback = 7 * 24 * time.Hour

// RouteSnapshot adalah agregasi metrik satu rute selama rentang waktu.
type RouteSnapshot struct {
	Route     string
	Method    string
	Requests  int64
	Errors    int64
	ErrorRate float64
	AvgMs     float64
	P95Ms     int64
	P99Ms     int64
	MaxMs     int64
}

// Snapshot adalah ringkasan metrik SLO pada satu rentang waktu.
type Snapshot struct {
	WindowStart time.Time
	WindowEnd   time.Time
	Requests    int64
	Errors      int64
	ErrorRate   float64
	AvgMs       float64
	P95Ms       int64
	P99Ms       int64
	MaxMs       int64
	Routes      []RouteSnapshot
}

// DashboardOptions memuat dependensi DashboardService.
type DashboardOptions struct {
	// Repository membaca metrik pada rentang waktu.
	Repository RangeReader
	// Now adalah sumber waktu. Nil memakai waktu UTC saat ini.
	Now func() time.Time
}

// DashboardService menyusun ringkasan metrik untuk halaman admin internal.
type DashboardService struct {
	repository RangeReader
	now        func() time.Time
}

// NewDashboardService membuat DashboardService.
func NewDashboardService(options DashboardOptions) *DashboardService {
	service := &DashboardService{
		repository: options.Repository,
		now:        options.Now,
	}
	if service.now == nil {
		service.now = func() time.Time { return time.Now().UTC() }
	}
	return service
}

// Snapshot membaca metrik selama lookback dan mengembalikan ringkasan
// agregat tingkat layanan dan per rute. Lookback yang melebihi MaxLookback
// dipotong; nilai nol atau negatif memakai DefaultLookback.
func (s *DashboardService) Snapshot(ctx context.Context, lookback time.Duration) (Snapshot, error) {
	if lookback <= 0 {
		lookback = DefaultLookback
	}
	if lookback > MaxLookback {
		lookback = MaxLookback
	}
	end := s.now().UTC()
	start := end.Add(-lookback)

	metrics, err := s.repository.ListWindow(ctx, domain.Window{Start: start, End: end})
	if err != nil {
		return Snapshot{}, fmt.Errorf("baca metrik slo: %w", err)
	}

	total := domain.Merge(metrics)
	summary := Snapshot{
		WindowStart: start,
		WindowEnd:   end,
		Requests:    total.Requests,
		Errors:      total.Errors,
		ErrorRate:   total.ErrorRate(),
		AvgMs:       averageMs(total),
		P95Ms:       total.Quantile(0.95).Milliseconds(),
		P99Ms:       total.Quantile(0.99).Milliseconds(),
		MaxMs:       total.LatencyMax.Milliseconds(),
	}
	summary.Routes = routeSnapshots(metrics)
	return summary, nil
}

// routeSnapshots menggabungkan bucket-bucket metrik per kombinasi rute dan
// metode, lalu mengurutkannya dari p95 tertinggi.
func routeSnapshots(metrics []domain.Metric) []RouteSnapshot {
	type key struct{ route, method string }
	grouped := map[key][]domain.Metric{}
	for _, metric := range metrics {
		k := key{metric.Route, metric.Method}
		grouped[k] = append(grouped[k], metric)
	}

	routes := make([]RouteSnapshot, 0, len(grouped))
	for k, list := range grouped {
		merged := domain.Merge(list)
		routes = append(routes, RouteSnapshot{
			Route:     k.route,
			Method:    k.method,
			Requests:  merged.Requests,
			Errors:    merged.Errors,
			ErrorRate: merged.ErrorRate(),
			AvgMs:     averageMs(merged),
			P95Ms:     merged.Quantile(0.95).Milliseconds(),
			P99Ms:     merged.Quantile(0.99).Milliseconds(),
			MaxMs:     merged.LatencyMax.Milliseconds(),
		})
	}
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].P95Ms != routes[j].P95Ms {
			return routes[i].P95Ms > routes[j].P95Ms
		}
		if routes[i].Route != routes[j].Route {
			return routes[i].Route < routes[j].Route
		}
		return routes[i].Method < routes[j].Method
	})
	return routes
}

// averageMs menghitung rata-rata latensi dalam milidetik.
func averageMs(metric domain.Metric) float64 {
	if metric.Requests == 0 {
		return 0
	}
	return float64(metric.LatencySum) / float64(metric.Requests) / float64(time.Millisecond)
}
