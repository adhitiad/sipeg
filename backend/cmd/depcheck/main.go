// Package main menyediakan perkakas verifikasi dependensi terautentikasi
// (PostgreSQL, Redis/Upstash, SMTP) untuk SIMPEG. Tidak memuat kredensial
// di repo; seluruh nilai dibaca dari environment.
package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/smtp"
	"os"
	"strings"
	"time"

	"simpeg/backend/internal/cache"
	"simpeg/backend/internal/database"

	"gorm.io/gorm"
)

type checkResult struct {
	Name    string `json:"name"`
	OK      bool   `json:"ok"`
	Detail  string `json:"detail,omitempty"`
	Latency string `json:"latency,omitempty"`
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	results := []checkResult{}
	allOK := true

	// PostgreSQL
	dbRes := checkPostgres(ctx)
	results = append(results, dbRes)
	if !dbRes.OK {
		allOK = false
	}

	// Redis (Upstash REST)
	redisRes := checkRedis(ctx)
	results = append(results, redisRes)
	if !redisRes.OK {
		allOK = false
	}

	// SMTP
	smtpRes := checkSMTP()
	results = append(results, smtpRes)
	if !smtpRes.OK {
		allOK = false
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(map[string]any{
		"checkedAt": time.Now().UTC().Format(time.RFC3339),
		"allOk":     allOK,
		"checks":    results,
	})

	if !allOK {
		os.Exit(1)
	}
}

func checkPostgres(ctx context.Context) checkResult {
	dsn := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dsn == "" {
		return checkResult{Name: "postgres", OK: false, Detail: "DATABASE_URL kosong"}
	}
	start := time.Now()
	db, err := database.OpenPostgres(dsn)
	if err != nil {
		return checkResult{Name: "postgres", OK: false, Detail: err.Error()}
	}
	sqlDB, err := db.DB()
	if err != nil {
		return checkResult{Name: "postgres", OK: false, Detail: err.Error()}
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		return checkResult{Name: "postgres", OK: false, Detail: err.Error()}
	}
	// verify goose version readable
	var version int
	row := sqlDB.QueryRowContext(ctx, `SELECT COALESCE((SELECT version_id FROM goose_db_version ORDER BY id DESC LIMIT 1),0)`)
	if err := row.Scan(&version); err != nil {
		// non-fatal: maybe table absent
		_ = err
	}
	_ = gorm.ErrRecordNotFound
	lat := time.Since(start)
	return checkResult{Name: "postgres", OK: true, Detail: fmt.Sprintf("goose_version=%d", version), Latency: lat.String()}
}

func checkRedis(ctx context.Context) checkResult {
	base := strings.TrimSpace(os.Getenv("UPSTASH_REDIS_REST_URL"))
	token := strings.TrimSpace(os.Getenv("UPSTASH_REDIS_REST_TOKEN"))
	if base == "" || token == "" {
		return checkResult{Name: "redis", OK: false, Detail: "UPSTASH_REDIS_REST_URL/TOKEN kosong (disabled)"}
	}
	start := time.Now()
	client := cache.NewRESTClient(base, token)
	// Upstash REST ping via GET /ping is not in client; use Get on a sentinel key (may miss).
	_, err := client.Get(ctx, "__depcheck_ping__")
	// A miss returns nil/empty usually; treat connection/auth errors as failure.
	// We also issue a raw ping by checking status via a HEAD-less approach: try Set+Delete roundtrip.
	if err != nil {
		// try ping endpoint manually
	}
	// Perform a safe roundtrip to validate auth + connectivity.
	if err := client.Set(ctx, "__depcheck_ping__", "1", 10*time.Second); err != nil {
		return checkResult{Name: "redis", OK: false, Detail: err.Error()}
	}
	_ = client.Delete(ctx, "__depcheck_ping__")
	lat := time.Since(start)
	return checkResult{Name: "redis", OK: true, Detail: "roundtrip ok", Latency: lat.String()}
}

func checkSMTP() checkResult {
	host := strings.TrimSpace(os.Getenv("SMTP_HOST"))
	if host == "" {
		return checkResult{Name: "smtp", OK: false, Detail: "SMTP_HOST kosong (disabled)"}
	}
	port := strings.TrimSpace(os.Getenv("SMTP_PORT"))
	if port == "" {
		port = "587"
	}
	user := strings.TrimSpace(os.Getenv("SMTP_USER"))
	pass := os.Getenv("SMTP_PASS")
	secure := strings.TrimSpace(os.Getenv("SMTP_SECURE"))

	start := time.Now()
	addr := host + ":" + port

	useTLS := false
	if secure == "true" {
		useTLS = true
	} else if secure == "" && port == "465" {
		useTLS = true
	}

	var c *smtp.Client
	var err error
	if useTLS {
		conn, derr := tls.Dial("tcp", addr, &tls.Config{ServerName: host})
		if derr != nil {
			return checkResult{Name: "smtp", OK: false, Detail: derr.Error()}
		}
		c, err = smtp.NewClient(conn, host)
		if err != nil {
			conn.Close()
			return checkResult{Name: "smtp", OK: false, Detail: err.Error()}
		}
	} else {
		conn, derr := net.DialTimeout("tcp", addr, 10*time.Second)
		if derr != nil {
			return checkResult{Name: "smtp", OK: false, Detail: derr.Error()}
		}
		c, err = smtp.NewClient(conn, host)
		if err != nil {
			conn.Close()
			return checkResult{Name: "smtp", OK: false, Detail: err.Error()}
		}
		if ok, _ := c.Extension("STARTTLS"); ok {
			if err := c.StartTLS(&tls.Config{ServerName: host}); err != nil {
				c.Close()
				return checkResult{Name: "smtp", OK: false, Detail: "starttls: " + err.Error()}
			}
		}
	}
	defer c.Close()

	if user != "" {
		auth := smtp.PlainAuth("", user, pass, host)
		if err := c.Auth(auth); err != nil {
			return checkResult{Name: "smtp", OK: false, Detail: "auth: " + err.Error()}
		}
	}
	lat := time.Since(start)
	return checkResult{Name: "smtp", OK: true, Detail: "connect+auth ok", Latency: lat.String()}
}
