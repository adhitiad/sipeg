// Package mq menghubungkan backend dengan broker RabbitMQ sebagai
// pengangkut pekerjaan asinkron terjadwal.
//
// Paket ini menggantikan peran QStash pada deployment self-hosted: penjadwal
// (cmd/jobsched) menerbitkan pesan pekerjaan ke exchange simpeg.jobs, dan
// konsumen di server menjalankan pekerjaannya. Bentuk payload sama dengan
// body callback QStash ({"kind": ...}) sehingga kedua transport menjaga
// kontrak yang sama.
//
// Pekerjaan yang dikirim tetap idempoten: broker memberikan at-least-once,
// bukan exactly-once. Pesan yang gagal dijalankan dikirim ke antrean
// dead-letter untuk diperiksa operator; tick cron berikutnya memicu eksekusi
// baru secara alami.
package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rabbitmq/amqp091-go"
)

// Nama topologi broker. Semuanya dideklarasikan idempoten saat startup.
const (
	// ExchangeJobs menerima pesan pekerjaan dengan routing key berupa kind.
	ExchangeJobs = "simpeg.jobs"
	// ExchangeDeadLetter menerima pesan yang gagal dijalankan.
	ExchangeDeadLetter = "simpeg.jobs.dlx"
	// QueueJobs menampung pesan pekerjaan yang siap dijalankan.
	QueueJobs = "simpeg.jobs"
	// QueueJobsDead menampung pesan gagal untuk diperiksa operator.
	QueueJobsDead = "simpeg.jobs.dead"
	// RoutingKeyDeadLetter adalah routing key pesan dead-letter.
	RoutingKeyDeadLetter = "simpeg.jobs.dead"
)

// Kind pekerjaan yang dikenali konsumen. Nilainya sama dengan jenis callback
// QStash karena payload kedua transport identik.
const (
	KindAuditCleanup  = "audit.cleanup"
	KindPayrollNotify = "payroll.notify"
	KindSLOEvaluate   = "slo.evaluate"
)

// message adalah bentuk payload pekerjaan yang dikirim penjadwal.
type message struct {
	Kind string `json:"kind"`
}

// encode menuliskan payload pekerjaan dalam JSON.
func encode(kind string) ([]byte, error) {
	body, err := json.Marshal(message{Kind: kind})
	if err != nil {
		return nil, fmt.Errorf("encode pesan pekerjaan: %w", err)
	}
	return body, nil
}

// decodeKind membaca kind dari payload pesan.
func decodeKind(body []byte) (string, error) {
	var payload message
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", fmt.Errorf("decode pesan pekerjaan: %w", err)
	}
	if payload.Kind == "" {
		return "", fmt.Errorf("pesan pekerjaan tanpa kind")
	}
	return payload.Kind, nil
}

// Connect membuka koneksi AMQP dengan heartbeat agar kehilangan koneksi
// terdeteksi cepat oleh pustaka klien.
func Connect(url string) (*amqp091.Connection, error) {
	connection, err := amqp091.DialConfig(url, amqp091.Config{
		Heartbeat: 10 * time.Second,
		Locale:    "en_US",
	})
	if err != nil {
		return nil, fmt.Errorf("koneksi rabbitmq: %w", err)
	}
	return connection, nil
}

// NewPublishChannel membuka channel penerbitan pada mode confirm.
//
// Mode confirm membuat broker mengirim konfirmasi untuk setiap pesan yang
// diterima, sehingga penjadwal tahu pesannya benar-benar tersimpan.
func NewPublishChannel(connection *amqp091.Connection) (*amqp091.Channel, error) {
	channel, err := connection.Channel()
	if err != nil {
		return nil, fmt.Errorf("buka channel penerbitan: %w", err)
	}
	if err := channel.Confirm(false); err != nil {
		channel.Close()
		return nil, fmt.Errorf("aktifkan mode confirm: %w", err)
	}
	return channel, nil
}

// DeclareTopology memastikan exchange, antrean, dan binding tersedia.
//
// Semua deklarasi idempoten: menjalankannya berulang pada topologi yang sama
// tidak mengubah apa pun. Antrean pekerjaan bersifat durable dan
// dead-lettering ke ExchangeDeadLetter bila pesan ditolak konsumen.
func DeclareTopology(channel *amqp091.Channel) error {
	if err := channel.ExchangeDeclare(ExchangeDeadLetter, "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("deklarasi exchange dead-letter: %w", err)
	}
	if err := channel.ExchangeDeclare(ExchangeJobs, "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("deklarasi exchange pekerjaan: %w", err)
	}
	if _, err := channel.QueueDeclare(QueueJobsDead, true, false, false, false, nil); err != nil {
		return fmt.Errorf("deklarasi antrean dead-letter: %w", err)
	}
	if err := channel.QueueBind(QueueJobsDead, RoutingKeyDeadLetter, ExchangeDeadLetter, false, nil); err != nil {
		return fmt.Errorf("binding antrean dead-letter: %w", err)
	}
	args := amqp091.Table{
		"x-dead-letter-exchange":    ExchangeDeadLetter,
		"x-dead-letter-routing-key": RoutingKeyDeadLetter,
	}
	if _, err := channel.QueueDeclare(QueueJobs, true, false, false, false, args); err != nil {
		return fmt.Errorf("deklarasi antrean pekerjaan: %w", err)
	}
	for _, kind := range []string{KindAuditCleanup, KindPayrollNotify, KindSLOEvaluate} {
		if err := channel.QueueBind(QueueJobs, kind, ExchangeJobs, false, nil); err != nil {
			return fmt.Errorf("binding pekerjaan %s: %w", kind, err)
		}
	}
	return nil
}

// Publish mengirim satu pesan pekerjaan dan menunggu konfirmasi broker.
//
// Pesan bersifat persistent sehingga bertahan saat broker dimulai ulang.
// Channel harus sudah berada pada mode confirm.
func Publish(ctx context.Context, channel *amqp091.Channel, kind string) error {
	body, err := encode(kind)
	if err != nil {
		return err
	}
	publishing := amqp091.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp091.Persistent,
		Timestamp:    time.Now().UTC(),
		Body:         body,
	}
	confirmation, err := channel.PublishWithDeferredConfirmWithContext(
		ctx, ExchangeJobs, kind, false, false, publishing,
	)
	if err != nil {
		return fmt.Errorf("terbitkan pekerjaan %s: %w", kind, err)
	}
	acked, err := confirmation.WaitContext(ctx)
	if err != nil {
		return fmt.Errorf("pekerjaan %s tidak selesai dikonfirmasi: %w", kind, err)
	}
	if !acked {
		return fmt.Errorf("pekerjaan %s tidak dikonfirmasi broker", kind)
	}
	return nil
}
