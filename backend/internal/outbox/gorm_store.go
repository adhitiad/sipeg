package outbox

import (
	"context"
	"errors"
	"fmt"
	"time"

	"simpeg/backend/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GormStore menyimpan antrean keluar di tabel outboxMessage.
type GormStore struct {
	DB *gorm.DB
}

// NewGormStore membuat store outbox.
func NewGormStore(db *gorm.DB) GormStore { return GormStore{DB: db} }

// Enqueue menambahkan pekerjaan baru. DedupeKey yang sama ditolak agar satu
// pekerjaan logis tidak pernah diproses dua kali.
func (s GormStore) Enqueue(ctx context.Context, kind, dedupeKey, payload string) (Message, error) {
	if s.DB == nil {
		return Message{}, errors.New("outbox database is required")
	}
	record := models.OutboxMessage{
		Kind:        kind,
		DedupeKey:   dedupeKey,
		Payload:     payload,
		Status:      StatusPending,
		AvailableAt: time.Now().UTC(),
	}
	err := s.DB.WithContext(ctx).Create(&record).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return Message{}, ErrDuplicate
		}
		return Message{}, fmt.Errorf("enqueue outbox message: %w", err)
	}
	return toMessage(record), nil
}

// Claim mengambil pesan yang siap diproses lalu mengunci setiap pesan agar
// tidak diklaim dua worker sekaligus.
func (s GormStore) Claim(ctx context.Context, batchSize int, now time.Time) ([]Message, error) {
	if s.DB == nil {
		return nil, errors.New("outbox database is required")
	}
	if batchSize <= 0 {
		return nil, nil
	}

	var claimed []Message
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var records []models.OutboxMessage
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("status = ? AND \"availableAt\" <= ?", StatusPending, now).
			Order("\"availableAt\" ASC").
			Limit(batchSize).
			Find(&records).Error; err != nil {
			return fmt.Errorf("claim outbox messages: %w", err)
		}
		if len(records) == 0 {
			return nil
		}

		ids := make([]string, 0, len(records))
		for _, record := range records {
			ids = append(ids, record.ID)
		}
		lockedAt := now
		if err := tx.Model(&models.OutboxMessage{}).
			Where("id IN ?", ids).
			Updates(map[string]any{"status": StatusProcessing, "lockedAt": lockedAt}).Error; err != nil {
			return fmt.Errorf("lock outbox messages: %w", err)
		}
		for _, record := range records {
			record.Status = StatusProcessing
			record.LockedAt = &lockedAt
			claimed = append(claimed, toMessage(record))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return claimed, nil
}

// Complete menandai pesan selesai diproses.
func (s GormStore) Complete(ctx context.Context, id string, now time.Time) error {
	if s.DB == nil {
		return errors.New("outbox database is required")
	}
	result := s.DB.WithContext(ctx).Model(&models.OutboxMessage{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":      StatusDone,
			"processedAt": now,
			"lockedAt":    nil,
			"lastError":   nil,
		})
	if result.Error != nil {
		return fmt.Errorf("complete outbox message: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// Fail mencatat kegagalan percobaan. Bila percobaan masih di bawah maxAttempts,
// pesan dijadwalkan ulang dengan waktu tersedia berikutnya. Setelah melewati
// batas, pesan diletakkan pada status FAILED sebagai dead-letter yang dapat
// ditinjau dan diulang kembali.
func (s GormStore) Fail(ctx context.Context, id, message string, nextAttemptAt, now time.Time, maxAttempts int) error {
	if s.DB == nil {
		return errors.New("outbox database is required")
	}

	var record models.OutboxMessage
	if err := s.DB.WithContext(ctx).Where("id = ?", id).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("read outbox message: %w", err)
	}

	attempts := record.Attempts + 1
	status := StatusPending
	updates := map[string]any{
		"status":    status,
		"attempts":  attempts,
		"lastError": message,
		"lockedAt":  nil,
	}
	if maxAttempts > 0 && attempts >= maxAttempts {
		updates["status"] = StatusFailed
	} else {
		updates["availableAt"] = nextAttemptAt
	}

	result := s.DB.WithContext(ctx).Model(&models.OutboxMessage{}).
		Where("id = ?", id).
		Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("fail outbox message: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// ReleaseLocked mengembalikan pesan yang menggantung di status PROCESSING ke
// status PENDING. Dipanggil saat worker berhenti sehingga pesan tidak hilang.
func (s GormStore) ReleaseLocked(ctx context.Context, now time.Time) error {
	if s.DB == nil {
		return errors.New("outbox database is required")
	}
	result := s.DB.WithContext(ctx).Model(&models.OutboxMessage{}).
		Where("status = ? AND \"lockedAt\" < ?", StatusProcessing, now).
		Updates(map[string]any{"status": StatusPending, "lockedAt": nil})
	if result.Error != nil {
		return fmt.Errorf("release locked outbox messages: %w", result.Error)
	}
	return nil
}

func toMessage(record models.OutboxMessage) Message {
	return Message{
		ID:          record.ID,
		Kind:        record.Kind,
		DedupeKey:   record.DedupeKey,
		Payload:     record.Payload,
		Status:      record.Status,
		Attempts:    record.Attempts,
		AvailableAt: record.AvailableAt,
		LockedAt:    record.LockedAt,
		LastError:   record.LastError,
		CreatedAt:   record.CreatedAt,
		ProcessedAt: record.ProcessedAt,
	}
}
