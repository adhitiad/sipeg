package mq

import (
	"context"
	"errors"
	"fmt"
	"time"

	"simpeg/backend/internal/observability"

	"github.com/rabbitmq/amqp091-go"
)

// JobFunc menjalankan satu pekerjaan dari pesan broker.
type JobFunc func(ctx context.Context) error

// ConsumerOptions memuat konfigurasi konsumen pekerjaan.
type ConsumerOptions struct {
	URL    string
	Logger *observability.Logger
}

// Consumer menerima pesan pekerjaan dari antrean simpeg.jobs dan menjalankan
// handler yang terdaftar untuk kind-nya.
//
// Konsumen sengaja mengakui (ack) pesan hanya setelah pekerjaan sukses.
// Kegagalan pekerjaan ditolak tanpa requeue sehingga pesan pindah ke antrean
// dead-letter; tick cron berikutnya dari penjadwal memicu eksekusi baru.
// Koneksi yang terputus dicoba lagi dengan jeda agar broker yang baru
// menyala tetap tersambung tanpa restart proses.
type Consumer struct {
	options  ConsumerOptions
	handlers map[string]JobFunc
}

// NewConsumer menyiapkan konsumen tanpa membuka koneksi.
func NewConsumer(options ConsumerOptions) *Consumer {
	return &Consumer{
		options:  options,
		handlers: make(map[string]JobFunc),
	}
}

// Register mendaftarkan handler untuk satu kind pekerjaan.
func (c *Consumer) Register(kind string, job JobFunc) {
	c.handlers[kind] = job
}

// Run menjalankan lingkaran konsumsi sampai context dibatalkan.
func (c *Consumer) Run(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			return nil
		}
		handled, err := c.consumeOnce(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			if c.options.Logger != nil {
				c.options.Logger.Error("mq_consumer_error", observability.Fields{"error": err.Error()})
			}
		}
		if !handled && err == nil {
			// Lingkaran selesai secara normal, misalnya context selesai.
			return nil
		}
		select {
		case <-time.After(5 * time.Second):
		case <-ctx.Done():
			return nil
		}
	}
}

// consumeOnce membuka koneksi, deklarasi topologi, dan mengonsumsi pesan
// sampai koneksi terputus atau context selesai. Ia mengembalikan true bila
// lingkaran konsumsi pernah berjalan.
func (c *Consumer) consumeOnce(ctx context.Context) (bool, error) {
	connection, err := Connect(c.options.URL)
	if err != nil {
		return false, err
	}
	defer connection.Close()

	channel, err := connection.Channel()
	if err != nil {
		return false, fmt.Errorf("buka channel konsumsi: %w", err)
	}
	defer channel.Close()

	if err := channel.Qos(1, 0, false); err != nil {
		return false, fmt.Errorf("atur qos konsumen: %w", err)
	}
	if err := DeclareTopology(channel); err != nil {
		return false, err
	}

	deliveries, err := channel.Consume(QueueJobs, "", false, false, false, false, nil)
	if err != nil {
		return false, fmt.Errorf("mulai konsumsi antrean: %w", err)
	}

	if c.options.Logger != nil {
		c.options.Logger.Info("mq_consumer_started", observability.Fields{"queue": QueueJobs})
	}
	for {
		select {
		case <-ctx.Done():
			return true, nil
		case delivery, open := <-deliveries:
			if !open {
				return true, errors.New("antrean konsumsi tertutup")
			}
			c.handle(ctx, channel, delivery)
		}
	}
}

// handle menjalankan satu pesan: decode kind, jalankan handler, lalu ack atau
// dead-letter. Nilai isi pesan tidak dicatat karena hanya berisi kind.
func (c *Consumer) handle(ctx context.Context, channel *amqp091.Channel, delivery amqp091.Delivery) {
	kind, err := decodeKind(delivery.Body)
	if err != nil {
		if c.options.Logger != nil {
			c.options.Logger.Error("mq_message_invalid", observability.Fields{"error": err.Error()})
		}
		_ = channel.Nack(delivery.DeliveryTag, false, false)
		return
	}

	job, registered := c.handlers[kind]
	if !registered {
		if c.options.Logger != nil {
			c.options.Logger.Error("mq_job_unknown", observability.Fields{"kind": kind})
		}
		_ = channel.Nack(delivery.DeliveryTag, false, false)
		return
	}

	if err := job(ctx); err != nil {
		if c.options.Logger != nil {
			c.options.Logger.Error("mq_job_failed", observability.Fields{
				"kind":  kind,
				"error": err.Error(),
			})
		}
		// Tanpa requeue: pesan pindah ke antrean dead-letter. Penjadwal
		// menerbitkan eksekusi baru pada tick berikutnya, sehingga pekerjaan
		// tidak hilang dan pesan rusak tidak berputar tanpa henti.
		_ = channel.Nack(delivery.DeliveryTag, false, false)
		return
	}

	if err := channel.Ack(delivery.DeliveryTag, false); err != nil {
		if c.options.Logger != nil {
			c.options.Logger.Warn("mq_ack_failed", observability.Fields{"kind": kind, "error": err.Error()})
		}
	}
	if c.options.Logger != nil {
		c.options.Logger.Info("mq_job_done", observability.Fields{"kind": kind})
	}
}
