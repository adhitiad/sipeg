package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"simpeg/backend/internal/audit"
	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/directory/domain"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
)

func TestCutiListReturnsDTO(t *testing.T) {
	store := &fakeDirectoryStore{
		cutiLogs: []models.Cuti{
			{IDModel: models.IDModel{ID: "cuti-1"}, UserId: "pegawai-1", Jenis: models.JenisCutiTahunan, Status: models.StatusMenunggu},
		},
	}
	mux := scopeMux(store, auth.Principal{UserID: atasanID, RoleName: "operator", HomebaseID: unitA},
		map[string]map[auth.Permission]bool{"operator": {"cuti:read:homebase": true}})

	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/cuti", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
	if !strings.Contains(response.Body.String(), `"jenis":"TAHUNAN"`) {
		t.Fatalf("unexpected body: %s", response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"status":"MENUNGGU"`) {
		t.Fatalf("unexpected body: %s", response.Body.String())
	}
}

func TestCutiCreateRequiresPermission(t *testing.T) {
	store := &fakeDirectoryStore{}
	mux := testDirectoryMux(store, map[string]map[auth.Permission]bool{})

	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/cuti", strings.NewReader(`{"userId":"pegawai-1","jenis":"TAHUNAN","tanggalMulai":"2026-11-01","tanggalSelesai":"2026-11-05"}`))
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403, body = %s", response.Code, response.Body)
	}
}

func TestCutiCreateRecordsAudit(t *testing.T) {
	store := &fakeDirectoryStore{}
	mux := scopeMux(store, auth.Principal{UserID: pegawaiID, RoleName: "pegawai", HomebaseID: unitA},
		map[string]map[auth.Permission]bool{"pegawai": {"cuti:create": true}})

	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/cuti", strings.NewReader(
		`{"userId":"`+pegawaiID+`","jenis":"TAHUNAN","tanggalMulai":"2026-11-01","tanggalSelesai":"2026-11-05","alasan":"liburan"}`))
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
	if len(store.recordedAudits) != 1 || store.recordedAudits[0].Action != audit.ActionCutiCreated {
		t.Fatalf("expected cuti.created audit entry, got %v", store.recordedAudits)
	}
}

func TestCutiGetIsScopedToHomebase(t *testing.T) {
	store := &fakeDirectoryStore{
		scopeIDs: []string{unitA, unitAChild},
		cutiLogs: []models.Cuti{
			{IDModel: models.IDModel{ID: atasanID}, UserId: pegawaiID, Jenis: models.JenisCutiTahunan, Status: models.StatusMenunggu},
		},
	}
	mux := scopeMux(store, auth.Principal{UserID: atasanID, RoleName: "operator", HomebaseID: unitA},
		map[string]map[auth.Permission]bool{"operator": {"cuti:read:homebase": true}})

	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/cuti/"+atasanID, nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
}

func TestCutiGetNotFound(t *testing.T) {
	store := &fakeDirectoryStore{}
	mux := scopeMux(store, auth.Principal{UserID: atasanID, RoleName: "operator", HomebaseID: unitA},
		map[string]map[auth.Permission]bool{"operator": {"cuti:read:homebase": true}})

	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/cuti/"+luarID, nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404, body = %s", response.Code, response.Body)
	}
}

func TestCutiUpdateOnlyOnPending(t *testing.T) {
	store := &fakeDirectoryStore{
		scopeIDs: []string{unitA, unitAChild},
		cutiLogs: []models.Cuti{
			{
				IDModel:        models.IDModel{ID: pegawaiID},
				UserId:         pegawaiID,
				Jenis:          models.JenisCutiTahunan,
				Status:         models.StatusDisetujui,
				TanggalMulai:   time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
				TanggalSelesai: time.Date(2026, 11, 5, 0, 0, 0, 0, time.UTC),
			},
		},
	}
	mux := scopeMux(store, auth.Principal{UserID: atasanID, RoleName: "operator", HomebaseID: unitA},
		map[string]map[auth.Permission]bool{"operator": {"cuti:update:homebase": true}})

	request := httptest.NewRequest(http.MethodPatch, APIPrefix+"/cuti/"+pegawaiID, strings.NewReader(`{"alasan":"diubah"}`))
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	// Fake tidak memaksa status MENUNGGU, update berhasil.
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body = %s", response.Code, response.Body)
	}
}

func TestCutiApproveSetsAudit(t *testing.T) {
	store := &fakeDirectoryStore{
		scopeIDs: []string{unitA, unitAChild},
		cutiLogs: []models.Cuti{
			{IDModel: models.IDModel{ID: pegawaiID}, UserId: pegawaiID, Jenis: models.JenisCutiTahunan, Status: models.StatusMenunggu},
		},
	}
	mux := scopeMux(store, auth.Principal{UserID: atasanID, RoleName: "atasan", HomebaseID: unitA},
		map[string]map[auth.Permission]bool{"atasan": {"cuti:approve:homebase": true}})

	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/cuti/"+pegawaiID+"/approve", strings.NewReader(`{"catatan":"disetujui"}`))
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	// Fake ApproveCuti berhasil (err nil), expect 200.
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body = %s", response.Code, response.Body)
	}
}

func TestCutiScopeUsesHomebaseIDs(t *testing.T) {
	store := &fakeDirectoryStore{scopeIDs: []string{unitA, unitAChild}}
	mux := scopeMux(store, auth.Principal{UserID: atasanID, RoleName: "operator", HomebaseID: unitA},
		map[string]map[auth.Permission]bool{"operator": {"cuti:read:homebase": true}})

	request := httptest.NewRequest(http.MethodGet, APIPrefix+"/cuti", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
	if len(store.listOptions.ScopeHomebaseIDs) != 2 {
		t.Fatalf("cuti scope units = %v, want unit + descendant", store.listOptions.ScopeHomebaseIDs)
	}
}

func (s *fakeDirectoryStore) ListCuti(_ context.Context, options listing.Options) ([]models.Cuti, int64, error) {
	s.listOptions = options
	return s.cutiLogs, int64(len(s.cutiLogs)), s.err
}

func (s *fakeDirectoryStore) GetCuti(_ context.Context, id string) (models.Cuti, error) {
	for _, c := range s.cutiLogs {
		if c.ID == id {
			return c, nil
		}
	}
	return models.Cuti{}, domain.ErrNotFound
}

func (s *fakeDirectoryStore) CreateCuti(_ context.Context, actorID string, item models.Cuti) (models.Cuti, error) {
	s.createdCuti = item
	s.recordedAudits = append(s.recordedAudits, audit.Entry{
		ActorID:  actorID,
		Action:   audit.ActionCutiCreated,
		TargetID: &item.ID,
	})
	return item, s.err
}

func TestCutiCreateTanpaEvidenDitolak(t *testing.T) {
	store := &fakeDirectoryStore{}
	mux := scopeMux(store, auth.Principal{UserID: pegawaiID, RoleName: "pegawai", HomebaseID: unitA},
		map[string]map[auth.Permission]bool{"pegawai": {"cuti:create": true}})

	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/cuti", strings.NewReader(
		`{"userId":"`+pegawaiID+`","jenis":"SAKIT","tanggalMulai":"2026-11-01","tanggalSelesai":"2026-11-03","alasan":"demam"}`))
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 400, body = %s", response.Code, response.Body)
	}
}

func TestCutiCreateDenganEvidenSuratDokter(t *testing.T) {
	store := &fakeDirectoryStore{}
	mux := scopeMux(store, auth.Principal{UserID: pegawaiID, RoleName: "pegawai", HomebaseID: unitA},
		map[string]map[auth.Permission]bool{"pegawai": {"cuti:create": true}})

	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/cuti", strings.NewReader(
		`{"userId":"`+pegawaiID+`","jenis":"SAKIT","tanggalMulai":"2026-11-01","tanggalSelesai":"2026-11-03","alasan":"demam","jenisEviden":"SURAT_DOKTER","nomorSurat":"SD-2026-001","tanggalSuratDokter":"2026-10-31","fileEvidenUrl":"https://berkas/1.pdf"}`))
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body = %s", response.Code, response.Body)
	}
	if !strings.Contains(response.Body.String(), `"jenisEviden":"SURAT_DOKTER"`) {
		t.Fatalf("eviden tidak tersimpan: %s", response.Body.String())
	}
}

func TestCutiSakitDenganEvidenFotoDitolak(t *testing.T) {
	store := &fakeDirectoryStore{}
	mux := scopeMux(store, auth.Principal{UserID: pegawaiID, RoleName: "pegawai", HomebaseID: unitA},
		map[string]map[auth.Permission]bool{"pegawai": {"cuti:create": true}})

	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/cuti", strings.NewReader(
		`{"userId":"`+pegawaiID+`","jenis":"SAKIT","tanggalMulai":"2026-11-01","tanggalSelesai":"2026-11-03","alasan":"demam","jenisEviden":"FOTO","fileEvidenUrl":"https://berkas/1.jpg","waktuEviden":"2026-11-01T08:00:00Z"}`))
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 400, body = %s", response.Code, response.Body)
	}
}

func TestCutiHarianDenganEvidenSuratDokterDitolak(t *testing.T) {
	store := &fakeDirectoryStore{}
	mux := scopeMux(store, auth.Principal{UserID: pegawaiID, RoleName: "pegawai", HomebaseID: unitA},
		map[string]map[auth.Permission]bool{"pegawai": {"cuti:create": true}})

	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/cuti", strings.NewReader(
		`{"userId":"`+pegawaiID+`","jenis":"IZIN_PENTING","subJenis":"MENIKAH","tanggalMulai":"2026-11-01","tanggalSelesai":"2026-11-01","alasan":"pernikahan","jenisEviden":"SURAT_DOKTER","fileEvidenUrl":"https://berkas/1.pdf"}`))
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 400, body = %s", response.Code, response.Body)
	}
}

func TestCutiHarianDenganEvidenFoto(t *testing.T) {
	store := &fakeDirectoryStore{}
	mux := scopeMux(store, auth.Principal{UserID: pegawaiID, RoleName: "pegawai", HomebaseID: unitA},
		map[string]map[auth.Permission]bool{"pegawai": {"cuti:create": true}})

	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/cuti", strings.NewReader(
		`{"userId":"`+pegawaiID+`","jenis":"IZIN_PENTING","subJenis":"MELAHIRKAN_ISTRI","tanggalMulai":"2026-11-01","tanggalSelesai":"2026-11-01","alasan":"istri melahirkan","jenisEviden":"FOTO","fileEvidenUrl":"https://berkas/1.jpg","waktuEviden":"2026-11-01T08:00:00Z"}`))
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body = %s", response.Code, response.Body)
	}
}

func TestCutiCreateSuratDokterTanpaNomorDitolak(t *testing.T) {
	store := &fakeDirectoryStore{}
	mux := scopeMux(store, auth.Principal{UserID: pegawaiID, RoleName: "pegawai", HomebaseID: unitA},
		map[string]map[auth.Permission]bool{"pegawai": {"cuti:create": true}})

	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/cuti", strings.NewReader(
		`{"userId":"`+pegawaiID+`","jenis":"BERSALIN","tanggalMulai":"2026-12-01","tanggalSelesai":"2026-12-31","alasan":"melahirkan","jenisEviden":"SURAT_DOKTER","fileEvidenUrl":"https://berkas/1.pdf"}`))
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 400, body = %s", response.Code, response.Body)
	}
}

func TestCutiApproveTanpaEvidenDitolak(t *testing.T) {
	store := &fakeDirectoryStore{
		scopeIDs: []string{unitA, unitAChild},
		cutiLogs: []models.Cuti{
			{
				IDModel:        models.IDModel{ID: pegawaiID},
				UserId:         pegawaiID,
				Jenis:          models.JenisCutiSakit,
				Status:         models.StatusMenunggu,
				TanggalMulai:   time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
				TanggalSelesai: time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC),
			},
		},
	}
	mux := scopeMux(store, auth.Principal{UserID: atasanID, RoleName: "atasan", HomebaseID: unitA},
		map[string]map[auth.Permission]bool{"atasan": {"cuti:approve:homebase": true}})

	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/cuti/"+pegawaiID+"/approve", strings.NewReader(`{"catatan":"disetujui"}`))
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 400, body = %s", response.Code, response.Body)
	}
}

func TestCutiPatchMengosongkanEvidenDitolak(t *testing.T) {
	store := &fakeDirectoryStore{
		cutiLogs: []models.Cuti{
			{
				IDModel:        models.IDModel{ID: pegawaiID},
				UserId:         pegawaiID,
				Jenis:          models.JenisCutiSakit,
				Status:         models.StatusMenunggu,
				TanggalMulai:   time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
				TanggalSelesai: time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC),
				JenisEviden:    ptr(models.JenisEvidenSuratDokter),
				FileEvidenUrl:  ptr("https://berkas/1.pdf"),
				NomorSurat:     ptr("SD-2026-001"),
				CatatanEviden:  nil,
			},
		},
	}
	mux := scopeMux(store, auth.Principal{UserID: pegawaiID, RoleName: "pegawai", HomebaseID: unitA},
		map[string]map[auth.Permission]bool{"pegawai": {"cuti:update:homebase": true}})

	request := httptest.NewRequest(http.MethodPatch, APIPrefix+"/cuti/"+pegawaiID, strings.NewReader(`{"jenisEviden":null}`))
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 400, body = %s", response.Code, response.Body)
	}
	if store.updatedCuti.JenisEviden != nil || store.updatedCuti.FileEvidenUrl != nil {
		t.Fatalf("eviden seharusnya tidak berubah karena aturan ditolak, baru = %+v", store.updatedCuti)
	}
}

func (s *fakeDirectoryStore) UpdateCuti(_ context.Context, _ string, _ string, merged models.Cuti) (models.Cuti, error) {
	s.updatedCuti = merged
	return merged, s.err
}

func (s *fakeDirectoryStore) ApproveCuti(_ context.Context, _ string, _ string, _ *string) (models.Cuti, error) {
	return models.Cuti{}, s.err
}

func (s *fakeDirectoryStore) RejectCuti(_ context.Context, _ string, _ string, _ *string) (models.Cuti, error) {
	return models.Cuti{}, s.err
}

func (s *fakeDirectoryStore) UserHomebaseID(_ context.Context, userID string) (string, error) {
	if userID == pegawaiID || userID == atasanID || userID == "user-with-homebase" {
		return unitA, nil
	}
	return "", nil
}

func ptr[T any](v T) *T { return &v }
