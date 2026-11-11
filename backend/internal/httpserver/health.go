package httpserver

import (
	"context"
	"errors"
	"net/http"
	"time"

	"simpeg/backend/internal/apierror"
	"simpeg/backend/internal/database"
)

// DependencyChecker memeriksa satu dependency wajib sebelum API siap menerima traffic.
type DependencyChecker interface {
	// Name mengembalikan nama dependency yang stabil untuk ditampilkan pada status readiness.
	Name() string
	// Check mengembalikan error bila dependency tidak siap.
	Check(ctx context.Context) error
}

// defaultReadinessTimeout membatasi waktu pemeriksaan setiap dependency.
const defaultReadinessTimeout = 3 * time.Second

type dependencyStatus struct {
	Status string `json:"status"`
}

type readinessResponse struct {
	Status string                      `json:"status"`
	Checks map[string]dependencyStatus `json:"checks"`
}

// NewReadinessHandler membuat handler readiness yang memeriksa dependency wajib.
// Liveness (/health) tetap terpisah dan tidak gagal hanya karena dependency sementara down.
func NewReadinessHandler(checkers []DependencyChecker, timeout time.Duration) http.Handler {
	if timeout <= 0 {
		timeout = defaultReadinessTimeout
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(checkers) == 0 {
			writeJSON(w, http.StatusOK, readinessResponse{Status: "ready", Checks: map[string]dependencyStatus{}})
			return
		}

		checks := make(map[string]dependencyStatus, len(checkers))
		ready := true
		for _, checker := range checkers {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			err := checker.Check(ctx)
			cancel()
			if err != nil {
				ready = false
				// Detail error sengaja tidak dikembalikan agar tidak membocorkan
				// connection string, host, kredensial, atau struktur internal.
				status := "unavailable"
				if errors.Is(err, database.ErrPoolSaturated) {
					status = "saturated"
				}
				checks[checker.Name()] = dependencyStatus{Status: status}
				continue
			}
			checks[checker.Name()] = dependencyStatus{Status: "ok"}
		}

		if !ready {
			apierror.WriteWith(w, http.StatusServiceUnavailable, apierror.Payload{
				Code:    apierror.CodeNotReady,
				Message: "API belum siap menerima traffic.",
				Rows:    readinessResponse{Status: "not_ready", Checks: checks},
			})
			return
		}
		writeJSON(w, http.StatusOK, readinessResponse{Status: "ready", Checks: checks})
	})
}
