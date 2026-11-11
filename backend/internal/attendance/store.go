package attendance

import (
	"context"
	"errors"
	"time"

	"simpeg/backend/internal/models"
)

var (
	ErrMachineNotFound = errors.New("attendance machine not found")
	ErrEventConflict   = errors.New("attendance idempotency key conflicts with stored event")
)

type Store interface {
	SaveEvent(context.Context, string, models.AttendanceEvent) (models.AttendanceEvent, bool, error)
}

func sameEventPayload(stored, incoming models.AttendanceEvent) bool {
	return stored.MachineId == incoming.MachineId &&
		stored.MachinePin == incoming.MachinePin &&
		stored.OccurredAt.Equal(normalizeEventTime(incoming.OccurredAt))
}

func normalizeEventTime(value time.Time) time.Time {
	return value.UTC().Round(time.Millisecond)
}
