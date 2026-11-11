package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"simpeg/backend/internal/apierror"
	"simpeg/backend/internal/audit"
	"simpeg/backend/internal/qstash"
)

// JobKindAuditCleanup menandai pekerjaan pembersihan jejak audit.
const JobKindAuditCleanup = "audit.cleanup"

// JobKindPayrollNotify menandai pekerjaan pemberitahuan slip gaji H+1.
const JobKindPayrollNotify = "payroll.notify"

// HeaderMessageID memuat id pesan QStash pada header callback.
const HeaderMessageID = "Upstash-Message-Id"

// AuditRecorder mencatat jejak audit untuk aksi yang berada di luar CRUD.
type AuditRecorder interface {
	Record(ctx context.Context, entry audit.Entry) error
}

// qstashAPI menangani callback pekerjaan asinkron dari QStash.
//
// Callback tidak memakai sesi bearer. Autentikasinya adalah signature QStash
// yang diverifikasi terhadap raw body permintaan, dan subjek signature harus
// cocok dengan URL tujuan yang dikonfigurasi.
type qstashAPI struct {
	options *QStashOptions
}

// qstashJob adalah bentuk payload pekerjaan yang dikirim QStash.
type qstashJob struct {
	Kind string `json:"kind"`
}

// RegisterQStashRoutes mendaftarkan endpoint callback QStash.
//
// Endpoint callback berada di bawah prefix versi API namun tidak memakai
// autentikasi sesi: signature QStash adalah satu-satunya pengesah. Callback
// pemberitahuan slip gaji hanya didaftarkan bila verifier dan pekerjanya
// keduanya dikonfigurasi.
func RegisterQStashRoutes(mux *http.ServeMux, options *QStashOptions) {
	if options == nil {
		return
	}
	api := qstashAPI{options: options}
	mux.Handle("POST "+APIPrefix+"/internal/qstash/audit/cleanup", http.HandlerFunc(api.auditCleanup))
	if options.PayrollVerifier != nil && options.PayrollReminder != nil {
		mux.Handle("POST "+APIPrefix+"/internal/qstash/payroll/notify", http.HandlerFunc(api.payrollNotify))
	}
}

// auditCleanup menjalankan pembersihan jejak audit yang kedaluwarsa.
//
// Pekerjaan ini idempoten: menjalankannya dua kali menghasilkan hasil yang
// sama karena baris yang sudah dihapus tidak akan ditemukan lagi.
func (api qstashAPI) auditCleanup(w http.ResponseWriter, r *http.Request) {
	if _, ok := api.verifiedBody(w, r, api.options.Verifier, JobKindAuditCleanup); !ok {
		return
	}
	if api.options.Cleaner == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
		return
	}

	deleted, err := api.options.Cleaner.Cleanup(r.Context())
	if err != nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
		return
	}

	note := "jejak audit kedaluwarsa dihapus"
	api.recordOutcome(r.Context(), audit.ActionAuditCleaned, r.Header.Get(HeaderMessageID), note)

	writeJSON(w, http.StatusOK, map[string]any{"deleted": deleted})
}

// payrollNotify menjalankan pemberitahuan ketersediaan slip gaji H+1.
//
// Pekerjaan ini idempoten: slip yang pesannya sudah diantrekan dilewati oleh
// kunci idempotensi antrean keluar, sehingga percobaan ulang QStash maupun
// eksekusi ganda tidak menghasilkan pemberitahuan ganda.
func (api qstashAPI) payrollNotify(w http.ResponseWriter, r *http.Request) {
	_, ok := api.verifiedBody(w, r, api.options.PayrollVerifier, JobKindPayrollNotify)
	if !ok {
		return
	}

	queued, err := api.options.PayrollReminder(r.Context())
	if err != nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
		return
	}

	note := fmt.Sprintf("pemberitahuan slip diantrekan: %d", queued)
	api.recordOutcome(r.Context(), audit.ActionPayrollNotified, r.Header.Get(HeaderMessageID), note)

	writeJSON(w, http.StatusOK, map[string]any{"queued": queued})
}

// verifiedBody membaca raw body, memverifikasi signature QStash, dan
// memeriksa jenis pekerjaan. Bila pemeriksaan gagal, fungsi menulis respons
// error dan mengembalikan ok=false.
func (api qstashAPI) verifiedBody(w http.ResponseWriter, r *http.Request, verifier *qstash.Verifier, expectedKind string) ([]byte, bool) {
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 64*1024))
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return nil, false
	}
	if verifier == nil {
		writeErrorResponse(w, http.StatusUnauthorized, apierror.CodeUnauthorized)
		return nil, false
	}
	if err := verifier.Verify(qstash.SignatureFromRequest(r), raw); err != nil {
		writeErrorResponse(w, http.StatusUnauthorized, apierror.CodeUnauthorized)
		return nil, false
	}

	var job qstashJob
	if err := json.Unmarshal(raw, &job); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return nil, false
	}
	if strings.TrimSpace(job.Kind) != expectedKind {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
		return nil, false
	}
	return raw, true
}

// recordOutcome mencatat hasil pekerjaan. Kegagalan pencatatan tidak
// menggagalkan pekerjaan karena jejak audit bukan hal kritis.
func (api qstashAPI) recordOutcome(ctx context.Context, action, messageID, note string) {
	if api.options.Recorder == nil {
		return
	}
	detail := note
	if messageID != "" {
		detail = note + " (pesan " + messageID + ")"
	}
	_ = api.options.Recorder.Record(ctx, audit.Entry{
		Action: action,
		Note:   &detail,
	})
}
