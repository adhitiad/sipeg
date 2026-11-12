// Package slo memuat pengoleksi metrik yang mengukur latensi dan galat
// setiap permintaan HTTP, lalu menyimpannya sebagai agregasi per rute per
// jendela waktu.
//
// Pengoleksi bekerja di dalam proses backend. Agregasi disimpan di memori
// sehingga jalur permintaan tidak menambah satu statement database per
// permintaan; penulisan ke database dilakukan berkala dalam jumlah kecil.
package slo

import (
	"context"
	"net/http"
	"sync"
	"time"

	"simpeg/backend/internal/observability"
	"simpeg/backend/internal/slo/domain"
)

// Store adalah port persistensi metrik.
type Store interface {
	// RecordBatch menyimpan agregasi metrik untuk sebuah jendela waktu.
	RecordBatch(ctx context.Context, window domain.Window, metrics []domain.Metric) error
}

// excludedPaths adalah path yang tidak diukur sebagai SLO layanan.
//
// Endpoint liveness dan readiness memeriksa proses dan dependensi, bukan
// perilaku layanan pengguna. Memasukkannya ke SLO akan mengaburkan tanda
// layanan yang sedang menurun.
var excludedPaths = map[string]bool{
	"/health": true,
	"/ready":  true,
}

// windowBucket adalah agregasi metrik untuk satu jendela waktu.
type windowBucket struct {
	window  domain.Window
	metrics map[string]*domain.Metric
}

// Collector menghimpun metrik permintaan dan menulisnya ke store.
type Collector struct {
	store Store
	// windowSize adalah panjang satu jendela agregasi.
	windowSize time.Duration
	logger     *observability.Logger

	mu      sync.Mutex
	windows map[time.Time]*windowBucket
}

// CollectorOptions memuat konfigurasi pengoleki metrik.
type CollectorOptions struct {
	// Store menerima agregasi metrik. Wajib diisi.
	Store Store
	// WindowSize adalah panjang jendela agregasi. Nil memakai lima menit.
	WindowSize time.Duration
	// Logger menerima catatan kegagalan penulisan metrik. Nil berarti
	// kegagalan tidak dicatat.
	Logger *observability.Logger
}

// NewCollector membuat pengoleki metrik.
func NewCollector(options CollectorOptions) *Collector {
	windowSize := options.WindowSize
	if windowSize <= 0 {
		windowSize = 5 * time.Minute
	}
	return &Collector{
		store:      options.Store,
		windowSize: windowSize,
		logger:     options.Logger,
		windows:    map[time.Time]*windowBucket{},
	}
}

// Record menambahkan satu sampel permintaan ke pengoleki.
//
// failed menandai permintaan yang berakhir dengan galat server. Sampel
// milik jendela yang sudah pernah ditulis tetap diterima, karena penulisan
// selalu menggantikan seluruh nilai dan tidak pernah menambahkannya.
func (c *Collector) Record(method, route string, status int, latency time.Duration) {
	if c == nil || c.store == nil {
		return
	}
	now := time.Now().UTC()
	start := now.Truncate(c.windowSize)
	failed := status >= http.StatusInternalServerError
	key := method + " " + route

	c.mu.Lock()
	bucket, ok := c.windows[start]
	if !ok {
		bucket = &windowBucket{
			window:  domain.Window{Start: start, End: start.Add(c.windowSize)},
			metrics: map[string]*domain.Metric{},
		}
		c.windows[start] = bucket
	}
	metric, ok := bucket.metrics[key]
	if !ok {
		created := domain.NewMetric(route, method)
		bucket.metrics[key] = &created
		metric = &created
	}
	metric.Observe(latency, failed)
	c.mu.Unlock()
}

// Flush menulis seluruh agregasi yang terkumpul ke store.
//
// Penulisan memakai penggantian penuh pada kombinasi jendela, metode, dan
// rute, sehingga flush berulang tidak menggandakan hitungan. Jendela yang
// sudah lewat tetap ditulis ulang, tetapi isinya sama dengan isi sebelumnya.
// Flush menulis metrik dari jendela yang sudah selesai ke store.
//
// Hanya jendela yang sudah lewat yang ditulis dan kemudian dibuang dari
// memori. Jendela yang sedang berjalan tetap dikumpulkan supaya isinya
// lengkap ketika jendela tersebut berakhir.
//
// Penulisan memakai penggantian penuh pada kombinasi jendela, metode, dan
// rute, sehingga flush berulang tidak menggandakan hitungan.
func (c *Collector) Flush(ctx context.Context) error {
	return c.flush(ctx, false)
}

// flushAll menulis seluruh jendela yang terkumpul, termasuk yang sedang
// berjalan. Fungsi ini dipakai saat proses berhenti supaya data terakhir
// tidak hilang.
func (c *Collector) flushAll(ctx context.Context) error {
	return c.flush(ctx, true)
}

// flush menjalankan penulisan metrik.
//
// When includeCurrent bernilai true, jendela yang sedang berjalan ikut
// ditulis. Nilai itu hanya dipakai saat proses berhenti.
func (c *Collector) flush(ctx context.Context, includeCurrent bool) error {
	if c == nil || c.store == nil {
		return nil
	}
	now := time.Now().UTC()
	c.mu.Lock()
	pending := make([]windowBucket, 0, len(c.windows))
	for start, bucket := range c.windows {
		if len(bucket.metrics) == 0 {
			delete(c.windows, start)
			continue
		}
		if !includeCurrent && bucket.window.End.After(now) {
			continue
		}
		pending = append(pending, *bucket)
		delete(c.windows, start)
	}
	c.mu.Unlock()

	var firstErr error
	for _, bucket := range pending {
		metrics := make([]domain.Metric, 0, len(bucket.metrics))
		for _, metric := range bucket.metrics {
			metrics = append(metrics, *metric)
		}
		if err := c.store.RecordBatch(ctx, bucket.window, metrics); err != nil {
			if c.logger != nil {
				c.logger.Error("slo_metric_write_failed", observability.Fields{
					"error": err.Error(),
				})
			}
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

// Run menulis metrik secara berkala sampai konteks dibatalkan.
//
// Interval penulisan disetel sama dengan panjang jendela sehingga setiap
// jendela ditulis setidaknya satu kali.
func (c *Collector) Run(ctx context.Context) {
	ticker := time.NewTicker(c.windowSize)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			// Jendela terakhir ditulis saat proses berhenti agar data tidak
			// hilang pada pergantian versi.
			flushCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			if err := c.flushAll(flushCtx); err != nil && c.logger != nil {
				c.logger.Error("slo_metric_final_flush_failed", observability.Fields{
					"error": err.Error(),
				})
			}
			cancel()
			return
		case <-ticker.C:
			flushCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
			if err := c.Flush(flushCtx); err != nil && c.logger != nil {
				c.logger.Error("slo_metric_flush_failed", observability.Fields{
					"error": err.Error(),
				})
			}
			cancel()
		}
	}
}

// Middleware mengukur setiap permintaan HTTP dan mencatatnya ke pengoleki.
//
// Rute yang diukur adalah pola rute dari ServeMux, bukan path mentah, agar
// jumlah kunci metrik tetap kecil dan tidak mengikutsertaan identifier.
func (c *Collector) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if excludedPaths[r.URL.Path] {
			next.ServeHTTP(w, r)
			return
		}
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		started := time.Now()
		next.ServeHTTP(recorder, r)
		c.Record(r.Method, routeOf(r), recorder.status, time.Since(started))
	})
}

// routeOf mengembalikan pola rute yang cocok pada permintaan.
//
// Pola ini hanya tersedia setelah ServeMux mencocokkan permintaan. Bila
// permintaan tidak tercocokkan, path mentah dipakai sebagai gantinya.
func routeOf(r *http.Request) string {
	if r.Pattern != "" {
		return r.Pattern
	}
	return r.URL.Path
}

// statusRecorder mencatat kode status respons tanpa mengubahnya.
type statusRecorder struct {
	http.ResponseWriter
	status  int
	written bool
}

// WriteHeader menyimpan kode status sebelum meneruskannya.
func (s *statusRecorder) WriteHeader(code int) {
	if !s.written {
		s.status = code
		s.written = true
	}
	s.ResponseWriter.WriteHeader(code)
}

// Write menandai respons sudah ditulis bila belum.
func (s *statusRecorder) Write(data []byte) (int, error) {
	s.written = true
	return s.ResponseWriter.Write(data)
}

// Unwrap mengembalikan writer asli sehingga http.ResponseController tetap
// dapat menjangkau antarmuka opsional seperti Flusher.
func (s *statusRecorder) Unwrap() http.ResponseWriter {
	return s.ResponseWriter
}
