package config

import (
	"os"
	"time"
)

// Nilai bawaan konfigurasi Service Level Objective.
const (
	defaultSLOWindowMinutes   = 5
	defaultSLOP95TargetMS     = 800
	defaultSLOErrorRateTarget = 0.01
	defaultSLOMinRequests     = 20
	defaultSLORetentionDays   = 30
)

// SLOConfig memuat konfigurasi Service Level Objective.
//
// Pengoleksi metrik dan pemeriksaan SLO hanya aktif bila Enabled disetel.
// Tanpa itu, backend tetap berfungsi penuh tanpa menulis metrik apa pun.
type SLOConfig struct {
	// Enabled mengaktifkan pengoleki metrik dan pemeriksaan SLO terjadwal.
	Enabled bool
	// WindowMinutes adalah panjang jendela agregasi metrik dalam menit.
	WindowMinutes int
	// P95TargetMS adalah latensi persentil 95 maksimum dalam milidetik.
	P95TargetMS int
	// ErrorRateTarget adalah laju kesalahan maksimum, dinyatakan sebagai
	// pecahan 0 sampai 1.
	ErrorRateTarget float64
	// MinRequests adalah jumlah permintaan minimum pada satu jendela
	// sebelum target dianggap terlampaui. Jendela dengan permintaan lebih
	// sedikit dianggap terlalu kecil untuk dinilai.
	MinRequests int64
	// AlertRecipients adalah daftar alamat email yang menerima alert.
	AlertRecipients []string
	// RetentionDays adalah masa simpan metrik dalam hari.
	RetentionDays int
}

// loadSLOConfig membaca konfigurasi SLO dari environment.
func loadSLOConfig() SLOConfig {
	target := envFloat("SLO_ERROR_RATE_TARGET", defaultSLOErrorRateTarget)
	if target < 0 || target > 1 {
		target = defaultSLOErrorRateTarget
	}
	window := envInt("SLO_WINDOW_MINUTES", defaultSLOWindowMinutes)
	if window < 1 {
		window = defaultSLOWindowMinutes
	}
	p95 := envInt("SLO_P95_TARGET_MS", defaultSLOP95TargetMS)
	if p95 < 1 {
		p95 = defaultSLOP95TargetMS
	}
	minRequests := envInt("SLO_MIN_REQUESTS", defaultSLOMinRequests)
	if minRequests < 1 {
		minRequests = defaultSLOMinRequests
	}
	retention := envInt("SLO_RETENTION_DAYS", defaultSLORetentionDays)
	if retention < 1 {
		retention = defaultSLORetentionDays
	}
	return SLOConfig{
		Enabled:         envBool("SLO_ENABLED", false),
		WindowMinutes:   window,
		P95TargetMS:     p95,
		ErrorRateTarget: target,
		MinRequests:     int64(minRequests),
		AlertRecipients: splitList(os.Getenv("SLO_ALERT_RECIPIENTS")),
		RetentionDays:   retention,
	}
}

// SLOEnabled melapor apakah pengoleki metrik SLO aktif.
func (c Config) SLOEnabled() bool {
	return c.SLO.Enabled
}

// SLOWindow mengembalikan panjang jendela agregasi metrik SLO.
func (c Config) SLOWindow() time.Duration {
	return time.Duration(c.SLO.WindowMinutes) * time.Minute
}
