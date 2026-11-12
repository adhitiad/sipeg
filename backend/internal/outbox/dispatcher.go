package outbox

import (
	"context"
	"errors"
	"math"
	"time"
)

// DispatcherOptions mengatur perilaku dispatcher outbox.
type DispatcherOptions struct {
	// BatchSize adalah jumlah pesan yang diklaim per siklus.
	BatchSize int
	// Interval adalah jeda antar siklus polling.
	Interval time.Duration
	// MaxAttempts adalah jumlah percobaan maksimum sebelum pesan menjadi
	// dead-letter berstatus FAILED.
	MaxAttempts int
	// BaseBackoff adalah jeda dasar untuk backoff eksponensial.
	BaseBackoff time.Duration
	// MaxBackoff membatasi jeda backoff maksimum.
	MaxBackoff time.Duration
	// LockTimeout adalah lama pesanan terkunci sebelum dianggap menggantung.
	LockTimeout time.Duration
	// Now dapat diganti pada pengujian.
	Now func() time.Time
	// Logger menerima catatan dispatcher. Nil berarti tanpa pencatatan.
	Logger Logger
}

// Logger adalah port pencatatan dispatcher. Field memakai map[string]any agar
// logger terstruktur pada paket observability dapat dipakai langsung.
type Logger interface {
	Info(event string, fields map[string]any)
	Error(event string, fields map[string]any)
}

// Dispatcher memproses antrean keluar dan mengirimnya ke tujuan akhirnya.
type Dispatcher struct {
	store     Store
	deliverer Deliverer
	options   DispatcherOptions
}

// NewDispatcher membuat dispatcher outbox.
func NewDispatcher(store Store, deliverer Deliverer, options DispatcherOptions) *Dispatcher {
	if options.BatchSize <= 0 {
		options.BatchSize = 20
	}
	if options.Interval <= 0 {
		options.Interval = 5 * time.Second
	}
	if options.MaxAttempts <= 0 {
		options.MaxAttempts = 5
	}
	if options.BaseBackoff <= 0 {
		options.BaseBackoff = 30 * time.Second
	}
	if options.MaxBackoff <= 0 {
		options.MaxBackoff = 30 * time.Minute
	}
	if options.LockTimeout <= 0 {
		options.LockTimeout = 5 * time.Minute
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	return &Dispatcher{store: store, deliverer: deliverer, options: options}
}

// RunClaimAndDeliver menjalankan satu siklus: klaim pesan, kirim, lalu tandai
// hasil. Mengembalikan jumlah pesan yang berhasil dikirim.
func (d *Dispatcher) RunClaimAndDeliver(ctx context.Context) (int, error) {
	now := d.options.Now()
	messages, err := d.store.Claim(ctx, d.options.BatchSize, now)
	if err != nil {
		return 0, err
	}

	delivered := 0
	for _, message := range messages {
		if err := d.deliverer.Deliver(ctx, message); err != nil {
			// Kesalahan permanen tidak diulang; pesan langsung menjadi dead-letter
			// agar dapat ditinjau dan diulang kembali secara manual.
			if IsPermanent(err) {
				if failErr := d.store.Fail(ctx, message.ID, err.Error(), now, now, 1); failErr != nil {
					return delivered, failErr
				}
				d.log("outbox_delivery_rejected", message, err)
				continue
			}
			nextAttempt := now.Add(d.backoff(message.Attempts + 1))
			if failErr := d.store.Fail(ctx, message.ID, err.Error(), nextAttempt, now, d.options.MaxAttempts); failErr != nil {
				return delivered, failErr
			}
			d.log("outbox_delivery_failed", message, err)
			continue
		}
		if err := d.store.Complete(ctx, message.ID, now); err != nil {
			return delivered, err
		}
		delivered++
		d.log("outbox_delivered", message, nil)
	}
	return delivered, nil
}

// RunClaimAndDeliverLoop menjalankan dispatcher terus-menerus sampai context
// dibatalkan.
func (d *Dispatcher) RunClaimAndDeliverLoop(ctx context.Context) error {
	ticker := time.NewTicker(d.options.Interval)
	defer ticker.Stop()

	for {
		if _, err := d.RunClaimAndDeliver(ctx); err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil
			}
			d.log("outbox_cycle_failed", Message{}, err)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

// ReleaseStaleLocked mengembalikan pesan yang menggantung ke antrean.
func (d *Dispatcher) ReleaseStaleLocked(ctx context.Context) error {
	cutoff := d.options.Now().Add(-d.options.LockTimeout)
	return d.store.ReleaseLocked(ctx, cutoff)
}

// backoff menghitung jeda percobaan berikutnya dengan pertumbuhan eksponensial
// yang dibatasi MaxBackoff.
func (d *Dispatcher) backoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	factor := math.Pow(2, float64(attempt-1))
	delay := time.Duration(float64(d.options.BaseBackoff) * factor)
	if delay > d.options.MaxBackoff || delay <= 0 {
		return d.options.MaxBackoff
	}
	return delay
}

func (d *Dispatcher) log(event string, message Message, err error) {
	if d.options.Logger == nil {
		return
	}
	// Kunci idempotensi tidak dicatat karena memuat identitas penerima.
	fields := map[string]any{
		"message_id": message.ID,
		"kind":       message.Kind,
		"attempts":   message.Attempts,
	}
	if err != nil {
		fields["error"] = err.Error()
		d.options.Logger.Error(event, fields)
		return
	}
	d.options.Logger.Info(event, fields)
}

// ErrDeliveryRejected menandakan pengirim menolak pesan secara permanen.
// Pesan seperti ini tidak layak dicoba ulang.
var ErrDeliveryRejected = errors.New("delivery rejected permanently")

// PermanentError menandai kesalahan yang tidak dapat diperbaiki dengan mencoba
// lagi. Pesan seperti itu langsung menjadi dead-letter.
type PermanentError interface {
	error
	Permanent() bool
}

// IsPermanent menandai apakah kesalahan tidak layak dicoba ulang.
func IsPermanent(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrDeliveryRejected) {
		return true
	}
	var permanent PermanentError
	return errors.As(err, &permanent) && permanent.Permanent()
}

// Describe merangkum kegagalan pengiriman untuk disimpan pada lastError.
func Describe(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
