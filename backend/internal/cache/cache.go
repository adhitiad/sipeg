// Package cache menyediakan cache data referensi non-sensitif di Redis.
//
// Aturan yang ditegakkan paket ini:
//
//   - Redis tidak pernah menjadi sumber kebenaran. Setiap pembacaan memakai
//     ReadThrough yang selalu memuat ulang dari sumber utama saat cache miss
//     atau saat Redis gagal.
//   - Hanya data referensi non-sensitif yang boleh dicache. Kunci yang
//     menyerupai sesi, penggajian, atau status pembayaran ditolak saat runtime.
//   - Semua kunci berada dalam satu namespace agar dapat dikosongkan menyeluruh.
//   - TTL pendek membatasi umur data basi.
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// ErrForbiddenKey menandakan kunci yang tidak boleh dicache.
var ErrForbiddenKey = errors.New("cache key is not allowed")

// ErrNotFound menandakan kunci tidak ada di cache.
var ErrNotFound = errors.New("cache key not found")

// forbiddenFragments adalah fragmen yang menandai kunci sebagai data sensitif.
// Pencocokan dilakukan pada nama kunci huruf kecil.
var forbiddenFragments = []string{
	"session", "token", "credential", "password",
	"gaji", "payroll", "slip", "salary", "pembayaran", "payment", "mutation",
}

// Config memuat konfigurasi cache.
type Config struct {
	// Namespace adalah awalan seluruh kunci cache.
	Namespace string
	// TTL adalah umur maksimum entri cache.
	TTL time.Duration
	// ForbiddenFragments menambah daftar fragmen kunci yang dilarang.
	ForbiddenFragments []string
}

// DefaultTTL adalah umur cache default bila konfigurasi tidak menyetelnya.
const DefaultTTL = 5 * time.Minute

// Cache menyimpan data referensi dengan aturan namespace dan TTL.
type Cache struct {
	client    Client
	config    Config
	forbidden []string
}

// New membuat cache dengan konfigurasi yang diberikan.
func New(client Client, config Config) *Cache {
	if config.TTL <= 0 {
		config.TTL = DefaultTTL
	}
	if config.Namespace == "" {
		config.Namespace = "simpeg"
	}
	forbidden := append([]string(nil), forbiddenFragments...)
	forbidden = append(forbidden, config.ForbiddenFragments...)
	return &Cache{client: client, config: config, forbidden: forbidden}
}

// Key membangun nama kunci lengkap dengan namespace.
func (c *Cache) Key(key string) string {
	return c.config.Namespace + ":" + strings.TrimPrefix(strings.TrimSpace(key), ":")
}

// IsAllowed memeriksa apakah kunci boleh dicache.
func (c *Cache) IsAllowed(key string) bool {
	lowered := strings.ToLower(key)
	for _, fragment := range c.forbidden {
		if strings.Contains(lowered, fragment) {
			return false
		}
	}
	return true
}

// Get membaca nilai cache dan memetakannya ke target.
// Mengembalikan ErrNotFound bila kunci tidak ada atau kedaluwarsa.
func (c *Cache) Get(ctx context.Context, key string, target any) error {
	if !c.IsAllowed(key) {
		return fmt.Errorf("%w: %s", ErrForbiddenKey, key)
	}
	raw, err := c.client.Get(ctx, c.Key(key))
	if err != nil {
		return err
	}
	if raw == "" {
		return ErrNotFound
	}
	if err := json.Unmarshal([]byte(raw), target); err != nil {
		return fmt.Errorf("decode cache entry: %w", err)
	}
	return nil
}

// Set menyimpan nilai cache dengan TTL.
func (c *Cache) Set(ctx context.Context, key string, value any) error {
	if !c.IsAllowed(key) {
		return fmt.Errorf("%w: %s", ErrForbiddenKey, key)
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode cache entry: %w", err)
	}
	return c.client.Set(ctx, c.Key(key), string(encoded), c.config.TTL)
}

// Delete menghapus satu entri cache.
func (c *Cache) Delete(ctx context.Context, key string) error {
	if !c.IsAllowed(key) {
		return fmt.Errorf("%w: %s", ErrForbiddenKey, key)
	}
	return c.client.Delete(ctx, c.Key(key))
}

// ReadThrough membaca nilai dari cache dan, bila tidak ditemukan atau cache
// bermasalah, memuat ulang dari sumber utama lalu mengisi cache.
//
// Fungsi ini menjamin cache tidak pernah menjadi sumber kebenaran: kegagalan
// Redis tidak membuat pembacaan gagal. Fungsi load mengisi target secara
// langsung sehingga bentuk data di cache dan sumber utama selalu sama.
func (c *Cache) ReadThrough(ctx context.Context, key string, target any, load func(ctx context.Context) error) error {
	if !c.IsAllowed(key) {
		return fmt.Errorf("%w: %s", ErrForbiddenKey, key)
	}

	err := c.Get(ctx, key, target)
	if err == nil {
		return nil
	}
	if !errors.Is(err, ErrNotFound) {
		// Cache bermasalah: lanjut ke sumber utama tanpa menggagalkan pembacaan.
		c.logFailure("cache_read_failed", key, err)
	}

	if loadErr := load(ctx); loadErr != nil {
		return loadErr
	}
	// Kegagalan menulis cache tidak menggagalkan pembacaan.
	if setErr := c.Set(ctx, key, target); setErr != nil {
		c.logFailure("cache_write_failed", key, setErr)
	}
	return nil
}

// logFailure mencatat kegagalan cache tanpa menggagalkan operasi.
func (c *Cache) logFailure(event, key string, err error) {
	if errors.Is(err, ErrForbiddenKey) {
		return
	}
	slog.Warn(event, "key", key, "error", err.Error())
}
