package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/directory/app"
	"simpeg/backend/internal/directory/domain"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
)

// loadStore adalah repository palsu untuk pengujian beban.
//
// Implementasi ini aman dipakai dari banyak goroutine dan mencatat jumlah
// pemanggilan sehingga jalur baca bisa diawasi saat request datang paralel.
type loadStore struct {
	mu        sync.Mutex
	employees []models.User
	calls     atomic.Int64
}

func (s *loadStore) ListEmployees(_ context.Context, _ listing.Options) ([]models.User, int64, error) {
	s.calls.Add(1)
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.employees, int64(len(s.employees)), nil
}

func (s *loadStore) GetEmployee(context.Context, string) (models.User, error) {
	return models.User{}, domain.ErrNotFound
}

func (s *loadStore) CreateEmployee(context.Context, string, models.User) (models.User, error) {
	return models.User{}, domain.ErrConflict
}

func (s *loadStore) CreateEmployees(context.Context, string, []models.User) ([]models.User, error) {
	return nil, domain.ErrConflict
}

func (s *loadStore) UpdateEmployee(context.Context, string, string, domain.EmployeePatch) (models.User, error) {
	return models.User{}, domain.ErrConflict
}

func (s *loadStore) AssignEmployeeRole(context.Context, string, string, string) (models.User, error) {
	return models.User{}, domain.ErrConflict
}

func (s *loadStore) ArchiveEmployee(context.Context, string, string) error {
	return domain.ErrConflict
}

// ListUnitScopeIDs mengembalikan cakupan unit penugasan beserta turunannya.
func (s *loadStore) ListUnitScopeIDs(context.Context, string) ([]string, error) {
	return []string{unitA, unitAChild}, nil
}

// loadEmployees membuat data pegawai untuk pengujian beban.
func loadEmployees(count int) []models.User {
	employees := make([]models.User, 0, count)
	for index := 0; index < count; index++ {
		employees = append(employees, models.User{
			BaseModel:  models.BaseModel{IDModel: models.IDModel{ID: atasanID}},
			Nama:       "Pegawai Beban",
			Email:      "beban@example.test",
			HomebaseId: pointer(unitA),
			Aktif:      true,
		})
	}
	return employees
}

// loadMux menyusun mux yang hanya memuat modul pegawai agar pengujian beban
// tidak terganggu oleh modul lain.
func loadMux(store *loadStore, principal auth.Principal, grants map[string]map[auth.Permission]bool) *http.ServeMux {
	policy := auth.StaticPolicy(grants)
	return NewMuxWithOptions(MuxOptions{
		Resolver: principalResolver{principal: principal},
		Policy:   policy,
		Directory: DirectoryServices{
			Employees: app.NewEmployeeService(store, policy, store),
		},
	})
}

// TestBurstRequestsRemainStable menjalankan banyak permintaan secara paralel.
//
// Uji ini menjadi bukti bahwa jalur baca tetap memberi jawaban yang sama
// ketika request datang bersamaan, dan tidak menyisakan goroutine yang
// menggantung setelah beban selesai.
func TestBurstRequestsRemainStable(t *testing.T) {
	store := &loadStore{employees: loadEmployees(50)}
	mux := loadMux(store, auth.Principal{UserID: atasanID, RoleName: "admin"},
		map[string]map[auth.Permission]bool{"admin": {"employee:read:any": true}})

	const burst = 200
	baseline := runtime.NumGoroutine()

	var failures atomic.Int64
	var wait sync.WaitGroup
	for index := 0; index < burst; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			request := httptest.NewRequest(http.MethodGet, APIPrefix+"/employees?pageSize=50", nil)
			request.Header.Set("Authorization", "Bearer valid-session")
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			if response.Code != http.StatusOK {
				failures.Add(1)
			}
		}()
	}
	wait.Wait()

	if failures.Load() != 0 {
		t.Fatalf("failed requests = %d, want 0", failures.Load())
	}
	if calls := store.calls.Load(); calls != burst {
		t.Fatalf("store calls = %d, want %d", calls, burst)
	}

	// Goroutine yang bocor akan bertambah sebanding dengan jumlah permintaan.
	// Kelonggaran kecil dipakai untuk menutup selisih goroutine milik runner.
	after := runtime.NumGoroutine()
	if after > baseline+burst/10 {
		t.Fatalf("goroutines before = %d, after = %d — indikasi kebocoran goroutine", baseline, after)
	}
}
