package qstash

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	signingKey     = "current-signing-key"
	nextSigningKey = "next-signing-key"
	destination    = "https://simpeg.example/api/v1/hooks/qstash"
)

func signCallback(t *testing.T, key string, body []byte, subject string, expiry time.Time) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  subject,
		"body": base64.StdEncoding.EncodeToString(body),
		"iss":  "Upstash",
		"exp":  expiry.Unix(),
		"iat":  time.Now().Unix(),
	})
	signed, err := token.SignedString([]byte(key))
	if err != nil {
		t.Fatalf("failed to sign callback: %v", err)
	}
	return signed
}

func TestVerifierAcceptsValidSignature(t *testing.T) {
	body := []byte(`{"event":"penggajian"}`)
	header := signCallback(t, signingKey, body, destination, time.Now().Add(5*time.Minute))
	verifier := NewVerifier([]string{signingKey, nextSigningKey}, destination)

	if err := verifier.Verify(header, body); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifierAcceptsNextSigningKeyDuringRotation(t *testing.T) {
	body := []byte(`{"event":"penggajian"}`)
	header := signCallback(t, nextSigningKey, body, destination, time.Now().Add(5*time.Minute))
	verifier := NewVerifier([]string{signingKey, nextSigningKey}, destination)

	if err := verifier.Verify(header, body); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifierRejectsWrongKey(t *testing.T) {
	body := []byte(`{"event":"penggajian"}`)
	header := signCallback(t, "unknown-key", body, destination, time.Now().Add(5*time.Minute))
	verifier := NewVerifier([]string{signingKey}, destination)

	if err := verifier.Verify(header, body); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("expected ErrInvalidSignature, got %v", err)
	}
}

func TestVerifierRejectsTamperedBody(t *testing.T) {
	original := []byte(`{"amount":1000000}`)
	tampered := []byte(`{"amount":9000000}`)
	header := signCallback(t, signingKey, original, destination, time.Now().Add(5*time.Minute))
	verifier := NewVerifier([]string{signingKey}, destination)

	if err := verifier.Verify(header, tampered); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("expected ErrInvalidSignature for tampered body, got %v", err)
	}
}

func TestVerifierRejectsDifferentDestination(t *testing.T) {
	body := []byte(`{"event":"penggajian"}`)
	header := signCallback(t, signingKey, body, "https://penyerang.example/hook", time.Now().Add(5*time.Minute))
	verifier := NewVerifier([]string{signingKey}, destination)

	if err := verifier.Verify(header, body); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("expected ErrInvalidSignature for wrong destination, got %v", err)
	}
}

func TestVerifierRejectsExpiredSignature(t *testing.T) {
	body := []byte(`{"event":"penggajian"}`)
	header := signCallback(t, signingKey, body, destination, time.Now().Add(-time.Minute))
	verifier := NewVerifier([]string{signingKey}, destination)

	if err := verifier.Verify(header, body); !errors.Is(err, ErrExpiredSignature) {
		t.Fatalf("expected ErrExpiredSignature, got %v", err)
	}
}

func TestVerifierRejectsMissingSignature(t *testing.T) {
	verifier := NewVerifier([]string{signingKey}, destination)
	if err := verifier.Verify("", []byte("{}")); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("expected ErrInvalidSignature, got %v", err)
	}
}

func TestVerifierRejectsUnsignedToken(t *testing.T) {
	// Token tanpa tanda tangan tidak boleh diterima.
	verifier := NewVerifier([]string{signingKey}, destination)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": destination})
	unsigned, err := token.SignedString([]byte(signingKey))
	if err != nil {
		t.Fatal(err)
	}
	// Tearing bagian tanda tangan.
	parts := strings.Split(unsigned, ".")
	broken := parts[0] + "." + parts[1] + "."
	if err := verifier.Verify(broken, []byte("{}")); err == nil {
		t.Fatal("expected error for unsigned token")
	}
}

func TestPublisherSendsAuthorizedRequest(t *testing.T) {
	var received *http.Request
	var body []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = r.Clone(context.Background())
		buffer := make([]byte, 4096)
		n, _ := r.Body.Read(buffer)
		body = buffer[:n]
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	publisher := NewPublisher("qstash-token")
	publisher.BaseURL = server.URL

	err := publisher.Publish(context.Background(), PublishRequest{
		Destination: destination,
		Body:        []byte(`{"hello":"world"}`),
		RetryCount:  3,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if received == nil {
		t.Fatal("expected request to be received")
	}
	if received.Header.Get("Authorization") != "Bearer qstash-token" {
		t.Fatalf("unexpected authorization header: %q", received.Header.Get("Authorization"))
	}
	if received.Header.Get("Upstash-Retries") != "3" {
		t.Fatalf("unexpected retries header: %q", received.Header.Get("Upstash-Retries"))
	}
	if !strings.Contains(string(body), base64.StdEncoding.EncodeToString([]byte(`{"hello":"world"}`))) {
		t.Fatalf("payload must carry the base64 body: %s", body)
	}
}

func TestPublisherRejectsRejectedRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid destination"}`))
	}))
	defer server.Close()

	publisher := NewPublisher("qstash-token")
	publisher.BaseURL = server.URL

	err := publisher.Publish(context.Background(), PublishRequest{Destination: destination})
	if !errors.Is(err, ErrPublishRejected) {
		t.Fatalf("expected ErrPublishRejected, got %v", err)
	}
}

func TestPublisherRequiresDestinationAndToken(t *testing.T) {
	publisher := NewPublisher("")
	if err := publisher.Publish(context.Background(), PublishRequest{Destination: destination}); err == nil {
		t.Fatal("expected error when token is missing")
	}
	publisher = NewPublisher("token")
	if err := publisher.Publish(context.Background(), PublishRequest{}); err == nil {
		t.Fatal("expected error when destination is missing")
	}
}

func TestSignatureFromRequest(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/hook", nil)
	request.Header.Set(HeaderSignature, "signed-value")
	if got := SignatureFromRequest(request); got != "signed-value" {
		t.Fatalf("signature = %q", got)
	}
}
