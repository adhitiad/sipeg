package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	. "simpeg/backend/internal/httpserver"
	"strings"
	"testing"

	"simpeg/backend/internal/me/app"
	"simpeg/backend/internal/me/domain"
	"simpeg/backend/internal/models"
)

type fakeMeStore struct {
	user      models.User
	findErr   error
	updateErr error
	gotInput  domain.UpdateProfileInput
}

func (f *fakeMeStore) FindProfile(_ context.Context, _ string) (models.User, error) {
	if f.findErr != nil {
		return models.User{}, f.findErr
	}
	return f.user, nil
}

func (f *fakeMeStore) UpdateProfile(_ context.Context, _ string, input domain.UpdateProfileInput) (models.User, error) {
	f.gotInput = input
	if f.updateErr != nil {
		return models.User{}, f.updateErr
	}
	updated := f.user
	if input.Nama != nil {
		updated.Nama = *input.Nama
	}
	if input.NoHp != nil {
		if *input.NoHp == "" {
			updated.NoHp = nil
		} else {
			updated.NoHp = input.NoHp
		}
	}
	f.user = updated
	return updated, nil
}

func newMeService(store *fakeMeStore) *app.Service {
	return app.NewService(store)
}

func patchMeRequest(t *testing.T, mux http.Handler, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPatch, APIPrefix+"/me", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	return res
}

func TestPatchMeRequiresSession(t *testing.T) {
	t.Parallel()
	mux := NewMuxWithOptions(MuxOptions{
		Resolver: testSessionResolver{},
		Me:       newMeService(&fakeMeStore{}),
	})

	res := patchMeRequest(t, mux, "", `{"nama":"Budi"}`)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", res.Code)
	}
}

func TestPatchMeUpdatesProfile(t *testing.T) {
	t.Parallel()
	store := &fakeMeStore{user: models.User{Nama: "Budi Lama"}}
	mux := NewMuxWithOptions(MuxOptions{
		Resolver: testSessionResolver{},
		Me:       newMeService(store),
	})

	res := patchMeRequest(t, mux, "valid-session", `{"nama":"  Budi   Santoso ","noHp":"0812-3456-7890"}`)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", res.Code, res.Body.String())
	}
	if store.gotInput.Nama == nil || *store.gotInput.Nama != "Budi Santoso" {
		t.Fatalf("expected normalized nama, got %+v", store.gotInput.Nama)
	}
	if store.gotInput.NoHp == nil || *store.gotInput.NoHp != "081234567890" {
		t.Fatalf("expected normalized noHp, got %+v", store.gotInput.NoHp)
	}
	if !strings.Contains(res.Body.String(), `"nama":"Budi Santoso"`) {
		t.Fatalf("expected updated profile in body, got %s", res.Body.String())
	}
	if !strings.Contains(res.Body.String(), `"noHp":"081234567890"`) {
		t.Fatalf("expected updated noHp in body, got %s", res.Body.String())
	}
}

func TestPatchMeRejectsEmptyPatch(t *testing.T) {
	t.Parallel()
	mux := NewMuxWithOptions(MuxOptions{
		Resolver: testSessionResolver{},
		Me:       newMeService(&fakeMeStore{}),
	})

	res := patchMeRequest(t, mux, "valid-session", `{}`)
	if res.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", res.Code)
	}
}

func TestPatchMeRejectsInvalidNoHp(t *testing.T) {
	t.Parallel()
	mux := NewMuxWithOptions(MuxOptions{
		Resolver: testSessionResolver{},
		Me:       newMeService(&fakeMeStore{}),
	})

	res := patchMeRequest(t, mux, "valid-session", `{"noHp":"bukan-nomor"}`)
	if res.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", res.Code)
	}
}

func TestPatchMeRejectsMalformedJSON(t *testing.T) {
	t.Parallel()
	mux := NewMuxWithOptions(MuxOptions{
		Resolver: testSessionResolver{},
		Me:       newMeService(&fakeMeStore{}),
	})

	res := patchMeRequest(t, mux, "valid-session", `{"nama":`)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.Code)
	}
	if !strings.Contains(res.Body.String(), `"code":"invalid_json"`) {
		t.Fatalf("expected invalid_json code, got %s", res.Body.String())
	}
}

func TestPatchMeRejectsUnknownField(t *testing.T) {
	t.Parallel()
	mux := NewMuxWithOptions(MuxOptions{
		Resolver: testSessionResolver{},
		Me:       newMeService(&fakeMeStore{}),
	})

	res := patchMeRequest(t, mux, "valid-session", `{"gajiPokok":123}`)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.Code)
	}
}

func TestPatchMeNotFoundWhenUserRecordMissing(t *testing.T) {
	t.Parallel()
	store := &fakeMeStore{updateErr: domain.ErrNotFound}
	mux := NewMuxWithOptions(MuxOptions{
		Resolver: testSessionResolver{},
		Me:       newMeService(store),
	})

	res := patchMeRequest(t, mux, "valid-session", `{"nama":"Budi"}`)
	if res.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", res.Code)
	}
}

func TestGetMeIncludesNamaAndNoHpWhenMeServiceWired(t *testing.T) {
	t.Parallel()
	store := &fakeMeStore{user: models.User{Nama: "Budi Santoso", NoHp: ptrString("081234567890")}}
	mux := NewMuxWithOptions(MuxOptions{
		Resolver: testSessionResolver{},
		Me:       newMeService(store),
	})

	req := httptest.NewRequest(http.MethodGet, APIPrefix+"/me", nil)
	req.Header.Set("Authorization", "Bearer valid-session")
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected JSON body, got %q", res.Body.String())
	}
	if body["nama"] != "Budi Santoso" || body["noHp"] != "081234567890" {
		t.Fatalf("expected nama and noHp enriched from record, got %v", body)
	}
}

func TestGetMeOmitsNamaWhenMeServiceNotWired(t *testing.T) {
	t.Parallel()
	mux := NewMux(testSessionResolver{})

	req := httptest.NewRequest(http.MethodGet, APIPrefix+"/me", nil)
	req.Header.Set("Authorization", "Bearer valid-session")
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
	if strings.Contains(res.Body.String(), `"nama"`) {
		t.Fatalf("expected no nama field without me service, got %s", res.Body.String())
	}
}

func ptrString(value string) *string { return &value }
