package auth

import (
	"context"
	"errors"
	"strings"

	"simpeg/backend/internal/audit"
	"simpeg/backend/internal/models"

	"gorm.io/gorm"
)

// ErrSessionNotFound berarti token sesi sudah tidak berlaku di database.
var ErrSessionNotFound = errors.New("session not found")

// SessionRevoker mencabut sesi aktif sehingga token tidak lagi berlaku.
type SessionRevoker interface {
	RevokeSession(ctx context.Context, token string, actor Principal) error
}

// GormSessionRevoker menghapus baris sesi di tabel session dan mencatat jejak audit.
type GormSessionRevoker struct {
	DB *gorm.DB
}

func (r GormSessionRevoker) RevokeSession(ctx context.Context, token string, actor Principal) error {
	if r.DB == nil || strings.TrimSpace(token) == "" {
		return ErrSessionNotFound
	}
	var session models.Session
	if err := r.DB.WithContext(ctx).Where("token = ?", token).First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrSessionNotFound
		}
		return err
	}

	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Where("token = ?", token).Delete(&models.Session{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrSessionNotFound
		}
		// Jejak audit mencatat bahwa sesi dicabut, bukan isi token.
		var unitID *string
		if actor.HomebaseID != "" {
			unitID = &actor.HomebaseID
		}
		return audit.Write(ctx, tx, audit.Entry{
			ActorID:  actor.UserID,
			Action:   audit.ActionSessionRevoked,
			TargetID: &session.ID,
			UnitID:   unitID,
		})
	})
}
