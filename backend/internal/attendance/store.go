package attendance

import (
	"context"
	"errors"
	"time"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/crypto"
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
	// Sort adalah kunci sort dari allowlist RekamSortFields.
	Sort string
	// Order adalah arah pengurutan: "asc" atau "desc".
	Order string
	// Status menyaring status presensi; kosong berarti semua status.
	Status string
	// Search menyaring nama pegawai pada rekam pribadi.
	Search string
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

// UnitDayRow adalah satu pegawai aktif dalam cakupan beserta catatan
// kehadirannya untuk sebuah tanggal. `Status` nil berarti belum tercatat.
type UnitDayRow struct {
	UserID   string
	Nama     string
	Nip      *crypto.Encrypted
	UnitID   *string
	UnitName *string
	Status   *models.StatusPresensi
	Note     *string
}

type RecordReader interface {
	ListMyRecords(context.Context, string, RecordFilter) ([]models.Presensi, int64, error)
	ListUnitSummary(context.Context, []string, RecordFilter) ([]StatusCount, error)
	ListUnitDay(context.Context, []string, time.Time) ([]UnitDayRow, error)
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
