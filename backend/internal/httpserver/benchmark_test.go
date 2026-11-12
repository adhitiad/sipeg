package httpserver

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/models"
)

// BenchmarkListEmployeesConversion mengukur biaya konversi DTO pada daftar
// pegawai. Jalur ini dipanggil setiap permintaan daftar, sehingga regresi di
// sini langsung terasa pada latensi.
func BenchmarkListEmployeesConversion(b *testing.B) {
	employees := make([]models.User, 0, 100)
	for index := 0; index < 100; index++ {
		employees = append(employees, models.User{
			BaseModel: models.BaseModel{IDModel: models.IDModel{ID: "id"}},
			Nama:      "Pegawai Benchmark",
			Email:     "pegawai@example.test",
			Aktif:     true,
		})
	}
	store := &fakeDirectoryStore{employees: employees}
	mux := scopeMux(store, auth.Principal{UserID: atasanID, RoleName: "admin"},
		map[string]map[auth.Permission]bool{"admin": {"employee:read:any": true}})

	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		request := httptest.NewRequest(http.MethodGet, APIPrefix+"/employees?pageSize=100", nil)
		request.Header.Set("Authorization", "Bearer valid-session")
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			b.Fatalf("status = %d", response.Code)
		}
	}
}

// BenchmarkAuditLogDTOConversion mengukur konversi jejak audit menjadi DTO.
func BenchmarkAuditLogDTOConversion(b *testing.B) {
	entries := make([]models.AuditLog, 0, 100)
	for index := 0; index < 100; index++ {
		entries = append(entries, models.AuditLog{
			IDModel: models.IDModel{ID: "log-id"},
			Aksi:    "employee.updated",
		})
	}
	store := &fakeDirectoryStore{auditLogs: entries}
	mux := scopeMux(store, auth.Principal{UserID: atasanID, RoleName: "admin"},
		map[string]map[auth.Permission]bool{"admin": {"audit:read": true}})

	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		request := httptest.NewRequest(http.MethodGet, APIPrefix+"/audit-logs?pageSize=100", nil)
		request.Header.Set("Authorization", "Bearer valid-session")
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			b.Fatalf("status = %d", response.Code)
		}
	}
}

// BenchmarkUnitScopeResolution mengukur biaya resolusi cakupan unit, yang
// dijalankan pada setiap permintaan milik operator dan atasan.
func BenchmarkUnitScopeResolution(b *testing.B) {
	store := &fakeDirectoryStore{scopeIDs: []string{unitA, unitAChild}}
	mux := scopeMux(store, auth.Principal{UserID: atasanID, RoleName: "atasan", HomebaseID: unitA},
		map[string]map[auth.Permission]bool{"atasan": {"employee:read:homebase": true}})

	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		request := httptest.NewRequest(http.MethodGet, APIPrefix+"/employees", nil)
		request.Header.Set("Authorization", "Bearer valid-session")
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			b.Fatalf("status = %d", response.Code)
		}
	}
}

// BenchmarkListEmployeesParallel mengukur throughput daftar pegawai saat
// banyak permintaan dilayani bersamaan.
//
// Benchmark ini memakai repository palsu yang aman untuk akses paralel, jadi
// angka yang dihasilkan menggambarkan kapasitas jalur HTTP dan serialisasi
// DTO, bukan latensi PostgreSQL.
func BenchmarkListEmployeesParallel(b *testing.B) {
	store := &loadStore{employees: loadEmployees(100)}
	mux := loadMux(store, auth.Principal{UserID: atasanID, RoleName: "admin"},
		map[string]map[auth.Permission]bool{"admin": {"employee:read:any": true}})

	var failures atomic.Int64
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(iteration *testing.PB) {
		for iteration.Next() {
			request := httptest.NewRequest(http.MethodGet, APIPrefix+"/employees?pageSize=100", nil)
			request.Header.Set("Authorization", "Bearer valid-session")
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			if response.Code != http.StatusOK {
				failures.Add(1)
			}
		}
	})
	b.StopTimer()
	if failures.Load() != 0 {
		b.Fatalf("failed requests = %d, want 0", failures.Load())
	}
}
