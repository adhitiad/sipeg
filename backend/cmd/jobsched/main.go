// Perintah jobsched adalah penjadwal pekerjaan asinkron untuk deployment
// self-hosted.
//
// Ia menggantikan peran jadwal QStash: setiap tick cron, satu pesan
// pekerjaan diterbitkan ke exchange simpeg.jobs di RabbitMQ, lalu server
// menjalankannya lewat konsumen internal. Jadwal memakai konstanta yang sama
// dengan jalur QStash (internal/jobs) dan dihitung dalam UTC sehingga
// 02:00/02:05 WIB tetap benar di mana pun container berjalan.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"simpeg/backend/internal/jobs"
	"simpeg/backend/internal/mq"
	"simpeg/backend/internal/observability"

	"github.com/rabbitmq/amqp091-go"
	"github.com/robfig/cron/v3"
)

// attempt menetapkan berapa kali penerbitan pesan diulang saat broker gagal
// menerima. Jeda antar percobaan 5 detik.
const attempt = 3

// envBool membaca nilai boolean environment dengan nilai bawaan.
func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	switch strings.ToLower(value) {
	case "1", "true", "yes":
		return true
	case "0", "false", "no":
		return false
	}
	return fallback
}

func main() {
	url := strings.TrimSpace(os.Getenv("RABBITMQ_URL"))
	if url == "" {
		log.Fatal("RABBITMQ_URL wajib diisi")
	}
	logger := observability.New(os.Stdout)

	// Mode alat operator: `jobsched publish <kind>` menerbitkan satu pesan
	// segera untuk verifikasi jalur broker tanpa menunggu tick cron.
	if len(os.Args) >= 3 && os.Args[1] == "publish" {
		if err := publishOnce(url, logger, os.Args[2]); err != nil {
			log.Fatal(err)
		}
		return
	}

	broker := newBroker(url, logger)
	defer broker.close()

	scheduler := cron.New(cron.WithLocation(time.UTC))
	register := func(spec, kind string) {
		_, err := scheduler.AddFunc(spec, func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			if err := broker.publish(ctx, kind); err != nil {
				logger.Error("jobsched_publish_failed", observability.Fields{"kind": kind, "error": err.Error()})
				return
			}
			logger.Info("jobsched_published", observability.Fields{"kind": kind})
		})
		if err != nil {
			log.Fatalf("jadwal tidak valid (%s): %v", spec, err)
		}
		logger.Info("jobsched_schedule_ready", observability.Fields{"kind": kind, "cron": spec})
	}
	register(jobs.CronAuditCleanup, mq.KindAuditCleanup)
	register(jobs.CronPayrollNotify, mq.KindPayrollNotify)
	// Pemeriksaan SLO hanya diterbitkan bila backend menjalankan pengoleki
	// metriknya; tanpa itu pesan berulang tiap lima menit hanya menumpuk di
	// dead-letter.
	if envBool("SLO_EVALUATE_ENABLED", false) {
		register(jobs.CronSLOEvaluate, mq.KindSLOEvaluate)
	}
	scheduler.Start()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)
	<-shutdown
	logger.Info("jobsched_stopping", nil)

	// Stop mengembalikan context yang selesai setelah pekerjaan yang sedang
	// berjalan selesai, sehingga penerbitan pesan tidak terpotong.
	stopDone := scheduler.Stop()
	<-stopDone.Done()
	broker.close()
	logger.Info("jobsched_stopped", nil)
}

// broker memelihara koneksi dan channel penerbitan yang tahan putus.
//
// Koneksi dibuka saat pertama kali dibutuhkan dan dibuka ulang bila channel
// pernah ditutup, sehingga broker yang dimulai ulang tidak mematikan penjadwal.
type broker struct {
	url    string
	logger *observability.Logger

	mu        sync.Mutex
	conn      *amqp091.Connection
	channel   *amqp091.Channel
	lastError string
}

func newBroker(url string, logger *observability.Logger) *broker {
	return &broker{url: url, logger: logger}
}

// publish menerbitkan satu pesan pekerjaan dengan beberapa percobaan.
func (b *broker) publish(ctx context.Context, kind string) error {
	var err error
	for i := 1; i <= attempt; i++ {
		if err = b.publishOnce(ctx, kind); err == nil {
			return nil
		}
		select {
		case <-time.After(5 * time.Second):
		case <-ctx.Done():
			return fmt.Errorf("terbitkan pekerjaan %s: %w", kind, ctx.Err())
		}
	}
	return fmt.Errorf("terbitkan pekerjaan %s setelah %d percobaan: %w", kind, attempt, err)
}

// publishOnce memastikan channel tersedia lalu menerbitkan satu pesan.
func (b *broker) publishOnce(ctx context.Context, kind string) error {
	channel, err := b.ensureChannel(ctx)
	if err != nil {
		return err
	}
	if err := mq.Publish(ctx, channel, kind); err != nil {
		b.reset(err.Error())
		return err
	}
	return nil
}

// ensureChannel membuka koneksi dan channel penerbitan, termasuk
// deklarasi topologi yang idempoten.
func (b *broker) ensureChannel(_ context.Context) (*amqp091.Channel, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.channel != nil && !b.channel.IsClosed() && b.conn != nil && !b.conn.IsClosed() {
		return b.channel, nil
	}
	b.dropLocked()
	connection, err := mq.Connect(b.url)
	if err != nil {
		return nil, err
	}
	channel, err := mq.NewPublishChannel(connection)
	if err != nil {
		connection.Close()
		return nil, err
	}
	if err := mq.DeclareTopology(channel); err != nil {
		channel.Close()
		connection.Close()
		return nil, err
	}
	b.conn = connection
	b.channel = channel
	if b.lastError != "" && b.logger != nil {
		b.logger.Info("jobsched_reconnected", observability.Fields{})
		b.lastError = ""
	}
	return channel, nil
}

// dropLocked menutup koneksi dan channel yang ada saat ini.
func (b *broker) dropLocked() {
	if b.channel != nil {
		b.channel.Close()
		b.channel = nil
	}
	if b.conn != nil {
		b.conn.Close()
		b.conn = nil
	}
}

// reset menutup koneksi dan channel yang bermasalah agar percobaan
// berikutnya membuka koneksi baru.
func (b *broker) reset(reason string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.dropLocked()
	b.lastError = reason
}

// close menutup koneksi dan channel saat proses berhenti.
func (b *broker) close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.dropLocked()
}

// publishOnce menerbitkan satu pesan pekerjaan segera untuk verifikasi.
//
// Hanya kind yang dikenal konsumen yang boleh dikirim supaya alat ini tidak
// membanjiri antrean dead-letter dengan pesan yang tidak berarti.
func publishOnce(url string, logger *observability.Logger, kind string) error {
	valid := map[string]bool{
		mq.KindAuditCleanup:  true,
		mq.KindPayrollNotify: true,
		mq.KindSLOEvaluate:   true,
	}
	if !valid[kind] {
		return fmt.Errorf("kind tidak dikenal %q (gunakan %s, %s, atau %s)",
			kind, mq.KindAuditCleanup, mq.KindPayrollNotify, mq.KindSLOEvaluate)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	broker := newBroker(url, logger)
	defer broker.close()
	if err := broker.publish(ctx, kind); err != nil {
		return err
	}
	logger.Info("jobsched_publish_once", observability.Fields{"kind": kind})
	return nil
}
