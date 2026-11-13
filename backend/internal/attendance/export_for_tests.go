package attendance

import "simpeg/backend/internal/models"

// SameEventPayload mengekspor sameEventPayload untuk pengujian di backend/tests.
func SameEventPayload(stored, incoming models.AttendanceEvent) bool {
	return sameEventPayload(stored, incoming)
}
