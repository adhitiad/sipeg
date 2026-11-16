package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"simpeg/backend/internal/notify"
)

// Default nilai operasional yang aman bila env tidak disetel.
//
// Bawaan pool koneksi sengaja kecil. Pooler sesi Supabase membatasi jumlah
// koneksi concurrently per proyek; ketika batas itu terlampaui, PostgreSQL
// menolak koneksi baru dengan SQLSTATE XX000 (EMAXCONNSESSION) dan backend
// menjawab 503. Bawaan 10 connections memberi ruang di bawah batas pooler
// tanpa mengorbankan throughput pada beban wajar.
const (
	defaultPort               = "8080"
	defaultDBMaxOpenConns     = 10
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
	defaultSessionLifetime    = 23 * time.Hour
)

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

	// SessionLifetime adalah masa berlaku sesi yang diterbitkan backend
	// (opaque session).
	SessionLifetime time.Duration
	// PIIEncryptionKey adalah kunci AES-256 (32 byte) untuk enkripsi kolom
	// PII (user.nik, user.nip, user.npwp, RiwayatPendidikan.noIjazah).
	// Enkripsi dilakukan di lapisan aplikasi (driver.Valuer/Scanner).
	// Produksi mewajibkan variabel ini; kosong di luar produksi berarti
	// jalur PII tidak dapat dipakai.
	PIIEncryptionKey []byte
	// AppBaseURL adalah origin aplikasi untuk tautan pada pemberitahuan.
	AppBaseURL string
	// PayrollSlipOrgName adalah nama instansi yang dicetak pada kop slip
	// gaji. Kosong berarti slip menampilkan label generik.
	PayrollSlipOrgName string

	// Google OAuth configuration for Better Auth runtime.
	GoogleClientID     string
	GoogleClientSecret string

	// AllowedEmailDomains dibatasi pada domain email yang diizinkan untuk akses.
	// Kosong berarti tidak ada pembatasan domain.
	AllowedEmailDomains []string

	// InviteOnly mengaktifkan kebijakan undangan. Jika true, akses hanya
	// diizinkan bagi pengguna yang menerima undangan (selain domain allowlist).
	InviteOnly bool

	// Konfigurasi SMTP untuk pengiriman email.
	SMTP SMTPConfig

	// Konfigurasi QStash untuk pekerjaan asinkron.
	QStashURL               string
	QStashSchedulesURL      string
	QStashToken             string
	QStashCurrentSigningKey string
	QStashNextSigningKey    string

	// RabbitMQURL adalah alamat broker AMQP untuk mode self-hosted. Bila
	// diisi, pekerjaan terjadwal berpindah dari callback QStash ke
	// konsumen RabbitMQ (lihat internal/mq dan cmd/jobsched).
	RabbitMQURL string

	// Konfigurasi Redis Upstash untuk cache data referensi.
	RedisURL   string
	RedisToken string

	// Konfigurasi penyedia hari libur nasional.
	HolidayAPIURL string
	HolidayAPIKey string

	// Konfigurasi Cloudinary untuk penyimpanan berkas dokumen pegawai.
	Cloudinary CloudinaryConfig

	// AuditRetentionDays adalah masa simpan jejak audit dalam hari. 0 = tidak ada pemotongan otomatis.
	AuditRetentionDays int

	// RPO adalah target waktu pemulihan data dalam jam.
	RPOHours int
	// RTO adalah target waktu pemulihan layanan dalam jam.
	RTOHours int
	// SLOTarget adalah target ketersediaan layanan dalam persen (99.9 = 99.9%).
	SLOTarget float64
	// SLO memuat konfigurasi pengoleksi metrik dan pemeriksaan target
	// layanan. Lihat internal/config/slo.go.
	SLO SLOConfig
}

// SMTPEnabled melapor apakah konfigurasi email sudah cukup untuk dipakai.
func (c Config) SMTPEnabled() bool {
	return c.SMTP.Enabled()
}

// QStashEnabled melapor apakah QStash sudah dikonfigurasi.
// QSTASH_URL boleh dikosongkan karena publisher memakai endpoint bawaan.
func (c Config) QStashEnabled() bool {
	return c.QStashToken != "" && c.QStashCurrentSigningKey != ""
}

// RabbitMQEnabled melapor apakah broker RabbitMQ dikonfigurasi untuk
// pekerjaan terjadwal self-hosted.
func (c Config) RabbitMQEnabled() bool {
	return c.RabbitMQURL != ""
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
	piiEncryptionKey, err := loadPIIEncryptionKey(os.Getenv("PII_ENCRYPTION_KEY"))
	if err != nil {
		return Config{}, err
	}
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
		PIIEncryptionKey:        piiEncryptionKey,
		SessionLifetime:         envDuration("SESSION_LIFETIME", defaultSessionLifetime),
		AppBaseURL:              strings.TrimSpace(os.Getenv("APP_BASE_URL")),
		PayrollSlipOrgName:      strings.TrimSpace(os.Getenv("PAYROLL_SLIP_ORG_NAME")),
		GoogleClientID:          strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID")),
		GoogleClientSecret:      strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_SECRET")),
		AllowedEmailDomains:     splitList(os.Getenv("ALLOWED_EMAIL_DOMAINS")),
		InviteOnly:              envBool("INVITE_ONLY", false),
		SMTP:                    loadSMTPConfig(),
		QStashURL:               strings.TrimSpace(os.Getenv("QSTASH_URL")),
		QStashSchedulesURL:      strings.TrimSpace(os.Getenv("QSTASH_SCHEDULES_URL")),
		QStashToken:             strings.TrimSpace(os.Getenv("QSTASH_TOKEN")),
		QStashCurrentSigningKey: strings.TrimSpace(os.Getenv("QSTASH_CURRENT_SIGNING_KEY")),
		QStashNextSigningKey:    strings.TrimSpace(os.Getenv("QSTASH_NEXT_SIGNING_KEY")),
		RabbitMQURL:             strings.TrimSpace(os.Getenv("RABBITMQ_URL")),
		RedisURL:                strings.TrimSpace(os.Getenv("UPSTASH_REDIS_REST_URL")),
		RedisToken:              strings.TrimSpace(os.Getenv("UPSTASH_REDIS_REST_TOKEN")),
		HolidayAPIURL:           strings.TrimSpace(os.Getenv("HOLIDAY_API_URL")),
		HolidayAPIKey:           strings.TrimSpace(os.Getenv("HOLIDAY_API_KEY")),
		Cloudinary: CloudinaryConfig{
			CloudName: strings.TrimSpace(os.Getenv("CLOUDINARY_CLOUD_NAME")),
			APIKey:    strings.TrimSpace(os.Getenv("CLOUDINARY_API_KEY")),
			APISecret: strings.TrimSpace(os.Getenv("CLOUDINARY_API_SECRET")),
		},
		AuditRetentionDays: envInt("AUDIT_RETENTION_DAYS", defaultAuditRetentionDays),
		RPOHours:           envInt("RPO_HOURS", defaultRPOHours),
		RTOHours:           envInt("RTO_HOURS", defaultRTOHours),
		SLOTarget:          envFloat("SLO_TARGET", defaultSLOTarget),
		SLO:                loadSLOConfig(),
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

// validEmailAddress memeriksa bentuk alamat email sederhana.
//
// Bentuk polos dan bentuk "Nama Tampilan <alamat@domain>" sama-sama
// diterima karena keduanya baku RFC 5322 dan lazim dipakai pada header
// From. Pemeriksaan lengkap dilakukan oleh pustaka email saat pengiriman.
func validEmailAddress(address string) bool {
	return notify.ValidateAddress(address) == nil
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
