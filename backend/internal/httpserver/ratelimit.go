package httpserver

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"simpeg/backend/internal/apierror"
)

// RateLimitOptions mengatur batas permintaan per klien.
type RateLimitOptions struct {
	// Limit adalah jumlah permintaan maksimum dalam satu window.
	Limit int
	// Window adalah panjang periode hitungan.
	Window time.Duration
	// StrictPaths menerima batas lebih ketat untuk path sensitif.
	StrictPaths []string
	// StrictLimit dan StrictWindow berlaku untuk StrictPaths.
	StrictLimit  int
	StrictWindow time.Duration
	// Now dapat diganti pada pengujian.
	Now func() time.Time
}

// RateLimiter membatasi permintaan per klien dengan hitungan window geser.
//
// Implementasi ini memakai memori per instance. Pada deployment lebih dari satu
// instance, batas efektif menjadi jumlah instance dikalikan batas per instance.
// Batas terdistribusi membutuhkan Redis, yang belum menjadi use case yang
// disetujui.
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*rateBucket
	options RateLimitOptions
}

type rateBucket struct {
	count int
	reset time.Time
}

// NewRateLimiter membuat limiter dengan opsi yang diberikan.
func NewRateLimiter(options RateLimitOptions) *RateLimiter {
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.Limit <= 0 {
		options.Limit = 120
	}
	if options.Window <= 0 {
		options.Window = time.Minute
	}
	return &RateLimiter{buckets: map[string]*rateBucket{}, options: options}
}

// Middleware membungkus handler dengan pembatasan laju.
func (l *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limit, window := l.limitFor(r.URL.Path)
		key := clientKey(r) + "|" + r.URL.Path
		now := l.options.Now()

		l.mu.Lock()
		bucket, exists := l.buckets[key]
		if !exists || now.After(bucket.reset) {
			bucket = &rateBucket{reset: now.Add(window)}
			l.buckets[key] = bucket
		}
		bucket.count++
		count := bucket.count
		reset := bucket.reset
		l.evictLocked(now)
		l.mu.Unlock()

		if count > limit {
			retryAfter := int(time.Until(reset).Seconds())
			if retryAfter < 1 {
				retryAfter = 1
			}
			w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
			writeErrorWith(w, http.StatusTooManyRequests, apierror.Payload{
				Code:    "rate_limited",
				Message: "Terlalu banyak permintaan. Coba lagi nanti.",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// limitFor mengembalikan batas dan window yang berlaku untuk sebuah path.
func (l *RateLimiter) limitFor(path string) (int, time.Duration) {
	for _, strict := range l.options.StrictPaths {
		if strings.HasPrefix(path, strict) {
			if l.options.StrictLimit > 0 {
				return l.options.StrictLimit, l.options.StrictWindow
			}
		}
	}
	return l.options.Limit, l.options.Window
}

// evictLocked membuang bucket yang sudah kedaluwarsa agar memori tidak tumbuh
// tanpa batas. Pemanggil harus memegang l.mu.
func (l *RateLimiter) evictLocked(now time.Time) {
	if len(l.buckets) < 1024 {
		return
	}
	for key, bucket := range l.buckets {
		if now.After(bucket.reset) {
			delete(l.buckets, key)
		}
	}
}

// clientKey menentukan identitas klien untuk perhitungan batas.
// Header proxy tidak dipercaya karena dapat dipalsukan klien.
func clientKey(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
