package notify

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// ErrPaymentNotConfirmed menandakan pembayaran belum tercatat sebagai
// terkonfirmasi sehingga notifikasi gaji tidak boleh dikirim.
var ErrPaymentNotConfirmed = errors.New("payment is not confirmed")

// PaymentConfirmationSource mencari bukti pembayaran yang sudah tersimpan.
type PaymentConfirmationSource interface {
	// ConfirmedPayment melaporkan apakah pembayaran untuk pengguna dan periode
	// tertentu sudah tercatat sebagai terkonfirmasi di server.
	ConfirmedPayment(ctx context.Context, userID, period string) (bool, error)
}

// SalaryNoticeSender mengirim notifikasi gaji dengan pengaman konfirmasi.
//
// Notifikasi gaji hanya boleh dikirim setelah backend menyimpan konfirmasi
// pembayaran. Slip yang baru diterbitkan bukan bukti pembayaran, sehingga
// pengaman ini memeriksa konfirmasi secara terpisah.
type SalaryNoticeSender struct {
	Sender        Sender
	Confirmations PaymentConfirmationSource
	// BaseURL adalah origin aplikasi yang dipakai untuk tautan halaman
	// terautentikasi pada isi email.
	BaseURL string
}

// SendSalaryPaidNotice mengirim notifikasi gaji untuk pengguna dan periode
// tertentu setelah pembayaran terkonfirmasi.
func (s *SalaryNoticeSender) SendSalaryPaidNotice(ctx context.Context, recipient, recipientName, userID, period string) error {
	if s.Confirmations == nil {
		return errors.New("payment confirmation source is not configured")
	}
	confirmed, err := s.Confirmations.ConfirmedPayment(ctx, userID, period)
	if err != nil {
		return fmt.Errorf("check payment confirmation: %w", err)
	}
	if !confirmed {
		return ErrPaymentNotConfirmed
	}
	if s.BaseURL == "" {
		return errors.New("base url is not configured")
	}
	link := strings.TrimRight(s.BaseURL, "/") + "/penggajian?periode=" + period
	return s.Sender.Send(ctx, SalaryPaidNotice(recipient, recipientName, period, link))
}

// ConfirmedPaymentFunc adalah adaptor fungsi biasa menjadi PaymentConfirmationSource.
type ConfirmedPaymentFunc func(ctx context.Context, userID, period string) (bool, error)

// ConfirmedPayment memanggil fungsi yang dibungkus.
func (f ConfirmedPaymentFunc) ConfirmedPayment(ctx context.Context, userID, period string) (bool, error) {
	return f(ctx, userID, period)
}
