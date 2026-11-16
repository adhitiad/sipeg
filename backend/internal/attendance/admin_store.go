// Package attendance memuat adapter GORM untuk AdminStore.
package attendance

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"simpeg/backend/internal/audit"
	"simpeg/backend/internal/models"
	"simpeg/backend/internal/timeutil"
)

// GormAdminStore adalah AdminStore berbasis GORM.
type GormAdminStore struct {
	DB *gorm.DB
}

// NewGormAdminStore membuat store administratif presensi.
func NewGormAdminStore(db *gorm.DB) *GormAdminStore {
	return &GormAdminStore{DB: db}
}

func isUniqueViolation(err error) bool {
	var pgErr interface{ GetSQLState() string }
	if errors.As(err, &pgErr) {
		return pgErr.GetSQLState() == "23505"
	}
	msg := err.Error()
	return strings.Contains(msg, "23505") || strings.Contains(msg, "duplicate key")
}

func optionalScopeIDs(scopeIDs []string) bool { return scopeIDs != nil }

func (s *GormAdminStore) ListPins(ctx context.Context, scopeIDs []string) ([]PinView, error) {
	type row struct {
		ID         string    `gorm:"column:id"`
		MachinePin string    `gorm:"column:machinePin"`
		UserID     string    `gorm:"column:userId"`
		Nama       string    `gorm:"column:nama"`
		Aktif      bool      `gorm:"column:aktif"`
		Catatan    *string   `gorm:"column:catatan"`
		CreatedAt  time.Time `gorm:"column:createdAt"`
	}
	rows := []row{}
	query := s.DB.WithContext(ctx).
		Table(`"mesinAbsensiPin" AS p`).
		Select(`p."id", p."machinePin", p."userId", u."nama", p."aktif", p."catatan", p."createdAt"`).
		Joins(`JOIN "user" AS u ON u."id" = p."userId"`).
		Order(`p."createdAt" DESC, p."id" DESC`).
		Limit(500)
	if optionalScopeIDs(scopeIDs) {
		if len(scopeIDs) == 0 {
			return []PinView{}, nil
		}
		query = query.Where(`u."homebaseId" IN ?`, scopeIDs)
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list machine pins: %w", err)
	}
	views := make([]PinView, 0, len(rows))
	for _, row := range rows {
		views = append(views, PinView{
			ID:         row.ID,
			MachinePin: row.MachinePin,
			UserID:     row.UserID,
			Nama:       row.Nama,
			Aktif:      row.Aktif,
			Catatan:    row.Catatan,
			CreatedAt:  timeutil.FormatDateTime(row.CreatedAt),
		})
	}
	return views, nil
}

func (s *GormAdminStore) userHomebase(ctx context.Context, tx *gorm.DB, userID string) (*string, error) {
	var row struct {
		HomebaseID *string `gorm:"column:homebaseId"`
	}
	handle := s.DB.WithContext(ctx)
	if tx != nil {
		handle = tx
	}
	if err := handle.Table(`"user"`).Select(`"homebaseId"`).Where("id = ?", userID).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("pegawai tidak ditemukan")
		}
		return nil, fmt.Errorf("read pegawai homebase: %w", err)
	}
	return row.HomebaseID, nil
}

func (s *GormAdminStore) CreatePin(ctx context.Context, machinePin, userID string, catatan *string, actorID string) (models.MesinAbsensiPin, error) {
	pin := models.MesinAbsensiPin{MachinePin: machinePin, UserId: userID, Aktif: true, Catatan: catatan}
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		homebase, err := s.userHomebase(ctx, tx, userID)
		if err != nil {
			return err
		}
		if err := tx.Create(&pin).Error; err != nil {
			if isUniqueViolation(err) {
				return ErrPinDuplicate
			}
			return fmt.Errorf("insert machine pin: %w", err)
		}
		return audit.Write(ctx, tx, audit.Entry{
			ActorID:  actorID,
			Action:   audit.ActionPinCreated,
			TargetID: &pin.ID,
			UnitID:   homebase,
			Note:     ptr("pemetaan machinePin mesin absensi"),
		})
	})
	return pin, err
}

func (s *GormAdminStore) DeletePin(ctx context.Context, pinID string, scopeIDs []string, actorID string) error {
	var row struct {
		PinID    string  `gorm:"column:pin_id"`
		UserID   string  `gorm:"column:user_id"`
		Homebase *string `gorm:"column:homebase"`
	}
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Table(`"mesinAbsensiPin" AS p`).
			Select(`p."id" AS pin_id, p."userId" AS user_id, u."homebaseId" AS homebase`).
			Joins(`JOIN "user" AS u ON u."id" = p."userId"`).
			Where(`p."id" = ?`, pinID).
			Take(&row).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPinNotFound
		}
		if err != nil {
			return fmt.Errorf("read machine pin: %w", err)
		}
		if optionalScopeIDs(scopeIDs) {
			found := false
			if row.Homebase != nil {
				for _, id := range scopeIDs {
					if id == *row.Homebase {
						found = true
						break
					}
				}
			}
			if !found {
				return ErrPinNotFound
			}
		}
		if err := tx.Where("id = ?", pinID).Delete(&models.MesinAbsensiPin{}).Error; err != nil {
			return fmt.Errorf("delete machine pin: %w", err)
		}
		return audit.Write(ctx, tx, audit.Entry{
			ActorID:  actorID,
			Action:   audit.ActionPinDeleted,
			TargetID: &row.PinID,
			UnitID:   row.Homebase,
			Note:     ptr("pemetaan machinePin mesin absensi"),
		})
	})
}

func (s *GormAdminStore) UserIdentityOf(ctx context.Context, userID string) (models.User, error) {
	var row struct {
		ID         string  `gorm:"column:id"`
		Nama       string  `gorm:"column:nama"`
		HomebaseID *string `gorm:"column:homebaseId"`
	}
	if err := s.DB.WithContext(ctx).Table(`"user"`).Select(`"id", "nama", "homebaseId"`).Where("id = ?", userID).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.User{}, errors.New("pegawai tidak ditemukan")
		}
		return models.User{}, fmt.Errorf("read pegawai: %w", err)
	}
	return models.User{IDModel: models.IDModel{ID: row.ID}, Nama: row.Nama, HomebaseId: row.HomebaseID}, nil
}

func (s *GormAdminStore) UserByEmail(ctx context.Context, email string) (models.User, error) {
	var row struct {
		ID         string  `gorm:"column:id"`
		Nama       string  `gorm:"column:nama"`
		HomebaseID *string `gorm:"column:homebaseId"`
	}
	if err := s.DB.WithContext(ctx).Table(`"user"`).Select(`"id", "nama", "homebaseId"`).Where(`lower("email") = lower(?)`, email).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.User{}, errors.New("pegawai tidak ditemukan")
		}
		return models.User{}, fmt.Errorf("read pegawai by email: %w", err)
	}
	return models.User{IDModel: models.IDModel{ID: row.ID}, Nama: row.Nama, HomebaseId: row.HomebaseID}, nil
}
