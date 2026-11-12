package httpserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"simpeg/backend/internal/qstash"
	"simpeg/backend/internal/slo/domain"
)

const (
	sloSigningKey = "current-signing-key"
	sloSubject    = "https://simpeg.unisubang.ac.id/api/v1/internal/qstash/slo/evaluate"
)

// fakeSLORecorder merekam hasil pemeriksaan SLO.
type fakeSLORecorder struct {
	evaluation domain.Evaluation
	alerted    bool
	err        error
	calls      int
}

// EvaluateAndNotify mengembalikan hasil yang sudah disiapkan.
func (f *fakeSLORecorder) EvaluateAndNotify(_ context.Context) (domain.Evaluation, bool, error) {
	f.calls++
	if f.err != nil {
		return domain.Evaluation{}, false, f.err
	}
	return f.evaluation, f.alerted, nil
}

// sloMux menyusun mux dengan callback pemeriksaan SLO.
func sloMux(evaluator SLOEvaluator, recorder AuditRecorder) *http.ServeMux {
	mux := http.NewServeMux()
	verifier := qstash.NewVerifier([]string{sloSigningKey}, sloSubject)
	RegisterSLORoutes(mux, &SLOOptions{
		Verifier:  verifier,
		Evaluator: evaluator,
		Recorder:  recorder,
	})
	return mux
}

func TestSLOCallbackRejectsMissingSignature(t *testing.T) {
	mux := sloMux(&fakeSLORecorder{}, &fakeAuditRecorder{})
	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/internal/qstash/slo/evaluate",
		strings.NewReader(`{"kind":"slo.evaluate"}`))
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", response.Code)
	}
}

func TestSLOCallbackRejectsWrongJobKind(t *testing.T) {
	body := []byte(`{"kind":"audit.cleanup"}`)
	header := signQStashCallback(t, sloSigningKey, body, sloSubject)

	mux := sloMux(&fakeSLORecorder{}, &fakeAuditRecorder{})
	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/internal/qstash/slo/evaluate",
		strings.NewReader(string(body)))
	request.Header.Set(qstash.HeaderSignature, header)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", response.Code)
	}
}

func TestSLOCallbackReturnsEvaluation(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	evaluator := &fakeSLORecorder{
		evaluation: domain.Evaluation{
			Window:    domain.Window{Start: now.Add(-5 * time.Minute), End: now},
			Requests:  120,
			Errors:    3,
			ErrorRate: 0.025,
			P95:       900 * time.Millisecond,
			P99:       1400 * time.Millisecond,
			P95Bound:  1000 * time.Millisecond,
			Breaches:  []domain.Breach{{Kind: domain.BreachLatency}},
		},
		alerted: true,
	}
	recorder := &fakeAuditRecorder{}
	mux := sloMux(evaluator, recorder)

	body := []byte(`{"kind":"slo.evaluate"}`)
	header := signQStashCallback(t, sloSigningKey, body, sloSubject)
	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/internal/qstash/slo/evaluate",
		strings.NewReader(string(body)))
	request.Header.Set(qstash.HeaderSignature, header)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body = %s", response.Code, response.Body.String())
	}
	payload := response.Body.String()
	for _, expected := range []string{`"requests":120`, `"errors":3`, `"breaches":1`, `"alerted":true`} {
		if !strings.Contains(payload, expected) {
			t.Errorf("respons tidak memuat %s:\n%s", expected, payload)
		}
	}
	if len(recorder.entries) != 1 {
		t.Fatalf("entri jejak audit = %d, ingin 1", len(recorder.entries))
	}
	if recorder.entries[0].Action != "slo.evaluated" {
		t.Fatalf("aksi jejak audit = %s, ingin slo.evaluated", recorder.entries[0].Action)
	}
}

func TestSLOCallbackReportsEvaluationFailure(t *testing.T) {
	evaluator := &fakeSLORecorder{err: errors.New("database tidak dapat dijangkau")}
	mux := sloMux(evaluator, &fakeAuditRecorder{})

	body := []byte(`{"kind":"slo.evaluate"}`)
	header := signQStashCallback(t, sloSigningKey, body, sloSubject)
	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/internal/qstash/slo/evaluate",
		strings.NewReader(string(body)))
	request.Header.Set(qstash.HeaderSignature, header)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", response.Code)
	}
}

func TestSLORoutesAreNotRegisteredWithoutDependencies(t *testing.T) {
	mux := http.NewServeMux()
	RegisterSLORoutes(mux, nil)
	RegisterSLORoutes(mux, &SLOOptions{})

	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/internal/qstash/slo/evaluate",
		strings.NewReader(`{"kind":"slo.evaluate"}`))
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", response.Code)
	}
}
