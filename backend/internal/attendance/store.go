package attendance

import (
	"context"
	"errors"
	"time"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/models"
)

var (
	ErrMachineNotFound = errors.New("attendance machine not found")
	ErrEventConflict   = errors.New("attendance idempotency key conflicts with stored event")
)

type Store interface {
	SaveEvent(context.Context, string, models.AttendanceEvent) (models.AttendanceEvent, bool, error)
}

type RecordFilter struct {
	From     time.Time
	Until    time.Time
	Page     int
	PageSize int
}

type RecordPage struct {
	Items      []models.Presensi
	Page       int
	PageSize   int
	TotalItems int64
	TotalPages int
}

type StatusCount struct {
	Date   time.Time
	Status models.StatusPresensi
	Count  int64
}

type RecordReader interface {
	ListMyRecords(context.Context, string, RecordFilter) ([]models.Presensi, int64, error)
	ListUnitSummary(context.Context, []string, RecordFilter) ([]StatusCount, error)
}

type UnitScopeReader interface {
	ListUnitScopeIDs(context.Context, string) ([]string, error)
}

type ReadService struct {
	reader RecordReader
	scope  UnitScopeReader
	policy auth.Policy
}

func NewReadService(reader RecordReader, scope UnitScopeReader, policy auth.Policy) *ReadService {
	return &ReadService{reader: reader, scope: scope, policy: policy}
}

func sameEventPayload(stored, incoming models.AttendanceEvent) bool {
	return stored.MachineId == incoming.MachineId &&
		stored.MachinePin == incoming.MachinePin &&
		stored.OccurredAt.Equal(normalizeEventTime(incoming.OccurredAt))
}

func normalizeEventTime(value time.Time) time.Time {
	return value.UTC().Round(time.Millisecond)
}
