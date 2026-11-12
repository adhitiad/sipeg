package outbox

import (
	"context"
	"errors"
	"testing"
	"time"
)

// recordingLogger menyimpan catatan dispatcher untuk pemeriksaan.
type recordingLogger struct {
	events []string
	fields []map[string]any
}

func (l *recordingLogger) Info(event string, fields map[string]any) {
	l.events = append(l.events, event)
	l.fields = append(l.fields, fields)
}

func (l *recordingLogger) Error(event string, fields map[string]any) {
	l.events = append(l.events, event)
	l.fields = append(l.fields, fields)
}

func TestDispatcherLogsDeliveryFailure(t *testing.T) {
	logger := &recordingLogger{}
	store := &fakeStore{claimed: []Message{{ID: "msg-1", Kind: "email.penggajian", Attempts: 1}}}
	dispatcher := NewDispatcher(store, DelivererFunc(func(context.Context, Message) error {
		return errors.New("smtp tidak tersedia")
	}), DispatcherOptions{Now: time.Now, Logger: logger})

	if _, err := dispatcher.RunClaimAndDeliver(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(logger.events) != 1 || logger.events[0] != "outbox_delivery_failed" {
		t.Fatalf("unexpected events: %v", logger.events)
	}
	fields := logger.fields[0]
	if fields["message_id"] != "msg-1" || fields["kind"] != "email.penggajian" {
		t.Fatalf("unexpected fields: %#v", fields)
	}
	if fields["error"] != "smtp tidak tersedia" {
		t.Fatalf("missing error detail: %#v", fields)
	}
}

func TestDispatcherDoesNotLogDedupeKey(t *testing.T) {
	logger := &recordingLogger{}
	store := &fakeStore{claimed: []Message{{ID: "msg-1", Kind: "email.penggajian", DedupeKey: "salary-paid:2026-11:user-1"}}}
	dispatcher := NewDispatcher(store, DelivererFunc(func(context.Context, Message) error {
		return nil
	}), DispatcherOptions{Now: time.Now, Logger: logger})

	if _, err := dispatcher.RunClaimAndDeliver(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(logger.events) != 1 || logger.events[0] != "outbox_delivered" {
		t.Fatalf("unexpected events: %v", logger.events)
	}
	if _, ok := logger.fields[0]["dedupe_key"]; ok {
		t.Fatalf("dedupe key must not be logged: %#v", logger.fields[0])
	}
}
