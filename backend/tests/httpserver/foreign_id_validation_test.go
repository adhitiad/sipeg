package httpserver

import (
	"net/http"
	"net/http/httptest"
	. "simpeg/backend/internal/httpserver"
	"strings"
	"testing"

	"simpeg/backend/internal/auth"
	directoryapp "simpeg/backend/internal/directory/app"
	tridharmaapp "simpeg/backend/internal/tridharma/app"
	tridharmarepository "simpeg/backend/internal/tridharma/repository"
)

// stubTridharmaRepository memenuhi tridharmarepository.Repository tanpa
// implementasi; pengujian penolakan validasi tidak menyentuh repository.
type stubTridharmaRepository struct {
	tridharmarepository.Repository
}

const foreignIDValidUUID = "11111111-1111-4111-8111-111111111111"

func postForeignID(t *testing.T, path, body, granted string) *httptest.ResponseRecorder {
	t.Helper()
	store := &fakeDirectoryStore{}
	mux := testDirectoryMux(store, map[string]map[auth.Permission]bool{
		"operator": {auth.Permission(granted): true},
	})
	request := httptest.NewRequest(http.MethodPost, APIPrefix+path, strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer valid-session")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	return response
}

func TestTridharmaCreateRejectsNonUUIDForeignID(t *testing.T) {
	store := &fakeDirectoryStore{}
	policy := auth.StaticPolicy(map[string]map[auth.Permission]bool{
		"operator": {"tridharma:evaluasi:manage:any": true},
	})
	mux := NewMuxWithOptions(MuxOptions{
		Resolver: testSessionResolver{},
		Policy:   policy,
		Directory: DirectoryServices{
			Employees: directoryapp.NewEmployeeService(store, policy, store),
			Tridharma: tridharmaapp.NewTridharmaService(stubTridharmaRepository{}, policy, store),
		},
	})
	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/tridharma/evaluasi",
		strings.NewReader(`{"userId":"`+foreignIDValidUUID+`","periodeAkademikId":"","nilai":90,"predikat":"A","tanggalEvaluasi":"2026-11-13"}`))
	request.Header.Set("Authorization", "Bearer valid-session")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
}

func TestEmployeeCreateRejectsNonUUIDHomebase(t *testing.T) {
	response := postForeignID(t, "/employees",
		`{"nama":"Pegawai Uji","email":"uji@example.test","homebaseId":"bukan-uuid"}`,
		"employee:create")
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
}

func TestOrganizationCreateRejectsNonUUIDParent(t *testing.T) {
	response := postForeignID(t, "/organizations",
		`{"name":"Unit Uji","parentId":"bukan-uuid"}`,
		"organization:manage")
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
}

func TestCutiCreateRejectsEmptyUserID(t *testing.T) {
	response := postForeignID(t, "/cuti",
		`{"userId":"","jenis":"TAHUNAN","tanggalMulai":"2026-12-01","tanggalSelesai":"2026-12-02"}`,
		"cuti:create")
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
}
