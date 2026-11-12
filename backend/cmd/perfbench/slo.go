package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"simpeg/backend/internal/jobs"

	"github.com/golang-jwt/jwt/v5"
)

// runSLO memanggil endpoint pemeriksaan SLO seperti yang dilakukan QStash.
//
// Callback QStash ditandatangani dengan kunci HS256 di atas raw body, dan
// subjek signature harus sama dengan URL tujuan. Fungsi ini meniru bentuk
// callback itu agar jalur verifikasi signature dan pemeriksaan SLO dapat
// diuji tanpa menunggu jadwal QStash.
func runSLO(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("slo", flag.ContinueOnError)
	url := flags.String("url", "", "alamat server yang dipanggil (kosong = APP_BASE_URL)")
	destination := flags.String("destination", "", "URL tujuan yang harus cocok dengan subjek signature")
	if err := flags.Parse(args); err != nil {
		return err
	}
	key, err := requireEnv("QSTASH_CURRENT_SIGNING_KEY")
	if err != nil {
		return err
	}
	appBase := envOr("APP_BASE_URL", "http://127.0.0.1:8080")
	target := *destination
	if target == "" {
		target = jobs.Destination(appBase, "/api/v1/internal/qstash/slo/evaluate")
	}
	endpoint := *url
	if endpoint == "" {
		endpoint = target
	}

	body := []byte(`{"kind":"slo.evaluate"}`)
	signature, err := signCallback(key, target, body)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Upstash-Signature", signature)

	client := &http.Client{Timeout: 30 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("panggil pemeriksaan slo: %w", err)
	}
	defer response.Body.Close()
	payload, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
	fmt.Printf("status: %d\n", response.StatusCode)
	fmt.Printf("alamat: %s\n", endpoint)
	fmt.Printf("subjek signature: %s\n", target)
	pretty, err := indentJSON(payload)
	if err != nil {
		fmt.Printf("body: %s\n", string(payload))
		return nil
	}
	fmt.Printf("body:\n%s\n", pretty)
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("pemeriksaan slo menjawab %d", response.StatusCode)
	}
	return nil
}

// signCallback menandatangani body dengan klaim yang dipakai QStash.
func signCallback(key, subject string, body []byte) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  subject,
		"body": base64.StdEncoding.EncodeToString(body),
		"iss":  "Upstash",
		"exp":  time.Now().Add(5 * time.Minute).Unix(),
		"iat":  time.Now().Unix(),
	})
	signed, err := token.SignedString([]byte(key))
	if err != nil {
		return "", fmt.Errorf("tandatangani callback: %w", err)
	}
	return signed, nil
}

// indentJSON memformat body JSON agar mudah dibaca.
func indentJSON(payload []byte) (string, error) {
	var value any
	if err := json.Unmarshal(payload, &value); err != nil {
		return "", err
	}
	formatted, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "", err
	}
	return string(formatted), nil
}
