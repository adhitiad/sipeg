package domain

import (
	"testing"
	"time"
)

func TestObserveCountsRequestsErrorsAndBuckets(t *testing.T) {
	metric := NewMetric("/api/v1/employees", "GET")
	metric.Observe(3*time.Millisecond, false)
	metric.Observe(120*time.Millisecond, true)
	metric.Observe(9*time.Second, false)

	if metric.Requests != 3 {
		t.Fatalf("permintaan = %d, ingin 3", metric.Requests)
	}
	if metric.Errors != 1 {
		t.Fatalf("galat = %d, ingin 1", metric.Errors)
	}
	if metric.LatencyMax != 9*time.Second {
		t.Fatalf("latensi maksimum = %s, ingin 9s", metric.LatencyMax)
	}
	// Bucket bersifat kumulatif. Sampel 3 ms masuk ke bucket 5 ms dan
	// terhitung pada seluruh bucket di atasnya.
	if metric.Buckets[0] != 1 {
		t.Fatalf("bucket 5ms = %d, ingin 1", metric.Buckets[0])
	}
	// Sampel 120 ms tidak masuk ke bucket 100 ms, baru ke bucket 200 ms.
	if metric.Buckets[4] != 1 {
		t.Fatalf("bucket 100ms = %d, ingin 1", metric.Buckets[4])
	}
	if metric.Buckets[5] != 2 {
		t.Fatalf("bucket 200ms = %d, ingin 2", metric.Buckets[5])
	}
	// Bucket luapan memuat seluruh permintaan karena bersifat kumulatif.
	if metric.Buckets[BucketOverflow] != 3 {
		t.Fatalf("bucket luapan = %d, ingin 3", metric.Buckets[BucketOverflow])
	}
}

func TestErrorRateReturnsZeroWithoutRequests(t *testing.T) {
	metric := NewMetric("/api/v1/me", "GET")
	if metric.ErrorRate() != 0 {
		t.Fatalf("laju galat = %f, ingin 0", metric.ErrorRate())
	}
}

func TestQuantileUpperNeverUnderstatesLatency(t *testing.T) {
	metric := NewMetric("/api/v1/me", "GET")
	for index := 0; index < 90; index++ {
		metric.Observe(10*time.Millisecond, false)
	}
	for index := 0; index < 10; index++ {
		metric.Observe(110*time.Millisecond, false)
	}

	upper := metric.QuantileUpper(0.50)
	if upper != 10*time.Millisecond {
		t.Fatalf("batas atas p50 = %s, ingin 10ms", upper)
	}
	upper = metric.QuantileUpper(0.95)
	if upper != 200*time.Millisecond {
		t.Fatalf("batas atas p95 = %s, ingin 200ms", upper)
	}
	// Perkiraan interpolasi berada di dalam bucket yang memuat persentil.
	estimate := metric.Quantile(0.50)
	if estimate < 5*time.Millisecond || estimate > 10*time.Millisecond {
		t.Fatalf("perkiraan p50 = %s, ingin di antara 5ms dan 10ms", estimate)
	}
}

func TestQuantileUpperUsesLastBoundaryForOverflowSamples(t *testing.T) {
	metric := NewMetric("/api/v1/me", "GET")
	metric.Observe(30*time.Second, false)

	if got := metric.QuantileUpper(0.99); got != BucketBoundaries[BucketOverflow-1] {
		t.Fatalf("batas atas p99 = %s, ingin %s", got, BucketBoundaries[BucketOverflow-1])
	}
}

func TestMergeCombinesMetrics(t *testing.T) {
	first := NewMetric("/a", "GET")
	first.Observe(10*time.Millisecond, false)
	second := NewMetric("/b", "GET")
	second.Observe(30*time.Millisecond, true)

	merged := Merge([]Metric{first, second})
	if merged.Requests != 2 {
		t.Fatalf("permintaan gabungan = %d, ingin 2", merged.Requests)
	}
	if merged.Errors != 1 {
		t.Fatalf("galat gabungan = %d, ingin 1", merged.Errors)
	}
	if merged.LatencyMax != 30*time.Millisecond {
		t.Fatalf("latensi maksimum gabungan = %s, ingin 30ms", merged.LatencyMax)
	}
	// Sampel 10 ms terhitung pada bucket 10 ms, bukan pada bucket 5 ms.
	if merged.Buckets[1] != 1 {
		t.Fatalf("bucket 10ms gabungan = %d, ingin 1", merged.Buckets[1])
	}
}

func TestEvaluateReportsLatencyBreach(t *testing.T) {
	metric := NewMetric("/api/v1/employees", "GET")
	for index := 0; index < 100; index++ {
		metric.Observe(200*time.Millisecond, false)
	}
	target := Target{P95: 100 * time.Millisecond, ErrorRate: 0.01, MinRequests: 20}

	evaluation := Evaluate(Window{}, []Metric{metric}, target)
	if evaluation.OK() {
		t.Fatal("evaluasi seharusnya melaporkan pelanggaran latensi")
	}
	if evaluation.Breaches[0].Kind != BreachLatency {
		t.Fatalf("jenis pelanggaran = %s, ingin %s", evaluation.Breaches[0].Kind, BreachLatency)
	}
	if evaluation.P95Bound != 200*time.Millisecond {
		t.Fatalf("batas atas p95 = %s, ingin 200ms", evaluation.P95Bound)
	}
}

func TestEvaluateReportsErrorRateBreach(t *testing.T) {
	metric := NewMetric("/api/v1/payroll/slips", "GET")
	for index := 0; index < 100; index++ {
		metric.Observe(10*time.Millisecond, index%10 == 0)
	}
	target := Target{P95: time.Second, ErrorRate: 0.05, MinRequests: 20}

	evaluation := Evaluate(Window{}, []Metric{metric}, target)
	if evaluation.OK() {
		t.Fatal("evaluasi seharusnya melaporkan pelanggaran laju galat")
	}
	if evaluation.Breaches[0].Kind != BreachErrorRate {
		t.Fatalf("jenis pelanggaran = %s, ingin %s", evaluation.Breaches[0].Kind, BreachErrorRate)
	}
}

func TestEvaluateIgnoresWindowBelowMinimumRequests(t *testing.T) {
	metric := NewMetric("/api/v1/me", "GET")
	metric.Observe(5*time.Second, true)
	target := Target{P95: 10 * time.Millisecond, ErrorRate: 0.001, MinRequests: 20}

	evaluation := Evaluate(Window{}, []Metric{metric}, target)
	if !evaluation.OK() {
		t.Fatal("jendela dengan permintaan terlalu sedikit tidak boleh dilaporkan")
	}
	if evaluation.Requests != 1 {
		t.Fatalf("permintaan = %d, ingin 1", evaluation.Requests)
	}
}

func TestEvaluateAcceptsTargetEqualToBucketBoundary(t *testing.T) {
	// Target 800 ms adalah batas bucket. Latensi tepat 800 ms tidak boleh
	// dianggap sebagai pelanggaran.
	metric := NewMetric("/api/v1/employees", "GET")
	for index := 0; index < 100; index++ {
		metric.Observe(800*time.Millisecond, false)
	}
	target := Target{P95: 800 * time.Millisecond, ErrorRate: 0.01, MinRequests: 20}

	evaluation := Evaluate(Window{}, []Metric{metric}, target)
	if !evaluation.OK() {
		t.Fatalf("latensi tepat pada target tidak boleh dianggap pelanggaran: %v", evaluation.Breaches)
	}
}

func TestEvaluateSortsRoutesByLatencyDescending(t *testing.T) {
	slow := NewMetric("/lambat", "GET")
	slow.Observe(500*time.Millisecond, false)
	fast := NewMetric("/cepat", "GET")
	fast.Observe(5*time.Millisecond, false)
	target := Target{P95: time.Second, ErrorRate: 0.5, MinRequests: 1}

	evaluation := Evaluate(Window{}, []Metric{fast, slow}, target)
	if len(evaluation.Routes) != 2 {
		t.Fatalf("jumlah rute = %d, ingin 2", len(evaluation.Routes))
	}
	if evaluation.Routes[0].Route != "/lambat" {
		t.Fatalf("rute pertama = %s, ingin /lambat", evaluation.Routes[0].Route)
	}
}

func TestFormatDurationAndPercent(t *testing.T) {
	if got := FormatDuration(1500 * time.Microsecond); got != "2 ms" {
		t.Fatalf("durasi = %s, ingin 2 ms", got)
	}
	if got := FormatDuration(2500 * time.Millisecond); got != "2.50 s" {
		t.Fatalf("durasi = %s, ingin 2.50 s", got)
	}
	if got := FormatPercent(0.0123); got != "1.23%" {
		t.Fatalf("persentase = %s, ingin 1.23%%", got)
	}
}
