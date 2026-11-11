package notify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"simpeg/backend/internal/outbox"
)

// Kind pesan outbox yang dipahami pengirim email.
const (
	// KindSalaryPaid menandai notifikasi gaji.
	KindSalaryPaid = "email.penggajian"
	// KindGeneric menandai pemberitahuan umum.
	KindGeneric = "email.umum"
	// KindSlipAvailable menandai pemberitahuan ketersediaan slip gaji.
	KindSlipAvailable = "email.slip-tersedia"
)

// ErrInvalidPayload menandai isi outbox yang tidak dapat diproses. Kesalahan
// ini permanen sehingga pesan tidak diulang tanpa henti.
var ErrInvalidPayload = errors.New("invalid email payload")

// permanentError membungkus kesalahan sebagai permanen bagi dispatcher.
type permanentError struct {
	err error
}

func (e permanentError) Error() string { return e.err.Error() }

// Unwrap membungkus kesalahan asal sehingga errors.Is tetap bekerja.
func (e permanentError) Unwrap() error { return e.err }

// Permanent menandai kesalahan tidak layak dicoba ulang.
func (e permanentError) Permanent() bool { return true }

// emailPayload adalah isi pesan outbox untuk email.
//
// Isi tidak pernah memuat nominal gaji: pesan gaji hanya membawa penerima,
// nama, periode, dan id pengguna. Isi pesan dibangun ulang di sisi
// pengirim sehingga nominal tidak pernah dapat diinjeksi lewat payload.
type emailPayload struct {
	Kind      string `json:"kind"`
	Recipient string `json:"recipient"`
	Name      string `json:"name"`
	Period    string `json:"period"`
	UserID    string `json:"userId"`
	Subject   string `json:"subject"`
	Body      string `json:"body"`
}

// SalaryPaidPayload menyusun payload notifikasi gaji.
func SalaryPaidPayload(recipient, recipientName, period, userID string) ([]byte, error) {
	return json.Marshal(emailPayload{
		Kind:      KindSalaryPaid,
		Recipient: recipient,
		Name:      recipientName,
		Period:    period,
		UserID:    userID,
	})
}

// GenericPayload menyusun payload pemberitahuan umum.
func GenericPayload(recipient, subject, body string) ([]byte, error) {
	return json.Marshal(emailPayload{
		Kind:      KindGeneric,
		Recipient: recipient,
		Subject:   subject,
		Body:      body,
	})
}

// EmailDeliverer mengirim pesan outbox bertipe email.
//
// Pengirim ini menjadi jembatan antara antrean keluar persisten dan SMTP:
// bisnis menulis baris outbox dalam transaksinya, dispatcher mengambilnya,
// dan pesan dikirim di sini. Kegagalan pengiriman dikembalikan ke dispatcher
// sehingga pesan dijadwalkan ulang dengan backoff.
type EmailDeliverer struct {
	Sender Sender
	// Confirmations memeriksa konfirmasi pembayaran untuk pesan gaji.
	// Bila nil, pesan gaji tidak dikirim sama sekali.
	Confirmations PaymentConfirmationSource
	// BaseURL adalah origin aplikasi untuk tautan halaman terautentikasi.
	BaseURL string
}

// Deliver mengirim satu pesan outbox sebagai email.
func (d *EmailDeliverer) Deliver(ctx context.Context, message outbox.Message) error {
	var payload emailPayload
	if err := json.Unmarshal([]byte(message.Payload), &payload); err != nil {
		return permanentError{err: fmt.Errorf("%w: %s", ErrInvalidPayload, err.Error())}
	}

	switch payload.Kind {
	case KindSalaryPaid:
		return d.deliverSalaryNotice(ctx, payload)
	case KindSlipAvailable:
		return d.deliverSlipAvailable(ctx, payload)
	case KindGeneric:
		return d.deliverGeneric(ctx, payload)
	default:
		return permanentError{err: fmt.Errorf("%w: unknown kind %q", ErrInvalidPayload, payload.Kind)}
	}
}

// deliverSalaryNotice mengirim notifikasi gaji setelah pembayaran terkonfirmasi.
func (d *EmailDeliverer) deliverSalaryNotice(ctx context.Context, payload emailPayload) error {
	if d.Confirmations == nil {
		return permanentError{err: fmt.Errorf("%w: sumber konfirmasi pembayaran belum dikonfigurasi", ErrInvalidPayload)}
	}
	confirmed, err := d.Confirmations.ConfirmedPayment(ctx, payload.UserID, payload.Period)
	if err != nil {
		return fmt.Errorf("periksa konfirmasi pembayaran: %w", err)
	}
	if !confirmed {
		return permanentError{err: ErrPaymentNotConfirmed}
	}
	if d.BaseURL == "" {
		return permanentError{err: fmt.Errorf("%w: base url tidak dikonfigurasi", ErrInvalidPayload)}
	}

	link := strings.TrimRight(d.BaseURL, "/") + "/penggajian?periode=" + payload.Period
	return d.Sender.Send(ctx, SalaryPaidNotice(payload.Recipient, payload.Name, payload.Period, link))
}

// deliverSlipAvailable mengirim pemberitahuan ketersediaan slip gaji.
//
// Pemberitahuan ini bukan bukti pembayaran: tidak ada pemeriksaan konfirmasi
// pembayaran di sini, dan isi pesannya tidak memuat nominal maupun klaim bahwa
// gaji sudah dibayar. Pesan hanya mengarahkan pegawai ke halaman penggajian.
func (d *EmailDeliverer) deliverSlipAvailable(ctx context.Context, payload emailPayload) error {
	if strings.TrimSpace(payload.Recipient) == "" || strings.TrimSpace(payload.Period) == "" {
		return permanentError{err: fmt.Errorf("%w: penerima dan periode wajib diisi", ErrInvalidPayload)}
	}
	if d.BaseURL == "" {
		return permanentError{err: fmt.Errorf("%w: base url tidak dikonfigurasi", ErrInvalidPayload)}
	}

	link := strings.TrimRight(d.BaseURL, "/") + "/penggajian?periode=" + payload.Period
	return d.Sender.Send(ctx, SlipAvailableNotice(payload.Recipient, payload.Name, payload.Period, link))
}

// deliverGeneric mengirim pemberitahuan umum tanpa data keuangan.
func (d *EmailDeliverer) deliverGeneric(ctx context.Context, payload emailPayload) error {
	if strings.TrimSpace(payload.Subject) == "" || strings.TrimSpace(payload.Body) == "" {
		return permanentError{err: fmt.Errorf("%w: subjek dan isi wajib diisi", ErrInvalidPayload)}
	}
	return d.Sender.Send(ctx, Message{
		To:             payload.Recipient,
		Subject:        payload.Subject,
		Body:           payload.Body,
		IdempotencyKey: payload.Kind + ":" + payload.Recipient + ":" + payload.Subject,
	})
}
