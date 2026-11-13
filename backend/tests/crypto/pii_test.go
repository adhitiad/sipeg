package crypto

import (
	. "simpeg/backend/internal/crypto"
	"strings"
	"testing"
)

func testKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, 32 /*keySize*/)
	for i := range key {
		key[i] = byte(i + 1)
	}
	return key
}

func TestConfigureRejectsWrongLength(t *testing.T) {
	if err := Configure([]byte("pendek")); err == nil {
		t.Fatal("expected error for short key")
	}
	if err := Configure(testKey(t)); err != nil {
		t.Fatalf("configure: %v", err)
	}
}

func TestRoundTrip(t *testing.T) {
	if err := Configure(testKey(t)); err != nil {
		t.Fatal(err)
	}
	plaintext := "3273010101900001"
	stored, err := Encrypted(plaintext).Value()
	if err != nil {
		t.Fatalf("value: %v", err)
	}
	if ct, ok := stored.(string); !ok || !strings.HasPrefix(ct, CiphertextPrefix) {
		t.Fatalf("stored value tidak ber-prefix %q: %v", CiphertextPrefix, stored)
	}
	var scanned Encrypted
	if err := scanned.Scan(stored); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if string(scanned) != plaintext {
		t.Fatalf("round trip mismatch: %q", scanned)
	}
}

func TestEncryptUsesRandomNonce(t *testing.T) {
	if err := Configure(testKey(t)); err != nil {
		t.Fatal(err)
	}
	first, err := Encrypt("sama")
	if err != nil {
		t.Fatal(err)
	}
	second, err := Encrypt("sama")
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("ciphertext tidak boleh sama untuk plaintext yang sama (nonce harus acak)")
	}
}

func TestScanRejectsPlaintext(t *testing.T) {
	if err := Configure(testKey(t)); err != nil {
		t.Fatal(err)
	}
	var scanned Encrypted
	if err := scanned.Scan("0123456789012345"); err == nil {
		t.Fatal("plaintext di database harus ditolak")
	}
}

func TestBlindIndexStableAndNormalized(t *testing.T) {
	if err := Configure(testKey(t)); err != nil {
		t.Fatal(err)
	}
	a, err := BlindIndex("01.234.567.8-901.000")
	if err != nil {
		t.Fatal(err)
	}
	b, err := BlindIndex(" 012345678901000 ")
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatalf("blind index harus cocok lintas format: %q vs %q", a, b)
	}
	c, err := BlindIndex("012345678901999")
	if err != nil {
		t.Fatal(err)
	}
	if c == a {
		t.Fatal("blind index untuk nilai berbeda tidak boleh sama")
	}
}

func TestPtrHelpersNilSafe(t *testing.T) {
	if got := PtrString(nil); got != nil {
		t.Fatal("nil Encrypted harus menjadi nil *string")
	}
	if got := Deref(nil); got != "" {
		t.Fatal("Deref(nil) harus \"\"")
	}
	if got := PtrEncrypted(nil); got != nil {
		t.Fatal("nil input harus menjadi nil *Encrypted")
	}
	empty := ""
	if got := PtrEncrypted(&empty); got != nil {
		t.Fatal("string kosong harus menjadi nil *Encrypted")
	}
}
