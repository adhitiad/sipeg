package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// e2eOptions memuat parameter subcommand e2e.
type e2eOptions struct {
	Duration    time.Duration
	Streams     int
	Requests    int
	Soak        bool
	ShowBody    bool
	KeepRunning bool
}

// endpoint adalah satu target permintaan HTTP.
type endpoint struct {
	Path   string
	Weight int
}

// endpoints adalah daftar endpoint yang diukur.
//
// Endpoint dipilih agar seluruh jalur baca terwakili: direktori, audit,
// cache, dan penggajian. Bobot mencerminkan freqkuensi pemakaian yang
// diharapkan: daftar pegawai paling sering, slip gaji paling jarang.
var endpoints = []endpoint{
	{Path: "/api/v1/me", Weight: 5},
	{Path: "/api/v1/employees?page=1&pageSize=20&sort=nama&order=asc", Weight: 30},
	{Path: "/api/v1/employees?page=1&pageSize=50&sort=nama&order=asc", Weight: 15},
	{Path: "/api/v1/employees?search=Pegawai+Sintetis+0001&page=1&pageSize=20", Weight: 10},
	{Path: "/api/v1/audit-logs?page=1&pageSize=20&sort=createdAt&order=desc", Weight: 20},
	{Path: "/api/v1/categories?page=1&pageSize=50", Weight: 10},
	{Path: "/api/v1/payroll/slips?page=1&pageSize=20&sort=periode&order=desc", Weight: 10},
}

// runE2E menjalankan server produksi lalu mengukur beban HTTP nyata.
func runE2E(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("e2e", flag.ContinueOnError)
	duration := flags.Duration("duration", 60*time.Second, "durasi uji (mode soak)")
	streams := flags.Int("streams", 16, "jumlah permintaan paralel")
	requests := flags.Int("requests", 300, "jumlah permintaan (mode burst)")
	soak := flags.Bool("soak", false, "mode soak: jalankan selama durasi tertentu")
	showBody := flags.Bool("show-body", false, "tampilkan body respons pada kegagalan")
	keepRunning := flags.Bool("keep-running", false, "biarkan server hidup setelah pengukuran")
	if err := flags.Parse(args); err != nil {
		return err
	}
	options := e2eOptions{
		Duration:    *duration,
		Streams:     *streams,
		Requests:    *requests,
		Soak:        *soak,
		ShowBody:    *showBody,
		KeepRunning: *keepRunning,
	}
	if options.Streams < 1 || options.Requests < 1 || options.Duration <= 0 {
		return fmt.Errorf("parameter uji tidak valid")
	}

	server, err := startServer(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if !options.KeepRunning {
			_ = server.stop()
		}
	}()

	token, err := issuePerfSession(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("server produksi berjalan di %s\n", server.BaseURL)
	fmt.Printf("uji: %s\n\n", describeMode(options))

	if options.Soak {
		return runSoak(ctx, server, token, options)
	}
	return runBurst(ctx, server, token, options)
}

// describeMode menjelaskan mode uji yang dijalankan.
func describeMode(options e2eOptions) string {
	if options.Soak {
		return fmt.Sprintf("soak selama %s dengan %d stream paralel", options.Duration, options.Streams)
	}
	return fmt.Sprintf("burst %d permintaan dengan %d stream paralel", options.Requests, options.Streams)
}

// requestStats menghimpun statistik respons HTTP.
type requestStats struct {
	mu        sync.Mutex
	latencies []time.Duration
	statuses  map[int]int
	errors    int
	firstBody string
}

// record menyimpan satu hasil permintaan.
func (s *requestStats) record(latency time.Duration, status int, body string, failed bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.latencies = append(s.latencies, latency)
	if s.statuses == nil {
		s.statuses = map[int]int{}
	}
	s.statuses[status]++
	if failed {
		s.errors++
		if s.firstBody == "" {
			s.firstBody = body
		}
	}
}

// snapshot merangkum statistik yang terkumpul.
func (s *requestStats) snapshot() (latencyStats, map[int]int, int, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	statuses := make(map[int]int, len(s.statuses))
	for code, count := range s.statuses {
		statuses[code] = count
	}
	return summarize(s.latencies), statuses, s.errors, s.firstBody
}

// runBurst menjalankan sejumlah permintaan lalu mencetak ringkasan.
func runBurst(ctx context.Context, server *serverProcess, token string, options e2eOptions) error {
	stats := &requestStats{}
	client := &http.Client{Timeout: 30 * time.Second}
	work := make(chan int)
	var group sync.WaitGroup
	for worker := 0; worker < options.Streams; worker++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for index := range work {
				executeRequest(ctx, client, server.BaseURL, token, endpoints[index%len(endpoints)], stats, options.ShowBody)
			}
		}()
	}
	started := time.Now()
	for index := 0; index < options.Requests; index++ {
		work <- index
	}
	close(work)
	group.Wait()
	elapsed := time.Since(started)
	return reportHTTP(stats, elapsed, options)
}

// runSoak menjalankan beban berkelanjutan selama durasi tertentu dan
// mencatat pergeseran latensi serta laju kegagalan.
func runSoak(ctx context.Context, server *serverProcess, token string, options e2eOptions) error {
	stats := &requestStats{}
	client := &http.Client{Timeout: 30 * time.Second}
	deadline := time.Now().Add(options.Duration)
	var counter atomic.Int64
	var group sync.WaitGroup
	for worker := 0; worker < options.Streams; worker++ {
		group.Add(1)
		go func(worker int) {
			defer group.Done()
			index := worker
			for time.Now().Before(deadline) {
				executeRequest(ctx, client, server.BaseURL, token, endpoints[index%len(endpoints)], stats, options.ShowBody)
				index++
				counter.Add(1)
			}
		}(worker)
	}
	started := time.Now()
	group.Wait()
	elapsed := time.Since(started)
	fmt.Printf("total permintaan: %d\n", counter.Load())
	return reportHTTP(stats, elapsed, options)
}

// executeRequest menjalankan satu permintaan HTTP dan mencatat hasilnya.
func executeRequest(
	ctx context.Context,
	client *http.Client,
	baseURL string,
	token string,
	target endpoint,
	stats *requestStats,
	showBody bool,
) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+target.Path, nil)
	if err != nil {
		stats.record(0, 0, err.Error(), true)
		return
	}
	request.Header.Set("Authorization", "Bearer "+token)
	started := time.Now()
	response, err := client.Do(request)
	elapsed := time.Since(started)
	if err != nil {
		stats.record(elapsed, 0, err.Error(), true)
		return
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
	failed := response.StatusCode >= 500
	stats.record(elapsed, response.StatusCode, string(body), failed)
	if showBody && failed {
		fmt.Printf("gagal %d: %s\n", response.StatusCode, truncateSQL(string(body), 300))
	}
}

// reportHTTP mencetak ringkasan hasil pengukuran HTTP.
func reportHTTP(stats *requestStats, elapsed time.Duration, options e2eOptions) error {
	latency, statuses, failures, body := stats.snapshot()
	fmt.Printf("durasi: %s, throughput: %.1f req/detik\n",
		formatDuration(elapsed), float64(latency.Count)/elapsed.Seconds())
	fmt.Printf("latensi min %s p50 %s p95 %s p99 %s maks %s\n",
		formatDuration(latency.Min), formatDuration(latency.P50),
		formatDuration(latency.P95), formatDuration(latency.P99),
		formatDuration(latency.Max))
	codes := make([]int, 0, len(statuses))
	for code := range statuses {
		codes = append(codes, code)
	}
	for index := 0; index < len(codes); index++ {
		for position := index + 1; position < len(codes); position++ {
			if codes[position] < codes[index] {
				codes[index], codes[position] = codes[position], codes[index]
			}
		}
	}
	fmt.Print("kode status: ")
	for _, code := range codes {
		fmt.Printf("%d=%d ", code, statuses[code])
	}
	fmt.Println()
	if failures > 0 {
		fmt.Printf("gagal: %d\n", failures)
		if body != "" {
			fmt.Printf("contoh respons: %s\n", truncateSQL(body, 300))
		}
	}
	return nil
}
