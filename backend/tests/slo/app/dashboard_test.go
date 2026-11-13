package app

import (
	"context"
	"errors"
	. "simpeg/backend/internal/slo/app"
	"testing"
	"time"

	"simpeg/backend/internal/slo/domain"
)

type fakeRangeRepository struct {
	metrics []domain.Metric
	window  domain.Window
	err     error
	calls   int
}

func (f *fakeRangeRepository) ListWindow(_ context.Context, window domain.Window) ([]domain.Metric, error) {
	f.calls++
	f.window = window
	return f.metrics, f.err
}

func TestDashboardSnapshotAggregatesAcrossBuckets(t *testing.T) {
	now := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	repository := &fakeRangeRepository{metrics: []domain.Metric{
		metricWithBuckets("/api/v1/employees", httpMethodGet, 100, 2, 50*time.Millisecond, 400*time.Millisecond, 200*time.Millisecond),
		metricWithBuckets("/api/v1/employees", httpMethodGet, 50, 1, 40*time.Millisecond, 300*time.Millisecond, 100*time.Millisecond),
		metricWithBuckets("/api/v1/me", httpMethodGet, 10, 0, 10*time.Millisecond, 60*time.Millisecond, 50*time.Millisecond),
	}}
	service := NewDashboardService(DashboardOptions{
		Repository: repository,
		Now:        func() time.Time { return now },
	})

	snapshot, err := service.Snapshot(context.Background(), 6*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if repository.window.End != now || repository.window.Start != now.Add(-6*time.Hour) {
		t.Fatalf("window = %+v", repository.window)
	}
	if snapshot.Requests != 160 || snapshot.Errors != 3 {
		t.Fatalf("requests=%d errors=%d, ingin 160/3", snapshot.Requests, snapshot.Errors)
	}
	if len(snapshot.Routes) != 2 {
		t.Fatalf("routes = %d, ingin 2", len(snapshot.Routes))
	}
	if snapshot.Routes[0].Route != "/api/v1/employees" {
		t.Fatalf("urutan rute melanggar p95 menurun: %+v", snapshot.Routes)
	}
	employees := snapshot.Routes[0]
	if employees.Requests != 150 || employees.Errors != 3 {
		t.Fatalf("agregasi rute employees = %+v", employees)
	}
}

func TestDashboardSnapshotClampsLookback(t *testing.T) {
	now := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	repository := &fakeRangeRepository{}
	service := NewDashboardService(DashboardOptions{
		Repository: repository,
		Now:        func() time.Time { return now },
	})

	if _, err := service.Snapshot(context.Background(), 1000*24*time.Hour); err != nil {
		t.Fatal(err)
	}
	if repository.window.End.Sub(repository.window.Start) != MaxLookback {
		t.Fatalf("lookback = %s, ingin %s", repository.window.End.Sub(repository.window.Start), MaxLookback)
	}

	defaultRepository := &fakeRangeRepository{}
	defaultService := NewDashboardService(DashboardOptions{
		Repository: defaultRepository,
		Now:        func() time.Time { return now },
	})
	if _, err := defaultService.Snapshot(context.Background(), 0); err != nil {
		t.Fatal(err)
	}
	if defaultRepository.window.End.Sub(defaultRepository.window.Start) != DefaultLookback {
		t.Fatalf("lookback default = %s, ingin %s", defaultRepository.window.End.Sub(defaultRepository.window.Start), DefaultLookback)
	}
}

func TestDashboardSnapshotEmptyMetrics(t *testing.T) {
	now := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	service := NewDashboardService(DashboardOptions{
		Repository: &fakeRangeRepository{},
		Now:        func() time.Time { return now },
	})
	snapshot, err := service.Snapshot(context.Background(), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Requests != 0 || snapshot.Errors != 0 {
		t.Fatalf("snapshot = %+v", snapshot)
	}
	if len(snapshot.Routes) != 0 {
		t.Fatalf("routes = %d, ingin 0", len(snapshot.Routes))
	}
}

func TestDashboardSnapshotPropagatesRepositoryError(t *testing.T) {
	service := NewDashboardService(DashboardOptions{
		Repository: &fakeRangeRepository{err: errors.New("database tidak dapat dijangkau")},
	})
	if _, err := service.Snapshot(context.Background(), time.Hour); err == nil {
		t.Fatal("error repository tidak diteruskan")
	}
}

const httpMethodGet = "GET"

// metricWithBuckets membangun metrik sintetis dengan latensi tersebar pada
// satu bucket dominan agar persentil dapat dihitung.
func metricWithBuckets(route, method string, requests, errors int64, avg, max, bucketUpper time.Duration) domain.Metric {
	metric := domain.NewMetric(route, method)
	metric.Requests = requests
	metric.Errors = errors
	metric.LatencySum = avg * time.Duration(requests)
	metric.LatencyMax = max
	index := len(domain.BucketBoundaries)
	for position, boundary := range domain.BucketBoundaries {
		if bucketUpper <= boundary {
			index = position
			break
		}
	}
	for position := index; position < len(metric.Buckets); position++ {
		metric.Buckets[position] = requests
	}
	return metric
}
