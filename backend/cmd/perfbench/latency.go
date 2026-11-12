package main

import (
	"context"
	"flag"
	"fmt"
	"sort"
	"time"
)

// latencyOptions memuat parameter subcommand latency.
type latencyOptions struct {
	Iterations int
	Warmup     int
	Concurrent int
}

// latencyStats memuat ringkasan distribusi latensi.
type latencyStats struct {
	Count int
	Min   time.Duration
	Max   time.Duration
	Mean  time.Duration
	P50   time.Duration
	P95   time.Duration
	P99   time.Duration
}

// runLatency mengukur distribusi latensi tiap skenario terhadap database
// nyata. Pengukuran memakai pemanggilan repository produksi sehingga angka
// mencakup waktu jaringan, waktu parsing driver, dan waktu eksekusi
// PostgreSQL.
func runLatency(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("latency", flag.ContinueOnError)
	iterations := flags.Int("iterations", 50, "jumlah iterasi per skenario")
	warmup := flags.Int("warmup", 3, "jumlah iterasi pemanasan")
	concurrent := flags.Int("concurrent", 1, "jumlah goroutine paralel per skenario")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *iterations < 1 || *warmup < 0 || *concurrent < 1 {
		return fmt.Errorf("parameter iterasi tidak valid")
	}
	db, err := openDB(ctx)
	if err != nil {
		return err
	}
	env, err := loadScenarioEnv(ctx, db)
	if err != nil {
		return err
	}
	options := latencyOptions{Iterations: *iterations, Warmup: *warmup, Concurrent: *concurrent}
	fmt.Printf("latensi %d iterasi per skenario, %d pemanasan, %d goroutine paralel\n\n",
		options.Iterations, options.Warmup, options.Concurrent)
	printLatencyHeader()
	scenarios := buildScenarios(env)
	for _, item := range scenarios {
		stats, err := measureScenario(ctx, item, options)
		if err != nil {
			return scenarioError(item.Name, err)
		}
		printLatencyRow(item, stats)
	}
	return nil
}

// measureScenario menjalankan satu skenario berulang lalu merangkum latensi.
func measureScenario(
	ctx context.Context,
	item scenario,
	options latencyOptions,
) (latencyStats, error) {
	for index := 0; index < options.Warmup; index++ {
		if err := item.Run(ctx); err != nil {
			return latencyStats{}, err
		}
	}
	samples := make([]time.Duration, 0, options.Iterations*options.Concurrent)
	if options.Concurrent == 1 {
		for index := 0; index < options.Iterations; index++ {
			started := time.Now()
			if err := item.Run(ctx); err != nil {
				return latencyStats{}, err
			}
			samples = append(samples, time.Since(started))
		}
		return summarize(samples), nil
	}

	perWorker := options.Iterations
	results := make(chan []time.Duration, options.Concurrent)
	for worker := 0; worker < options.Concurrent; worker++ {
		go func() {
			local := make([]time.Duration, 0, perWorker)
			for index := 0; index < perWorker; index++ {
				started := time.Now()
				if err := item.Run(ctx); err != nil {
					results <- local
					return
				}
				local = append(local, time.Since(started))
			}
			results <- local
		}()
	}
	for worker := 0; worker < options.Concurrent; worker++ {
		samples = append(samples, <-results...)
	}
	return summarize(samples), nil
}

// summarize menghitung persentil dari kumpulan sampel latensi.
func summarize(samples []time.Duration) latencyStats {
	if len(samples) == 0 {
		return latencyStats{}
	}
	sorted := append([]time.Duration(nil), samples...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	var total time.Duration
	for _, sample := range sorted {
		total += sample
	}
	return latencyStats{
		Count: len(sorted),
		Min:   sorted[0],
		Max:   sorted[len(sorted)-1],
		Mean:  total / time.Duration(len(sorted)),
		P50:   percentile(sorted, 0.50),
		P95:   percentile(sorted, 0.95),
		P99:   percentile(sorted, 0.99),
	}
}

// percentile menghitung persentil dengan interpolasi linier pada sampel
// yang sudah terurut.
func percentile(sorted []time.Duration, fraction float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	if len(sorted) == 1 {
		return sorted[0]
	}
	position := fraction * float64(len(sorted)-1)
	lower := int(position)
	upper := lower + 1
	if upper >= len(sorted) {
		return sorted[len(sorted)-1]
	}
	weight := position - float64(lower)
	return time.Duration(float64(sorted[lower])*(1-weight) + float64(sorted[upper])*weight)
}

// printLatencyHeader mencetak judul tabel hasil.
func printLatencyHeader() {
	fmt.Printf("%-28s %6s %10s %10s %10s %10s %10s\n",
		"SKENARIO", "N", "MIN", "P50", "P95", "P99", "MAKS")
	fmt.Println(repeat("-", 90))
}

// printLatencyRow mencetak satu baris hasil.
func printLatencyRow(item scenario, stats latencyStats) {
	fmt.Printf("%-28s %6d %10s %10s %10s %10s %10s\n",
		item.Name, stats.Count,
		formatDuration(stats.Min),
		formatDuration(stats.P50),
		formatDuration(stats.P95),
		formatDuration(stats.P99),
		formatDuration(stats.Max))
}

// repeat mengulang karakter sejumlah n kali.
func repeat(character string, count int) string {
	result := make([]byte, 0, count)
	for index := 0; index < count; index++ {
		result = append(result, character[0])
	}
	return string(result)
}
