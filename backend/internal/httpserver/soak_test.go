package httpserver

import (
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"simpeg/backend/internal/auth"
)

// TestSoakReadPath adalah uji soak pada jalur baca HTTP.
//
// Uji ini berjalan secara opt-in lewat variabel lingkungan SIMPEG_SOAK=1
// supaya pipeline CI tetap cepat. Durasi dan jumlah aliran request paralel
// dapat diatur lewat SIMPEG_SOAK_DURATION dan SIMPEG_SOAK_STREAMS.
//
// Hasil soak memakai repository palsu, sehingga yang diuji adalah stabilitas
// jalur HTTP: degradasi latensi, pertumbuhan memori, dan error. Uji soak
// terhadap PostgreSQL, Redis, QStash, dan SMTP tetap memerlukan staging.
func TestSoakReadPath(t *testing.T) {
	if os.Getenv("SIMPEG_SOAK") != "1" {
		t.Skip("uji soak dijalankan dengan SIMPEG_SOAK=1")
	}

	duration := soakDuration()
	streams := soakStreams()

	store := &loadStore{employees: loadEmployees(100)}
	mux := loadMux(store, auth.Principal{UserID: atasanID, RoleName: "admin"},
		map[string]map[auth.Permission]bool{"admin": {"employee:read:any": true}})

	var before runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)

	deadline := time.Now().Add(duration)
	var failures atomic.Int64
	var requests atomic.Int64
	histograms := make([]*latencyHistogram, streams)
	var wait sync.WaitGroup

	for stream := 0; stream < streams; stream++ {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			histogram := newLatencyHistogram()
			histograms[index] = histogram
			for time.Now().Before(deadline) {
				started := time.Now()
				httpRequest := httptest.NewRequest(http.MethodGet, APIPrefix+"/employees?pageSize=100", nil)
				httpRequest.Header.Set("Authorization", "Bearer valid-session")
				response := httptest.NewRecorder()
				mux.ServeHTTP(response, httpRequest)
				histogram.add(time.Since(started))
				requests.Add(1)
				if response.Code != http.StatusOK {
					failures.Add(1)
				}
			}
		}(stream)
	}
	wait.Wait()

	var after runtime.MemStats
	runtime.ReadMemStats(&after)

	merged := newLatencyHistogram()
	for _, histogram := range histograms {
		if histogram != nil {
			merged.merge(histogram)
		}
	}

	total := requests.Load()
	t.Logf("soak: durasi=%s streams=%d requests=%d failures=%d throughput=%.0f req/detik",
		duration, streams, total, failures.Load(), float64(total)/duration.Seconds())
	t.Logf("soak latensi: p50=%s p95=%s p99=%s max=%s (bucket %s)",
		merged.percentile(0.50), merged.percentile(0.95), merged.percentile(0.99), merged.max, latencyBucketSize)
	t.Logf("soak memori: heap=%d KB -> %d KB, totalAlloc=%d MB, gc=%d",
		before.HeapAlloc/1024, after.HeapAlloc/1024,
		(after.TotalAlloc-before.TotalAlloc)/(1024*1024), after.NumGC-before.NumGC)

	if failures.Load() != 0 {
		t.Fatalf("gagal=%d, stabilitas jalur baca tidak tercapai", failures.Load())
	}
	if total == 0 {
		t.Fatal("tidak ada request yang tercatat")
	}
	// Degradasi latensi yang tidak terkendali menandakan sumber daya yang
	// menumpuk pada jalur baca.
	if merged.percentile(0.99) > 500*time.Millisecond {
		t.Fatalf("p99=%s, jauh di atas batas 500ms", merged.percentile(0.99))
	}
}

// Parameter histogram latensi.
const (
	latencyBucketSize  = 50 * time.Microsecond
	latencyBucketCount = 2000
)

// latencyHistogram menyimpan distribusi latensi dalam bucket berkuran tetap.
//
// Histogram dipakai agar harness soak tidak menyimpan seluruh sampel latensi
// di memori, sehingga pertumbuhan memori yang terukur benar-benar berasal
// dari jalur yang diuji.
type latencyHistogram struct {
	buckets  []int64
	overflow int64
	max      time.Duration
}

func newLatencyHistogram() *latencyHistogram {
	return &latencyHistogram{buckets: make([]int64, latencyBucketCount)}
}

func (h *latencyHistogram) add(value time.Duration) {
	if value > h.max {
		h.max = value
	}
	index := int(value / latencyBucketSize)
	if index >= len(h.buckets) {
		h.overflow++
		return
	}
	h.buckets[index]++
}

func (h *latencyHistogram) merge(other *latencyHistogram) {
	for index, count := range other.buckets {
		h.buckets[index] += count
	}
	h.overflow += other.overflow
	if other.max > h.max {
		h.max = other.max
	}
}

func (h *latencyHistogram) percentile(fraction float64) time.Duration {
	total := h.overflow
	for _, count := range h.buckets {
		total += count
	}
	if total == 0 {
		return 0
	}
	target := int64(float64(total) * fraction)
	if target >= total {
		target = total - 1
	}
	seen := int64(0)
	for index, count := range h.buckets {
		seen += count
		if seen > target {
			return time.Duration(index) * latencyBucketSize
		}
	}
	return h.max
}

// soakDuration membaca durasi soak dari lingkungan, default 30 detik.
func soakDuration() time.Duration {
	value := os.Getenv("SIMPEG_SOAK_DURATION")
	if value == "" {
		return 30 * time.Second
	}
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return 30 * time.Second
	}
	return time.Duration(seconds) * time.Second
}

// soakStreams membaca jumlah aliran request paralel dari lingkungan.
func soakStreams() int {
	value := os.Getenv("SIMPEG_SOAK_STREAMS")
	if value == "" {
		return 16
	}
	count, err := strconv.Atoi(value)
	if err != nil || count <= 0 {
		return 16
	}
	return count
}
