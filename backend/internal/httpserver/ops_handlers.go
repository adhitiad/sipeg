package httpserver

import (
	"context"
	"net/http"
	"time"

	"simpeg/backend/internal/apierror"
	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/outbox"
)

// OutboxStatsReader membaca ringkasan antrean keluar untuk pemantauan
// operasional.
type OutboxStatsReader interface {
	Stats(ctx context.Context) (outbox.Stats, error)
}

// OpsStatusDTO adalah bentuk publik status operasional untuk pemantauan.
// Nilai ini hanya memuat agregat; isi payload, kunci idempotensi, dan
// identitas penerima tidak pernah disertakan.
type OpsStatusDTO struct {
	CheckedAt time.Time      `json:"checkedAt"`
	Outbox    OutboxStatsDTO `json:"outbox"`
}

// OutboxStatsDTO adalah bentuk publik ringkasan antrean keluar.
type OutboxStatsDTO struct {
	Pending            int64            `json:"pending"`
	Processing         int64            `json:"processing"`
	Done               int64            `json:"done"`
	Failed             int64            `json:"failed"`
	OldestPendingAt    *time.Time       `json:"oldestPendingAt,omitempty"`
	OldestPendingAgeMs *int64           `json:"oldestPendingAgeMs,omitempty"`
	FailedByKind       map[string]int64 `json:"failedByKind"`
}

// RegisterOpsRoutes mendaftarkan endpoint status operasional admin.
//
// Endpoint bersifat read-only dan dibatasi permission ops:read (admin).
// Akses tanpa izin ditolak 403 (deny-by-default).
func RegisterOpsRoutes(mux *http.ServeMux, resolver auth.SessionResolver, policy auth.Policy, outboxReader OutboxStatsReader) {
	if outboxReader == nil {
		return
	}
	api := opsAPI{outbox: outboxReader}
	protected := func(permission auth.Permission, handler http.Handler) http.Handler {
		return auth.Authenticate(resolver)(auth.RequirePermission(policy, permission, handler))
	}
	mux.Handle("GET "+APIPrefix+"/admin/ops/status", protected("ops:read", http.HandlerFunc(api.status)))
}

type opsAPI struct {
	outbox OutboxStatsReader
}

// status mengembalikan ringkasan status operasional saat ini.
func (api opsAPI) status(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()
	stats, err := api.outbox.Stats(r.Context())
	if err != nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
		return
	}

	dto := OpsStatusDTO{
		CheckedAt: now,
		Outbox: OutboxStatsDTO{
			Pending:      stats.Pending,
			Processing:   stats.Processing,
			Done:         stats.Done,
			Failed:       stats.Failed,
			FailedByKind: stats.FailedByKind,
		},
	}
	if stats.OldestPendingAt != nil {
		t := stats.OldestPendingAt.UTC()
		dto.Outbox.OldestPendingAt = &t
		age := now.Sub(t).Milliseconds()
		if age < 0 {
			age = 0
		}
		dto.Outbox.OldestPendingAgeMs = &age
	}
	if dto.Outbox.FailedByKind == nil {
		dto.Outbox.FailedByKind = map[string]int64{}
	}

	writeJSON(w, http.StatusOK, dto)
}
