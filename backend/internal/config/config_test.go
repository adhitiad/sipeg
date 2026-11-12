package config

import (
	"strings"
	"testing"
	"time"
)

func setBaseEnv(t *testing.T) {
	t.Helper()
	// DSN tanpa kredensial: konfigurasi hanya butuh nilai non-kosong, dan
	// pemindaian kebocoran connection string di CI menolak pola password.
	t.Setenv("DATABASE_URL", "postgres://simpeg@localhost:5432/simpeg")
	t.Setenv("APP_ENV", "development")
	t.Setenv("PORT", "")
	t.Setenv("BETTER_AUTH_SECRET", "")
	t.Setenv("TRUSTED_ORIGINS", "")
	t.Setenv("DB_MAX_OPEN_CONNS", "")
	t.Setenv("DB_MAX_IDLE_CONNS", "")
	t.Setenv("DB_CONN_MAX_LIFETIME", "")
	t.Setenv("DB_STATEMENT_TIMEOUT", "")
}

func TestLoadAppliesSafeDefaults(t *testing.T) {
	setBaseEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != "8080" {
		t.Fatalf("port = %q, want 8080", cfg.Port)
	}
	if cfg.AppEnv != "development" {
		t.Fatalf("app env = %q, want development", cfg.AppEnv)
	}
	if cfg.DBMaxOpenConns != defaultDBMaxOpenConns || cfg.DBMaxIdleConns != defaultDBMaxIdleConns {
		t.Fatalf("unexpected pool defaults: %+v", cfg)
	}
	if cfg.DBConnMaxLifetime != defaultDBConnMaxLifetime || cfg.DBStatementTimeout != defaultDBStatementTimeout {
		t.Fatalf("unexpected timeout defaults: %+v", cfg)
	}
	if len(cfg.TrustedOrigins) != 0 {
		t.Fatalf("expected no trusted origins, got %v", cfg.TrustedOrigins)
	}
	if cfg.SessionLifetime != defaultSessionLifetime {
		t.Fatalf("session lifetime = %v, want %v", cfg.SessionLifetime, defaultSessionLifetime)
	}
}

func TestLoadParsesTrustedOrigins(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("TRUSTED_ORIGINS", "https://a.example, https://b.example ,")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.TrustedOrigins) != 2 {
		t.Fatalf("trusted origins = %v, want 2 entries", cfg.TrustedOrigins)
	}
	if cfg.TrustedOrigins[1] != "https://b.example" {
		t.Fatalf("unexpected trusted origin: %q", cfg.TrustedOrigins[1])
	}
}

func TestLoadRejectsWildcardTrustedOrigin(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("TRUSTED_ORIGINS", "*")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "wildcard") {
		t.Fatalf("expected wildcard rejection, got: %v", err)
	}
}

func TestProductionRequiresStrongSecret(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("TRUSTED_ORIGINS", "https://simpeg.example")

	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "BETTER_AUTH_SECRET") {
		t.Fatalf("expected missing secret error, got: %v", err)
	}

	t.Setenv("BETTER_AUTH_SECRET", "pendek")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "at least") {
		t.Fatalf("expected weak secret error, got: %v", err)
	}
}

func TestProductionRejectsKnownWeakSecret(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("TRUSTED_ORIGINS", "https://simpeg.example")
	t.Setenv("BETTER_AUTH_SECRET", strings.Repeat("change-me", 6))

	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "well-known") {
		t.Fatalf("expected weak secret rejection, got: %v", err)
	}
}

func TestProductionRequiresTrustedOrigins(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("BETTER_AUTH_SECRET", strings.Repeat("x", 40))

	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "TRUSTED_ORIGINS") {
		t.Fatalf("expected missing origins error, got: %v", err)
	}
}

func TestProductionAcceptsValidConfiguration(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("BETTER_AUTH_SECRET", strings.Repeat("x", 40))
	t.Setenv("TRUSTED_ORIGINS", "https://simpeg.example")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.IsProduction() {
		t.Fatal("expected production mode")
	}
}

func TestLoadRejectsInvalidPoolValues(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("DB_MAX_IDLE_CONNS", "100")

	if _, err := Load(); err == nil {
		t.Fatal("expected pool validation error")
	}
}

func TestLoadParsesDurations(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("DB_CONN_MAX_LIFETIME", "90s")
	t.Setenv("DB_STATEMENT_TIMEOUT", "5s")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DBConnMaxLifetime != 90*time.Second || cfg.DBStatementTimeout != 5*time.Second {
		t.Fatalf("unexpected durations: %+v", cfg)
	}
}

func TestSenderAddressIsValidated(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("SMTP_HOST", "smtp.example.test")
	t.Setenv("SMTP_FROM", "no-reply@simpeg.unisubang.ac.id")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.SMTPEnabled() {
		t.Fatal("expected SMTP to be enabled")
	}
}

func TestSenderAddressRejectsInvalidValues(t *testing.T) {
	for _, invalid := range []string{
		"bukan-email",
		"@simpeg.unisubang.ac.id",
		"no-reply@",
		"no-reply@localhost",
		"no reply@simpeg.unisubang.ac.id",
	} {
		setBaseEnv(t)
		t.Setenv("SMTP_HOST", "smtp.example.test")
		t.Setenv("SMTP_FROM", invalid)

		if _, err := Load(); err == nil {
			t.Errorf("sender %q must be rejected", invalid)
		}
	}
}

func TestSenderAddressRequiredWhenHostConfigured(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("SMTP_HOST", "smtp.example.test")
	t.Setenv("SMTP_FROM", "")

	if _, err := Load(); err == nil {
		t.Fatal("expected error when SMTP_FROM is missing")
	}
}

func TestSMTPPortIsValidated(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("SMTP_HOST", "smtp.example.test")
	t.Setenv("SMTP_FROM", "no-reply@simpeg.unisubang.ac.id")
	t.Setenv("SMTP_PORT", "0")

	if _, err := Load(); err == nil {
		t.Fatal("expected error for invalid SMTP port")
	}
}

func TestIntegrationEnablementFlags(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("SMTP_HOST", "smtp.example.test")
	t.Setenv("SMTP_FROM", "no-reply@simpeg.unisubang.ac.id")
	t.Setenv("QSTASH_TOKEN", "token")
	t.Setenv("QSTASH_CURRENT_SIGNING_KEY", "key")
	t.Setenv("UPSTASH_REDIS_REST_URL", "https://redis.example.test")
	t.Setenv("UPSTASH_REDIS_REST_TOKEN", "token")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.SMTPEnabled() || !cfg.QStashEnabled() || !cfg.RedisEnabled() {
		t.Fatalf("expected all integrations enabled: %+v", cfg)
	}
}

func TestIntegrationSecretsMustBeComplete(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("QSTASH_TOKEN", "token")
	if _, err := Load(); err == nil {
		t.Fatal("expected error when signing key is missing")
	}

	setBaseEnv(t)
	t.Setenv("UPSTASH_REDIS_REST_URL", "https://redis.example.test")
	if _, err := Load(); err == nil {
		t.Fatal("expected error when redis token is missing")
	}
}

func TestAllowedEmailDomainsRequiresGoogleOAuth(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("ALLOWED_EMAIL_DOMAINS", "unisubang.ac.id")
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "GOOGLE_CLIENT_ID") {
		t.Fatalf("expected Google OAuth required error, got: %v", err)
	}
}

func TestInviteOnlyRequiresGoogleOAuth(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("INVITE_ONLY", "true")
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "GOOGLE_CLIENT_ID") {
		t.Fatalf("expected Google OAuth required error for invite only, got: %v", err)
	}
}

func TestGoogleOAuthConfigIsLoaded(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("GOOGLE_CLIENT_ID", "id-123")
	t.Setenv("GOOGLE_CLIENT_SECRET", "secret-xyz")
	t.Setenv("ALLOWED_EMAIL_DOMAINS", "unisubang.ac.id")
	t.Setenv("INVITE_ONLY", "false")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.GoogleClientID != "id-123" || cfg.GoogleClientSecret != "secret-xyz" {
		t.Fatalf("Google OAuth not loaded correctly: %+v", cfg)
	}
	if len(cfg.AllowedEmailDomains) != 1 || cfg.AllowedEmailDomains[0] != "unisubang.ac.id" {
		t.Fatalf("allowed domains not loaded: %v", cfg.AllowedEmailDomains)
	}
}

func TestSessionLifetimeDefaultIs23Hours(t *testing.T) {
	setBaseEnv(t)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.SessionLifetime != 23*time.Hour {
		t.Fatalf("expected session lifetime 23h, got %v", cfg.SessionLifetime)
	}
}
