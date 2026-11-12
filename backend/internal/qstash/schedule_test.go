package qstash

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateScheduleSendsCronAndDestination(t *testing.T) {
	var (
		receivedPath   string
		receivedCron   string
		receivedRetry  string
		receivedMethod string
		receivedBody   string
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer qstash-token" {
			t.Errorf("authorization = %q", r.Header.Get("Authorization"))
		}
		receivedPath = r.URL.Path
		receivedCron = r.Header.Get("Upstash-Cron")
		receivedRetry = r.Header.Get("Upstash-Retries")
		receivedMethod = r.Header.Get("Upstash-Method")
		raw, _ := io.ReadAll(r.Body)
		receivedBody = string(raw)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"scheduleId":"sched-1"}`))
	}))
	defer server.Close()

	publisher := NewPublisher("qstash-token")
	publisher.SchedulesURL = server.URL

	id, err := publisher.CreateSchedule(context.Background(), ScheduleRequest{
		Destination: destination,
		Body:        `{"kind":"audit.cleanup"}`,
		Cron:        "0 19 * * *",
		Retries:     3,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "sched-1" {
		t.Fatalf("schedule id = %q", id)
	}
	// API v2 QStash menerima destination pada path dan cron pada header.
	if !strings.HasSuffix(receivedPath, "/v2/schedules/"+destination) {
		t.Fatalf("path = %q, ingin berakhiran /v2/schedules/%s", receivedPath, destination)
	}
	if receivedCron != "0 19 * * *" {
		t.Fatalf("cron = %q", receivedCron)
	}
	if receivedRetry != "3" {
		t.Fatalf("percobaan ulang = %q, ingin 3", receivedRetry)
	}
	if receivedMethod != http.MethodPost {
		t.Fatalf("metode = %q, ingin POST", receivedMethod)
	}
	if receivedBody != `{"kind":"audit.cleanup"}` {
		t.Fatalf("body = %q", receivedBody)
	}
}

func TestCreateScheduleRequiresDestinationCronAndToken(t *testing.T) {
	publisher := NewPublisher("token")
	if _, err := publisher.CreateSchedule(context.Background(), ScheduleRequest{Cron: "0 19 * * *"}); err == nil {
		t.Fatal("expected error when destination is missing")
	}
	if _, err := publisher.CreateSchedule(context.Background(), ScheduleRequest{Destination: destination}); err == nil {
		t.Fatal("expected error when cron is missing")
	}
	if _, err := NewPublisher("").CreateSchedule(context.Background(), ScheduleRequest{Destination: destination, Cron: "0 19 * * *"}); err == nil {
		t.Fatal("expected error when token is missing")
	}
}

func TestListSchedulesDecodesRegisteredSchedules(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"scheduleId":"sched-1","destination":"https://simpeg.example/hook","cron":"0 19 * * *"}]`))
	}))
	defer server.Close()

	publisher := NewPublisher("qstash-token")
	publisher.SchedulesURL = server.URL

	schedules, err := publisher.ListSchedules(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(schedules) != 1 || schedules[0].ID != "sched-1" {
		t.Fatalf("schedules = %+v", schedules)
	}
}

func TestDeleteScheduleRequiresID(t *testing.T) {
	publisher := NewPublisher("qstash-token")
	if err := publisher.DeleteSchedule(context.Background(), ""); err == nil {
		t.Fatal("expected error when schedule id is missing")
	}
}
