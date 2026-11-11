package holiday

import (
	"context"
	"fmt"
	"time"

	"simpeg/backend/internal/audit"
	"simpeg/backend/internal/models"

	"gorm.io/gorm"
)

// GormStore menyimpan kalender hari libur pada tabel `HariLibur`.
type GormStore struct {
	DB *gorm.DB
}

// ReplaceYear mengganti seluruh entri kalender sebuah tahun.
//
// Penghapusan dan penulisan berada dalam satu transaksi sehingga pembaca
// tidak pernah melihat kalender setengah terisi. Jejak audit ditulis pada
// transaksi yang sama. Nilai balik adalah jumlah entri yang tersimpan.
func (s GormStore) ReplaceYear(ctx context.Context, actorID string, year int, entries []Entry) (int, error) {
	if s.DB == nil {
		return 0, fmt.Errorf("holiday store requires a database handle")
	}
	from := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(year, time.December, 31, 23, 59, 59, 999999999, time.UTC)

	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("tanggal >= ? AND tanggal <= ?", from, to).Delete(&models.HariLibur{}).Error; err != nil {
			return fmt.Errorf("clear holiday calendar: %w", err)
		}
		rows := make([]models.HariLibur, 0, len(entries))
		for _, entry := range entries {
			rows = append(rows, models.HariLibur{Tanggal: entry.Tanggal, Nama: entry.Nama})
		}
		if len(rows) > 0 {
			if err := tx.Create(&rows).Error; err != nil {
				return fmt.Errorf("write holiday calendar: %w", err)
			}
		}
		note := fmt.Sprintf("kalender hari libur %d diganti: %d entri", year, len(rows))
		return audit.Write(ctx, tx, audit.Entry{
			ActorID: actorID,
			Action:  audit.ActionHolidayRefreshed,
			Note:    &note,
		})
	})
	if err != nil {
		return 0, err
	}
	return len(entries), nil
}

// ListYear membaca kalender hari libur sebuah tahun.
func (s GormStore) ListYear(ctx context.Context, year int) ([]Entry, error) {
	if s.DB == nil {
		return nil, fmt.Errorf("holiday store requires a database handle")
	}
	var holidays []models.HariLibur
	from := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(year, time.December, 31, 23, 59, 59, 999999999, time.UTC)
	if err := s.DB.WithContext(ctx).
		Where("tanggal >= ? AND tanggal <= ?", from, to).
		Order("tanggal ASC").
		Find(&holidays).Error; err != nil {
		return nil, fmt.Errorf("list holiday calendar: %w", err)
	}
	entries := make([]Entry, 0, len(holidays))
	for _, holiday := range holidays {
		entries = append(entries, Entry{Tanggal: holiday.Tanggal, Nama: holiday.Nama})
	}
	return entries, nil
}
