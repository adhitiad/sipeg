package notify

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"simpeg/backend/internal/outbox"
)

func TestSlipAvailablePayloadContainsNoAmount(t *testing.T) {
	payload, err := SlipAvailablePayload("budi@example.test", "Budi Santoso", "2026-10", "user-1")
	if err != nil {
		t.Fatal(err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"amount", "total", "nominal", "gajiPokok", "totalGaji"} {
		if _, exists := decoded[forbidden]; exists {
			t.Fatalf("payload leaked field %q", forbidden)
		}
	}
}

func TestSlipAvailableNoticeHasNoPaymentClaim(t *testing.T) {
	message := SlipAvailableNotice("budi@example.test", "Budi Santoso", "Oktober 2026", "https://simpeg.example.test/penggajian?periode=2026-10")
	combined := message.Subject + "\n" + message.Body

	for _, forbidden := range []string{"dibayar", "pembayaran", "sudah masuk", "transfer", "Rp"} {
		if strings.Contains(strings.ToLower(combined), strings.ToLower(forbidden)) {
			t.Fatalf("notice mengandung klaim pembayaran atau nominal (%q): %s", forbidden, combined)
		}
	}
	for _, required := range []string{"Budi Santoso", "Oktober 2026", "https://simpeg.example.test/penggajian?periode=2026-10"} {
		if !strings.Contains(combined, required) {
			t.Fatalf("notice tidak memuat %q: %s", required, combined)
		}
	}
	if message.IdempotencyKey == "" {
		t.Fatal("idempotency key wajib diisi")
	}
}

func TestEmailDelivererSendsSlipAvailableWithoutConfirmationSource(t *testing.T) {
	transport := &fakeTransport{}
	// Deliverer sengaja dibuat tanpa Confirmations: pemberitahuan
	// ketersediaan slip bukan bukti pembayaran dan tidak boleh bergantung
	// pada sumber konfirmasi pembayaran.
	deliverer := &EmailDeliverer{
		Sender:  NewSMTPSenderWithTransport(SMTPConfig{From: "noreply@simpeg.example.test"}, transport),
		BaseURL: "https://simpeg.example.test",
	}

	payload, err := SlipAvailablePayload("pegawai@example.test", "Siti", "2026-10", "user-9")
	if err != nil {
		t.Fatal(err)
	}
	err = deliverer.Deliver(context.Background(), outbox.Message{
		ID: "msg-1", Kind: KindSlipAvailable, DedupeKey: "k-1", Payload: string(payload),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(transport.sent) != 1 {
		t.Fatalf("expected 1 email, got %d", len(transport.sent))
	}
	if !contains(string(transport.sent[0]), "Siti") {
		t.Fatalf("email tidak memuat penerima: %s", transport.sent[0])
	}
}

func TestEmailDelivererRejectsSlipAvailableWithoutBaseURL(t *testing.T) {
	transport := &fakeTransport{}
	deliverer := &EmailDeliverer{Sender: NewSMTPSenderWithTransport(SMTPConfig{From: "noreply@simpeg.example.test"}, transport)}

	payload, err := SlipAvailablePayload("pegawai@example.test", "Siti", "2026-10", "user-9")
	if err != nil {
		t.Fatal(err)
	}
	err = deliverer.Deliver(context.Background(), outbox.Message{
		ID: "msg-2", Kind: KindSlipAvailable, DedupeKey: "k-2", Payload: string(payload),
	})
	if err == nil {
		t.Fatal("expected permanent error for missing base URL")
	}
	if len(transport.sent) != 0 {
		t.Fatal("email tidak boleh terkirim tanpa base URL")
	}
}
