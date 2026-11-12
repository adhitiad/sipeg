package qstash

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestPublishAppendsPublishPathToRegionalRoot memastikan endpoint regional
// mendapat jalur "/v2/publish".
//
// Nilai BaseURL pada konfigurasi produksi adalah root API regional, bukan
// endpoint lengkap. Tanpa penambahan jalur, URL tujuan menempel langsung ke
// root dan QStash menjawab 404.
func TestPublishAppendsPublishPathToRegionalRoot(t *testing.T) {
	var receivedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	publisher := &Publisher{BaseURL: server.URL, Token: "token"}
	err := publisher.Publish(context.Background(), PublishRequest{
		Destination: "https://simpeg.example/api/v1/hooks/qstash",
		Body:        []byte(`{"kind":"audit.cleanup"}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(receivedPath, "/v2/publish/") {
		t.Fatalf("path permintaan = %s, ingin diawali /v2/publish/", receivedPath)
	}
	if !strings.Contains(receivedPath, "https://simpeg.example/api/v1/hooks/qstash") {
		t.Fatalf("path permintaan = %s, ingin memuat URL tujuan", receivedPath)
	}
}

// TestPublishKeepsExplicitPublishEndpoint memastikan endpoint yang sudah
// memuat jalur publish tidak digandakan.
func TestPublishKeepsExplicitPublishEndpoint(t *testing.T) {
	var receivedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	publisher := &Publisher{BaseURL: server.URL + "/v2/publish", Token: "token"}
	err := publisher.Publish(context.Background(), PublishRequest{
		Destination: "https://simpeg.example/hook",
	})
	if err != nil {
		t.Fatal(err)
	}

	if strings.Count(receivedPath, "/v2/publish") != 1 {
		t.Fatalf("path permintaan = %s, ingin satu kemunculan /v2/publish", receivedPath)
	}
}

// TestPublishEndpointDefaultsToGlobalEndpoint memastikan tanpa konfigurasi
// endpoint bawaan dipakai.
func TestPublishEndpointDefaultsToGlobalEndpoint(t *testing.T) {
	publisher := &Publisher{}
	if got := publisher.publishEndpoint(); got != DefaultPublishURL {
		t.Fatalf("endpoint = %s, ingin %s", got, DefaultPublishURL)
	}
}

// TestSchedulesURLFollowsRegionalRoot memastikan endpoint jadwal mengikuti
// region token.
//
// Tanpa ini, jadwal dibuat pada region global padahal token dimiliki region
// lain sehingga penjadwalan selalu gagal.
func TestSchedulesURLFollowsRegionalRoot(t *testing.T) {
	publisher := &Publisher{BaseURL: "https://qstash-us-east-1.upstash.io"}
	got := publisher.schedulesURL()
	if got != "https://qstash-us-east-1.upstash.io/v2/schedules" {
		t.Fatalf("endpoint jadwal = %s, ingin root regional + /v2/schedules", got)
	}
}

// TestSchedulesURLPrefersExplicitValue memastikan nilai eksplisit dipakai
// apa adanya.
func TestSchedulesURLPrefersExplicitValue(t *testing.T) {
	publisher := &Publisher{
		BaseURL:      "https://qstash-us-east-1.upstash.io",
		SchedulesURL: "https://custom.example/v2/schedules",
	}
	if got := publisher.schedulesURL(); got != "https://custom.example/v2/schedules" {
		t.Fatalf("endpoint jadwal = %s, ingin https://custom.example/v2/schedules", got)
	}
}

// TestSchedulesURLDefaultsToGlobalEndpoint memastikan tanpa konfigurasi
// endpoint bawaan dipakai.
func TestSchedulesURLDefaultsToGlobalEndpoint(t *testing.T) {
	publisher := &Publisher{}
	if got := publisher.schedulesURL(); got != DefaultSchedulesURL {
		t.Fatalf("endpoint = %s, ingin %s", got, DefaultSchedulesURL)
	}
}

// TestCreateScheduleUsesDerivedEndpoint memastikan jadwal dikirim ke endpoint
// turunan dari root regional dengan destination pada path.
func TestCreateScheduleUsesDerivedEndpoint(t *testing.T) {
	var receivedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"scheduleId": "sched-1"})
	}))
	defer server.Close()

	publisher := &Publisher{BaseURL: server.URL, Token: "token"}
	id, err := publisher.CreateSchedule(context.Background(), ScheduleRequest{
		Destination: "https://simpeg.example/hook",
		Cron:        "*/5 * * * *",
	})
	if err != nil {
		t.Fatal(err)
	}
	if id != "sched-1" {
		t.Fatalf("id jadwal = %s, ingin sched-1", id)
	}
	if !strings.HasPrefix(receivedPath, "/v2/schedules/") {
		t.Fatalf("path permintaan = %s, ingin diawali /v2/schedules/", receivedPath)
	}
}
