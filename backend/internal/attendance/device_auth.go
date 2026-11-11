package attendance

import (
	"context"
	"crypto/subtle"
	"errors"
	"time"

	"gorm.io/gorm"
	"simpeg/backend/internal/models"
)

var (
	// ErrInvalidDeviceCredentials dikembalikan bila kredensial perangkat tidak valid.
	ErrInvalidDeviceCredentials = errors.New("invalid device credentials")
)

// DeviceAuthenticator memverifikasi kredensial perangkat mesin absensi.
//
// Implementasi inti adalah GORM (tabel machineCredential); nanti dapat
// diganti dengan implementasi yang sesuai dengan protokol vendor setelah
// kontrak API/SDK vendor tersedia.
type DeviceAuthenticator interface {
	// AuthenticateDevice memverifikasi deviceID dan deviceToken terhadap
	// tabel machineCredential. Mengembalikan mesin bila valid, atau
	// ErrInvalidDeviceCredentials bila tidak.
	AuthenticateDevice(ctx context.Context, deviceID, deviceToken string) (*models.MesinAbsensi, error)

	// RecordLastUsed mencatat waktu penggunaan terakhir kredensial perangkat.
	RecordLastUsed(ctx context.Context, machineID string) error
}

// GormDeviceAuthenticator memverifikasi kredensial perangkat melalui GORM.
type GormDeviceAuthenticator struct {
	DB *gorm.DB
}

// NewGormDeviceAuthenticator membuat authentikator perangkat GORM.
func NewGormDeviceAuthenticator(db *gorm.DB) *GormDeviceAuthenticator {
	return &GormDeviceAuthenticator{DB: db}
}

// AuthenticateDevice mencari kredensial aktif untuk deviceID dan membandingkan
// deviceToken dengan credentialHash menggunakan perbandingan konstan waktu.
func (a *GormDeviceAuthenticator) AuthenticateDevice(ctx context.Context, deviceID, deviceToken string) (*models.MesinAbsensi, error) {
	if a.DB == nil {
		return nil, errors.New("device authenticator database is required")
	}

	var credential models.MachineCredential
	if err := a.DB.WithContext(ctx).
		Where("machine_id = ? AND active = ?", deviceID, true).
		First(&credential).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidDeviceCredentials
		}
		return nil, err
	}

	if subtle.ConstantTimeCompare([]byte(deviceToken), []byte(credential.CredentialHash)) != 1 {
		return nil, ErrInvalidDeviceCredentials
	}

	var machine models.MesinAbsensi
	if err := a.DB.WithContext(ctx).First(&machine, "id = ?", credential.MachineId).Error; err != nil {
		return nil, err
	}
	if !machine.Aktif {
		return nil, ErrInvalidDeviceCredentials
	}

	_ = a.RecordLastUsed(ctx, machine.ID)

	return &machine, nil
}

// RecordLastUsed memperbarui lastUsedAt pada kredensial perangkat.
func (a *GormDeviceAuthenticator) RecordLastUsed(ctx context.Context, machineID string) error {
	if a.DB == nil {
		return errors.New("device authenticator database is required")
	}
	return a.DB.WithContext(ctx).
		Model(&models.MachineCredential{}).
		Where("machine_id = ? AND active = ?", machineID, true).
		Update("last_used_at", time.Now().UTC()).Error
}
