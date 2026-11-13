package repository

import (
	"os"
	"testing"

	"simpeg/backend/internal/crypto"
)

// TestMain mengonfigurasi kunci PII deterministik untuk seluruh tes paket
// ini. driver.Valuer/Scanner kolom terenkripsi dipanggil oleh GORM saat
// membangun dan membaca query, termasuk pada pengujian sqlmock.
func TestMain(m *testing.M) {
	key := make([]byte, 32)
	for index := range key {
		key[index] = byte(index + 1)
	}
	if err := crypto.Configure(key); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}
