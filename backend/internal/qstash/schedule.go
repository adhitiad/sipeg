package qstash

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DefaultSchedulesURL adalah endpoint jadwal berkala QStash.
const DefaultSchedulesURL = "https://qstash.upstash.io/v2/schedules"

// ErrScheduleNotFound menandai jadwal tidak ditemukan pada QStash.
var ErrScheduleNotFound = errors.New("qstash schedule not found")

// ScheduleRequest menjelaskan satu jadwal pekerjaan berkala.
type ScheduleRequest struct {
	// Destination adalah URL tujuan callback.
	Destination string
	// Body adalah isi payload yang dikirim pada setiap pemicu.
	Body string
	// Cron adalah ekspresi jadwal berkala dalam zona UTC.
	Cron string
	// Retries adalah jumlah percobaan ulang oleh QStash.
	Retries int
}

// Schedule adalah satu jadwal yang terdaftar pada QStash.
type Schedule struct {
	ID          string `json:"scheduleId"`
	Destination string `json:"destination"`
	Cron        string `json:"cron"`
}

// CreateSchedule mendaftarkan jadwal berkala dan mengembalikan id jadwal.
//
// Jadwal membuat QStash memanggil destination secara berulang sesuai cron,
// sehingga pekerjaan asinkron dijalankan dari luar instance backend.
//
// API v2 QStash memakai bentuk yang sama dengan publikasi: destination
// merupakan bagian dari path, sedangkan cron, percobaan ulang, dan metode
// dikirim sebagai header Upstash. Bentuk lama yang menaruh destination dan
// cron di dalam body ditolak QStash dengan "invalid destination url".
func (p *Publisher) CreateSchedule(ctx context.Context, request ScheduleRequest) (string, error) {
	if strings.TrimSpace(request.Destination) == "" {
		return "", errors.New("qstash schedule destination is required")
	}
	if strings.TrimSpace(request.Cron) == "" {
		return "", errors.New("qstash schedule cron is required")
	}
	if strings.TrimSpace(p.Token) == "" {
		return "", errors.New("qstash token is not configured")
	}

	body := request.Body
	if body == "" {
		body = "{}"
	}
	target, err := url.Parse(
		strings.TrimRight(p.schedulesURL(), "/") + "/" + strings.TrimPrefix(request.Destination, "/"),
	)
	if err != nil {
		return "", fmt.Errorf("build qstash schedule url: %w", err)
	}

	request2, err := http.NewRequestWithContext(ctx, http.MethodPost, target.String(), strings.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build qstash schedule request: %w", err)
	}
	request2.Header.Set("Authorization", "Bearer "+p.Token)
	request2.Header.Set("Content-Type", "application/json")
	request2.Header.Set("Upstash-Cron", request.Cron)
	request2.Header.Set("Upstash-Method", http.MethodPost)
	if request.Retries > 0 {
		request2.Header.Set("Upstash-Retries", fmt.Sprintf("%d", request.Retries))
	}

	response, err := p.send(request2)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	var created struct {
		ScheduleID string `json:"scheduleId"`
	}
	if err := json.NewDecoder(response.Body).Decode(&created); err != nil {
		return "", fmt.Errorf("decode qstash schedule response: %w", err)
	}
	if created.ScheduleID == "" {
		return "", errors.New("qstash did not return a schedule id")
	}
	return created.ScheduleID, nil
}

// ListSchedules mengembalikan seluruh jadwal yang terdaftar pada QStash.
func (p *Publisher) ListSchedules(ctx context.Context) ([]Schedule, error) {
	if strings.TrimSpace(p.Token) == "" {
		return nil, errors.New("qstash token is not configured")
	}
	response, err := p.do(ctx, http.MethodGet, p.schedulesURL(), nil)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	var schedules []Schedule
	if err := json.NewDecoder(response.Body).Decode(&schedules); err != nil {
		return nil, fmt.Errorf("decode qstash schedule list: %w", err)
	}
	return schedules, nil
}

// DeleteSchedule membatalkan jadwal yang terdaftar pada QStash.
func (p *Publisher) DeleteSchedule(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return errors.New("qstash schedule id is required")
	}
	if strings.TrimSpace(p.Token) == "" {
		return errors.New("qstash token is not configured")
	}
	response, err := p.do(ctx, http.MethodDelete, strings.TrimRight(p.schedulesURL(), "/")+"/"+id, nil)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	return nil
}

// schedulesURL mengembalikan endpoint jadwal yang dipakai publisher.
//
// Nilai SchedulesURL dipakai apa adanya bila sudah memuat jalur jadwal.
// Bila kosong, endpoint diturunkan dari BaseURL supaya token regional dan
// jadwal berada pada region yang sama. Tanpa ini, jadwal akan dibuat pada
// region global padahal token dimiliki region lain.
func (p *Publisher) schedulesURL() string {
	if trimmed := strings.TrimRight(strings.TrimSpace(p.SchedulesURL), "/"); trimmed != "" {
		return withSchedulesPath(trimmed)
	}
	base := strings.TrimRight(strings.TrimSpace(p.BaseURL), "/")
	if base == "" {
		return DefaultSchedulesURL
	}
	return withSchedulesPath(base)
}

// withSchedulesPath menambahkan jalur jadwal bila belum ada.
func withSchedulesPath(base string) string {
	if strings.HasSuffix(base, "/v2/schedules") {
		return base
	}
	return base + "/v2/schedules"
}

// do menjalankan satu permintaan HTTP ke QStash dan memeriksa statusnya.
func (p *Publisher) do(ctx context.Context, method, endpoint string, payload []byte) (*http.Response, error) {
	var body io.Reader
	if payload != nil {
		body = bytes.NewReader(payload)
	}
	request, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("build qstash request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+p.Token)
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := p.send(request)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// send menjalankan satu permintaan HTTP ke QStash dan memeriksa statusnya.
func (p *Publisher) send(request *http.Request) (*http.Response, error) {
	client := p.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("call qstash: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode > 299 {
		detail, _ := io.ReadAll(io.LimitReader(response.Body, 2048))
		response.Body.Close()
		return nil, fmt.Errorf("%w: status %d: %s", ErrPublishRejected, response.StatusCode, strings.TrimSpace(string(detail)))
	}
	return response, nil
}
