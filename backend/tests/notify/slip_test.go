package notify

import (
	"context"
	"encoding/base64"
	"encoding/json"
	. "simpeg/backend/internal/notify"
	"strings"
	"testing"

	"simpeg/backend/internal/outbox"
)

// decodeMIMEParts memecah pesan multipart/related, mendekode setiap bagian
// base64 menjadi teks polos, dan menggabungkan hasilnya untuk asersi.
func decodeMIMEParts(t *testing.T, raw string) string {
	t.Helper()
	var out strings.Builder
	for _, part := range strings.Split(raw, "--") {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" || strings.HasPrefix(trimmed, "rel-") && strings.Count(trimmed, "\r\n") < 3 {
			// boundary penutup: lewati
		}
		if !strings.Contains(trimmed, "Content-Transfer-Encoding: base64") {
			out.WriteString(trimmed)
			out.WriteString("\n")
			continue
		}
		body := trimmed[strings.Index(trimmed, "\r\n\r\n")+4:]
		body = strings.ReplaceAll(body, "\r\n", "")
		decoded, err := base64.StdEncoding.DecodeString(body)
		if err != nil {
			out.WriteString(trimmed)
			out.WriteString("\n")
			continue
		}
		out.Write(decoded)
		out.WriteString("\n")
	}
	return out.String()
}

func TestSlipAvailablePayloadContainsNoAmount(t *testing.T) {
	payload, err := SlipAvailablePayload("budi@example.test", "Budi Santoso", "2026-10", "user-1", "token-test-1")
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
	message, err := SlipAvailableNotice("budi@example.test", "Budi Santoso", "Oktober 2026", "https://simpeg.example.test/verify/slip/token-test-1")
	if err != nil {
		t.Fatal(err)
	}
	combined := message.Subject + "\n" + message.Body + "\n" + message.HTMLBody

	for _, forbidden := range []string{"dibayar", "pembayaran", "sudah masuk", "transfer", "Rp"} {
		if strings.Contains(strings.ToLower(combined), strings.ToLower(forbidden)) {
			t.Fatalf("notice mengandung klaim pembayaran atau nominal (%q): %s", forbidden, combined)
		}
	}
	for _, required := range []string{"Budi Santoso", "Oktober 2026", "https://simpeg.example.test/verify/slip/token-test-1"} {
		if !strings.Contains(combined, required) {
			t.Fatalf("notice tidak memuat %q: %s", required, combined)
		}
	}
	if !strings.Contains(message.HTMLBody, "cid:slip-qr") || len(message.InlineImages) != 1 {
		t.Fatal("notice harus memuat QR inline (HTML + image/png)")
	}
	if message.InlineImages[0].ContentType != "image/png" || len(message.InlineImages[0].Content) == 0 {
		t.Fatal("inline image harus PNG dan tidak kosong")
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

	payload, err := SlipAvailablePayload("pegawai@example.test", "Siti", "2026-10", "user-9", "token-test-9")
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
	sent := string(transport.sent[0])
	if !strings.Contains(sent, "multipart/related") || !strings.Contains(sent, "image/png") {
		t.Fatalf("email H-1 harus multipart/related dengan gambar QR: %s", sent)
	}
	decoded := decodeMIMEParts(t, sent)
	if !strings.Contains(decoded, "/verify/slip/token-test-9") {
		t.Fatalf("email H-1 harus memuat tautan verifikasi slip: %s", decoded)
	}
	if !strings.Contains(decoded, "Siti") {
		t.Fatalf("email tidak memuat penerima: %s", decoded)
	}
	if !strings.Contains(decoded, "cid:slip-qr") {
		t.Fatalf("email H-1 harus mereferensikan QR inline: %s", decoded)
	}
}

func TestEmailDelivererRejectsSlipAvailableWithoutBaseURL(t *testing.T) {
	transport := &fakeTransport{}
	deliverer := &EmailDeliverer{Sender: NewSMTPSenderWithTransport(SMTPConfig{From: "noreply@simpeg.example.test"}, transport)}

	payload, err := SlipAvailablePayload("pegawai@example.test", "Siti", "2026-10", "user-9", "token-test-9")
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
