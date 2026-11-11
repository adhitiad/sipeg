package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"simpeg/backend/internal/attendance"
	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/models"
)

const (
	testAttendanceMachineID = "00000000-0000-4000-8000-000000000001"
	testAttendanceEventID   = "00000000-0000-4000-8000-000000000002"
)

type fakeAttendanceStore struct {
	events map[string]models.AttendanceEvent
	err    error
}

func (s *fakeAttendanceStore) SaveEvent(_ context.Context, _ string, event models.AttendanceEvent) (models.AttendanceEvent, bool, error) {
	if s.err != nil {
		return models.AttendanceEvent{}, false, s.err
	}
	if event.MachineId != testAttendanceMachineID {
		return models.AttendanceEvent{}, false, attendance.ErrMachineNotFound
	}
	if s.events == nil {
		s.events = make(map[string]models.AttendanceEvent)
	}
	key := event.MachineId + "/" + event.IdempotencyKey
	if stored, ok := s.events[key]; ok {
		if !stored.OccurredAt.Equal(event.OccurredAt) || stored.MachinePin != event.MachinePin {
			return models.AttendanceEvent{}, false, attendance.ErrEventConflict
		}
		return stored, true, nil
	}
	event.ID = testAttendanceEventID
	s.events[key] = event
	return event, false, nil
}

func attendanceMux(store attendance.Store, policy auth.Policy) *http.ServeMux {
	return NewMuxWithServices(testSessionResolver{}, policy, DirectoryServices{}, store)
}

func attendanceRequest(body string) *http.Request {
	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/attendance/events", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer valid-session")
	request.Header.Set("Content-Type", "application/json")
	return request
}

func TestAttendanceEventIngestRequiresSessionAndExplicitPermission(t *testing.T) {
	store := &fakeAttendanceStore{}
	mux := attendanceMux(store, nil)

	unauthenticated := httptest.NewRecorder()
	mux.ServeHTTP(unauthenticated, httptest.NewRequest(http.MethodPost, APIPrefix+"/attendance/events", strings.NewReader(`{}`)))
	if unauthenticated.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated request status = %d, want %d", unauthenticated.Code, http.StatusUnauthorized)
	}

	forbidden := httptest.NewRecorder()
	mux.ServeHTTP(forbidden, attendanceRequest(`{}`))
	if forbidden.Code != http.StatusForbidden {
		t.Fatalf("request without a grant status = %d, want %d", forbidden.Code, http.StatusForbidden)
	}
	if len(store.events) != 0 {
		t.Fatal("denied requests must not reach persistence")
	}
}

func TestAttendanceEventRetryReturnsExistingEvent(t *testing.T) {
	store := &fakeAttendanceStore{}
	policy := auth.StaticPolicy{"operator": {attendanceIngestPermission: true}}
	mux := attendanceMux(store, policy)
	body := `{"machineId":"00000000-0000-4000-8000-000000000001","machinePin":"000042","occurredAt":"2026-11-10T09:00:00+07:00","idempotencyKey":"source-event-1"}`

	first := httptest.NewRecorder()
	mux.ServeHTTP(first, attendanceRequest(body))
	if first.Code != http.StatusCreated {
		t.Fatalf("first request status = %d, want %d: %s", first.Code, http.StatusCreated, first.Body.String())
	}
	second := httptest.NewRecorder()
	mux.ServeHTTP(second, attendanceRequest(body))
	if second.Code != http.StatusOK {
		t.Fatalf("retry status = %d, want %d: %s", second.Code, http.StatusOK, second.Body.String())
	}

	var response attendanceEventResponse
	if err := json.Unmarshal(second.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.EventID != testAttendanceEventID || !response.Duplicate {
		t.Fatalf("unexpected retry response: %+v", response)
	}
	if len(store.events) != 1 {
		t.Fatalf("stored event count = %d, want 1", len(store.events))
	}
	for _, event := range store.events {
		if event.MachinePin != "000042" || event.OccurredAt.Location() != time.UTC {
			t.Fatalf("event facts were not preserved and normalized: %+v", event)
		}
	}
}

func TestAttendanceEventKeyReuseWithChangedFactsConflicts(t *testing.T) {
	store := &fakeAttendanceStore{}
	policy := auth.StaticPolicy{"operator": {attendanceIngestPermission: true}}
	mux := attendanceMux(store, policy)
	firstBody := `{"machineId":"00000000-0000-4000-8000-000000000001","machinePin":"000042","occurredAt":"2026-11-10T09:00:00+07:00","idempotencyKey":"source-event-1"}`
	changedBody := `{"machineId":"00000000-0000-4000-8000-000000000001","machinePin":"000043","occurredAt":"2026-11-10T09:00:00+07:00","idempotencyKey":"source-event-1"}`

	created := httptest.NewRecorder()
	mux.ServeHTTP(created, attendanceRequest(firstBody))
	conflict := httptest.NewRecorder()
	mux.ServeHTTP(conflict, attendanceRequest(changedBody))
	if created.Code != http.StatusCreated || conflict.Code != http.StatusConflict {
		t.Fatalf("statuses = (%d, %d), want (%d, %d)", created.Code, conflict.Code, http.StatusCreated, http.StatusConflict)
	}
	if len(store.events) != 1 {
		t.Fatalf("stored event count = %d, want 1", len(store.events))
	}
}

func TestAttendanceEventRejectsInvalidOrUnapprovedFields(t *testing.T) {
	store := &fakeAttendanceStore{}
	policy := auth.StaticPolicy{"operator": {attendanceIngestPermission: true}}
	mux := attendanceMux(store, policy)
	valid := `{"machineId":"00000000-0000-4000-8000-000000000001","machinePin":"000042","occurredAt":"2026-11-10T09:00:00+07:00","idempotencyKey":"source-event-1"}`
	tests := []struct {
		name string
		body string
		want int
	}{
		{name: "missing PIN", body: `{"machineId":"00000000-0000-4000-8000-000000000001","machinePin":" ","occurredAt":"2026-11-10T09:00:00+07:00","idempotencyKey":"source-event-1"}`, want: http.StatusUnprocessableEntity},
		{name: "timestamp without offset", body: `{"machineId":"00000000-0000-4000-8000-000000000001","machinePin":"000042","occurredAt":"2026-11-10T09:00:00","idempotencyKey":"source-event-1"}`, want: http.StatusBadRequest},
		{name: "biometric field is not accepted", body: strings.TrimSuffix(valid, "}") + `,"template":"sensitive"}`, want: http.StatusBadRequest},
		{name: "invalid machine ID", body: strings.Replace(valid, testAttendanceMachineID, "not-a-uuid", 1), want: http.StatusUnprocessableEntity},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, attendanceRequest(test.body))
			if response.Code != test.want {
				t.Fatalf("status = %d, want %d: %s", response.Code, test.want, response.Body.String())
			}
		})
	}
	if len(store.events) != 0 {
		t.Fatal("invalid requests must not be persisted")
	}
}

func TestAttendanceEventUnknownMachineAndStoreErrorsAreSafe(t *testing.T) {
	policy := auth.StaticPolicy{"operator": {attendanceIngestPermission: true}}
	body := `{"machineId":"00000000-0000-4000-8000-000000000099","machinePin":"000042","occurredAt":"2026-11-10T09:00:00+07:00","idempotencyKey":"source-event-1"}`
	unknownMachine := httptest.NewRecorder()
	attendanceMux(&fakeAttendanceStore{}, policy).ServeHTTP(unknownMachine, attendanceRequest(body))
	if unknownMachine.Code != http.StatusNotFound {
		t.Fatalf("unknown machine status = %d, want %d", unknownMachine.Code, http.StatusNotFound)
	}

	failedStore := httptest.NewRecorder()
	attendanceMux(&fakeAttendanceStore{err: errors.New("database details")}, policy).ServeHTTP(failedStore, attendanceRequest(strings.Replace(body, "000000000099", "000000000001", 1)))
	if failedStore.Code != http.StatusServiceUnavailable || strings.Contains(failedStore.Body.String(), "database details") {
		t.Fatalf("store error response was not safely mapped: %d %s", failedStore.Code, failedStore.Body.String())
	}
}
