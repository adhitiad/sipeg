package config

import (
	"encoding/base64"
	"fmt"
	"strings"
)

// loadPIIEncryptionKey mendekode PII_ENCRYPTION_KEY (base64, tepat 32 byte /
// AES-256). Kosong berarti tidak dikonfigurasi — hanya sah di luar
// production; jalur yang menyentuh PII akan gagal dengan galat eksplisit.
func loadPIIEncryptionKey(raw string) ([]byte, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	key, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		if key, err = base64.RawStdEncoding.DecodeString(raw); err != nil {
			return nil, fmt.Errorf("PII_ENCRYPTION_KEY bukan base64 yang valid")
		}
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("PII_ENCRYPTION_KEY harus tepat 32 byte ter-decode (dapat %d byte)", len(key))
	}
	return key, nil
}

// PIIEnabled melapor apakah kunci enkripsi PII dikonfigurasi.
func (c Config) PIIEnabled() bool {
	return len(c.PIIEncryptionKey) == 32
}
