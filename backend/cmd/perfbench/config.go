package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"simpeg/backend/internal/database"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// envOr mengembalikan nilai environment atau nilai bawaan bila kosong.
func envOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

// requireEnv membaca nilai environment yang wajib ada.
func requireEnv(key string) (string, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return "", fmt.Errorf("%s wajib diisi", key)
	}
	return value, nil
}

// envInt membaca bilangan bulat dari environment dengan nilai bawaan.
func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

// openDB membuka koneksi PostgreSQL dengan pool yang cukup untuk pengukuran
// beban. Pool produksi (DB_MAX_OPEN_CONNS) dipakai bila tersedia.
func openDB(ctx context.Context) (*gorm.DB, error) {
	dsn, err := requireEnv("DATABASE_URL")
	if err != nil {
		return nil, err
	}
	db, err := database.OpenPostgres(dsn)
	if err != nil {
		return nil, fmt.Errorf("buka koneksi database: %w", err)
	}
	// Logger bawaan GORM mencetak peringatan kueri lambat ke stderr. Pada
	// alat pengukuran log itu menutupi hasil, sehingga dimatikan.
	db.Logger = logger.Discard
	maxOpen := envInt("DB_MAX_OPEN_CONNS", 25)
	if err := database.ConfigurePool(db, database.PoolOptions{
		MaxOpenConns:    maxOpen,
		MaxIdleConns:    envInt("DB_MAX_IDLE_CONNS", 5),
		ConnMaxLifetime: 30 * time.Minute,
	}); err != nil {
		return nil, fmt.Errorf("konfigurasi pool: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("ambil handle database: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		return nil, errors.New("ping database gagal")
	}
	return db, nil
}

// backendDir mencari direktori modul backend dari lokasi kerja saat ini.
func backendDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for attempt := 0; attempt < 5; attempt++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", errors.New("direktori modul backend tidak ditemukan")
}

// buildServer membangun binari cmd/server ke path sementara. Server asli
// dipakai agar pengukuran memakai rantai handler produksi tanpa duplikasi.
func buildServer() (string, error) {
	dir, err := backendDir()
	if err != nil {
		return "", err
	}
	output := filepath.Join(os.TempDir(), fmt.Sprintf("simpeg-server-%d.exe", os.Getpid()))
	cmd := exec.Command("go", "build", "-o", output, "./cmd/server")
	cmd.Dir = dir
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("build cmd/server: %w", err)
	}
	return output, nil
}
