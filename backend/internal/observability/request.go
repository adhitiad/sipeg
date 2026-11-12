package observability

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// RequestIDHeader adalah header yang dipakai untuk korelasi permintaan.
const RequestIDHeader = "X-Request-ID"

// maxRequestIDLength membatasi panjang request ID yang diterima dari klien.
const maxRequestIDLength = 128

type requestIDKey struct{}

// RequestIDFromContext mengembalikan request ID yang tersimpan di context.
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// WithRequestID menyimpan request ID ke context.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}

// RequestIDMiddleware memakai request ID dari klien bila valid, atau membuat
// baru bila tidak ada/nilai tidak aman, lalu meneruskannya ke context dan header respons.
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := sanitizeRequestID(r.Header.Get(RequestIDHeader))
		if id == "" {
			id = uuid.NewString()
		}
		w.Header().Set(RequestIDHeader, id)
		wrapped := &requestIDWriter{ResponseWriter: w, id: id}
		next.ServeHTTP(wrapped, r.WithContext(WithRequestID(r.Context(), id)))
	})
}

// requestIDWriter membawa request ID ke handler tanpa membocorkan context ke luar paket.
type requestIDWriter struct {
	http.ResponseWriter
	id string
}

// RequestIDValue memungkinkan handler membaca request ID dari ResponseWriter.
func (w *requestIDWriter) RequestIDValue() string { return w.id }

// RequestIDFromWriter membaca request ID dari ResponseWriter bila tersedia.
func RequestIDFromWriter(w http.ResponseWriter) string {
	carrier, ok := w.(interface{ RequestIDValue() string })
	if !ok {
		return ""
	}
	return carrier.RequestIDValue()
}

// sanitizeRequestID menerima hanya karakter yang aman untuk log dan header.
// Nilai dari klien yang memuat karakter kontrol atau terlalu panjang dibuang.
func sanitizeRequestID(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || len(trimmed) > maxRequestIDLength {
		return ""
	}
	for _, r := range trimmed {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '-', r == '_', r == '.', r == ':':
		default:
			return ""
		}
	}
	return trimmed
}

// statusRecorder merekam status dan ukuran respons tanpa mengubah semantik.
type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (s *statusRecorder) WriteHeader(status int) {
	if s.status == 0 {
		s.status = status
	}
	s.ResponseWriter.WriteHeader(status)
}

func (s *statusRecorder) Write(data []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}
	written, err := s.ResponseWriter.Write(data)
	s.bytes += written
	return written, err
}

// RequestIDValue meneruskan request ID dari writer yang dibungkus.
func (s *statusRecorder) RequestIDValue() string {
	return RequestIDFromWriter(s.ResponseWriter)
}

// AccessLogOptions mengatur isi log akses.
type AccessLogOptions struct {
	Logger *Logger
}

// AccessLogMiddleware mencatat satu entri log terstruktur per permintaan.
// Header kredensial, body, dan query string tidak pernah dicatat.
func AccessLogMiddleware(options AccessLogOptions) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := options.Logger.Now()
			recorder := &statusRecorder{ResponseWriter: w}
			sink := &ActorSink{}
			next.ServeHTTP(recorder, r.WithContext(WithActorSink(r.Context(), sink)))

			fields := Fields{
				"request_id":  RequestIDFromContext(r.Context()),
				"method":      r.Method,
				"path":        r.URL.Path,
				"status":      recorder.status,
				"bytes":       recorder.bytes,
				"duration_ms": time.Since(started).Milliseconds(),
			}
			// Sink menangkap principal dari middleware autentikasi yang berada
			// di dalam router, yang tidak dapat dibaca dari context luar.
			if userID, role, ok := sink.Load(); ok {
				fields["user_id"] = userID
				fields["role"] = role
			}
			if recorder.status >= http.StatusInternalServerError {
				options.Logger.Error("http_request", fields)
				return
			}
			options.Logger.Info("http_request", fields)
		})
	}
}
