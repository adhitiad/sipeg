package holiday

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"simpeg/backend/internal/timeutil"
)

// Provider mengambil kalender hari libur sebuah tahun dari penyedia.
type Provider interface {
	FetchYear(ctx context.Context, year int) ([]Entry, error)
}

// Nilai type dari penyedia yang menandai hari libur. Entri "Observance"
// (peringatan tanpa libur) sengaja dilewati.
const (
	typePublicHoliday   = "Public Holiday"
	typeNationalHoliday = "National Holiday"
	typeJointHoliday    = "Joint Holiday"
)

const defaultTimeout = 10 * time.Second

const maxPages = 10

// APICoIDClient mengambil kalender dari https://use.api.co.id.
//
// Kunci API dikirim pada header `x-api-co-id`. Endpoint mengembalikan
// pagination; halaman diambil berurutan sampai habis.
type APICoIDClient struct {
	// BaseURL adalah endpoint daftar hari libur.
	BaseURL string
	// APIKey adalah kunci dari dashboard api.co.id.
	APIKey string
	// HTTPClient menentukan klien HTTP. Bila nil, klien dengan batas waktu
	// 10 detik dipakai.
	HTTPClient *http.Client
}

type apiCoIDHoliday struct {
	Date string `json:"date"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type apiCoIDResponse struct {
	IsSuccess bool             `json:"is_success"`
	Message   string           `json:"message"`
	Data      []apiCoIDHoliday `json:"data"`
	Paging    struct {
		TotalPage int `json:"total_page"`
	} `json:"paging"`
}

// FetchYear mengambil seluruh hari libur tahun yang diminta.
func (c *APICoIDClient) FetchYear(ctx context.Context, year int) ([]Entry, error) {
	if err := ValidateYear(year); err != nil {
		return nil, err
	}
	if strings.TrimSpace(c.BaseURL) == "" || strings.TrimSpace(c.APIKey) == "" {
		return nil, ErrProviderNotConfigured
	}
	client := c.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: defaultTimeout}
	}

	seen := map[string]bool{}
	var entries []Entry
	for page := 1; page <= maxPages; page++ {
		var response apiCoIDResponse
		if err := c.getPage(ctx, client, year, page, &response); err != nil {
			return nil, err
		}
		if !response.IsSuccess {
			return nil, ErrProvider{Detail: strings.TrimSpace(response.Message)}
		}
		for _, item := range response.Data {
			entry, keep, err := parseAPICoIDHoliday(item, year, seen)
			if err != nil {
				return nil, err
			}
			if keep {
				seen[entry.Tanggal.Format("2006-01-02")] = true
				entries = append(entries, entry)
			}
		}
		if page >= response.Paging.TotalPage || len(response.Data) == 0 {
			break
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Tanggal.Before(entries[j].Tanggal) })
	return entries, nil
}

func (c *APICoIDClient) getPage(ctx context.Context, client *http.Client, year, page int, target *apiCoIDResponse) error {
	parsed, err := url.Parse(c.BaseURL)
	if err != nil {
		return ErrProvider{Detail: "base url tidak sah"}
	}
	query := parsed.Query()
	query.Set("year", fmt.Sprintf("%d", year))
	query.Set("page", fmt.Sprintf("%d", page))
	parsed.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return ErrProvider{Detail: "gagal menyusun permintaan"}
	}
	request.Header.Set("x-api-co-id", c.APIKey)
	request.Header.Set("Accept", "application/json")

	response, err := client.Do(request)
	if err != nil {
		return ErrProvider{Detail: "permintaan gagal"}
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return ErrProvider{Detail: fmt.Sprintf("status HTTP %d", response.StatusCode)}
	}
	const maxBody = 1 << 20
	body, err := io.ReadAll(io.LimitReader(response.Body, maxBody))
	if err != nil {
		return ErrProvider{Detail: "gagal membaca respons"}
	}
	if err := json.Unmarshal(body, target); err != nil {
		return ErrProvider{Detail: "respons tidak dapat dibaca"}
	}
	return nil
}

// parseAPICoIDHoliday memetakan satu entri penyedia menjadi Entry kalender.
// Entri Observance, tanggal di luar tahun yang diminta, dan tanggal ganda
// diabaikan. Entri dengan data tidak lengkap ditolak.
func parseAPICoIDHoliday(item apiCoIDHoliday, year int, seen map[string]bool) (Entry, bool, error) {
	kind := strings.TrimSpace(item.Type)
	if kind != typePublicHoliday && kind != typeNationalHoliday && kind != typeJointHoliday {
		return Entry{}, false, nil
	}
	name := strings.TrimSpace(item.Name)
	if name == "" {
		return Entry{}, false, ErrProvider{Detail: "entri tanpa nama"}
	}
	date, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(item.Date), time.UTC)
	if err != nil {
		return Entry{}, false, ErrProvider{Detail: fmt.Sprintf("tanggal tidak sah pada entri %q", name)}
	}
	if date.Year() != year {
		return Entry{}, false, nil
	}
	key := date.Format("2006-01-02")
	if seen[key] {
		return Entry{}, false, nil
	}
	return Entry{Tanggal: timeutil.DateOnly(date), Nama: name}, true, nil
}
