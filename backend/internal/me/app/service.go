// Package app menjalankan use case self-service profil principal. Principal
// hanya boleh mengubah profilnya sendiri; cakupan ditegakkan dengan membatasi
// seluruh mutasi ke identifier dari sesi terverifikasi.
package app

import (
	"context"
	"fmt"

	"simpeg/backend/internal/me/domain"
	"simpeg/backend/internal/models"
)

// Store adalah port persistensi profil mandiri. Implementasi adapter GORM
// menulis perubahan dan jejak audit dalam satu transaksi.
type Store interface {
	UpdateProfile(ctx context.Context, userID string, input domain.UpdateProfileInput) (models.User, error)
	FindProfile(ctx context.Context, userID string) (models.User, error)
}

// Service menjalankan use case profil mandiri.
type Service struct {
	store Store
}

// NewService menyusun service dengan port penyimpanan yang diberikan.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Get mengembalikan profil principal. Respons API hanya memakai kolom
// non-rahasia dari record ini.
func (s *Service) Get(ctx context.Context, userID string) (models.User, error) {
	if userID == "" {
		return models.User{}, fmt.Errorf("%w: user id must not be empty", domain.ErrValidation)
	}
	user, err := s.store.FindProfile(ctx, userID)
	if err != nil {
		return models.User{}, fmt.Errorf("find profile: %w", err)
	}
	return user, nil
}

// Update mengubah kolom profil yang boleh diubah principal sendiri.
func (s *Service) Update(ctx context.Context, userID string, input domain.UpdateProfileInput) (models.User, error) {
	if userID == "" {
		return models.User{}, fmt.Errorf("%w: user id must not be empty", domain.ErrValidation)
	}
	normalized, err := domain.Normalize(input)
	if err != nil {
		return models.User{}, err
	}
	user, err := s.store.UpdateProfile(ctx, userID, normalized)
	if err != nil {
		return models.User{}, fmt.Errorf("update profile: %w", err)
	}
	return user, nil
}
