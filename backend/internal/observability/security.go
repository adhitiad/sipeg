package observability

import (
	"net"
	"net/http"
	"strings"
)

// SecurityEventOptions mengatur pencatatan peristiwa keamanan.
type SecurityEventOptions struct {
	// Logger menerima peristiwa keamanan. Nil berarti tanpa pencatatan.
	Logger *Logger
	// AuthPaths adalah prefix path yang diperlakukan sebagai permukaan
	// autentikasi. Respons 401 pada path tersebut ditandai sebagai login
	// gagal agar mudah dipilah dari kegagalan token pada endpoint lain.
	AuthPaths []string
}

// SecurityEventMiddleware mencatat kegagalan autentikasi, penolakan akses,
// dan pembatasan laju sebagai peristiwa keamanan yang terpisah dari log akses.
//
// Yang dicatat hanya path URL tanpa query string. Header kredensial, body,
// dan nilai query string tidak pernah masuk log.
func SecurityEventMiddleware(options SecurityEventOptions) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			recorder := &statusRecorder{ResponseWriter: w}
			sink, ok := ActorSinkFromContext(r.Context())
			if !ok {
				sink = &ActorSink{}
				r = r.WithContext(WithActorSink(r.Context(), sink))
			}
			next.ServeHTTP(recorder, r)

			event := securityEvent(recorder.status)
			if event == "" {
				return
			}
			fields := Fields{
				"request_id": RequestIDFromContext(r.Context()),
				"event":      event,
				"method":     r.Method,
				"path":       r.URL.Path,
				"status":     recorder.status,
				"client_ip":  hostOf(r.RemoteAddr),
			}
			if isAuthPath(r.URL.Path, options.AuthPaths) {
				fields["auth_surface"] = true
			}
			// Penolakan akses terjadi setelah autentikasi berhasil, sehingga
			// pelaku dapat diketahui. Kegagalan autentikasi tidak punya aktor.
			if userID, role, found := sink.Load(); found {
				fields["user_id"] = userID
				fields["role"] = role
			}
			options.Logger.Warn(event, fields)
		})
	}
}

// securityEvent memetakan status respons ke nama peristiwa keamanan.
func securityEvent(status int) string {
	switch status {
	case http.StatusUnauthorized:
		return "auth_failed"
	case http.StatusForbidden:
		return "access_denied"
	case http.StatusTooManyRequests:
		return "rate_limited"
	default:
		return ""
	}
}

// isAuthPath memeriksa apakah path berada pada permukaan autentikasi.
func isAuthPath(path string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// hostOf mengambil host dari alamat remote tanpa port.
func hostOf(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}
	return host
}
