package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"simpeg/backend/internal/models"
)

var ErrInvalidSession = errors.New("invalid session")

type GormSessionResolver struct {
	DB *gorm.DB
}

func (r GormSessionResolver) ResolvePrincipal(ctx context.Context, token string) (Principal, error) {
	if r.DB == nil || strings.TrimSpace(token) == "" {
		return Principal{}, ErrInvalidSession
	}

	var session models.Session
	err := r.DB.WithContext(ctx).Where("token = ?", token).First(&session).Error
	if errors.Is(err, gorm.ErrRecordNotFound) || (err == nil && !session.ExpiresAt.After(time.Now())) {
		return Principal{}, ErrInvalidSession
	}
	if err != nil {
		return Principal{}, fmt.Errorf("resolve session: %w", err)
	}
	return r.ResolveUserPrincipal(ctx, session.UserId)
}

// ResolveUserPrincipal memuat ulang status akun, role, dan homebase dari
// PostgreSQL. Klaim JWT hanya boleh menyumbang userId; klaim otorisasi dalam
// token tidak menjadi sumber kebenaran.
func (r GormSessionResolver) ResolveUserPrincipal(ctx context.Context, userID string) (Principal, error) {
	if r.DB == nil || strings.TrimSpace(userID) == "" {
		return Principal{}, ErrInvalidSession
	}

	var user models.User
	err := r.DB.WithContext(ctx).
		Select("id", "email", "aktif", "roleId", "homebaseId").
		First(&user, "id = ?", userID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Principal{}, ErrInvalidSession
	}
	if err != nil {
		return Principal{}, fmt.Errorf("resolve session user: %w", err)
	}
	if !user.Aktif {
		return Principal{}, ErrInvalidSession
	}

	principal := Principal{
		UserID:     user.ID,
		Email:      user.Email,
		HomebaseID: valueOrEmpty(user.HomebaseId),
	}
	if user.RoleId == nil || *user.RoleId == "" {
		return principal, nil
	}

	var role models.Role
	if err := r.DB.WithContext(ctx).First(&role, "id = ?", *user.RoleId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Principal{}, ErrInvalidSession
		}
		return Principal{}, fmt.Errorf("resolve session role: %w", err)
	}
	principal.RoleName = role.Name
	return principal, nil
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
