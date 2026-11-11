package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Default nilai operasional yang aman bila env tidak disetel.
const (
	defaultPort               = "8080"
	defaultDBMaxOpenConns     = 25
	defaultDBMaxIdleConns     = 5
	defaultDBConnMaxLifetime  = 30 * time.Minute
	defaultDBStatementTimeout = 15 * time.Second
	defaultRateLimitRequests  = 120
	defaultRateLimitWindow    = time.Minute
	defaultRateLimitAuth      = 10
	defaultAuditRetentionDays = 90   // 3 bulan
	defaultRPOHours           = 24   // 1 hari
	defaultRTOHours           = 4    // 4 jam
	defaultSLOTarget          = 99.9 // 99.9%
)

// minSecretLength adalah panjang minimum shared secret produksi.
const minSecretLength = 32

// Config memuat seluruh konfigurasi runtime backend.
type Config struct {
	DatabaseURL      string
	AppEnv           string
	Port             string
	BetterAuthSecret string
	// TrustedOrigins adalah origin browser yang diizinkan memanggil API.
	// Kosong berarti tidak ada akses lintas origin.
	TrustedOrigins     []string
	DBMaxOpenConns     int
	DBMaxIdleConns     int
	DBConnMaxLifetime  time.Duration
	DBStatementTimeout time.Duration
	// RateLimitRequests dan RateLimitWindow membatasi jumlah permintaan per klien.
	RateLimitRequests int
	RateLimitWindow   time.Duration
	// RateLimitAuthRequests dan RateLimitAuthWindow berlaku untuk endpoint
	// autentikasi yang lebih sensitif.
	RateLimitAuthRequests int
	RateLimitAuthWindow   time.Duration
	// AppBaseURL adalah origin aplikasi untuk tautan pada pemberitahuan.
	AppBaseURL string

	// Konfigurasi SMTP untuk pengiriman email.
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	SMTPSecure   bool
	SMTPFrom     string

	// Konfigurasi QStash untuk pekerjaan asinkron.
	QStashURL               string
	QStashSchedulesURL      string
	QStashToken             string
	QStashCurrentSigningKey string
	QStashNextSigningKey    string

	// Konfigurasi Redis Upstash untuk cache data referensi.
	RedisURL   string
	RedisToken string

	// Konfigurasi penyedia hari libur nasional.
	HolidayAPIURL string
	HolidayAPIKey string

	// AuditRetentionDays adalah masa simpan jejak audit dalam hari. 0 = tidak ada pemotongan otomatis.
	AuditRetentionDays int

	// RPO adalah target waktu pemulihan data dalam jam.
	RPOHours int
	// RTO adalah target waktu pemulihan layanan dalam jam.
	RTOHours int
	// SLOTarget adalah target ketersediaan layanan dalam persen (99.9 = 99.9%).
	SLOTarget float64
}

// SMTPEnabled melapor apakah konfigurasi email sudah cukup untuk dipakai.
func (c Config) SMTPEnabled() bool {
	return c.SMTPHost != "" && c.SMTPFrom != ""
}

// QStashEnabled melapor apakah QStash sudah dikonfigurasi.
// QSTASH_URL boleh dikosongkan karena publisher memakai endpoint bawaan.
func (c Config) QStashEnabled() bool {
	return c.QStashToken != "" && c.QStashCurrentSigningKey != ""
}

// RedisEnabled melapor apakah Redis Upstash sudah dikonfigurasi.
func (c Config) RedisEnabled() bool {
	return c.RedisURL != "" && c.RedisToken != ""
}

// HolidayProviderEnabled melapor apakah penyedia hari libur sudah
// dikonfigurasi. Pembacaan kalender dari database tidak memerlukan ini.
func (c Config) HolidayProviderEnabled() bool {
	return c.HolidayAPIURL != "" && c.HolidayAPIKey != ""
}

// IsProduction melaporkan apakah backend berjalan pada mode production.
func (c Config) IsProduction() bool {
	return strings.EqualFold(c.AppEnv, "production")
}

// Load membaca konfigurasi dari environment dan memvalidasinya.
func Load() (Config, error) {
	cfg := Config{
		DatabaseURL:             strings.TrimSpace(os.Getenv("DATABASE_URL")),
		AppEnv:                  strings.TrimSpace(os.Getenv("APP_ENV")),
		Port:                    strings.TrimSpace(os.Getenv("PORT")),
		BetterAuthSecret:        strings.TrimSpace(os.Getenv("BETTER_AUTH_SECRET")),
		TrustedOrigins:          splitList(os.Getenv("TRUSTED_ORIGINS")),
		DBMaxOpenConns:          envInt("DB_MAX_OPEN_CONNS", defaultDBMaxOpenConns),
		DBMaxIdleConns:          envInt("DB_MAX_IDLE_CONNS", defaultDBMaxIdleConns),
		DBConnMaxLifetime:       envDuration("DB_CONN_MAX_LIFETIME", defaultDBConnMaxLifetime),
		DBStatementTimeout:      envDuration("DB_STATEMENT_TIMEOUT", defaultDBStatementTimeout),
		RateLimitRequests:       envInt("RATE_LIMIT_REQUESTS", defaultRateLimitRequests),
		RateLimitWindow:         envDuration("RATE_LIMIT_WINDOW", defaultRateLimitWindow),
		RateLimitAuthRequests:   envInt("RATE_LIMIT_AUTH_REQUESTS", defaultRateLimitAuth),
		RateLimitAuthWindow:     envDuration("RATE_LIMIT_AUTH_WINDOW", defaultRateLimitWindow),
		AppBaseURL:              strings.TrimSpace(os.Getenv("APP_BASE_URL")),
		SMTPHost:                strings.TrimSpace(os.Getenv("SMTP_HOST")),
		SMTPPort:                envInt("SMTP_PORT", 587),
		SMTPUser:                strings.TrimSpace(os.Getenv("SMTP_USER")),
		SMTPPassword:            os.Getenv("SMTP_PASS"),
		SMTPSecure:              envBool("SMTP_SECURE", true),
		SMTPFrom:                strings.TrimSpace(os.Getenv("SMTP_FROM")),
		QStashURL:               strings.TrimSpace(os.Getenv("QSTASH_URL")),
		QStashSchedulesURL:      strings.TrimSpace(os.Getenv("QSTASH_SCHEDULES_URL")),
		QStashToken:             strings.TrimSpace(os.Getenv("QSTASH_TOKEN")),
		QStashCurrentSigningKey: strings.TrimSpace(os.Getenv("QSTASH_CURRENT_SIGNING_KEY")),
		QStashNextSigningKey:    strings.TrimSpace(os.Getenv("QSTASH_NEXT_SIGNING_KEY")),
		RedisURL:                strings.TrimSpace(os.Getenv("UPSTASH_REDIS_REST_URL")),
		RedisToken:              strings.TrimSpace(os.Getenv("UPSTASH_REDIS_REST_TOKEN")),
		HolidayAPIURL:           strings.TrimSpace(os.Getenv("HOLIDAY_API_URL")),
		HolidayAPIKey:           strings.TrimSpace(os.Getenv("HOLIDAY_API_KEY")),
		AuditRetentionDays:      envInt("AUDIT_RETENTION_DAYS", defaultAuditRetentionDays),
		RPOHours:                envInt("RPO_HOURS", defaultRPOHours),
		RTOHours:                envInt("RTO_HOURS", defaultRTOHours),
		SLOTarget:               envFloat("SLO_TARGET", defaultSLOTarget),
	}

	if cfg.AppEnv == "" {
		cfg.AppEnv = "development"
	}
	if cfg.Port == "" {
		cfg.Port = defaultPort
	}
	if cfg.DatabaseURL == "" {
		return cfg, fmt.Errorf("DATABASE_URL is required")
	}
	if err := cfg.validate(); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// validate menolak konfigurasi yang tidak aman, terutama pada production.
func (c Config) validate() error {
	if c.DBMaxOpenConns < 1 {
		return fmt.Errorf("DB_MAX_OPEN_CONNS must be at least 1")
	}
	if c.DBMaxIdleConns < 0 || c.DBMaxIdleConns > c.DBMaxOpenConns {
		return fmt.Errorf("DB_MAX_IDLE_CONNS must be between 0 and DB_MAX_OPEN_CONNS")
	}
	if c.DBConnMaxLifetime <= 0 {
		return fmt.Errorf("DB_CONN_MAX_LIFETIME must be positive")
	}
	if c.DBStatementTimeout <= 0 {
		return fmt.Errorf("DB_STATEMENT_TIMEOUT must be positive")
	}
	if c.RateLimitRequests < 1 {
		return fmt.Errorf("RATE_LIMIT_REQUESTS must be at least 1")
	}
	if c.RateLimitWindow <= 0 {
		return fmt.Errorf("RATE_LIMIT_WINDOW must be positive")
	}
	if c.RateLimitAuthRequests < 1 {
		return fmt.Errorf("RATE_LIMIT_AUTH_REQUESTS must be at least 1")
	}
	if c.RateLimitAuthWindow <= 0 {
		return fmt.Errorf("RATE_LIMIT_AUTH_WINDOW must be positive")
	}
	if c.SMTPHost != "" && c.SMTPFrom == "" {
		return fmt.Errorf("SMTP_FROM is required when SMTP_HOST is configured")
	}
	if c.SMTPFrom != "" && !validEmailAddress(c.SMTPFrom) {
		return fmt.Errorf("SMTP_FROM must be a valid email address")
	}
	if c.SMTPHost != "" && c.SMTPPort < 1 || c.SMTPHost != "" && c.SMTPPort > 65535 {
		return fmt.Errorf("SMTP_PORT must be between 1 and 65535")
	}
	if c.QStashToken != "" && c.QStashCurrentSigningKey == "" {
		return fmt.Errorf("QSTASH_CURRENT_SIGNING_KEY is required when QSTASH_TOKEN is configured")
	}
	if c.RedisURL != "" && c.RedisToken == "" {
		return fmt.Errorf("UPSTASH_REDIS_REST_TOKEN is required when UPSTASH_REDIS_REST_URL is configured")
	}

	// Audit retention harus non-negatif.
	if c.AuditRetentionDays < 0 {
		return fmt.Errorf("AUDIT_RETENTION_DAYS must be non-negative")
	}
	if c.RPOHours < 1 {
		return fmt.Errorf("RPO_HOURS must be at least 1")
	}
	if c.RTOHours < 1 {
		return fmt.Errorf("RTO_HOURS must be at least 1")
	}
	if c.SLOTarget <= 0 || c.SLOTarget > 100 {
		return fmt.Errorf("SLO_TARGET must be between 0 and 100")
	}

	for _, origin := range c.TrustedOrigins {
		if origin == "*" {
			return fmt.Errorf("TRUSTED_ORIGINS must not contain a wildcard origin")
		}
	}

	if !c.IsProduction() {
		return nil
	}

	// Production tidak boleh berjalan dengan secret lemah atau tanpa daftar origin.
	if c.BetterAuthSecret == "" {
		return fmt.Errorf("BETTER_AUTH_SECRET is required in production")
	}
	if len(c.BetterAuthSecret) < minSecretLength {
		return fmt.Errorf("BETTER_AUTH_SECRET must be at least %d characters in production", minSecretLength)
	}
	if isKnownWeakSecret(c.BetterAuthSecret) {
		return fmt.Errorf("BETTER_AUTH_SECRET must not use a well-known default value")
	}
	if len(c.TrustedOrigins) == 0 {
		return fmt.Errorf("TRUSTED_ORIGINS is required in production")
	}
	return nil
}

// weakSecrets adalah nilai yang sering tertinggal di konfigurasi contoh.
var weakSecrets = map[string]bool{
	"change-me":          true,
	"changeme":           true,
	"secret":             true,
	"secret123":          true,
	"better-auth-secret": true,
	"development-secret": true,
	"insecure":           true,
}

// isKnownWeakSecret menolak secret yang memuat pola bawaan yang sering
// tertinggal di konfigurasi contoh, termasuk secret yang mengulang pola itu.
func isKnownWeakSecret(secret string) bool {
	lowered := strings.ToLower(secret)
	for pattern := range weakSecrets {
		if strings.Contains(lowered, pattern) {
			return true
		}
	}
	return false
}

// splitList memecah nilai env yang dipisahkan koma dan membuang entri kosong.
func splitList(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func envInt(name string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return parsed
}

func envDuration(name string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return parsed
}

// validEmailAddress memeriksa bentuk alamat email sederhana. Pemeriksaan
// lengkap dilakukan oleh pustaka email saat pengiriman.
func validEmailAddress(address string) bool {
	trimmed := strings.TrimSpace(address)
	at := strings.Index(trimmed, "@")
	if at <= 0 || at == len(trimmed)-1 {
		return false
	}
	local, domain := trimmed[:at], trimmed[at+1:]
	if strings.ContainsAny(local, " \t\r\n") || strings.ContainsAny(domain, " \t\r\n") {
		return false
	}
	if !strings.Contains(domain, ".") || strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return false
	}
	return true
}

func envBool(name string, fallback bool) bool {
	raw := strings.ToLower(strings.TrimSpace(os.Getenv(name)))
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return parsed
}

// envFloat membaca nilai float dari environment variable.
func envFloat(name string, fallback float64) float64 {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return fallback
	}
	return parsed
}
