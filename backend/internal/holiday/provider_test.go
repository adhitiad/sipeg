package holiday

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPICoIDClientFetchesAndFiltersHolidays(t *testing.T) {
	var requestedKey string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedKey = r.Header.Get("x-api-co-id")
		if r.URL.Query().Get("year") != "2026" {
			t.Errorf("year = %q, want 2026", r.URL.Query().Get("year"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"is_success": true,
			"message": "Success",
			"data": [
				{"id": "h1", "date": "2026-01-01", "name": "Tahun Baru 2026", "type": "Public Holiday", "is_national_holiday": true},
				{"id": "h2", "date": "2026-02-17", "name": "Tahun Baru Imlek 2577", "type": "Observance"},
				{"id": "h3", "date": "2026-03-20", "name": "Hari Raya Nyepi", "type": "National Holiday"},
				{"id": "h4", "date": "2027-05-01", "name": "Hari Buruh (tahun lain)", "type": "Public Holiday"},
				{"id": "h5", "date": "2026-03-20", "name": "Hari Raya Nyepi (duplikat)", "type": "Joint Holiday"}
			],
			"paging": {"page": 1, "size": 100, "total_item": 4, "total_page": 1}
		}`))
	}))
	defer server.Close()

	client := &APICoIDClient{BaseURL: server.URL, APIKey: "test-key"}
	entries, err := client.FetchYear(context.Background(), 2026)
	if err != nil {
		t.Fatal(err)
	}
	if requestedKey != "test-key" {
		t.Fatalf("api key header = %q", requestedKey)
	}
	if len(entries) != 2 {
		t.Fatalf("entries = %+v", entries)
	}
	if entries[0].Tanggal.Format("2006-01-02") != "2026-01-01" || entries[0].Nama != "Tahun Baru 2026" {
		t.Fatalf("entry[0] = %+v", entries[0])
	}
	if entries[1].Nama != "Hari Raya Nyepi" {
		t.Fatalf("entry[1] = %+v", entries[1])
	}
}

func TestAPICoIDClientFollowsPagination(t *testing.T) {
	pages := map[string]string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		w.Header().Set("Content-Type", "application/json")
		switch page {
		case "1":
			pages["1"] = "seen"
			_, _ = w.Write([]byte(`{
				"is_success": true, "message": "Success",
				"data": [{"id": "h1", "date": "2026-01-01", "name": "Tahun Baru 2026", "type": "Public Holiday"}],
				"paging": {"page": 1, "size": 100, "total_item": 2, "total_page": 2}
			}`))
		case "2":
			pages["2"] = "seen"
			_, _ = w.Write([]byte(`{
				"is_success": true, "message": "Success",
				"data": [{"id": "h2", "date": "2026-08-17", "name": "Hari Kemerdekaan RI", "type": "Public Holiday"}],
				"paging": {"page": 2, "size": 100, "total_item": 2, "total_page": 2}
			}`))
		}
	}))
	defer server.Close()

	client := &APICoIDClient{BaseURL: server.URL, APIKey: "test-key"}
	entries, err := client.FetchYear(context.Background(), 2026)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || pages["1"] == "" || pages["2"] == "" {
		t.Fatalf("entries = %+v, pages = %v", entries, pages)
	}
}

func TestAPICoIDClientRequiresConfiguration(t *testing.T) {
	client := &APICoIDClient{}
	if _, err := client.FetchYear(context.Background(), 2026); !errors.Is(err, ErrProviderNotConfigured) {
		t.Fatalf("err = %v, want provider not configured", err)
	}
}

func TestAPICoIDClientSurfacesFailureResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"is_success": false, "message": "Invalid API key"}`))
	}))
	defer server.Close()

	client := &APICoIDClient{BaseURL: server.URL, APIKey: "wrong"}
	var providerErr ErrProvider
	_, err := client.FetchYear(context.Background(), 2026)
	if !errors.As(err, &providerErr) {
		t.Fatalf("err = %v, want provider error", err)
	}
}

func TestAPICoIDClientRejectsMalformedEntry(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"is_success": true, "message": "Success",
			"data": [{"id": "h1", "date": "bukan-tanggal", "name": "Entri Rusak", "type": "Public Holiday"}],
			"paging": {"page": 1, "size": 100, "total_item": 1, "total_page": 1}
		}`))
	}))
	defer server.Close()

	client := &APICoIDClient{BaseURL: server.URL, APIKey: "test-key"}
	var providerErr ErrProvider
	_, err := client.FetchYear(context.Background(), 2026)
	if !errors.As(err, &providerErr) {
		t.Fatalf("err = %v, want provider error", err)
	}
}

func TestValidateYearRejectsOutOfRange(t *testing.T) {
	for _, year := range []int{1999, 2101} {
		if err := ValidateYear(year); !errors.Is(err, ErrInvalidYear) {
			t.Fatalf("year %d: err = %v, want invalid year", year, err)
		}
	}
	if err := ValidateYear(2026); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseAPICoIDHolidayKeepsRequestedYearOnly(t *testing.T) {
	seen := map[string]bool{}
	_, keep, err := parseAPICoIDHoliday(apiCoIDHoliday{Date: "2025-12-25", Name: "Natal 2025", Type: typePublicHoliday}, 2026, seen)
	if err != nil || keep {
		t.Fatalf("entry di luar tahun harus dilewati: keep=%v err=%v", keep, err)
	}
}

func TestParseAPICoIDHolidayRejectsDuplicateDate(t *testing.T) {
	seen := map[string]bool{"2026-01-01": true}
	_, keep, err := parseAPICoIDHoliday(apiCoIDHoliday{Date: "2026-01-01", Name: "Duplikat", Type: typePublicHoliday}, 2026, seen)
	if err != nil || keep {
		t.Fatalf("entri duplikat harus dilewati: keep=%v err=%v", keep, err)
	}
}
