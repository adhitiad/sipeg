package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"simpeg/backend/internal/cache"
	"simpeg/backend/internal/notify"
	"simpeg/backend/internal/qstash"
)

// depsOptions memuat parameter subcommand deps.
type depsOptions struct {
	Redis       bool
	QStash      bool
	SMTP        bool
	Count       int
	Concurrency int
}

// loadResult memuat hasil satu uji beban.
type loadResult struct {
	Name       string
	Count      int
	Errors     int
	Elapsed    time.Duration
	Stats      latencyStats
	Throughput float64
	FirstError string
}

// runDeps menjalankan uji beban nyata terhadap Redis, QStash, dan SMTP.
//
// Uji ini menyentuh layanan pihak ketiga. Redis hanya dipakai untuk cache
// data referensi dan kuncinya dihapus setelah pengukuran. QStash menerima
// pesan uji yang tujuannya memang tidak pernah dieksekusi. SMTP mengirim
// email sungguhan; penerima bawaan adalah alamat akun SMTP sendiri sehingga
// uji beban tidak mengirimi pihak ketiga.
func runDeps(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("deps", flag.ContinueOnError)
	useRedis := flags.Bool("redis", false, "ukur beban cache Redis")
	useQStash := flags.Bool("qstash", false, "ukur beban publikasi QStash")
	useSMTP := flags.Bool("smtp", false, "ukur beban pengiriman SMTP")
	count := flags.Int("count", 100, "jumlah operasi per uji")
	concurrency := flags.Int("concurrency", 8, "jumlah operasi paralel")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if !*useRedis && !*useQStash && !*useSMTP {
		return fmt.Errorf("pilih minimal satu uji: --redis, --qstash, atau --smtp")
	}
	options := depsOptions{
		Redis:       *useRedis,
		QStash:      *useQStash,
		SMTP:        *useSMTP,
		Count:       *count,
		Concurrency: *concurrency,
	}
	if options.Count < 1 || options.Concurrency < 1 {
		return fmt.Errorf("jumlah operasi tidak valid")
	}
	if options.Redis {
		if err := runRedisLoad(ctx, options); err != nil {
			return err
		}
	}
	if options.QStash {
		if err := runQStashLoad(ctx, options); err != nil {
			return err
		}
	}
	if options.SMTP {
		if err := runSMTPLoad(ctx, options); err != nil {
			return err
		}
	}
	return nil
}

// runLoad menjalankan operasi berulang secara paralel dan merangkum hasilnya.
func runLoad(
	ctx context.Context,
	name string,
	count int,
	concurrency int,
	operation func(ctx context.Context, index int) error,
) loadResult {
	var (
		mu       sync.Mutex
		samples  = make([]time.Duration, 0, count)
		failures int
		firstErr string
	)
	started := time.Now()
	work := make(chan int)
	var group sync.WaitGroup
	for worker := 0; worker < concurrency; worker++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for index := range work {
				operationStarted := time.Now()
				err := operation(ctx, index)
				elapsed := time.Since(operationStarted)
				mu.Lock()
				samples = append(samples, elapsed)
				if err != nil {
					failures++
					if firstErr == "" {
						firstErr = err.Error()
					}
				}
				mu.Unlock()
			}
		}()
	}
	for index := 0; index < count; index++ {
		work <- index
	}
	close(work)
	group.Wait()
	elapsed := time.Since(started)
	result := loadResult{
		Name:       name,
		Count:      count,
		Errors:     failures,
		Elapsed:    elapsed,
		Stats:      summarize(samples),
		Throughput: float64(count) / elapsed.Seconds(),
		FirstError: firstErr,
	}
	return result
}

// printLoadResult mencetak hasil satu uji beban.
func printLoadResult(result loadResult) {
	fmt.Printf("=== %s ===\n", result.Name)
	fmt.Printf("  operasi: %d, gagal: %d, durasi: %s, throughput: %.1f ops/detik\n",
		result.Count, result.Errors, formatDuration(result.Elapsed), result.Throughput)
	fmt.Printf("  latensi min %s p50 %s p95 %s p99 %s maks %s\n",
		formatDuration(result.Stats.Min),
		formatDuration(result.Stats.P50),
		formatDuration(result.Stats.P95),
		formatDuration(result.Stats.P99),
		formatDuration(result.Stats.Max))
	if result.FirstError != "" {
		fmt.Printf("  contoh kesalahan: %s\n", truncateSQL(result.FirstError, 160))
	}
	fmt.Println()
}

// runRedisLoad mengukur latensi tulis dan baca cache Redis, lalu membanding
// kannya dengan pembacaan yang sama langsung dari PostgreSQL.
func runRedisLoad(ctx context.Context, options depsOptions) error {
	baseURL, err := requireEnv("UPSTASH_REDIS_REST_URL")
	if err != nil {
		return err
	}
	token, err := requireEnv("UPSTASH_REDIS_REST_TOKEN")
	if err != nil {
		return err
	}
	store := cache.New(cache.NewRESTClient(baseURL, token), cache.Config{
		Namespace: "simpeg:perfbench",
		TTL:       5 * time.Minute,
	})
	key := fmt.Sprintf("cache-probe:%d", time.Now().UnixNano())
	defer func() { _ = store.Delete(context.Background(), key) }()

	payload := map[string]string{"probe": "kategori-pegawai", "isi": strings.Repeat("x", 512)}
	setResult := runLoad(ctx, "redis.set", options.Count, options.Concurrency,
		func(ctx context.Context, _ int) error {
			return store.Set(ctx, key, payload)
		})
	printLoadResult(setResult)

	getResult := runLoad(ctx, "redis.get", options.Count, options.Concurrency,
		func(ctx context.Context, _ int) error {
			var target map[string]string
			return store.Get(ctx, key, &target)
		})
	printLoadResult(getResult)

	return compareCacheWithDatabase(ctx, options)
}

// compareCacheWithDatabase membandingkan pembacaan daftar kategori dari cache
// dengan pembacaan langsung dari PostgreSQL.
func compareCacheWithDatabase(ctx context.Context, options depsOptions) error {
	db, err := openDB(ctx)
	if err != nil {
		return err
	}
	env, err := loadScenarioEnv(ctx, db)
	if err != nil {
		return err
	}
	scenarios := buildScenarios(env)
	for _, item := range scenarios {
		if item.Name != "categories.list" {
			continue
		}
		databaseResult := runLoad(ctx, "postgres.categories.list", options.Count, options.Concurrency,
			func(ctx context.Context, _ int) error {
				return item.Run(ctx)
			})
		printLoadResult(databaseResult)
	}
	return nil
}

// runQStashLoad mengukur latensi publikasi pekerjaan ke QStash.
//
// Pesan uji dikirim ke tujuan sekali pakai yang memang tidak pernah
// dieksekusi. Keberhasilan publikasi diukur; pengiriman tidak diukur karena
// tujuan uji tidak dapat dijangkau.
func runQStashLoad(ctx context.Context, options depsOptions) error {
	token, err := requireEnv("QSTASH_TOKEN")
	if err != nil {
		return err
	}
	publisher := &qstash.Publisher{
		BaseURL: envOr("QSTASH_URL", qstash.DefaultPublishURL),
		Token:   token,
	}
	destination := envOr("PERFBENCH_QSTASH_DESTINATION", "https://example.com/slo")
	body, err := json.Marshal(map[string]any{
		"kinds":  []string{"slo.evaluate"},
		"waktu":  time.Now().UTC().Format(time.RFC3339),
		"sumber": "perfbench",
	})
	if err != nil {
		return fmt.Errorf("susun payload uji: %w", err)
	}
	result := runLoad(ctx, "qstash.publish", options.Count, options.Concurrency,
		func(ctx context.Context, index int) error {
			return publisher.Publish(ctx, qstash.PublishRequest{
				Destination: destination,
				Body:        body,
				Headers:     map[string]string{"X-Perfbench-Index": fmt.Sprintf("%d", index)},
				RetryCount:  0,
			})
		})
	printLoadResult(result)
	fmt.Printf("  tujuan uji: %s (pesan uji tidak dieksekusi)\n\n", destination)
	return nil
}

// runSMTPLoad mengukur latensi pengiriman email lewat SMTP.
//
// Penerima bawaan adalah alamat akun SMTP sendiri agar uji beban tidak
// mengirim email ke pihak ketiga. Daftar penerima lain dapat diberikan lewat
// PERFBENCH_SMTP_RECIPIENTS sebagai daftar dipisah koma.
func runSMTPLoad(ctx context.Context, options depsOptions) error {
	host, err := requireEnv("SMTP_HOST")
	if err != nil {
		return err
	}
	from, err := requireEnv("SMTP_FROM")
	if err != nil {
		return err
	}
	port := envInt("SMTP_PORT", 587)
	user := os.Getenv("SMTP_USER")
	password := os.Getenv("SMTP_PASS")
	recipients := smtpRecipients(user)
	if len(recipients) == 0 {
		return fmt.Errorf("tidak ada penerima SMTP yang dapat ditentukan")
	}
	sender := notify.NewSMTPSender(notify.SMTPConfig{
		Host:     host,
		Port:     port,
		Username: user,
		Password: password,
		From:     from,
		UseTLS:   os.Getenv("SMTP_SECURE") == "true" || (os.Getenv("SMTP_SECURE") == "" && port == 465),
		Timeout:  20 * time.Second,
	})
	result := runLoad(ctx, "smtp.send", options.Count, options.Concurrency,
		func(ctx context.Context, index int) error {
			recipient := recipients[index%len(recipients)]
			message := notify.GenericNotice(
				recipient,
				"[perfbench] Uji beban SMTP SIMPEG",
				"Email ini dikirim oleh uji beban performa SIMPEG. Tidak ada aksi yang diperlukan.",
				fmt.Sprintf("perfbench:%d:%d", time.Now().UnixNano(), index),
			)
			return sender.Send(ctx, message)
		})
	printLoadResult(result)
	fmt.Printf("  penerima uji: %s\n\n", strings.Join(recipients, ", "))
	return nil
}

// smtpRecipients menentukan daftar penerima uji beban.
func smtpRecipients(account string) []string {
	if raw := strings.TrimSpace(os.Getenv("PERFBENCH_SMTP_RECIPIENTS")); raw != "" {
		parts := strings.Split(raw, ",")
		recipients := make([]string, 0, len(parts))
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				recipients = append(recipients, trimmed)
			}
		}
		return recipients
	}
	if trimmed := strings.TrimSpace(account); trimmed != "" {
		return []string{trimmed}
	}
	return nil
}
