package database

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// PoolOptions mengatur batas koneksi dan batas waktu statement database.
type PoolOptions struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	// StatementTimeout membatasi durasi satu statement. Zero berarti tanpa batas.
	StatementTimeout time.Duration
}

// ConfigurePool menerapkan pengaturan pool koneksi dan batas waktu statement.
func ConfigurePool(db *gorm.DB, options PoolOptions) error {
	if db == nil {
		return fmt.Errorf("database handle is required")
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if options.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(options.MaxOpenConns)
	}
	if options.MaxIdleConns >= 0 {
		sqlDB.SetMaxIdleConns(options.MaxIdleConns)
	}
	if options.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(options.ConnMaxLifetime)
	}
	return nil
}

// PoolStats merangkum kondisi pool koneksi untuk keperluan readiness.
type PoolStats struct {
	OpenConnections int
	InUse           int
	Idle            int
	MaxOpen         int
}

// Saturated melaporkan apakah pool koneksi sedang tidak punya koneksi bebas.
func (s PoolStats) Saturated() bool {
	return s.MaxOpen > 0 && s.InUse >= s.MaxOpen
}

// Stats mengembalikan ringkasan pool koneksi saat ini.
func Stats(db *gorm.DB) (PoolStats, error) {
	if db == nil {
		return PoolStats{}, fmt.Errorf("database handle is required")
	}
	sqlDB, err := db.DB()
	if err != nil {
		return PoolStats{}, err
	}
	stats := sqlDB.Stats()
	return PoolStats{
		OpenConnections: stats.OpenConnections,
		InUse:           stats.InUse,
		Idle:            stats.Idle,
		MaxOpen:         stats.MaxOpenConnections,
	}, nil
}
