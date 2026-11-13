package app

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"simpeg/backend/internal/me/app"
	"simpeg/backend/internal/me/domain"
	"simpeg/backend/internal/models"
)

func ptr(value string) *string { return &value }

type fakeStore struct {
	user      models.User
	findErr   error
	updateErr error

	gotUserID   string
	gotInput    domain.UpdateProfileInput
	findCalls   int
	updateCalls int
}

func (f *fakeStore) FindProfile(_ context.Context, userID string) (models.User, error) {
	f.findCalls++
	if f.findErr != nil {
		return models.User{}, f.findErr
	}
	return f.user, nil
}

func (f *fakeStore) UpdateProfile(_ context.Context, userID string, input domain.UpdateProfileInput) (models.User, error) {
	f.updateCalls++
	f.gotUserID = userID
	f.gotInput = input
	if f.updateErr != nil {
		return models.User{}, f.updateErr
	}
	updated := f.user
	updated.Nama = "Budi Santoso"
	if input.NoHp != nil && *input.NoHp == "" {
		updated.NoHp = nil
	}
	return updated, nil
}

func TestGetReturnsProfile(t *testing.T) {
	t.Parallel()
	store := &fakeStore{user: models.User{Nama: "Budi Santoso", NoHp: ptr("081234567890")}}
	service := app.NewService(store)

	user, err := service.Get(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if user.Nama != "Budi Santoso" || user.NoHp == nil || *user.NoHp != "081234567890" {
		t.Fatalf("unexpected profile: %+v", user)
	}
}

func TestGetRejectsEmptyUserID(t *testing.T) {
	t.Parallel()
	service := app.NewService(&fakeStore{})

	_, err := service.Get(context.Background(), "")
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestUpdateNormalizesBeforeCallingStore(t *testing.T) {
	t.Parallel()
	store := &fakeStore{user: models.User{Nama: "Lama"}}
	service := app.NewService(store)

	_, err := service.Update(context.Background(), "user-1", domain.UpdateProfileInput{
		Nama: ptr("  Budi   Santoso "),
		NoHp: ptr("0812-3456-7890"),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if store.gotUserID != "user-1" {
		t.Fatalf("expected store called with user-1, got %q", store.gotUserID)
	}
	if store.gotInput.Nama == nil || *store.gotInput.Nama != "Budi Santoso" {
		t.Fatalf("expected normalized nama, got %+v", store.gotInput.Nama)
	}
	if store.gotInput.NoHp == nil || *store.gotInput.NoHp != "081234567890" {
		t.Fatalf("expected normalized noHp, got %+v", store.gotInput.NoHp)
	}
}

func TestUpdateRejectsInvalidInputWithoutTouchingStore(t *testing.T) {
	t.Parallel()
	store := &fakeStore{}
	service := app.NewService(store)

	_, err := service.Update(context.Background(), "user-1", domain.UpdateProfileInput{})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
	if store.updateCalls != 0 {
		t.Fatal("store must not be called for invalid input")
	}
}

func TestUpdateWrapsStoreError(t *testing.T) {
	t.Parallel()
	store := &fakeStore{updateErr: fmt.Errorf("update user profile: %w", domain.ErrNotFound)}
	service := app.NewService(store)

	_, err := service.Update(context.Background(), "user-1", domain.UpdateProfileInput{Nama: ptr("Budi")})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected wrapped ErrNotFound, got %v", err)
	}
}
