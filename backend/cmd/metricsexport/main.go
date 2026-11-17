// Package main menyediakan exporter metrik Prometheus untuk SIMPEG.
//
// Exporter ini membaca agregat operasional (outbox, metrik SLO) langsung
// dari PostgreSQL dan mengeksposnya pada endpoint /metrics dalam format
// Prometheus text. Tujuan: sinyal aman untuk Prometheus tanpa membocorkan
// payload, dedupe key, penerima, atau data pegawai. Token/DSN dibaca dari
// environment; tidak ada kredensial di repo.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"simpeg/backend/internal/cache"
	"simpeg/backend/internal/database"
	"simpeg/backend/internal/outbox"
	"simpeg/backend/internal/slo/app"
	slorepo "simpeg/backend/internal/slo/repository"

	"gorm.io/gorm"
)

func main() {
	dsn := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_URL wajib diisi")
		os.Exit(1)
	}
	db, err := database.OpenPostgres(dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gagal membuka database: %v\n", err)
		os.Exit(1)
	}

	port := strings.TrimSpace(os.Getenv("EXPORTER_PORT"))
	if port == "" {
		port = "9101"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		renderMetrics(w, r, db)
	})
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	addr := ":" + port
	fmt.Printf("metricsexport listening on %s\n", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}

func renderMetrics(w http.ResponseWriter, _ *http.Request, db *gorm.DB) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var b strings.Builder

	// Outbox aggregates.
	reader := outbox.NewGormStore(db)
	stats, err := reader.Stats(ctx)
	if err != nil {
		fmt.Fprintf(&b, "# error reading outbox stats: %v\n", err)
	} else {
		b.WriteString("# HELP simpeg_outbox_messages Jumlah pesan antrean keluar per status.\n")
		b.WriteString("# TYPE simpeg_outbox_messages gauge\n")
		b.WriteString(fmt.Sprintf("simpeg_outbox_messages{status=\"pending\"} %d\n", stats.Pending))
		b.WriteString(fmt.Sprintf("simpeg_outbox_messages{status=\"processing\"} %d\n", stats.Processing))
		b.WriteString(fmt.Sprintf("simpeg_outbox_messages{status=\"done\"} %d\n", stats.Done))
		b.WriteString(fmt.Sprintf("simpeg_outbox_messages{status=\"failed\"} %d\n", stats.Failed))
		if stats.OldestPendingAt != nil {
			age := time.Now().UTC().Sub(stats.OldestPendingAt.UTC()).Seconds()
			if age < 0 {
				age = 0
			}
			b.WriteString("# HELP simpeg_outbox_oldest_pending_age_seconds Usia pesan pending tertua (detik).\n")
			b.WriteString("# TYPE simpeg_outbox_oldest_pending_age_seconds gauge\n")
			b.WriteString(fmt.Sprintf("simpeg_outbox_oldest_pending_age_seconds %.3f\n", age))
		}
		if len(stats.FailedByKind) > 0 {
			b.WriteString("# HELP simpeg_outbox_failed_by_kind Jumlah pesan FAILED per jenis pekerjaan.\n")
			b.WriteString("# TYPE simpeg_outbox_failed_by_kind gauge\n")
			for kind, count := range stats.FailedByKind {
				b.WriteString(fmt.Sprintf("simpeg_outbox_failed_by_kind{kind=%q} %d\n", kind, count))
			}
			// SMTP failures = sum of kinds prefixed with email.
			var smtpFailed int64
			for kind, count := range stats.FailedByKind {
				if strings.HasPrefix(strings.ToLower(kind), "email.") {
					smtpFailed += count
				}
			}
			b.WriteString("# HELP simpeg_smtp_failed_total Jumlah pesan email yang gagal permanen.\n")
			b.WriteString("# TYPE simpeg_smtp_failed_total gauge\n")
			b.WriteString(fmt.Sprintf("simpeg_smtp_failed_total %d\n", smtpFailed))
		}
	}

	// SLO aggregates (last 24h). Only if sloMetric has rows.
	repo := slorepo.NewGormStore(db)
	dashboard := app.NewDashboardService(app.DashboardOptions{Repository: repo})
	snapshot, err := dashboard.Snapshot(ctx, 24*time.Hour)
	if err == nil && snapshot.Requests > 0 {
		b.WriteString("# HELP simpeg_slo_requests_total Jumlah permintaan pada jendela SLO 24 jam.\n")
		b.WriteString("# TYPE simpeg_slo_requests_total gauge\n")
		b.WriteString(fmt.Sprintf("simpeg_slo_requests_total %d\n", snapshot.Requests))
		b.WriteString("# HELP simpeg_slo_errors_total Jumlah galat 5xx pada jendela SLO 24 jam.\n")
		b.WriteString("# TYPE simpeg_slo_errors_total gauge\n")
		b.WriteString(fmt.Sprintf("simpeg_slo_errors_total %d\n", snapshot.Errors))
		b.WriteString("# HELP simpeg_slo_error_rate Laju galat pada jendela SLO 24 jam.\n")
		b.WriteString("# TYPE simpeg_slo_error_rate gauge\n")
		b.WriteString(fmt.Sprintf("simpeg_slo_error_rate %.6f\n", snapshot.ErrorRate))
		b.WriteString("# HELP simpeg_slo_p95_ms Latensi p95 (ms) pada jendela SLO 24 jam.\n")
		b.WriteString("# TYPE simpeg_slo_p95_ms gauge\n")
		b.WriteString(fmt.Sprintf("simpeg_slo_p95_ms %d\n", snapshot.P95Ms))
		b.WriteString("# HELP simpeg_slo_p99_ms Latensi p99 (ms) pada jendela SLO 24 jam.\n")
		b.WriteString("# TYPE simpeg_slo_p99_ms gauge\n")
		b.WriteString(fmt.Sprintf("simpeg_slo_p99_ms %d\n", snapshot.P99Ms))
	}

	// Health dependency eksternal (Redis, SMTP) — agregat boolean.
	b.WriteString("# HELP simpeg_dependency_up Status kesehatan dependency eksternal (1=sehat).\n")
	b.WriteString("# TYPE simpeg_dependency_up gauge\n")
	// Redis
	redisUp := 0
	if base := strings.TrimSpace(os.Getenv("UPSTASH_REDIS_REST_URL")); base != "" {
		if tok := strings.TrimSpace(os.Getenv("UPSTASH_REDIS_REST_TOKEN")); tok != "" {
			c := cache.NewRESTClient(base, tok)
			if err := c.Set(ctx, "__metricsexport_ping__", "1", 10*time.Second); err == nil {
				_ = c.Delete(ctx, "__metricsexport_ping__")
				redisUp = 1
			}
		}
	}
	b.WriteString(fmt.Sprintf("simpeg_dependency_up{dependency=\"redis\"} %d\n", redisUp))
	// SMTP (TCP connect)
	smtpUp := 0
	if host := strings.TrimSpace(os.Getenv("SMTP_HOST")); host != "" {
		port := strings.TrimSpace(os.Getenv("SMTP_PORT"))
		if port == "" {
			port = "587"
		}
		conn, err := net.DialTimeout("tcp", host+":"+port, 5*time.Second)
		if err == nil {
			conn.Close()
			smtpUp = 1
		}
	}
	b.WriteString(fmt.Sprintf("simpeg_dependency_up{dependency=\"smtp\"} %d\n", smtpUp))

	// Keep JSON encoding import used? ensure no unused.
	_ = json.Marshal

	_, _ = w.Write([]byte(b.String()))
}
