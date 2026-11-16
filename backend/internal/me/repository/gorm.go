// Package repository menyimpan profil mandiri principal lewat GORM. Perubahan
// dan jejak audit ditulis dalam satu transaksi.
package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"simpeg/backend/internal/audit"
	"simpeg/backend/internal/me/domain"
	"simpeg/backend/internal/models"
	"simpeg/backend/internal/timeutil"

	"gorm.io/gorm"
)

// GormStore adalah adapter persistensi profil mandiri.
type GormStore struct {
	DB *gorm.DB
}

// FindProfile membaca record user principal.
func (s GormStore) FindProfile(ctx context.Context, userID string) (models.User, error) {
	var user models.User
	if err := s.DB.WithContext(ctx).
		Select("id", "nama", "no_hp", "homebaseId", "updatedAt").
		First(&user, "id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.User{}, domain.ErrNotFound
		}
		return models.User{}, fmt.Errorf("read user profile: %w", err)
	}
	return user, nil
}

// UpdateProfile mengubah kolom profil yang boleh diubah principal sendiri dan
// menulis jejak audit dalam satu transaksi.
func (s GormStore) UpdateProfile(ctx context.Context, userID string, input domain.UpdateProfileInput) (models.User, error) {
	var user models.User
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{"updatedAt": timeutil.NowUTC()}
		if input.Nama != nil {
			updates["nama"] = *input.Nama
		}
		if input.NoHp != nil {
			if *input.NoHp == "" {
				updates["no_hp"] = nil
			} else {
				updates["no_hp"] = *input.NoHp
			}
		}
		result := tx.Model(&models.User{}).Where("id = ?", userID).Updates(updates)
		if result.Error != nil {
			return fmt.Errorf("update user profile: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return domain.ErrNotFound
		}
		if err := tx.Select("id", "nama", "no_hp", "homebaseId", "updatedAt").First(&user, "id = ?", userID).Error; err != nil {
			return fmt.Errorf("read updated user profile: %w", err)
		}
		targetID := user.ID
		note := "pembaruan profil mandiri (fields=" + strings.Join(input.Fields(), ",") + ")"
		return audit.Write(ctx, tx, audit.Entry{
			ActorID:  userID,
			Action:   audit.ActionProfileUpdated,
			TargetID: &targetID,
			UnitID:   user.HomebaseId,
			Note:     &note,
		})
	})
	if err != nil {
		return models.User{}, err
	}
	return user, nil
}
