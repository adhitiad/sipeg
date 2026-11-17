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
	// Limit adalah jumlah permintaan maksimum per klien dalam satu window.
	Limit int
	// Window adalah panjang periode hitungan umum.
	Window time.Duration
	// StrictPaths menerima batas tambahan yang lebih ketat untuk path sensitif.
	StrictPaths []string
	// StrictLimit dan StrictWindow berlaku sebagai kuota tambahan untuk StrictPaths.
	StrictLimit  int
	StrictWindow time.Duration
	// Now dapat diganti pada pengujian.
	Now func() time.Time
}

// RateLimiter membatasi permintaan per klien menggunakan kuota umum dan,
// bila cocok, kuota tambahan untuk path sensitif.
//
// Implementasi ini memakai memori per instance. Pada deployment lebih dari satu
// instance, batas efektif tetap terpisah per instance; gunakan limiter terdistribusi
// di edge/gateway sebelum melakukan horizontal scaling.
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
	if options.StrictLimit > 0 && options.StrictWindow <= 0 {
		options.StrictWindow = options.Window
	}
	return &RateLimiter{buckets: map[string]*rateBucket{}, options: options}
}

// Middleware membungkus handler dengan pembatasan laju.
func (l *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		now := l.options.Now()
		client := clientKey(r)

		l.mu.Lock()
		general := l.consumeLocked(client+"|general", now, l.options.Window)
		blocked := general.count > l.options.Limit
		reset := general.reset

		if strictPath, ok := l.strictPath(r.URL.Path); ok && l.options.StrictLimit > 0 {
			strict := l.consumeLocked(client+"|strict:"+strictPath, now, l.options.StrictWindow)
			if strict.count > l.options.StrictLimit {
				blocked = true
				if strict.reset.After(reset) {
					reset = strict.reset
				}
			}
		}
		l.evictLocked(now)
		l.mu.Unlock()

		if blocked {
			remaining := reset.Sub(now)
			retryAfter := int((remaining + time.Second - 1) / time.Second)
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

// consumeLocked menaikkan bucket yang hanya dapat diakses saat l.mu terkunci.
func (l *RateLimiter) consumeLocked(key string, now time.Time, window time.Duration) *rateBucket {
	bucket, exists := l.buckets[key]
	if !exists || !now.Before(bucket.reset) {
		bucket = &rateBucket{reset: now.Add(window)}
		l.buckets[key] = bucket
	}
	bucket.count++
	return bucket
}

// strictPath mengembalikan prefix konfigurasi terpanjang yang cocok pada
// batas segmen path, agar prefix seperti /auth tidak cocok dengan /authentication.
func (l *RateLimiter) strictPath(path string) (string, bool) {
	matched := ""
	for _, strict := range l.options.StrictPaths {
		prefix := strings.TrimRight(strict, "/")
		if prefix == "" {
			continue
		}
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			if len(prefix) > len(matched) {
				matched = prefix
			}
		}
	}
	return matched, matched != ""
}

// evictLocked membuang bucket yang sudah kedaluwarsa agar memori tidak tumbuh
// tanpa batas. Pemanggil harus memegang l.mu.
func (l *RateLimiter) evictLocked(now time.Time) {
	if len(l.buckets) < 1024 {
		return
	}
	for key, bucket := range l.buckets {
		if !now.Before(bucket.reset) {
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
