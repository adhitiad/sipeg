package jobs

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"simpeg/backend/internal/qstash"
)

func TestEnsureScheduleReusesExistingDestination(t *testing.T) {
	created := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			created = true
			_, _ = w.Write([]byte(`{"scheduleId":"sched-new"}`))
			return
		}
		schedules := []qstash.Schedule{{ID: "sched-1", Destination: "https://simpeg.example/hook"}}
		_ = json.NewEncoder(w).Encode(schedules)
	}))
	defer server.Close()

	publisher := qstash.NewPublisher("qstash-token")
	publisher.SchedulesURL = server.URL

	id, err := EnsureSchedule(context.Background(), publisher, qstash.ScheduleRequest{
		Destination: "https://simpeg.example/hook",
		Cron:        CronAuditCleanup,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "sched-1" {
		t.Fatalf("schedule id = %q, want the existing schedule", id)
	}
	if created {
		t.Fatal("no new schedule may be created when the destination already exists")
	}
}

func TestEnsureScheduleCreatesMissingSchedule(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			_, _ = w.Write([]byte(`{"scheduleId":"sched-new"}`))
			return
		}
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	publisher := qstash.NewPublisher("qstash-token")
	publisher.SchedulesURL = server.URL

	id, err := EnsureSchedule(context.Background(), publisher, qstash.ScheduleRequest{
		Destination: "https://simpeg.example/hook",
		Cron:        CronAuditCleanup,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "sched-new" {
		t.Fatalf("schedule id = %q", id)
	}
}

func TestEnsureScheduleRequiresPublisher(t *testing.T) {
	if _, err := EnsureSchedule(context.Background(), nil, qstash.ScheduleRequest{}); err == nil {
		t.Fatal("expected error when publisher is missing")
	}
}

func TestDestinationJoinsBaseURLAndPath(t *testing.T) {
	got := Destination("https://simpeg.unisubang.ac.id/", "/api/v1/internal/qstash/audit/cleanup")
	want := "https://simpeg.unisubang.ac.id/api/v1/internal/qstash/audit/cleanup"
	if got != want {
		t.Fatalf("destination = %q, want %q", got, want)
	}
}
