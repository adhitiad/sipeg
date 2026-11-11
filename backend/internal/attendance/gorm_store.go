package attendance

import (
	"context"
	"errors"

	"simpeg/backend/internal/audit"
	"simpeg/backend/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GormStore struct {
	DB *gorm.DB
}

// SaveEvent menyimpan fakta tap ternormalisasi dan mencatat jejak audit.
// ActorID adalah principal API yang mengirim event; event perangkat belum
// memakai autentikasi perangkat sehingga unit cakupan belum dapat ditentukan.
func (s GormStore) SaveEvent(ctx context.Context, actorID string, event models.AttendanceEvent) (models.AttendanceEvent, bool, error) {
	if s.DB == nil {
		return models.AttendanceEvent{}, false, errors.New("attendance database is required")
	}
	event.OccurredAt = normalizeEventTime(event.OccurredAt)

	result := s.DB.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "machineId"}, {Name: "idempotencyKey"}},
			DoNothing: true,
		}).
		Create(&event)
	if errors.Is(result.Error, gorm.ErrForeignKeyViolated) {
		return models.AttendanceEvent{}, false, ErrMachineNotFound
	}
	if result.Error != nil {
		return models.AttendanceEvent{}, false, result.Error
	}
	if result.RowsAffected == 1 {
		if err := audit.Write(ctx, s.DB, audit.Entry{
			ActorID:  actorID,
			Action:   audit.ActionAttendanceEventIngested,
			TargetID: &event.ID,
		}); err != nil {
			return models.AttendanceEvent{}, false, err
		}
		return event, false, nil
	}

	var stored models.AttendanceEvent
	err := s.DB.WithContext(ctx).
		Where(`"machineId" = ? AND "idempotencyKey" = ?`, event.MachineId, event.IdempotencyKey).
		Take(&stored).Error
	if err != nil {
		return models.AttendanceEvent{}, false, err
	}
	if !sameEventPayload(stored, event) {
		return models.AttendanceEvent{}, false, ErrEventConflict
	}
	return stored, true, nil
}
