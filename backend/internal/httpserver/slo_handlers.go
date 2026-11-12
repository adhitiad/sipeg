package httpserver

import (
	"context"
	"net/http"
	"time"

	"simpeg/backend/internal/apierror"
	"simpeg/backend/internal/audit"
	"simpeg/backend/internal/observability"
	"simpeg/backend/internal/qstash"
	"simpeg/backend/internal/slo/domain"
)

// JobKindSLOEvaluate menandai pekerjaan pemeriksaan target SLO.
const JobKindSLOEvaluate = "slo.evaluate"

// SLOEvaluator mengevaluasi target SLO pada jendela terakhir.
type SLOEvaluator interface {
	// EvaluateAndNotify menilai jendela terakhir lalu mengantrekan alert
	// bila target terlampaui. Nilai balik kedua memberi tahu apakah
	// alert unleashed.
	EvaluateAndNotify(ctx context.Context) (domain.Evaluation, bool, error)
}

// SLOOptions memuat dependensi endpoint pemeriksaan SLO.
type SLOOptions struct {
	// Verifier memeriksa signature callback terhadap raw body.
	Verifier *qstash.Verifier
	// Evaluator menjalankan pemeriksaan target SLO.
	Evaluator SLOEvaluator
	// Recorder mencatat hasil pemeriksaan ke jejak audit.
	Recorder AuditRecorder
	// Logger menerima catatan kegagalan pemeriksaan.
	Logger *observability.Logger
}

// RegisterSLORoutes mendaftarkan endpoint callback pemeriksaan SLO.
//
// Endpoint memakai autentikasi signature QStash, bukan sesi bearer, sama
// seperti callback pekerjaan lain.
func RegisterSLORoutes(mux *http.ServeMux, options *SLOOptions) {
	if options == nil || options.Verifier == nil || options.Evaluator == nil {
		return
	}
	api := sloAPI{options: options}
	mux.Handle("POST "+APIPrefix+"/internal/qstash/slo/evaluate", http.HandlerFunc(api.evaluate))
}

// sloAPI menangani callback pemeriksaan SLO dari QStash.
type sloAPI struct {
	options *SLOOptions
}

// evaluate menjalankan pemeriksaan target SLO.
//
// Pekerjaan ini idempoten: alert mengantre dengan kunci idempotensi yang
// memuat awal jendela, sehingga percobaan ulang QStash tidak menghasilkan
// email kedua untuk jendela yang sama.
func (api sloAPI) evaluate(w http.ResponseWriter, r *http.Request) {
	if _, ok := api.verifiedBody(w, r); !ok {
		return
	}
	evaluation, alerted, err := api.options.Evaluator.EvaluateAndNotify(r.Context())
	if err != nil {
		api.logFailure("slo_evaluate_failed", err)
		writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
		return
	}

	note := api.summarize(evaluation, alerted)
	api.recordOutcome(r.Context(), note)

	writeJSON(w, http.StatusOK, map[string]any{
		"windowStart": evaluation.Window.Start.UTC().Format(time.RFC3339),
		"windowEnd":   evaluation.Window.End.UTC().Format(time.RFC3339),
		"requests":    evaluation.Requests,
		"errors":      evaluation.Errors,
		"errorRate":   evaluation.ErrorRate,
		"p95Ms":       evaluation.P95.Milliseconds(),
		"p99Ms":       evaluation.P99.Milliseconds(),
		"breaches":    len(evaluation.Breaches),
		"alerted":     alerted,
	})
}

// summarize merumuskan hasil pemeriksaan untuk jejak audit.
func (api sloAPI) summarize(evaluation domain.Evaluation, alerted bool) string {
	note := "pemeriksaan SLO: "
	if alerted {
		note += "alert diantrekan"
	} else if evaluation.OK() {
		note += "target terpenuhi"
	} else {
		note += "target terlampaui tanpa alert"
	}
	return note
}

// verifiedBody membaca raw body, memverifikasi signature QStash, dan
// memeriksa jenis pekerjaan.
func (api sloAPI) verifiedBody(w http.ResponseWriter, r *http.Request) ([]byte, bool) {
	raw, ok := readRawBody(w, r)
	if !ok {
		return nil, false
	}
	if err := api.options.Verifier.Verify(qstash.SignatureFromRequest(r), raw); err != nil {
		api.logRejectedCallback(r)
		writeErrorResponse(w, http.StatusUnauthorized, apierror.CodeUnauthorized)
		return nil, false
	}
	var job qstashJob
	if err := decodeJob(raw, &job); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return nil, false
	}
	if job.Kind != JobKindSLOEvaluate {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
		return nil, false
	}
	return raw, true
}

// recordOutcome mencatat hasil pemeriksaan ke jejak audit.
//
// Kegagalan pencatatan tidak menggagalkan pemeriksaan karena jejak audit
// bukan hal kritis.
func (api sloAPI) recordOutcome(ctx context.Context, note string) {
	if api.options.Recorder == nil {
		return
	}
	_ = api.options.Recorder.Record(ctx, audit.Entry{
		Action: audit.ActionSLOEvaluated,
		Note:   &note,
	})
}

// logRejectedCallback mencatat callback yang ditolak karena signature tidak
// valid. Nilai signature tidak pernah dicatat karena merupakan rahasia.
func (api sloAPI) logRejectedCallback(r *http.Request) {
	if api.options.Logger == nil {
		return
	}
	api.options.Logger.Warn("webhook_signature_rejected", observability.Fields{
		"request_id": observability.RequestIDFromContext(r.Context()),
		"job":        JobKindSLOEvaluate,
		"method":     r.Method,
		"path":       r.URL.Path,
	})
}

// logFailure mencatat kegagalan pemeriksaan.
func (api sloAPI) logFailure(event string, err error) {
	if api.options.Logger == nil {
		return
	}
	api.options.Logger.Error(event, observability.Fields{"error": err.Error()})
}
