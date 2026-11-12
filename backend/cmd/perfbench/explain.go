package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"strings"
	"time"
)

// runExplain menjalankan EXPLAIN (ANALYZE, BUFFERS) atas statement SQL yang
// benar-benar dibangkitkan repository produksi.
//
// Statement direkam saat skenario dijalankan, lalu statement yang sama
// dijalankan ulang dengan awalan EXPLAIN. Karena itu angka yang tampil
// berasal dari SQL yang sama dengan yang diterima PostgreSQL di produksi.
func runExplain(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("explain", flag.ContinueOnError)
	scenario := flags.String("scenario", "", "nama skenario tunggal (kosong = semua)")
	verbose := flags.Bool("verbose", false, "tampilkan plan lengkap")
	if err := flags.Parse(args); err != nil {
		return err
	}
	db, err := openDB(ctx)
	if err != nil {
		return err
	}
	env, err := loadScenarioEnv(ctx, db)
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("ambil handle database: %w", err)
	}
	recorder := newSQLRecorder(db)
	scenarios := buildScenarios(env)
	fmt.Printf("penjelasan kueri pada %s\n\n", time.Now().UTC().Format(time.RFC3339))
	for _, item := range scenarios {
		if *scenario != "" && item.Name != *scenario {
			continue
		}
		recorder.start()
		runErr := item.Run(ctx)
		statements := recorder.stop()
		if runErr != nil {
			return scenarioError(item.Name, runErr)
		}
		fmt.Printf("=== %s ===\n%s\n", item.Name, item.Description)
		if len(statements) == 0 {
			fmt.Println("  (tidak ada statement yang terekam)")
			continue
		}
		for index, statement := range statements {
			if err := printExplain(ctx, sqlDB, statement, *verbose); err != nil {
				return scenarioError(item.Name, err)
			}
			if index < len(statements)-1 {
				fmt.Println()
			}
		}
		fmt.Println()
	}
	return nil
}

// printExplain menjalankan satu statement dalam mode EXPLAIN dan mencetak
// hasilnya.
func printExplain(
	ctx context.Context,
	sqlDB *sql.DB,
	statement capturedStatement,
	verbose bool,
) error {
	query := "EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT) " + statement.SQL
	started := time.Now()
	rows, err := sqlDB.QueryContext(ctx, query, statement.Args...)
	if err != nil {
		return fmt.Errorf("jalankan explain: %w", err)
	}
	defer rows.Close()
	plan, err := collectPlan(rows)
	if err != nil {
		return err
	}
	fmt.Printf("  [%d] %s\n", 1, truncateSQL(statement.SQL, 110))
	fmt.Printf("      statement: %s\n", formatDuration(time.Since(started)))
	for _, line := range plan {
		if !verbose && isNoiseLine(line) {
			continue
		}
		fmt.Printf("      %s\n", line)
	}
	return nil
}

// collectPlan membaca seluruh baris hasil EXPLAIN.
func collectPlan(rows *sql.Rows) ([]string, error) {
	lines := make([]string, 0, 32)
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			return nil, fmt.Errorf("baca hasil explain: %w", err)
		}
		lines = append(lines, line)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("baca hasil explain: %w", err)
	}
	return lines, nil
}

// isNoiseLine menyaring baris plan yang tidak informatif untuk ringkasan.
func isNoiseLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	switch {
	case strings.HasPrefix(trimmed, "Buffers:"):
		return true
	case strings.HasPrefix(trimmed, "Planning:"):
		return true
	case strings.HasPrefix(trimmed, "Worker 0:"):
		return true
	case trimmed == "":
		return true
	}
	return false
}

// truncateSQL memotong SQL panjang agar tabel hasil tetap terbaca.
func truncateSQL(query string, limit int) string {
	normalized := strings.Join(strings.Fields(query), " ")
	if len(normalized) <= limit {
		return normalized
	}
	return normalized[:limit-1] + "…"
}

// formatDuration merumuskan durasi dalam satuan yang mudah dibaca.
func formatDuration(value time.Duration) string {
	switch {
	case value < time.Millisecond:
		return fmt.Sprintf("%.3f ms", float64(value.Microseconds())/1000)
	case value < time.Second:
		return fmt.Sprintf("%.2f ms", float64(value.Microseconds())/1000)
	default:
		return fmt.Sprintf("%.2f s", value.Seconds())
	}
}
