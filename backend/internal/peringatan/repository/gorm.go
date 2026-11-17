package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"simpeg/backend/internal/audit"
	"simpeg/backend/internal/models"
)

// GormStore mengimplementasikan Store dengan PostgreSQL.
type GormStore struct {
	DB *gorm.DB
}

// auditWrite menulis satu entri jejak audit dalam transaksi.
func auditWrite(ctx context.Context, tx *gorm.DB, entry audit.Entry) error {
	return audit.Write(ctx, tx, entry)
}

// ErrNotFound menandai peringatan yang tidak ditemukan.
var ErrNotFound = errors.New("peringatan not found")

// EmployeeHomebaseID mengembalikan unit penugasan pegawai; nil bila belum diisi.
func (s GormStore) EmployeeHomebaseID(ctx context.Context, userID string) (*string, error) {
	var row struct {
		HomebaseID *string `gorm:"column:homebaseId"`
	}
	if err := s.DB.WithContext(ctx).Table(`"user"`).Select(`"homebaseId"`).Where(`"id" = ?`, userID).Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("read homebase pegawai: %w", err)
	}
	return row.HomebaseID, nil
}

// EmployeeExists melaporkan apakah pegawai dikenal.
func (s GormStore) EmployeeExists(ctx context.Context, userID string) (bool, error) {
	var count int64
	if err := s.DB.WithContext(ctx).Table(`"user"`).Where(`"id" = ?`, userID).Count(&count).Error; err != nil {
		return false, fmt.Errorf("cek pegawai: %w", err)
	}
	return count > 0, nil
}

// Create menulis peringatan SP1 baru dalam satu transaksi + jejak audit.
func (s GormStore) Create(ctx context.Context, actor string, input CreateInput, unitID *string) (models.Peringatan, error) {
	item := models.Peringatan{
		UserId:        input.UserID,
		Tingkat:       input.Tingkat,
		Alasan:        strings.TrimSpace(input.Alasan),
		NomorSurat:    input.NomorSurat,
		Tanggal:       input.Tanggal,
		Penerbit:      strings.TrimSpace(input.Penerbit),
		Sanksi:        input.Sanksi,
		SanksiCatatan: input.SanksiCatatan,
		CreatedBy:     &actor,
	}
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&item).Error; err != nil {
			return fmt.Errorf("create peringatan: %w", err)
		}
		targetID := item.ID
		note := "SP1 diterbitkan: " + string(input.Tingkat) + " untuk pegawai " + input.UserID
		return auditWrite(ctx, tx, audit.Entry{
			ActorID:  actor,
			Action:   audit.ActionPeringatanCreated,
			TargetID: &targetID,
			UnitID:   unitID,
			Note:     &note,
		})
	})
	return item, err
}

// Get membaca satu peringatan.
func (s GormStore) Get(ctx context.Context, id string) (models.Peringatan, error) {
	var item models.Peringatan
	if err := s.DB.WithContext(ctx).Where("id = ?", id).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Peringatan{}, ErrNotFound
		}
		return models.Peringatan{}, fmt.Errorf("read peringatan: %w", err)
	}
	return item, nil
}

// List membaca peringatan sesuai filter dan cakupan.
func (s GormStore) List(ctx context.Context, options ListOptions) ([]models.Peringatan, int64, error) {
	query := s.DB.WithContext(ctx).Model(&models.Peringatan{})
	if options.UserID != "" {
		query = query.Where(`"userId" = ?`, options.UserID)
	}
	if options.Tingkat != "" {
		query = query.Where(`"tingkat" = ?`, options.Tingkat)
	}
	if len(options.HomebaseIDs) > 0 {
		query = query.Where(`"userId" IN (SELECT "id" FROM "user" WHERE "homebaseId" IN ?)`, options.HomebaseIDs)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count peringatan: %w", err)
	}
	if options.Limit == 0 {
		options.Limit = 50
	}
	var items []models.Peringatan
	if err := query.Order(`"createdAt" DESC`).Limit(options.Limit).Offset(options.Offset).Find(&items).Error; err != nil {
		return nil, 0, fmt.Errorf("list peringatan: %w", err)
	}
	return items, total, nil
}
