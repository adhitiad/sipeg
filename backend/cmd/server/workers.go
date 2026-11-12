// Pekerjaan latar yang menyertai server HTTP: pengirim email outbox,
// konsumen pekerjaan RabbitMQ, dan pendaftaran jadwal QStash.
//
// Semua pekerjaan idempoten sehingga pengulangan (restart, redelivery,
// percobaan ulang) tidak menghasilkan efek ganda.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"simpeg/backend/internal/audit"
	"simpeg/backend/internal/config"
	"simpeg/backend/internal/httpserver"
	"simpeg/backend/internal/jobs"
	"simpeg/backend/internal/mq"
	"simpeg/backend/internal/notify"
	"simpeg/backend/internal/observability"
	"simpeg/backend/internal/outbox"
	payrollapp "simpeg/backend/internal/payroll/app"
	"simpeg/backend/internal/qstash"
	sloapp "simpeg/backend/internal/slo/app"
	"simpeg/backend/internal/slo/domain"
	"simpeg/backend/internal/timeutil"

	"gorm.io/gorm"
)

// auditRecorder mencatat jejak audit untuk pekerjaan latar.
type auditRecorder interface {
	Record(ctx context.Context, entry audit.Entry) error
}

// startOutboxWorker menjalankan pengirim email dari antrean keluar
// persisten. Worker hanya berjalan bila SMTP dikonfigurasi.
func startOutboxWorker(ctx context.Context, cfg config.Config, db *gorm.DB, payroll *payrollapp.Service, logger *observability.Logger) {
	if !cfg.SMTPEnabled() {
		return
	}
	emailSender := notify.NewSMTPSender(notify.SMTPConfig{
		Host:     cfg.SMTP.Host,
		Port:     cfg.SMTP.Port,
		Username: cfg.SMTP.User,
		Password: cfg.SMTP.Password,
		From:     cfg.SMTP.From,
		UseTLS:   cfg.SMTP.Secure,
		Timeout:  15 * time.Second,
	})
	deliverer := &notify.EmailDeliverer{
		Sender:  emailSender,
		BaseURL: cfg.AppBaseURL,
		// Sumber konfirmasi pembayaran adalah modul penggajian: email gaji
		// hanya terkirim bila pembayaran sudah tercatat di database.
		Confirmations: payroll,
	}
	dispatcher := outbox.NewDispatcher(outbox.NewGormStore(db), deliverer, outbox.DispatcherOptions{
		BatchSize:   20,
		Interval:    10 * time.Second,
		MaxAttempts: 5,
		BaseBackoff: 30 * time.Second,
		MaxBackoff:  30 * time.Minute,
		Logger:      logger,
	})
	go func() {
		if err := dispatcher.RunClaimAndDeliverLoop(ctx); err != nil {
			logger.Error("outbox_worker_stopped", observability.Fields{"error": err.Error()})
		}
	}()
	logger.Info("outbox_worker_started", observability.Fields{"interval_seconds": 10})
}

// startJobConsumer menjalankan konsumen RabbitMQ untuk pekerjaan terjadwal
// mode self-hosted. Handler-nya menjalankan logika yang sama dengan callback
// QStash sehingga kedua transport setara.
func startJobConsumer(ctx context.Context, cfg config.Config, cleaner *audit.Cleaner, recorder auditRecorder, payroll *payrollapp.Service, sloEvaluator *sloapp.Service, logger *observability.Logger) {
	if !cfg.RabbitMQEnabled() {
		return
	}
	consumer := mq.NewConsumer(mq.ConsumerOptions{URL: cfg.RabbitMQURL, Logger: logger})
	consumer.Register(mq.KindAuditCleanup, func(ctx context.Context) error {
		if _, err := cleaner.Cleanup(ctx); err != nil {
			return err
		}
		recordJobOutcome(ctx, recorder, audit.ActionAuditCleaned, "jejak audit kedaluwarsa dihapus")
		return nil
	})
	consumer.Register(mq.KindPayrollNotify, func(ctx context.Context) error {
		queued, err := payroll.ProcessReminders(ctx, timeutil.NowUTC())
		if err != nil {
			return err
		}
		recordJobOutcome(ctx, recorder, audit.ActionPayrollNotified, fmt.Sprintf("pemberitahuan slip diantrekan: %d", queued))
		return nil
	})
	if sloEvaluator != nil {
		consumer.Register(mq.KindSLOEvaluate, func(ctx context.Context) error {
			evaluation, alerted, err := sloEvaluator.EvaluateAndNotify(ctx)
			if err != nil {
				return err
			}
			recordJobOutcome(ctx, recorder, audit.ActionSLOEvaluated, summarizeSLO(evaluation, alerted))
			return nil
		})
	}
	go func() {
		if err := consumer.Run(ctx); err != nil {
			logger.Error("mq_consumer_stopped", observability.Fields{"error": err.Error()})
		}
	}()
	logger.Info("mq_consumer_configured", observability.Fields{"url_env": "RABBITMQ_URL"})
}

// recordJobOutcome mencatat hasil pekerjaan. Kegagalan pencatatan tidak
// menggagalkan pekerjaan karena jejak audit bukan hal kritis.
func recordJobOutcome(ctx context.Context, recorder auditRecorder, action, note string) {
	if recorder == nil {
		return
	}
	_ = recorder.Record(ctx, audit.Entry{Action: action, Note: &note})
}

// summarizeSLO merumuskan hasil pemeriksaan untuk jejak audit, sama dengan
// ringkasan pada handler callback SLO.
func summarizeSLO(evaluation domain.Evaluation, alerted bool) string {
	note := "pemeriksaan SLO: "
	switch {
	case alerted:
		note += "alert diantrekan"
	case evaluation.OK():
		note += "target terpenuhi"
	default:
		note += "target terlampaui tanpa alert"
	}
	return note
}

// ensureQStashSchedules mendaftarkan jadwal QStash untuk mode cloud. Pada
// mode self-hosted (RABBITMQ_URL diisi), pendaftaran ini dilewati karena
// penjadwal cmd/jobsched yang mengambil peran tersebut.
func ensureQStashSchedules(ctx context.Context, cfg config.Config, publisher *qstash.Publisher, sloEvaluator *sloapp.Service, logger *observability.Logger) {
	if publisher == nil || cfg.RabbitMQEnabled() {
		return
	}
	ensureSchedule := func(callbackPath, cron, kind string) {
		destination := jobs.Destination(cfg.AppBaseURL, callbackPath)
		payload, err := json.Marshal(map[string]string{"kind": kind})
		if err != nil {
			logger.Error("qstash_job_payload_failed", observability.Fields{"error": err.Error()})
			return
		}
		scheduleID, err := jobs.EnsureSchedule(ctx, publisher, qstash.ScheduleRequest{
			Destination: destination,
			Body:        string(payload),
			Cron:        cron,
			Retries:     3,
		})
		if err != nil {
			logger.Error("qstash_schedule_failed", observability.Fields{"error": err.Error(), "destination": destination})
			return
		}
		logger.Info("qstash_schedule_ready", observability.Fields{
			"schedule_id": scheduleID,
			"destination": destination,
			"cron":        cron,
		})
	}
	ensureSchedule(httpserver.APIPrefix+"/internal/qstash/audit/cleanup", jobs.CronAuditCleanup, httpserver.JobKindAuditCleanup)
	ensureSchedule(httpserver.APIPrefix+"/internal/qstash/payroll/notify", jobs.CronPayrollNotify, httpserver.JobKindPayrollNotify)
	if sloEvaluator != nil {
		ensureSchedule(httpserver.APIPrefix+"/internal/qstash/slo/evaluate", jobs.CronSLOEvaluate, httpserver.JobKindSLOEvaluate)
	}
}
