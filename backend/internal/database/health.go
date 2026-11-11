package database

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

// DependencyName adalah nama dependency database pada status readiness.
const DependencyName = "postgres"

// ErrPoolSaturated menandakan pool koneksi tidak lagi punya koneksi bebas.
// Backend masih dapat melayani permintaan, namun capacidad cadangan habis.
var ErrPoolSaturated = errors.New("database connection pool is saturated")

// PingChecker memeriksa koneksi PostgreSQL untuk endpoint readiness.
type PingChecker struct {
	DB *gorm.DB
}

// Name mengembalikan nama dependency untuk ditampilkan pada status readiness.
func (c PingChecker) Name() string { return DependencyName }

// Check menjalankan ping database dengan batas waktu dari context dan memeriksa
// apakah pool koneksi sedang jenuh.
func (c PingChecker) Check(ctx context.Context) error {
	if c.DB == nil {
		return errors.New("database handle is not configured")
	}
	sqlDB, err := c.DB.DB()
	if err != nil {
		return err
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		return err
	}
	stats, err := Stats(c.DB)
	if err != nil {
		return err
	}
	if stats.Saturated() {
		return ErrPoolSaturated
	}
	return nil
}
