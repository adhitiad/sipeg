package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

// employeeImportRowCap adalah batas baris data per berkas impor (lihat
// employeeImportRowLimit di paket httpserver).
const employeeImportRowCap = 1000

// runImportExport mengukur jalur impor dan ekspor pegawai terhadap server
// asli. Ekspor diukur sebagai GET /api/v1/employees/export (seluruh pegawai
// aktif, tanpa pagination) dan impor sebagai POST /api/v1/employees/import
// dengan berkas CSV multipart buatan perkakas ini.
//
// Database harus sudah diisi data sintetis (`perfbench seed`). Pengukuran
// impor menambah baris ke tabel "user"; jalankan pada database uji yang
// dapat dibuang, bukan pada database yang dipakai sehari-hari.
func runImportExport(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("importexport", flag.ContinueOnError)
	importRows := flags.Int("import-rows", employeeImportRowCap, "jumlah baris pegawai per berkas impor (maks 1000)")
	importRuns := flags.Int("import-runs", 1, "jumlah percobaan impor")
	exportRuns := flags.Int("export-runs", 5, "jumlah percobaan ekspor")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *importRows < 1 || *importRows > employeeImportRowCap || *importRuns < 1 || *exportRuns < 1 {
		return fmt.Errorf("parameter impor/ekspor tidak valid")
	}

	db, err := openDB(ctx)
	if err != nil {
		return err
	}
	var total int64
	if err := db.Raw(`SELECT count(*) FROM "user" WHERE "user"."aktif" = $1`, true).Scan(&total).Error; err != nil {
		return fmt.Errorf("hitung pegawai aktif: %w", err)
	}

	server, err := startServer(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = server.stop() }()

	token, err := issuePerfSession(ctx)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 180 * time.Second}

	fmt.Printf("pegawai aktif di database: %d\n", total)
	fmt.Printf("server produksi berjalan di %s\n\n", server.BaseURL)

	fmt.Printf("ekspor GET /api/v1/employees/export (%d percobaan)\n", *exportRuns)
	exportLatencies := make([]time.Duration, 0, *exportRuns)
	var exportRows int
	for run := 1; run <= *exportRuns; run++ {
		latency, rows, payload, err := measureExport(ctx, client, server.BaseURL, token)
		if err != nil {
			return fmt.Errorf("ekspor percobaan %d: %w", run, err)
		}
		exportLatencies = append(exportLatencies, latency)
		exportRows = rows
		fmt.Printf("  percobaan %d: %s, %d baris, %.1f KB\n", run, formatDuration(latency), rows, float64(payload)/1024)
	}
	exportStats := summarize(exportLatencies)
	fmt.Printf("  ringkasan: min %s p50 %s p95 %s maks %s, %.1f baris/detik\n\n",
		formatDuration(exportStats.Min),
		formatDuration(exportStats.P50),
		formatDuration(exportStats.P95),
		formatDuration(exportStats.Max),
		float64(exportRows)/exportStats.P50.Seconds(),
	)
	if total > 0 && exportRows != int(total) {
		return fmt.Errorf("jumlah baris ekspor %d tidak sama dengan pegawai aktif %d", exportRows, total)
	}

	fmt.Printf("impor POST /api/v1/employees/import (berkas CSV, %d baris, %d percobaan)\n", *importRows, *importRuns)
	importLatencies := make([]time.Duration, 0, *importRuns)
	for run := 1; run <= *importRuns; run++ {
		latency, imported, err := measureImport(ctx, client, server.BaseURL, token, *importRows, run)
		if err != nil {
			return fmt.Errorf("impor percobaan %d: %w", run, err)
		}
		importLatencies = append(importLatencies, latency)
		fmt.Printf("  percobaan %d: %s, %d pegawai diimpor, %.1f pegawai/detik\n",
			run, formatDuration(latency), imported, float64(imported)/latency.Seconds())
	}
	importStats := summarize(importLatencies)
	fmt.Printf("  ringkasan: min %s p50 %s p95 %s maks %s\n",
		formatDuration(importStats.Min),
		formatDuration(importStats.P50),
		formatDuration(importStats.P95),
		formatDuration(importStats.Max),
	)
	return nil
}

// measureExport memanggil endpoint ekspor sekali dan mengembalikan latensi,
// jumlah baris data, dan ukuran payload berkas CSV.
func measureExport(ctx context.Context, client *http.Client, baseURL, token string) (time.Duration, int, int64, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/v1/employees/export", nil)
	if err != nil {
		return 0, 0, 0, err
	}
	request.Header.Set("Authorization", "Bearer "+token)
	started := time.Now()
	response, err := client.Do(request)
	elapsed := time.Since(started)
	if err != nil {
		return 0, 0, 0, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("baca isi ekspor: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		return 0, 0, 0, fmt.Errorf("ekspor mengembalikan status %d: %s", response.StatusCode, truncateSQL(string(body), 300))
	}
	return elapsed, countExportRows(body), int64(len(body)), nil
}

// measureImport mengimpor satu berkas CSV multipart dan mengembalikan
// latensi beserta jumlah pegawai yang diimpor.
func measureImport(ctx context.Context, client *http.Client, baseURL, token string, rows, run int) (time.Duration, int, error) {
	payload, err := generateImportCSV(rows, run)
	if err != nil {
		return 0, 0, err
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "pegawai-impor.csv")
	if err != nil {
		return 0, 0, err
	}
	if _, err := part.Write(payload); err != nil {
		return 0, 0, err
	}
	if err := writer.Close(); err != nil {
		return 0, 0, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/v1/employees/import", &body)
	if err != nil {
		return 0, 0, err
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	started := time.Now()
	response, err := client.Do(request)
	elapsed := time.Since(started)
	if err != nil {
		return 0, 0, err
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return 0, 0, err
	}
	if response.StatusCode != http.StatusCreated {
		return 0, 0, fmt.Errorf("impor mengembalikan status %d: %s", response.StatusCode, truncateSQL(string(responseBody), 300))
	}
	var result map[string]int
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return 0, 0, fmt.Errorf("baca respons impor: %w", err)
	}
	return elapsed, result["imported"], nil
}

// countExportRows menghitung baris data CSV ekspor (tanpa baris header dan
// BOM). Baris kosong diabaikan.
func countExportRows(body []byte) int {
	rows := 0
	for _, line := range strings.Split(string(body), "\n") {
		if strings.TrimSpace(strings.TrimPrefix(line, "\xef\xbb\xbf")) == "" {
			continue
		}
		rows++
	}
	if rows == 0 {
		return 0
	}
	return rows - 1
}

// generateImportCSV membuat berkas CSV dengan kolom sesuai template impor.
// Alamat email dan NIP dibuat unik per percobaan agar impor kedua dan
// seterusnya tidak gagal pada pemeriksaan identitas ganda.
func generateImportCSV(rows, run int) ([]byte, error) {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	if err := writer.Write([]string{"nama", "email", "nip", "jabatanStruktural", "jabatanFungsional"}); err != nil {
		return nil, err
	}
	for index := 0; index < rows; index++ {
		email := fmt.Sprintf("impor-%d-%06d@impor-sintetis.test", run, index)
		nip := fmt.Sprintf("9%09d", run*1_000_000+index)
		record := []string{
			fmt.Sprintf("Pegawai Impor %d %06d", run, index),
			email,
			nip,
			"Staf Administrasi",
			"",
		}
		if err := writer.Write(record); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	return buffer.Bytes(), writer.Error()
}
