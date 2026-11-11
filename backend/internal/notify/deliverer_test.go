package notify

import (
	"context"
	"errors"
	"testing"

	"simpeg/backend/internal/outbox"
)

func TestEmailDelivererSendsGenericMessage(t *testing.T) {
	transport := &fakeTransport{}
	deliverer := &EmailDeliverer{
		Sender:  NewSMTPSenderWithTransport(SMTPConfig{From: "noreply@simpeg.unisubang.ac.id"}, transport),
		BaseURL: "https://simpeg.unisubang.ac.id",
	}

	payload, err := GenericPayload("pegawai@example.test", "Pengingat", "Isi pengingat.")
	if err != nil {
		t.Fatal(err)
	}
	err = deliverer.Deliver(context.Background(), outbox.Message{
		ID: "msg-1", Kind: "email.umum", DedupeKey: "k-1", Payload: string(payload),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(transport.sent) != 1 {
		t.Fatalf("expected 1 email, got %d", len(transport.sent))
	}
	if !contains(string(transport.sent[0]), "Pengingat") {
		t.Fatalf("unexpected email: %s", transport.sent[0])
	}
}

func TestEmailDelivererRejectsMalformedPayloadPermanently(t *testing.T) {
	transport := &fakeTransport{}
	deliverer := &EmailDeliverer{
		Sender:  NewSMTPSenderWithTransport(SMTPConfig{From: "noreply@simpeg.unisubang.ac.id"}, transport),
		BaseURL: "https://simpeg.unisubang.ac.id",
	}

	err := deliverer.Deliver(context.Background(), outbox.Message{Payload: "bukan json"})
	if err == nil {
		t.Fatal("expected error for malformed payload")
	}
	if !outbox.IsPermanent(err) {
		t.Fatal("malformed payload must be permanent so it is not retried forever")
	}
	if len(transport.sent) != 0 {
		t.Fatal("no email may be sent for malformed payload")
	}
}

func TestEmailDelivererRejectsUnknownKindPermanently(t *testing.T) {
	deliverer := &EmailDeliverer{
		Sender:  NewSMTPSenderWithTransport(SMTPConfig{From: "noreply@simpeg.unisubang.ac.id"}, &fakeTransport{}),
		BaseURL: "https://simpeg.unisubang.ac.id",
	}

	err := deliverer.Deliver(context.Background(), outbox.Message{Payload: `{"kind":"tidak.dikenal"}`})
	if !outbox.IsPermanent(err) {
		t.Fatal("unknown kind must be permanent")
	}
}

func TestEmailDelivererDoesNotSendSalaryNoticeWithoutConfirmationSource(t *testing.T) {
	transport := &fakeTransport{}
	deliverer := &EmailDeliverer{
		Sender:  NewSMTPSenderWithTransport(SMTPConfig{From: "noreply@simpeg.unisubang.ac.id"}, transport),
		BaseURL: "https://simpeg.unisubang.ac.id",
		// Confirmations sengaja nil: sumber konfirmasi pembayaran belum ada.
	}

	payload, err := SalaryPaidPayload("pegawai@example.test", "Budi", "2026-11", "user-1")
	if err != nil {
		t.Fatal(err)
	}
	err = deliverer.Deliver(context.Background(), outbox.Message{Payload: string(payload)})
	if !errors.Is(err, ErrInvalidPayload) {
		t.Fatalf("expected ErrInvalidPayload, got %v", err)
	}
	if !outbox.IsPermanent(err) {
		t.Fatal("missing confirmation source must be permanent")
	}
	if len(transport.sent) != 0 {
		t.Fatal("no email may be sent without a confirmation source")
	}
}

func TestEmailDelivererDoesNotSendSalaryNoticeBeforeConfirmation(t *testing.T) {
	transport := &fakeTransport{}
	deliverer := &EmailDeliverer{
		Sender:  NewSMTPSenderWithTransport(SMTPConfig{From: "noreply@simpeg.unisubang.ac.id"}, transport),
		BaseURL: "https://simpeg.unisubang.ac.id",
		Confirmations: ConfirmedPaymentFunc(func(context.Context, string, string) (bool, error) {
			return false, nil
		}),
	}

	payload, err := SalaryPaidPayload("pegawai@example.test", "Budi", "2026-11", "user-1")
	if err != nil {
		t.Fatal(err)
	}
	err = deliverer.Deliver(context.Background(), outbox.Message{Payload: string(payload)})
	if !errors.Is(err, ErrPaymentNotConfirmed) {
		t.Fatalf("expected ErrPaymentNotConfirmed, got %v", err)
	}
	if len(transport.sent) != 0 {
		t.Fatal("no email may be sent before payment is confirmed")
	}
}

func TestEmailDelivererSendsSalaryNoticeAfterConfirmation(t *testing.T) {
	transport := &fakeTransport{}
	deliverer := &EmailDeliverer{
		Sender:  NewSMTPSenderWithTransport(SMTPConfig{From: "noreply@simpeg.unisubang.ac.id"}, transport),
		BaseURL: "https://simpeg.unisubang.ac.id",
		Confirmations: ConfirmedPaymentFunc(func(context.Context, string, string) (bool, error) {
			return true, nil
		}),
	}

	payload, err := SalaryPaidPayload("pegawai@example.test", "Budi", "2026-11", "user-1")
	if err != nil {
		t.Fatal(err)
	}
	err = deliverer.Deliver(context.Background(), outbox.Message{Payload: string(payload)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(transport.sent) != 1 {
		t.Fatalf("expected 1 email, got %d", len(transport.sent))
	}
	body := string(transport.sent[0])
	if !contains(body, "https://simpeg.unisubang.ac.id/penggajian?periode=2026-11") {
		t.Fatalf("email must link to the authenticated page: %s", body)
	}
}

func TestSalaryPayloadCarriesNoAmount(t *testing.T) {
	payload, err := SalaryPaidPayload("pegawai@example.test", "Budi", "2026-11", "user-1")
	if err != nil {
		t.Fatal(err)
	}
	text := string(payload)
	for _, forbidden := range []string{"amount", "nominal", "gajiPokok", "total"} {
		if contains(text, forbidden) {
			t.Errorf("payload memuat %q: %s", forbidden, text)
		}
	}
}

func contains(value, substring string) bool {
	for index := 0; index+len(substring) <= len(value); index++ {
		if value[index:index+len(substring)] == substring {
			return true
		}
	}
	return false
}
