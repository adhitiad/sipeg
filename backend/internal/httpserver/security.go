package httpserver

import (
	"context"
	"net/http"
	"strings"
	"time"

	"simpeg/backend/internal/apierror"
)

// CORSOptions mengatur akses lintas origin.
type CORSOptions struct {
	// TrustedOrigins adalah daftar origin yang diizinkan. Kosong berarti
	// tidak ada respons CORS sama sekali sehingga browser menolak cross-origin.
	TrustedOrigins []string
	// AllowedMethods daftar method yang diizinkan pada preflight.
	AllowedMethods []string
	// AllowedHeaders daftar header yang diizinkan pada preflight.
	AllowedHeaders []string
}

// DefaultCORSMethods adalah method yang diizinkan untuk API.
func DefaultCORSMethods() []string {
	return []string{
		http.MethodGet, http.MethodPost, http.MethodPatch,
		http.MethodDelete, http.MethodOptions,
	}
}

// DefaultCORSHeaders adalah header yang diizinkan pada preflight.
func DefaultCORSHeaders() []string {
	return []string{"Authorization", "Content-Type", "X-API-Version", "X-Request-ID"}
}

// CORSMiddleware membolehkan hanya origin yang tercantum. Origin di luar
// daftar tidak mendapat header CORS sehingga browser memblokir respons.
func CORSMiddleware(options CORSOptions) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(options.TrustedOrigins))
	for _, origin := range options.TrustedOrigins {
		allowed[strings.ToLower(strings.TrimRight(origin, "/"))] = true
	}
	methods := strings.Join(options.AllowedMethods, ", ")
	headers := strings.Join(options.AllowedHeaders, ", ")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := strings.ToLower(strings.TrimRight(r.Header.Get("Origin"), "/"))
			// Vary Origin wajib agar cache tidak-serving respons origin lain.
			w.Header().Add("Vary", "Origin")

			if origin == "" || !allowed[origin] {
				if r.Method == http.MethodOptions {
					// Preflight dari origin tak dikenal ditolak.
					writeErrorResponse(w, http.StatusForbidden, apierror.CodeForbidden)
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", methods)
			w.Header().Set("Access-Control-Allow-Headers", headers)
			w.Header().Set("Access-Control-Max-Age", "600")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// SecurityHeadersMiddleware menambahkan header keamanan dasar pada setiap respons.
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := w.Header()
		header.Set("X-Content-Type-Options", "nosniff")
		header.Set("X-Frame-Options", "DENY")
		header.Set("Referrer-Policy", "no-referrer")
		header.Set("Cross-Origin-Opener-Policy", "same-origin")
		header.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(w, r)
	})
}

// StatementTimeoutMiddleware membungkus context permintaan dengan batas waktu
// statement database. Driver PostgreSQL membatalkan query yang melewati batas,
// sehingga satu query lambat tidak dapat menahan seluruh worker.
func StatementTimeoutMiddleware(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if timeout <= 0 {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
