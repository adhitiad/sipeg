// Package repository menyimpan adapter GORM untuk pembacaan slip via token
// verifikasi publik.
package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"simpeg/backend/internal/models"
	"simpeg/backend/internal/payroll/domain"
)

// FindSlipByVerificationToken membaca slip beserta data pegawai untuk halaman
// verifikasi publik. Token invalid menghasilkan ErrInvalidSlipToken; token
// valid tetapi tidak dikenal menghasilkan domain.ErrNotFound.
func (s GormStore) FindSlipByVerificationToken(ctx context.Context, token string) (models.Gaji, error) {
	if !domain.ValidSlipVerificationToken(token) {
		return models.Gaji{}, domain.ErrInvalidSlipToken
	}
	var slip models.Gaji
	if err := s.DB.WithContext(ctx).Preload("Pegawai").Where(`"kodeVerifikasi" = ?`, token).First(&slip).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Gaji{}, domain.ErrNotFound
		}
		return models.Gaji{}, fmt.Errorf("find slip by verification token: %w", err)
	}
	return slip, nil
}
