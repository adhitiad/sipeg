package attendance

import (
	"testing"
	"time"

	"simpeg/backend/internal/models"
)

func TestSameEventPayloadAcceptsEquivalentTimeOffsets(t *testing.T) {
	stored := models.AttendanceEvent{
		MachineId:  "machine-1",
		MachinePin: "000042",
		OccurredAt: time.Date(2026, 11, 10, 2, 0, 0, 123000000, time.UTC),
	}
	incoming := models.AttendanceEvent{
		MachineId:  "machine-1",
		MachinePin: "000042",
		OccurredAt: time.Date(2026, 11, 10, 9, 0, 0, 123456000, time.FixedZone("UTC+7", 7*60*60)),
	}

	if !sameEventPayload(stored, incoming) {
		t.Fatal("equivalent event instants should normalize to the stored millisecond precision")
	}
}

func TestSameEventPayloadRejectsChangedTapFacts(t *testing.T) {
	stored := models.AttendanceEvent{
		MachineId:  "machine-1",
		MachinePin: "000042",
		OccurredAt: time.Date(2026, 11, 10, 2, 0, 0, 0, time.UTC),
	}
	changedPin := stored
	changedPin.MachinePin = "000043"
	changedTime := stored
	changedTime.OccurredAt = changedTime.OccurredAt.Add(time.Second)

	if sameEventPayload(stored, changedPin) || sameEventPayload(stored, changedTime) {
		t.Fatal("a reused idempotency key with changed tap facts must conflict")
	}
}
