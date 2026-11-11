package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client adalah antarmuka penyimpanan cache.
type Client interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

// RESTClient memakai API REST Upstash sehingga tidak memerlukan driver Redis
// khusus dan dapat berjalan di lingkungan serverless.
type RESTClient struct {
	// BaseURL adalah root API Upstash, mis. "https://xyz.upstash.io".
	BaseURL string
	// Token adalah token API Upstash.
	Token string
	// HTTPClient adalah client yang dipakai.
	HTTPClient *http.Client
}

// NewRESTClient membuat client REST Upstash.
func NewRESTClient(baseURL, token string) *RESTClient {
	return &RESTClient{BaseURL: baseURL, Token: token}
}

// Get membaca nilai mentah dari cache.
func (c *RESTClient) Get(ctx context.Context, key string) (string, error) {
	endpoint, err := c.endpoint("get", key)
	if err != nil {
		return "", err
	}
	request, err := c.newRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	response, err := c.client().Do(request)
	if err != nil {
		return "", fmt.Errorf("cache get: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("cache get: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return "", fmt.Errorf("cache get: status %d", response.StatusCode)
	}

	var payload struct {
		Result string `json:"result"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", fmt.Errorf("cache get: decode response: %w", err)
	}
	return payload.Result, nil
}

// Set menyimpan nilai mentah dengan TTL.
func (c *RESTClient) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	endpoint, err := c.endpoint("set", key)
	if err != nil {
		return err
	}
	if ttl > 0 {
		endpoint += "?EX=" + fmt.Sprintf("%d", int(ttl.Seconds()))
	}
	request, err := c.newRequest(ctx, http.MethodPost, endpoint, strings.NewReader(value))
	if err != nil {
		return err
	}
	response, err := c.client().Do(request)
	if err != nil {
		return fmt.Errorf("cache set: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return fmt.Errorf("cache set: status %d", response.StatusCode)
	}
	return nil
}

// Delete menghapus satu kunci.
func (c *RESTClient) Delete(ctx context.Context, key string) error {
	endpoint, err := c.endpoint("del", key)
	if err != nil {
		return err
	}
	request, err := c.newRequest(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return err
	}
	response, err := c.client().Do(request)
	if err != nil {
		return fmt.Errorf("cache delete: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return fmt.Errorf("cache delete: status %d", response.StatusCode)
	}
	return nil
}

func (c *RESTClient) endpoint(operation, key string) (string, error) {
	if strings.TrimSpace(c.BaseURL) == "" {
		return "", fmt.Errorf("cache base url is not configured")
	}
	return strings.TrimRight(c.BaseURL, "/") + "/" + operation + "/" + url.PathEscape(key), nil
}

func (c *RESTClient) newRequest(ctx context.Context, method, endpoint string, body io.Reader) (*http.Request, error) {
	request, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("cache request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+c.Token)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	return request, nil
}

func (c *RESTClient) client() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 5 * time.Second}
}
