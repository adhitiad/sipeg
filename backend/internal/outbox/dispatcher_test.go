package outbox

import (
	"context"
	"errors"
	"testing"
	"time"
)

// --- Dispatcher ---

type fakeStore struct {
	claimed    []Message
	completed  []string
	failed     []string
	failedNext []time.Time
	failErr    error
	claimErr   error
}

func (s *fakeStore) Enqueue(context.Context, string, string, string) (Message, error) {
	return Message{}, nil
}

func (s *fakeStore) EnqueueBatch(context.Context, []Message) (int, error) {
	return 0, nil
}

func (s *fakeStore) Claim(context.Context, int, time.Time) ([]Message, error) {
	return s.claimed, s.claimErr
}

func (s *fakeStore) Complete(_ context.Context, id string, _ time.Time) error {
	s.completed = append(s.completed, id)
	return nil
}

func (s *fakeStore) Fail(_ context.Context, id, _ string, nextAttemptAt, _ time.Time, _ int) error {
	s.failed = append(s.failed, id)
	s.failedNext = append(s.failedNext, nextAttemptAt)
	return s.failErr
}

func (s *fakeStore) ReleaseLocked(context.Context, time.Time) error { return nil }

func TestDispatcherDeliversAndCompletesMessages(t *testing.T) {
	now := time.Now().UTC()
	store := &fakeStore{claimed: []Message{{ID: "msg-1", Kind: "email.penggajian", DedupeKey: "key-1"}}}
	delivered := []string{}
	dispatcher := NewDispatcher(store, DelivererFunc(func(_ context.Context, message Message) error {
		delivered = append(delivered, message.ID)
		return nil
	}), DispatcherOptions{Now: func() time.Time { return now }})

	count, err := dispatcher.RunClaimAndDeliver(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 1 {
		t.Fatalf("delivered = %d, want 1", count)
	}
	if len(delivered) != 1 || delivered[0] != "msg-1" {
		t.Fatalf("unexpected deliveries: %v", delivered)
	}
	if len(store.completed) != 1 || store.completed[0] != "msg-1" {
		t.Fatalf("unexpected completions: %v", store.completed)
	}
	if len(store.failed) != 0 {
		t.Fatalf("no message should fail: %v", store.failed)
	}
}

func TestDispatcherMarksFailedMessageForRetry(t *testing.T) {
	now := time.Now().UTC()
	store := &fakeStore{claimed: []Message{{ID: "msg-1", Attempts: 0}}}
	dispatcher := NewDispatcher(store, DelivererFunc(func(context.Context, Message) error {
		return errors.New("smtp tidak tersedia")
	}), DispatcherOptions{Now: func() time.Time { return now }, BaseBackoff: 10 * time.Second})

	count, err := dispatcher.RunClaimAndDeliver(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Fatalf("delivered = %d, want 0", count)
	}
	if len(store.failed) != 1 || store.failed[0] != "msg-1" {
		t.Fatalf("unexpected failures: %v", store.failed)
	}
	if len(store.failedNext) != 1 {
		t.Fatalf("expected next attempt time, got %v", store.failedNext)
	}
	if !store.failedNext[0].After(now) {
		t.Fatalf("next attempt must be in the future, got %v", store.failedNext[0])
	}
}

func TestDispatcherBackoffGrowsAndIsCapped(t *testing.T) {
	dispatcher := NewDispatcher(&fakeStore{}, DelivererFunc(func(context.Context, Message) error { return nil }),
		DispatcherOptions{BaseBackoff: time.Second, MaxBackoff: 8 * time.Second})

	if delay := dispatcher.backoff(1); delay != time.Second {
		t.Fatalf("backoff(1) = %v, want 1s", delay)
	}
	if delay := dispatcher.backoff(2); delay != 2*time.Second {
		t.Fatalf("backoff(2) = %v, want 2s", delay)
	}
	if delay := dispatcher.backoff(3); delay != 4*time.Second {
		t.Fatalf("backoff(3) = %v, want 4s", delay)
	}
	if delay := dispatcher.backoff(10); delay != 8*time.Second {
		t.Fatalf("backoff(10) = %v, want capped 8s", delay)
	}
}

func TestDispatcherStopsOnContextCancel(t *testing.T) {
	dispatcher := NewDispatcher(&fakeStore{}, DelivererFunc(func(context.Context, Message) error { return nil }),
		DispatcherOptions{Interval: 10 * time.Millisecond})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := dispatcher.RunClaimAndDeliverLoop(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIsPermanentDetectsRejectedDelivery(t *testing.T) {
	if !IsPermanent(ErrDeliveryRejected) {
		t.Fatal("expected delivery rejection to be permanent")
	}
	if IsPermanent(errors.New("temporary")) {
		t.Fatal("temporary failure must not be permanent")
	}
}

func TestPermanentErrorIsDetectedAndUnwrapped(t *testing.T) {
	underlying := errors.New("isi rusak")
	wrapped := permanentError{err: underlying}

	if !IsPermanent(wrapped) {
		t.Fatal("expected permanent error to be detected")
	}
	if !errors.Is(wrapped, underlying) {
		t.Fatal("permanent error must unwrap to its cause")
	}
	if IsPermanent(nil) {
		t.Fatal("nil error must not be permanent")
	}
}

func TestDispatcherSendsPermanentFailureStraightToDeadLetter(t *testing.T) {
	now := time.Now().UTC()
	store := &fakeStore{claimed: []Message{{ID: "msg-1", Attempts: 0}}}
	dispatcher := NewDispatcher(store, DelivererFunc(func(context.Context, Message) error {
		return permanentError{err: errors.New("isi rusak")}
	}), DispatcherOptions{Now: func() time.Time { return now }, BaseBackoff: time.Second})

	if _, err := dispatcher.RunClaimAndDeliver(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(store.failed) != 1 {
		t.Fatalf("expected the message to be failed, got %v", store.failed)
	}
	// Dead-letter permanen tidak dijadwalkan ulang: waktu berikutnya sama dengan sekarang.
	if len(store.failedNext) != 1 || !store.failedNext[0].Equal(now) {
		t.Fatalf("permanent failure must not be rescheduled, got %v", store.failedNext)
	}
}

// permanentError menandai kesalahan permanen pada pengujian.
type permanentError struct{ err error }

func (e permanentError) Error() string   { return e.err.Error() }
func (e permanentError) Unwrap() error   { return e.err }
func (e permanentError) Permanent() bool { return true }
