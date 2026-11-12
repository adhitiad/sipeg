package mq

import (
	"context"
	"testing"
)

func TestEncodeDecodeKind(t *testing.T) {
	for _, kind := range []string{KindAuditCleanup, KindPayrollNotify, KindSLOEvaluate} {
		body, err := encode(kind)
		if err != nil {
			t.Fatalf("encode %s: %v", kind, err)
		}
		decoded, err := decodeKind(body)
		if err != nil {
			t.Fatalf("decode %s: %v", kind, err)
		}
		if decoded != kind {
			t.Errorf("kind berubah: %s menjadi %s", kind, decoded)
		}
	}
}

func TestDecodeKindRejectsEmptyKind(t *testing.T) {
	if _, err := decodeKind([]byte(`{}`)); err == nil {
		t.Error("pesan tanpa kind harus ditolak")
	}
}

func TestDecodeKindRejectsInvalidJSON(t *testing.T) {
	if _, err := decodeKind([]byte(`bukan json`)); err == nil {
		t.Error("payload bukan JSON harus ditolak")
	}
}

func TestRegisterDispatchesRegisteredHandler(t *testing.T) {
	consumer := NewConsumer(ConsumerOptions{})
	called := ""
	consumer.Register(KindSLOEvaluate, func(_ context.Context) error {
		called = "slo"
		return nil
	})
	job, registered := consumer.handlers[KindSLOEvaluate]
	if !registered {
		t.Fatal("handler slo.evaluate tidak terdaftar")
	}
	if err := job(nil); err != nil {
		t.Fatalf("handler: %v", err)
	}
	if called != "slo" {
		t.Error("handler salah dipanggil")
	}
}
