package audit

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// ErrRetentionNotSet menandakan retention days belum dikonfigurasi (0 = tidak ada pemotongan).
var ErrRetentionNotSet = errors.New("audit retention not configured")

// Cleaner membersihkan entri audit yang lebih lama dari retensi yang dikonfigurasi.
type Cleaner struct {
	db *gorm.DB
	// RetentionDays adalah umur simpan jejak audit dalam hari. 0 = tidak ada pemotongan.
	RetentionDays int
}

// NewCleaner membuat cleaner audit.
func NewCleaner(db *gorm.DB, retentionDays int) *Cleaner {
	return &Cleaner{db: db, RetentionDays: retentionDays}
}

// Cleanup menghapus entri audit yang lebih lama dari RetentionDays hari.
// Mengembalikan jumlah baris yang dihapus. Jika RetentionDays <= 0, tidak ada
// baris yang dihapus dan fungsi mengembalikan 0, nil.
func (c *Cleaner) Cleanup(ctx context.Context) (int64, error) {
	if c.RetentionDays <= 0 {
		return 0, nil
	}
	if c.db == nil {
		return 0, errors.New("database handle is required")
	}

	cutoff := c.ExpiredSince()
	result := c.db.WithContext(ctx).Exec(
		"DELETE FROM \"auditLog\" WHERE \"createdAt\" < ?",
		cutoff,
	)
	if result.Error != nil {
		return 0, fmt.Errorf("delete expired audit logs: %w", result.Error)
	}
	return result.RowsAffected, nil
}

// ExpiredSince mengembalikan waktu batas: entri lebih lama dari waktu ini akan
// dihapus oleh Cleanup. Mengembalikan waktu nol jika RetentionDays <= 0.
func (c *Cleaner) ExpiredSince() time.Time {
	if c.RetentionDays <= 0 {
		return time.Time{}
	}
	return time.Now().Add(-time.Duration(c.RetentionDays) * 24 * time.Hour)
}
