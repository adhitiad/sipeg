package notify

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"
)

type fakeTransport struct {
	sent       [][]byte
	recipients [][]string
	err        error
}

func (t *fakeTransport) Send(_ context.Context, raw []byte, _ string, recipients []string) error {
	if t.err != nil {
		return t.err
	}
	t.sent = append(t.sent, raw)
	t.recipients = append(t.recipients, recipients)
	return nil
}

func TestSMTPSenderSendsMessage(t *testing.T) {
	transport := &fakeTransport{}
	sender := NewSMTPSenderWithTransport(SMTPConfig{From: "noreply@example.test"}, transport)

	err := sender.Send(context.Background(), GenericNotice(
		"pegawai@example.test", "Halo", "Isi pesan", "notif-1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(transport.sent) != 1 {
		t.Fatalf("expected 1 email, got %d", len(transport.sent))
	}
	body := string(transport.sent[0])
	for _, expected := range []string{"From: noreply@example.test", "To: pegawai@example.test", "Subject: Halo", "Isi pesan"} {
		if !strings.Contains(body, expected) {
			t.Errorf("email body missing %q:\n%s", expected, body)
		}
	}
}

func TestSMTPSenderRejectsMessageWithoutRecipient(t *testing.T) {
	transport := &fakeTransport{}
	sender := NewSMTPSenderWithTransport(SMTPConfig{From: "noreply@example.test"}, transport)

	err := sender.Send(context.Background(), GenericNotice("", "Halo", "Isi", "notif-1"))
	if !errors.Is(err, ErrNoRecipients) {
		t.Fatalf("expected ErrNoRecipients, got %v", err)
	}
	if len(transport.sent) != 0 {
		t.Fatal("no email may be sent without recipient")
	}
}

func TestSMTPSenderRejectsMessageWithoutIdempotencyKey(t *testing.T) {
	transport := &fakeTransport{}
	sender := NewSMTPSenderWithTransport(SMTPConfig{From: "noreply@example.test"}, transport)

	err := sender.Send(context.Background(), GenericNotice("pegawai@example.test", "Halo", "Isi", ""))
	if !errors.Is(err, ErrMissingIdempotencyKey) {
		t.Fatalf("expected ErrMissingIdempotencyKey, got %v", err)
	}
}

func TestSMTPSenderRequiresSenderAddress(t *testing.T) {
	transport := &fakeTransport{}
	sender := NewSMTPSenderWithTransport(SMTPConfig{}, transport)

	err := sender.Send(context.Background(), GenericNotice("pegawai@example.test", "Halo", "Isi", "notif-1"))
	if err == nil {
		t.Fatal("expected error when sender address is not configured")
	}
}

func TestSMTPSenderPropagatesTransportFailure(t *testing.T) {
	transport := &fakeTransport{err: errors.New("smtp tidak tersedia")}
	sender := NewSMTPSenderWithTransport(SMTPConfig{From: "noreply@example.test"}, transport)

	err := sender.Send(context.Background(), GenericNotice("pegawai@example.test", "Halo", "Isi", "notif-1"))
	if err == nil {
		t.Fatal("expected transport error")
	}
}

func TestSalaryPaidNoticeNeverContainsAmount(t *testing.T) {
	message := SalaryPaidNotice("pegawai@example.test", "Budi", "2026-11", "https://simpeg.example/penggajian")

	body := message.Body
	// Pola nominal rupiah, angka ribuan, dan mata uang asing.
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)rp\s?[\d.,]+`),
		regexp.MustCompile(`\d{1,3}([.,]\d{3})+(,\d+)?`),
		regexp.MustCompile(`(?i)\b(idr|usd|rp)\b`),
	}
	for _, pattern := range patterns {
		if pattern.MatchString(body) {
			t.Errorf("email memuat pola nominal %q: %s", pattern, body)
		}
	}
	if !strings.Contains(body, "https://simpeg.example/penggajian") {
		t.Errorf("email harus memuat tautan ke halaman terautentikasi: %s", body)
	}
	if !strings.Contains(body, "2026-11") {
		t.Errorf("email harus memuat periode: %s", body)
	}
}

func TestSalaryPaidNoticeHasStableIdempotencyKey(t *testing.T) {
	first := SalaryPaidNotice("pegawai@example.test", "Budi", "2026-11", "https://simpeg.example")
	second := SalaryPaidNotice("pegawai@example.test", "Budi", "2026-11", "https://simpeg.example")
	other := SalaryPaidNotice("pegawai@example.test", "Budi", "2026-12", "https://simpeg.example")

	if first.IdempotencyKey != second.IdempotencyKey {
		t.Fatal("idempotency key must be stable for the same event")
	}
	if first.IdempotencyKey == other.IdempotencyKey {
		t.Fatal("different periods must produce different idempotency keys")
	}
}

func TestSalaryNoticeIsNotSentWithoutConfirmedPayment(t *testing.T) {
	transport := &fakeTransport{}
	sender := &SalaryNoticeSender{
		Sender: NewSMTPSenderWithTransport(SMTPConfig{From: "noreply@example.test"}, transport),
		Confirmations: ConfirmedPaymentFunc(func(context.Context, string, string) (bool, error) {
			return false, nil
		}),
		BaseURL: "https://simpeg.example",
	}

	err := sender.SendSalaryPaidNotice(context.Background(), "pegawai@example.test", "Budi", "user-1", "2026-11")
	if !errors.Is(err, ErrPaymentNotConfirmed) {
		t.Fatalf("expected ErrPaymentNotConfirmed, got %v", err)
	}
	if len(transport.sent) != 0 {
		t.Fatal("no email may be sent before payment is confirmed")
	}
}

func TestSalaryNoticeIsSentAfterConfirmedPayment(t *testing.T) {
	transport := &fakeTransport{}
	sender := &SalaryNoticeSender{
		Sender: NewSMTPSenderWithTransport(SMTPConfig{From: "noreply@example.test"}, transport),
		Confirmations: ConfirmedPaymentFunc(func(context.Context, string, string) (bool, error) {
			return true, nil
		}),
		BaseURL: "https://simpeg.example",
	}

	err := sender.SendSalaryPaidNotice(context.Background(), "pegawai@example.test", "Budi", "user-1", "2026-11")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(transport.sent) != 1 {
		t.Fatalf("expected 1 email, got %d", len(transport.sent))
	}
	body := string(transport.sent[0])
	if !strings.Contains(body, "https://simpeg.example/penggajian?periode=2026-11") {
		t.Fatalf("email must link to the authenticated page: %s", body)
	}
}

func TestSalaryNoticePropagatesConfirmationFailure(t *testing.T) {
	transport := &fakeTransport{}
	sender := &SalaryNoticeSender{
		Sender: NewSMTPSenderWithTransport(SMTPConfig{From: "noreply@example.test"}, transport),
		Confirmations: ConfirmedPaymentFunc(func(context.Context, string, string) (bool, error) {
			return false, errors.New("database tidak tersedia")
		}),
		BaseURL: "https://simpeg.example",
	}

	err := sender.SendSalaryPaidNotice(context.Background(), "pegawai@example.test", "Budi", "user-1", "2026-11")
	if err == nil {
		t.Fatal("expected error when confirmation lookup fails")
	}
	if len(transport.sent) != 0 {
		t.Fatal("no email may be sent when confirmation lookup fails")
	}
}

func TestSMTPConfigUsesSafeTimeoutByDefault(t *testing.T) {
	sender := NewSMTPSender(SMTPConfig{})
	if sender.config.Timeout <= 0 {
		t.Fatal("expected a positive default timeout")
	}
	if sender.config.Timeout > time.Minute {
		t.Fatalf("default timeout too long: %v", sender.config.Timeout)
	}
}
