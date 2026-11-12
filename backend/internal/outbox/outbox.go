// Package outbox menyediakan antrean keluar persisten di PostgreSQL.
//
// Transaksi bisnis menulis baris outbox dalam transaksi yang sama, sehingga
// pekerjaan tidak hilang saat queue eksternal gagal. Baris yang sama tidak
// pernah dikirim dua kali untuk satu kunci idempotensi.
package outbox

import (
	"context"
	"errors"
	"time"
)

// Status baris outbox.
const (
	StatusPending    = "PENDING"
	StatusProcessing = "PROCESSING"
	StatusDone       = "DONE"
	StatusFailed     = "FAILED"
)

// ErrDuplicate menandakan kunci idempotensi sudah dipakai.
var ErrDuplicate = errors.New("outbox dedupe key already exists")

// ErrNotFound menandakan baris outbox tidak ada.
var ErrNotFound = errors.New("outbox message not found")

// Message adalah satu pekerjaan dalam antrean.
type Message struct {
	ID          string
	Kind        string
	DedupeKey   string
	Payload     string
	Status      string
	Attempts    int
	AvailableAt time.Time
	LockedAt    *time.Time
	LastError   *string
	CreatedAt   time.Time
	ProcessedAt *time.Time
}

// NewMessage membuat pesan antrean dengan status awal PENDING.
//
// Pesan yang dibuat lewat fungsi ini siap diantrekan satu per satu maupun
// sekaligus lewat EnqueueBatch.
func NewMessage(kind, dedupeKey, payload string) Message {
	return Message{
		Kind:        kind,
		DedupeKey:   dedupeKey,
		Payload:     payload,
		Status:      StatusPending,
		AvailableAt: time.Now().UTC(),
	}
}

// Store menyimpan dan mengambil baris outbox.
type Store interface {
	// Enqueue menambahkan pekerjaan baru. DedupeKey yang sama tidak boleh
	// menghasilkan baris kedua.
	Enqueue(ctx context.Context, kind, dedupeKey, payload string) (Message, error)
	// EnqueueBatch menambahkan banyak pekerjaan sekaligus. Kunci idempotensi
	// yang sudah ada dilewati tanpa error. Nilai balik adalah jumlah baris
	// yang benar-benar diantrekan.
	EnqueueBatch(ctx context.Context, messages []Message) (int, error)
	// Claim mengambil sampai batch pesan siap diproses dan menguncinya.
	Claim(ctx context.Context, batchSize int, now time.Time) ([]Message, error)
	// Complete menandai pesan selesai diproses.
	Complete(ctx context.Context, id string, now time.Time) error
	// Fail menandai percobaan gagal dan menjadwalkan percobaan berikutnya.
	// Pesan yang melebihi maxAttempts diletakkan pada status FAILED sebagai
	// dead-letter yang dapat ditinjau dan diulang kembali.
	Fail(ctx context.Context, id string, message string, nextAttemptAt time.Time, now time.Time, maxAttempts int) error
	// ReleaseLocked mengembalikan pesan yang terkunci ke status PENDING.
	ReleaseLocked(ctx context.Context, now time.Time) error
}

// Deliverer mengirim satu pesan outbox ke tujuan akhirnya.
type Deliverer interface {
	Deliver(ctx context.Context, message Message) error
}

// DelivererFunc adalah adaptor fungsi biasa menjadi Deliverer.
type DelivererFunc func(ctx context.Context, message Message) error

// Deliver memanggil fungsi yang dibungkus.
func (f DelivererFunc) Deliver(ctx context.Context, message Message) error { return f(ctx, message) }
