package httpserver

import (
	"encoding/csv"
	"errors"
	"net/http"
	"net/http/httptest"
	. "simpeg/backend/internal/httpserver"
	"slices"
	"strings"
	"testing"

	"simpeg/backend/internal/audit"
	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/crypto"
	"simpeg/backend/internal/models"
)

func TestEmployeeExportWritesCSVWithHeadersAndRows(t *testing.T) {
	// 25 baris melebihi ukuran halaman bawaan (20) dan batas atas (100)
	// sehingga ekspor berbasis halaman akan tertangkap di sini.
	employees := make([]models.User, 0, 25)
	for index := 0; index < 25; index++ {
		employees = append(employees, models.User{
			BaseModel:    models.BaseModel{IDModel: models.IDModel{ID: userID(index)}},
			Nama:         "Pegawai " + strings.Repeat("x", index),
			Email:        "pegawai" + userID(index) + "@example.test",
			Nip:          crypto.PtrEncrypted(pointer("198001012005011001")),
			Nik:          crypto.PtrEncrypted(pointer("3201234567890001")),
			Npwp:         crypto.PtrEncrypted(pointer("01.234.567.8-901.000")),
			BankRekening: pointer("1234567890"),
			Aktif:        true,
		})
	}
	store := &fakeDirectoryStore{employees: employees}
	mux := scopeMux(store, auth.Principal{UserID: atasanID, RoleName: "admin", HomebaseID: unitA},
		map[string]map[auth.Permission]bool{"admin": {"employee:export": true, "employee:read:any": true}})

	// Parameter sort, page, dan pageSize diabaikan oleh ekspor.
	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/employees/export?sort=nip&page=2&pageSize=5", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
	contentType := response.Header().Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/csv") {
		t.Fatalf("content type = %q", contentType)
	}
	if !strings.Contains(response.Header().Get("Content-Disposition"), "attachment; filename=") {
		t.Fatalf("content disposition = %q", response.Header().Get("Content-Disposition"))
	}
	body := response.Body.String()
	if !strings.HasPrefix(body, "\xef\xbb\xbf") {
		t.Fatalf("expected UTF-8 BOM, body starts with %q", body[:3])
	}
	records, err := csv.NewReader(strings.NewReader(strings.TrimPrefix(body, "\xef\xbb\xbf"))).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 26 {
		t.Fatalf("expected header + 25 rows, got %d records", len(records))
	}
	wantHeader := []string{"nama", "email", "nip", "jabatanStruktural", "jabatanFungsional"}
	if !slices.Equal(records[0], wantHeader) {
		t.Fatalf("unexpected header: %v", records[0])
	}
	for _, record := range records[1:] {
		if record[2] != "198001012005011001" {
			t.Fatalf("expected NIP column, got %q", record[2])
		}
	}
	// Kolom sensitif tidak boleh muncul di berkas ekspor.
	for _, leaked := range []string{"3201234567890001", "01.234.567.8-901.000", "1234567890"} {
		if strings.Contains(body, leaked) {
			t.Fatalf("sensitive value leaked into export: %s", leaked)
		}
	}
	if len(store.recordedAudits) != 1 || store.recordedAudits[0].Action != audit.ActionEmployeeExported {
		t.Fatalf("expected employee.exported entry, got %v", store.recordedAudits)
	}
	note := store.recordedAudits[0].Note
	if note == nil || *note != "25 baris diekspor" {
		t.Fatalf("audit note = %v, want %q", note, "25 baris diekspor")
	}
}

func TestEmployeeExportWithoutRowsWritesHeaderOnly(t *testing.T) {
	store := &fakeDirectoryStore{employees: []models.User{}}
	mux := scopeMux(store, auth.Principal{UserID: atasanID, RoleName: "admin"},
		map[string]map[auth.Permission]bool{"admin": {"employee:export": true, "employee:read:any": true}})

	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/employees/export", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
	body := response.Body.String()
	if !strings.HasPrefix(body, "\xef\xbb\xbf") {
		t.Fatalf("expected UTF-8 BOM, body starts with %q", body[:3])
	}
	records, err := csv.NewReader(strings.NewReader(strings.TrimPrefix(body, "\xef\xbb\xbf"))).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("expected header-only CSV, got %d records", len(records))
	}
}

// TestEmployeeExportScopeFollowsOperatorHomebase memastikan ekspor operator
// mengikuti cakupan unit penugasan beserta turunannya.
func TestEmployeeExportScopeFollowsOperatorHomebase(t *testing.T) {
	store := &fakeDirectoryStore{scopeIDs: []string{unitA, unitAChild}}
	mux := scopeMux(store, auth.Principal{UserID: atasanID, RoleName: "operator", HomebaseID: unitA},
		map[string]map[auth.Permission]bool{"operator": {"employee:export": true, "employee:read:homebase": true}})

	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/employees/export", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
	if len(store.streamOptions.ScopeHomebaseIDs) != 2 {
		t.Fatalf("export scope units = %v, want unit + descendant", store.streamOptions.ScopeHomebaseIDs)
	}
}

// TestEmployeeExportReportsStreamFailure memastikan kegagalan membaca pegawai
// sebelum baris pertama tidak berakhir sebagai berkas CSV kosong berstatus 200.
func TestEmployeeExportReportsStreamFailure(t *testing.T) {
	store := &fakeDirectoryStore{err: errors.New("stream gagal")}
	mux := scopeMux(store, auth.Principal{UserID: atasanID, RoleName: "admin"},
		map[string]map[auth.Permission]bool{"admin": {"employee:export": true, "employee:read:any": true}})

	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/employees/export", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500, body = %s", response.Code, response.Body)
	}
	if strings.Contains(response.Body.String(), "nama,email") {
		t.Fatalf("kegagalan stream tidak boleh menghasilkan header CSV: %s", response.Body)
	}
}

// TestEmployeeExportRequiresReadGrant memastikan izin ekspor saja tidak
// membuka data: tanpa permission baca pada pegawai, lapisan aplikasi menolak
// dengan 403, bukan mengirim berkas.
func TestEmployeeExportRequiresReadGrant(t *testing.T) {
	store := &fakeDirectoryStore{employees: []models.User{
		{BaseModel: models.BaseModel{IDModel: models.IDModel{ID: pegawaiID}}, Nama: "Saya", HomebaseId: pointer(unitA), Aktif: true},
	}}
	mux := scopeMux(store, auth.Principal{UserID: atasanID, RoleName: "admin", HomebaseID: unitA},
		map[string]map[auth.Permission]bool{"admin": {"employee:export": true}})

	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/employees/export", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403, body = %s", response.Code, response.Body)
	}
	if strings.Contains(response.Body.String(), "nama,email") {
		t.Fatalf("izin ekspor tanpa baca tidak boleh membuka berkas: %s", response.Body)
	}
}

func TestEmployeeExportRejectedWithoutPermission(t *testing.T) {
	store := &fakeDirectoryStore{}
	mux := testDirectoryMux(store, map[string]map[auth.Permission]bool{})

	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/employees/export", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403, body = %s", response.Code, response.Body)
	}
}

func TestEmployeeExportRejectsInvalidFilters(t *testing.T) {
	tests := []struct {
		name   string
		query  string
		grants map[string]map[auth.Permission]bool
	}{
		{"homebaseId bukan UUID", "?homebaseId=bukan-uuid", map[string]map[auth.Permission]bool{"admin": {"employee:export": true}}},
		{"search terlalu panjang", "?search=" + strings.Repeat("a", 257), map[string]map[auth.Permission]bool{"admin": {"employee:export": true}}},
		{"tanpa izin ekspor", "", map[string]map[auth.Permission]bool{}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := &fakeDirectoryStore{}
			mux := scopeMux(store, auth.Principal{UserID: atasanID, RoleName: "admin"}, test.grants)

			request := httptest.NewRequest(http.MethodGet, APIPrefix+"/employees/export"+test.query, nil)
			request.Header.Set("Authorization", "Bearer valid-session")
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)

			want := http.StatusBadRequest
			if test.name == "tanpa izin ekspor" {
				want = http.StatusForbidden
			}
			if response.Code != want {
				t.Fatalf("status = %d, want %d, body = %s", response.Code, want, response.Body)
			}
		})
	}
}

// userID menyusun ID pengguna unik per indeks agar baris dapat dibedakan.
func userID(index int) string {
	return "eeeeeeee-eeee-4eee-8eee-" + strings.Repeat("0", 11) + string(rune('0'+index%10))
}
