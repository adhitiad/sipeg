package app

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"simpeg/backend/internal/notify"
	"simpeg/backend/internal/outbox"
	"simpeg/backend/internal/slo/domain"
)

// fakeRepository menyimpan metrik dalam memori.
type fakeRepository struct {
	metrics []domain.Metric
	window  domain.Window
	err     error
}

// ListWindow mengembalikan metrik yang sudah disiapkan.
func (f *fakeRepository) ListWindow(_ context.Context, window domain.Window) ([]domain.Metric, error) {
	f.window = window
	return f.metrics, f.err
}

// fakeOutbox mencatat pesan yang diantrekan.
type fakeOutbox struct {
	messages []outbox.Message
	err      error
}

// EnqueueBatch mencatat pesan antrean.
func (f *fakeOutbox) EnqueueBatch(_ context.Context, messages []outbox.Message) (int, error) {
	if f.err != nil {
		return 0, f.err
	}
	f.messages = append(f.messages, messages...)
	return len(messages), nil
}

func TestPreviousWindowExcludesRunningWindow(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 7, 30, 0, time.UTC)
	service := NewService(Options{
		WindowSize: 5 * time.Minute,
		Now:        func() time.Time { return now },
	})

	window := service.PreviousWindow()
	if !window.End.Equal(time.Date(2026, 3, 10, 12, 5, 0, 0, time.UTC)) {
		t.Fatalf("akhir jendela = %s, ingin 12:05", window.End)
	}
	if !window.Start.Equal(time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("awal jendela = %s, ingin 12:00", window.Start)
	}
}

func TestEvaluateAndNotifySkipsWhenTargetMet(t *testing.T) {
	metric := domain.NewMetric("/api/v1/me", "GET")
	metric.Observe(10*time.Millisecond, false)
	repository := &fakeRepository{metrics: []domain.Metric{metric}}
	outboxStore := &fakeOutbox{}
	service := NewService(Options{
		Repository: repository,
		Outbox:     outboxStore,
		Target:     domain.Target{P95: time.Second, ErrorRate: 0.05, MinRequests: 1},
		WindowSize: 5 * time.Minute,
	})

	evaluation, alerted, err := service.EvaluateAndNotify(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if alerted {
		t.Fatal("alert tidak seharusnya_disable saat target terpenuhi")
	}
	if !evaluation.OK() {
		t.Fatal("target seharusnya terpenuhi")
	}
	if len(outboxStore.messages) != 0 {
		t.Fatalf("pesan antrean = %d, ingin 0", len(outboxStore.messages))
	}
}

func TestEvaluateAndNotifyEnqueuesAlertOnBreach(t *testing.T) {
	metric := domain.NewMetric("/api/v1/employees", "GET")
	for index := 0; index < 50; index++ {
		metric.Observe(2*time.Second, index%2 == 0)
	}
	repository := &fakeRepository{metrics: []domain.Metric{metric}}
	outboxStore := &fakeOutbox{}
	service := NewService(Options{
		Repository: repository,
		Outbox:     outboxStore,
		Target:     domain.Target{P95: 500 * time.Millisecond, ErrorRate: 0.05, MinRequests: 10},
		WindowSize: 5 * time.Minute,
		Recipients: []string{"ops@example.test", "  ", "second@example.test"},
	})

	evaluation, alerted, err := service.EvaluateAndNotify(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !alerted {
		t.Fatal("alert seharusnya diantrekan")
	}
	if evaluation.OK() {
		t.Fatal("evaluasi seharusnya melaporkan pelanggaran")
	}
	if len(outboxStore.messages) != 2 {
		t.Fatalf("pesan antrean = %d, ingin 2 (penerima kosong dilewati)", len(outboxStore.messages))
	}
	if outboxStore.messages[0].Kind != notify.KindGeneric {
		t.Fatalf("jenis pesan = %s, ingin %s", outboxStore.messages[0].Kind, notify.KindGeneric)
	}
	if !strings.HasPrefix(outboxStore.messages[0].DedupeKey, "slo-alert:") {
		t.Fatalf("kunci idempotensi = %s, ingin diawali slo-alert:", outboxStore.messages[0].DedupeKey)
	}
	if outboxStore.messages[0].DedupeKey == outboxStore.messages[1].DedupeKey {
		t.Fatal("kunci idempotensi kedua penerima harus berbeda")
	}

	var payload struct {
		Recipient string `json:"recipient"`
		Subject   string `json:"subject"`
		Body      string `json:"body"`
	}
	if err := json.Unmarshal([]byte(outboxStore.messages[0].Payload), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Recipient != "ops@example.test" {
		t.Fatalf("penerima = %s, ingin ops@example.test", payload.Recipient)
	}
	if !strings.Contains(payload.Body, "terlampaui") {
		t.Fatalf("isi alert tidak menjelaskan pelanggaran:\n%s", payload.Body)
	}
}

func TestEvaluateAndNotifyWithoutOutboxDoesNotFail(t *testing.T) {
	metric := domain.NewMetric("/api/v1/me", "GET")
	metric.Observe(5*time.Second, true)
	repository := &fakeRepository{metrics: []domain.Metric{metric}}
	service := NewService(Options{
		Repository: repository,
		Target:     domain.Target{P95: time.Second, ErrorRate: 0.01, MinRequests: 1},
		WindowSize: 5 * time.Minute,
		Recipients: []string{"ops@example.test"},
	})

	evaluation, alerted, err := service.EvaluateAndNotify(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if alerted {
		t.Fatal("tanpa antrean keluar, alert tidak seharusnya dilaporkan terkirim")
	}
	if evaluation.OK() {
		t.Fatal("evaluasi seharusnya melaporkan pelanggaran")
	}
}

func TestEvaluateAndNotifyPropagatesRepositoryError(t *testing.T) {
	service := NewService(Options{
		Repository: &fakeRepository{err: context.DeadlineExceeded},
		Target:     domain.Target{P95: time.Second, ErrorRate: 0.01, MinRequests: 1},
		WindowSize: 5 * time.Minute,
	})

	if _, _, err := service.EvaluateAndNotify(context.Background()); err == nil {
		t.Fatal("kesalahan repository harus diteruskan")
	}
}
