package models

import "time"

// OutboxMessage adalah antrean keluar persisten. Baris ditulis dalam
// transaksi bisnis yang sama sehingga pekerjaan tidak hilang saat queue
// eksternal gagal.
type OutboxMessage struct {
	IDModel
	Kind      string `gorm:"type:varchar(100)"`
	DedupeKey string `gorm:"type:varchar(200);uniqueIndex:outbox_dedupe_key"`
	Payload   string `gorm:"type:text"`
	// Status adalah PENDING, PROCESSING, DONE, atau FAILED.
	Status      string     `gorm:"type:varchar(20);index:outbox_dispatch_idx"`
	Attempts    int        `gorm:"default:0"`
	AvailableAt time.Time  `gorm:"index:outbox_dispatch_idx;type:timestamp(3)"`
	LockedAt    *time.Time `gorm:"type:timestamp(3)"`
	LastError   *string    `gorm:"type:text"`
	CreatedAt   time.Time  `gorm:"column:createdAt;type:timestamp(3)"`
	ProcessedAt *time.Time `gorm:"type:timestamp(3)"`
}
