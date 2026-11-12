// Package app memuat use case Service Level Objective: mengevaluasi metrik
// sebuah jendela terhadap target, lalu mengantrekan alert bila target
// terlampaui.
//
// Notifikasi alert/antrean keluar persisten, bukan dikirim langsung, sehingga
// percobaan ulang callback QStash tidak menghasilkan email ganda dan email
// tetap terkirim meski server sedang dimatikan.
package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"simpeg/backend/internal/notify"
	"simpeg/backend/internal/outbox"
	"simpeg/backend/internal/slo/domain"
)

// Repository adalah port pembacaan metrik SLO.
type Repository interface {
	// ListWindow membaca seluruh metrik pada satu jendela waktu.
	ListWindow(ctx context.Context, window domain.Window) ([]domain.Metric, error)
}

// Outbox adalah port antrean keluar persisten untuk alert.
type Outbox interface {
	// EnqueueBatch.antrekan banyak pesan sekaligus. Kunci idempotensi yang
	// sudah ada dilewati tanpa error.
	EnqueueBatch(ctx context.Context, messages []outbox.Message) (int, error)
}

// Service mengevaluasi target SLO dan mengantrekan alert.
type Service struct {
	repository Repository
	outbox     Outbox
	target     domain.Target
	// windowSize adalah panjang satu jendela agregasi metrik.
	windowSize time.Duration
	// recipients adalah daftar alamat yang menerima alert.
	recipients []string
	// now adalah sumber waktu, dapat diganti pada pengujian.
	now func() time.Time
}

// Options memuat dependensi use case SLO.
type Options struct {
	// Repository membaca metrik pada jendela waktu.
	Repository Repository
	// Outbox mengantrekan alert. Bila nil, alert tidak/antre dan use case
	// hanya melaporkan hasil evaluasi.
	Outbox Outbox
	// Target adalah batas SLO yang dinilai.
	Target domain.Target
	// WindowSize adalah panjang jendela agregasi metrik.
	WindowSize time.Duration
	// Recipients adalah daftar penerima alert.
	Recipients []string
	// Now adalah sumber waktu. Nil memakai waktu UTC saat ini.
	Now func() time.Time
}

// NewService membuat use case SLO.
func NewService(options Options) *Service {
	service := &Service{
		repository: options.Repository,
		outbox:     options.Outbox,
		target:     options.Target,
		windowSize: options.WindowSize,
		recipients: append([]string(nil), options.Recipients...),
		now:        options.Now,
	}
	if service.windowSize <= 0 {
		service.windowSize = 5 * time.Minute
	}
	if service.now == nil {
		service.now = func() time.Time { return time.Now().UTC() }
	}
	return service
}

// PreviousWindow mengembalikan jendela waktu lengkap terakhir.
//
// Jendela yang sedang berjalan sengaja dikecualikan agar metrik yang belum
// lengkap tidak dinilai sebagai pelanggaran target.
func (s *Service) PreviousWindow() domain.Window {
	current := s.now().UTC().Truncate(s.windowSize)
	return domain.Window{Start: current.Add(-s.windowSize), End: current}
}

// Evaluate menilai metrik sebuah jendela terhadap target.
func (s *Service) Evaluate(ctx context.Context, window domain.Window) (domain.Evaluation, error) {
	metrics, err := s.repository.ListWindow(ctx, window)
	if err != nil {
		return domain.Evaluation{}, fmt.Errorf("baca metrik slo: %w", err)
	}
	return domain.Evaluate(window, metrics, s.target), nil
}

// EvaluateAndNotify menilai jendela terakhir lalu mengantrekan alert bila
// target terlampaui.
//
// Nilai balik kedua memberi tahu apakah alert unleashed. Bila tidak ada
// penerima atau antrean keluar tidak dikonfigurasi, alert tidak/antre dan
// hasil evaluasi tetap dikembalikan.
func (s *Service) EvaluateAndNotify(ctx context.Context) (domain.Evaluation, bool, error) {
	window := s.PreviousWindow()
	evaluation, err := s.Evaluate(ctx, window)
	if err != nil {
		return domain.Evaluation{}, false, err
	}
	if evaluation.OK() {
		return evaluation, false, nil
	}
	if s.outbox == nil || len(s.recipients) == 0 {
		return evaluation, false, nil
	}

	messages := s.buildAlertMessages(window, evaluation)
	queued, err := s.outbox.EnqueueBatch(ctx, messages)
	if err != nil {
		return evaluation, false, fmt.Errorf("antrekan alert slo: %w", err)
	}
	return evaluation, queued > 0, nil
}

// buildAlertMessages menyusun pesan alert untuk setiap penerima.
//
// Kunci idempotensi memakai awal jendela sehingga percobaan ulang callback
// pada jendela yang sama tidak menghasilkan email kedua.
func (s *Service) buildAlertMessages(window domain.Window, evaluation domain.Evaluation) []outbox.Message {
	subject := "Peringatan SLO SIMPEG: target layanan terlampaui"
	body := buildAlertBody(evaluation)
	dedupePrefix := "slo-alert:" + window.Start.UTC().Format(time.RFC3339) + ":"
	messages := make([]outbox.Message, 0, len(s.recipients))
	for _, recipient := range s.recipients {
		trimmed := strings.TrimSpace(recipient)
		if trimmed == "" {
			continue
		}
		payload, err := notify.GenericPayload(trimmed, subject, body)
		if err != nil {
			continue
		}
		messages = append(messages, outbox.NewMessage(
			notify.KindGeneric,
			dedupePrefix+trimmed,
			string(payload),
		))
	}
	return messages
}

// buildAlertBody menyusun isi alert dalam Bahasa Indonesia.
func buildAlertBody(evaluation domain.Evaluation) string {
	var builder strings.Builder
	builder.WriteString("Target level layanan SIMPEG terlampaui pada jendela berikut.\n\n")
	builder.WriteString(evaluation.Describe())
	builder.WriteString("\nPelanggaran:\n")
	for _, breach := range evaluation.Breaches {
		builder.WriteString(fmt.Sprintf("- %s %s: terukur %s, target %s\n",
			breach.Method, breach.Route, breach.Observed, breach.Target))
	}
	if len(evaluation.Routes) > 0 {
		builder.WriteString("\nLima rute terlambat:\n")
		for index, route := range evaluation.Routes {
			if index >= 5 {
				break
			}
			builder.WriteString(fmt.Sprintf("- %s %s: p95 %s, p99 %s, permintaan %d, galat %d\n",
				route.Method, route.Route,
				domain.FormatDuration(route.P95),
				domain.FormatDuration(route.P99),
				route.Requests, route.Errors))
		}
	}
	builder.WriteString("\nPesan ini dikirim otomatis oleh pemeriksaan SLO terjadwal.")
	return builder.String()
}
