package config

import (
	"fmt"
	"strings"
)

// minSecretLength adalah panjang minimum shared secret produksi.
const minSecretLength = 32

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
	if c.SessionLifetime <= 0 {
		return fmt.Errorf("SESSION_LIFETIME must be positive")
	}
	if err := c.SMTP.validate(); err != nil {
		return err
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

	// If email domain allowlist or invite-only mode is enabled, Google OAuth
	// must be configured. This ensures eligibility checks have a provider.
	if len(c.AllowedEmailDomains) > 0 || c.InviteOnly {
		if c.GoogleClientID == "" || c.GoogleClientSecret == "" {
			return fmt.Errorf("GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET are required when ALLOWED_EMAIL_DOMAINS is set or INVITE_ONLY is true")
		}
	}

	if !c.IsProduction() {
		return nil
	}

	// Production tidak boleh berjalan dengan secret lemah atau tanpa daftar origin.
	if len(c.PIIEncryptionKey) == 0 {
		return fmt.Errorf("PII_ENCRYPTION_KEY is required in production")
	}
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
