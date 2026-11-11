package httpserver

import (
	"net/http"

	"simpeg/backend/internal/apierror"
	"simpeg/backend/internal/observability"
)

// writeErrorResponse menulis respons error dengan bentuk seragam.
// Semua endpoint memakai helper ini supaya kode, message, dan requestId konsisten.
func writeErrorResponse(w http.ResponseWriter, status int, code string) {
	writeErrorWith(w, status, apierror.Payload{Code: code})
}

// writeErrorWith menulis respons error dengan payload lengkap.
func writeErrorWith(w http.ResponseWriter, status int, payload apierror.Payload) {
	if payload.RequestID == "" {
		payload.RequestID = observability.RequestIDFromWriter(w)
	}
	apierror.WriteWith(w, status, payload)
}

// writeValidationError menulis respons validasi dengan bentuk fields seragam.
func writeValidationError(w http.ResponseWriter) {
	writeErrorWith(w, http.StatusUnprocessableEntity, apierror.Payload{
		Code:    apierror.CodeValidation,
		Message: "Permintaan tidak valid.",
		Fields:  map[string]string{"request": "invalid"},
	})
}

// plainErrorInterceptor mengganti respons teks biasa dari ServeMux (404 dan 405)
// dengan JSON error yang seragam.
type plainErrorInterceptor struct {
	http.ResponseWriter
	replaced bool
}

func (w *plainErrorInterceptor) WriteHeader(status int) {
	if status != http.StatusNotFound && status != http.StatusMethodNotAllowed {
		w.ResponseWriter.WriteHeader(status)
		return
	}
	w.replaced = true
	code := apierror.CodeNotFound
	if status == http.StatusMethodNotAllowed {
		code = apierror.CodeMethodNotAllowed
	}
	apierror.WriteWith(w.ResponseWriter, status, apierror.Payload{
		Code:      code,
		RequestID: observability.RequestIDFromWriter(w.ResponseWriter),
	})
}

func (w *plainErrorInterceptor) Write(data []byte) (int, error) {
	if w.replaced {
		// Body bawaan ServeMux sudah dibuang; JSON error yang menggantikannya yang dikirim.
		return len(data), nil
	}
	return w.ResponseWriter.Write(data)
}

// RequestIDValue meneruskan request ID dari writer yang dibungkus.
func (w *plainErrorInterceptor) RequestIDValue() string {
	return observability.RequestIDFromWriter(w.ResponseWriter)
}

// JSONErrorMiddleware memastikan respons 404 dan 405 memakai bentuk JSON yang sama
// dengan endpoint lain.
func JSONErrorMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(&plainErrorInterceptor{ResponseWriter: w}, r)
	})
}
