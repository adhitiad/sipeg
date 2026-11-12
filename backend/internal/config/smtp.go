package config

import (
	"fmt"
	"os"
	"strings"
)

// smtpImplicitTLSPort adalah porta server yang menerima TLS langsung sejak
// koneksi pertama, tanpa perlu perintah STARTTLS.
const smtpImplicitTLSPort = 465

// SMTPConfig memuat konfigurasi pengiriman email.
//
// Mode keamanan ditentukan oleh nilai Secure:
//   - Secure benar berarti TLS langsung sejak koneksi pertama. Mode ini
//     dipakai server pada porta 465.
//   - Secure salah berarti koneksi polos yang dinaikkan ke TLS lewat
//     perintah STARTTLS bila server menawarkannya. Mode ini dipakai server
//     pada porta 587.
//
// Bila SMTP_SECURE tidak disetel, mode ditebak dari porta: porta 465 memakai
// TLS langsung, porta lain memakai STARTTLS.
type SMTPConfig struct {
	// Host adalah nama host server SMTP.
	Host string
	// Port adalah porta server SMTP.
	Port int
	// User adalah nama pengguna autentikasi. Kosong berarti tanpa
	// autentikasi.
	User string
	// Password adalah sandi autentikasi.
	Password string
	// Secure menentukan mode keamanan koneksi.
	Secure bool
	// From adalah alamat pengirim.
	From string
}

// loadSMTPConfig membaca konfigurasi SMTP dari environment.
func loadSMTPConfig() SMTPConfig {
	port := envInt("SMTP_PORT", 587)
	return SMTPConfig{
		Host:     strings.TrimSpace(os.Getenv("SMTP_HOST")),
		Port:     port,
		User:     strings.TrimSpace(os.Getenv("SMTP_USER")),
		Password: os.Getenv("SMTP_PASS"),
		Secure:   envBool("SMTP_SECURE", port == smtpImplicitTLSPort),
		From:     strings.TrimSpace(os.Getenv("SMTP_FROM")),
	}
}

// Enabled melapor apakah konfigurasi email sudah cukup untuk dipakai.
func (c SMTPConfig) Enabled() bool {
	return c.Host != "" && c.From != ""
}

// validate menolak konfigurasi SMTP yang tidak dapat dipakai.
func (c SMTPConfig) validate() error {
	if c.Host != "" && c.From == "" {
		return fmt.Errorf("SMTP_FROM is required when SMTP_HOST is configured")
	}
	if c.From != "" && !validEmailAddress(c.From) {
		return fmt.Errorf("SMTP_FROM must be a valid email address")
	}
	if c.Host != "" && (c.Port < 1 || c.Port > 65535) {
		return fmt.Errorf("SMTP_PORT must be between 1 and 65535")
	}
	return nil
}
