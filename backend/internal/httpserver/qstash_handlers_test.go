package httpserver

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"simpeg/backend/internal/audit"
	"simpeg/backend/internal/qstash"

	"github.com/golang-jwt/jwt/v5"
)

const (
	qstashSigningKey = "current-signing-key"
	qstashSubject    = "https://simpeg.unisubang.ac.id/api/v1/internal/qstash/audit/cleanup"
)

// fakeAuditRecorder merekam entri jejak audit untuk pemeriksaan.
type fakeAuditRecorder struct {
	entries []audit.Entry
}

func (r *fakeAuditRecorder) Record(_ context.Context, entry audit.Entry) error {
	r.entries = append(r.entries, entry)
	return nil
}

func signQStashCallback(t *testing.T, key string, body []byte, subject string) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  subject,
		"body": base64.StdEncoding.EncodeToString(body),
		"iss":  "Upstash",
		"exp":  time.Now().Add(5 * time.Minute).Unix(),
		"iat":  time.Now().Unix(),
	})
	signed, err := token.SignedString([]byte(key))
	if err != nil {
		t.Fatalf("failed to sign callback: %v", err)
	}
	return signed
}

func qstashMux(recorder AuditRecorder) *http.ServeMux {
	mux := http.NewServeMux()
	verifier := qstash.NewVerifier([]string{qstashSigningKey}, qstashSubject)
	RegisterQStashRoutes(mux, &QStashOptions{Verifier: verifier, Cleaner: audit.NewCleaner(nil, 0), Recorder: recorder})
	return mux
}

func TestQStashCallbackRejectsMissingSignature(t *testing.T) {
	mux := qstashMux(&fakeAuditRecorder{})
	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/internal/qstash/audit/cleanup",
		strings.NewReader(`{"kind":"audit.cleanup"}`))
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", response.Code)
	}
}

func TestQStashCallbackRejectsTamperedBody(t *testing.T) {
	body := []byte(`{"kind":"audit.cleanup"}`)
	header := signQStashCallback(t, qstashSigningKey, body, qstashSubject)

	mux := qstashMux(&fakeAuditRecorder{})
	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/internal/qstash/audit/cleanup",
		strings.NewReader(`{"kind":"audit.cleanup","force":true}`))
	request.Header.Set(qstash.HeaderSignature, header)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", response.Code)
	}
}

func TestQStashCallbackRejectsUnknownJobKind(t *testing.T) {
	body := []byte(`{"kind":"lain"}`)
	header := signQStashCallback(t, qstashSigningKey, body, qstashSubject)

	mux := qstashMux(&fakeAuditRecorder{})
	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/internal/qstash/audit/cleanup",
		strings.NewReader(string(body)))
	request.Header.Set(qstash.HeaderSignature, header)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", response.Code)
	}
}

func TestQStashCallbackRunsCleanupAndRecordsOutcome(t *testing.T) {
	body := []byte(`{"kind":"audit.cleanup"}`)
	header := signQStashCallback(t, qstashSigningKey, body, qstashSubject)

	recorder := &fakeAuditRecorder{}
	mux := qstashMux(recorder)
	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/internal/qstash/audit/cleanup",
		strings.NewReader(string(body)))
	request.Header.Set(qstash.HeaderSignature, header)
	request.Header.Set(HeaderMessageID, "msg-1")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
	if !strings.Contains(response.Body.String(), `"deleted":0`) {
		t.Fatalf("unexpected body: %s", response.Body)
	}
	if len(recorder.entries) != 1 {
		t.Fatalf("audit entries = %d, want 1", len(recorder.entries))
	}
	if recorder.entries[0].Action != audit.ActionAuditCleaned {
		t.Fatalf("action = %q", recorder.entries[0].Action)
	}
}

const payrollNotifySubject = "https://simpeg.unisubang.ac.id/api/v1/internal/qstash/payroll/notify"

func payrollNotifyMux(recorder AuditRecorder) *http.ServeMux {
	mux := http.NewServeMux()
	verifier := qstash.NewVerifier([]string{qstashSigningKey}, payrollNotifySubject)
	RegisterQStashRoutes(mux, &QStashOptions{
		PayrollVerifier: verifier,
		PayrollReminder: func(context.Context) (int, error) { return 3, nil },
		Recorder:        recorder,
	})
	return mux
}

func TestPayrollNotifyCallbackRunsReminderAndRecordsOutcome(t *testing.T) {
	body := []byte(`{"kind":"payroll.notify"}`)
	header := signQStashCallback(t, qstashSigningKey, body, payrollNotifySubject)

	recorder := &fakeAuditRecorder{}
	mux := payrollNotifyMux(recorder)
	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/internal/qstash/payroll/notify",
		strings.NewReader(string(body)))
	request.Header.Set(qstash.HeaderSignature, header)
	request.Header.Set(HeaderMessageID, "msg-9")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
	if !strings.Contains(response.Body.String(), `"queued":3`) {
		t.Fatalf("body = %s", response.Body)
	}
	if len(recorder.entries) != 1 {
		t.Fatalf("audit entries = %d, want 1", len(recorder.entries))
	}
	if recorder.entries[0].Action != audit.ActionPayrollNotified {
		t.Fatalf("action = %q", recorder.entries[0].Action)
	}
}

func TestPayrollNotifyCallbackRejectsWrongJobKind(t *testing.T) {
	body := []byte(`{"kind":"audit.cleanup"}`)
	header := signQStashCallback(t, qstashSigningKey, body, payrollNotifySubject)

	mux := payrollNotifyMux(&fakeAuditRecorder{})
	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/internal/qstash/payroll/notify",
		strings.NewReader(string(body)))
	request.Header.Set(qstash.HeaderSignature, header)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", response.Code)
	}
}

func TestPayrollNotifyCallbackRejectsBadSignature(t *testing.T) {
	body := []byte(`{"kind":"payroll.notify"}`)

	mux := payrollNotifyMux(&fakeAuditRecorder{})
	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/internal/qstash/payroll/notify",
		strings.NewReader(string(body)))
	request.Header.Set(qstash.HeaderSignature, "not-a-token")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", response.Code)
	}
}
