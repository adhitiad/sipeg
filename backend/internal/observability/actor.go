package observability

import (
	"context"
	"sync"
)

// ActorSink menyimpan identitas pemohon selama satu permintaan.
//
// Autentikasi berjalan di dalam router, sehingga principal hanya tersedia pada
// context yang diteruskan ke handler. Middleware di luar router memakai sink
// ini untuk membaca identitas pemohon setelah handler selesai.
type ActorSink struct {
	mu     sync.Mutex
	userID string
	role   string
	found  bool
}

// Observe mencatat identitas pemohon yang sudah terverifikasi.
func (s *ActorSink) Observe(userID, role string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.userID = userID
	s.role = role
	s.found = true
}

// Load mengembalikan identitas pemohon bila sudah terverifikasi.
func (s *ActorSink) Load() (string, string, bool) {
	if s == nil {
		return "", "", false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.userID, s.role, s.found
}

type actorSinkKey struct{}

// WithActorSink menyimpan sink identitas pemohon pada context permintaan.
func WithActorSink(ctx context.Context, sink *ActorSink) context.Context {
	return context.WithValue(ctx, actorSinkKey{}, sink)
}

// ActorSinkFromContext mengembalikan sink identitas pemohon dari context.
func ActorSinkFromContext(ctx context.Context) (*ActorSink, bool) {
	sink, ok := ctx.Value(actorSinkKey{}).(*ActorSink)
	return sink, ok
}
