package qstash

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DefaultPublishURL adalah endpoint publikasi QStash.
const DefaultPublishURL = "https://qstash.upstash.io/v2/publish"

// ErrPublishRejected menandiekan QStash menolak penerbitan.
var ErrPublishRejected = errors.New("qstash rejected the publish request")

// PublishRequest menjelaskan satu pekerjaan yang dikirim ke QStash.
type PublishRequest struct {
	// Destination adalah URL tujuan callback.
	Destination string
	// Body adalah isi payload.
	Body []byte
	// Headers adalah header tambahan yang diteruskan ke callback.
	Headers map[string]string
	// RetryCount adalah jumlah percobaan ulang oleh QStash.
	RetryCount int
	// NotBefore menunda pengiriman sampai waktu tertentu.
	NotBefore time.Time
}

// Publisher mempublikasikan pekerjaan ke QStash.
type Publisher struct {
	// BaseURL adalah endpoint publikasi. Kosong memakai DefaultPublishURL.
	BaseURL string
	// SchedulesURL adalah endpoint jadwal berkala. Kosong memakai
	// DefaultSchedulesURL.
	SchedulesURL string
	// Token adalah token API QStash.
	Token string
	// HTTPClient adalah client yang dipakai. Kosong memakai client bawaan.
	HTTPClient *http.Client
}

// NewPublisher membuat publisher QStash.
func NewPublisher(token string) *Publisher {
	return &Publisher{Token: token}
}

// base64Encode mengodekan payload ke base64 standar.
func base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// Publish mengirim satu pekerjaan ke QStash.
func (p *Publisher) Publish(ctx context.Context, request PublishRequest) error {
	if strings.TrimSpace(request.Destination) == "" {
		return errors.New("qstash destination is required")
	}
	if strings.TrimSpace(p.Token) == "" {
		return errors.New("qstash token is not configured")
	}

	body := request.Body
	if len(body) == 0 {
		body = []byte("{}")
	}
	payload, err := json.Marshal(map[string]any{
		"url":    request.Destination,
		"body":   base64Encode(body),
		"method": http.MethodPost,
	})
	if err != nil {
		return fmt.Errorf("encode qstash publish request: %w", err)
	}

	endpoint := p.BaseURL
	if endpoint == "" {
		endpoint = DefaultPublishURL
	}
	target, err := url.Parse(strings.TrimRight(endpoint, "/") + "/" + strings.TrimPrefix(request.Destination, "/"))
	if err != nil {
		return fmt.Errorf("build qstash url: %w", err)
	}

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, target.String(), bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build qstash request: %w", err)
	}
	httpRequest.Header.Set("Authorization", "Bearer "+p.Token)
	httpRequest.Header.Set("Content-Type", "application/json")
	if request.RetryCount > 0 {
		httpRequest.Header.Set("Upstash-Retries", fmt.Sprintf("%d", request.RetryCount))
	}
	if !request.NotBefore.IsZero() {
		httpRequest.Header.Set("Upstash-Not-Before", fmt.Sprintf("%d", request.NotBefore.Unix()))
	}
	for name, value := range request.Headers {
		httpRequest.Header.Set("Upstash-Forward-"+name, value)
	}

	client := p.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	response, err := client.Do(httpRequest)
	if err != nil {
		return fmt.Errorf("publish to qstash: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode > 299 {
		detail, _ := io.ReadAll(io.LimitReader(response.Body, 2048))
		return fmt.Errorf("%w: status %d: %s", ErrPublishRejected, response.StatusCode, strings.TrimSpace(string(detail)))
	}
	return nil
}
