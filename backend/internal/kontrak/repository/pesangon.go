// Aturan pesangonAturan: CRUD master data faktor kompensasi (admin input).
package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"simpeg/backend/internal/models"
)

// PesangonStore adalah port persistensi pesangonAturan.
type PesangonStore interface {
	ListRules(ctx context.Context, kategoriStatusID string) ([]models.PesangonAturan, error)
	CreateRule(ctx context.Context, rule models.PesangonAturan) (models.PesangonAturan, error)
	UpdateRule(ctx context.Context, id string, rule models.PesangonAturan) (models.PesangonAturan, error)
	DeleteRule(ctx context.Context, id string) error
}

// PesangonGormStore mengimplementasikan PesangonStore dengan PostgreSQL.
type PesangonGormStore struct {
	DB *gorm.DB
}

// ListRules membaca aturan per kategori status.
func (s PesangonGormStore) ListRules(ctx context.Context, kategoriStatusID string) ([]models.PesangonAturan, error) {
	query := s.DB.WithContext(ctx).Model(&models.PesangonAturan{})
	if kategoriStatusID != "" {
		query = query.Where(`"kategoriStatusId" = ?`, kategoriStatusID)
	}
	var rows []models.PesangonAturan
	if err := query.Order(`"masaKerjaTahun" ASC`).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list pesangonAturan: %w", err)
	}
	return rows, nil
}

// CreateRule menulis atau mengganti baris (kategoriStatusId, masaKerjaTahun).
func (s PesangonGormStore) CreateRule(ctx context.Context, rule models.PesangonAturan) (models.PesangonAturan, error) {
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing models.PesangonAturan
		err := tx.Where(`"kategoriStatusId" = ? AND "masaKerjaTahun" = ?`, rule.KategoriStatusId, rule.MasaKerjaTahun).First(&existing).Error
		switch {
		case err == nil:
			existing.PesangonBulan = rule.PesangonBulan
			existing.PenghargaanBulan = rule.PenghargaanBulan
			existing.GantiKerugianBulan = rule.GantiKerugianBulan
			existing.SumberRegulasi = rule.SumberRegulasi
			if err := tx.Save(&existing).Error; err != nil {
				return fmt.Errorf("update pesangonAturan: %w", err)
			}
			rule = existing
			return nil
		case errors.Is(err, gorm.ErrRecordNotFound):
			return tx.Create(&rule).Error
		default:
			return err
		}
	})
	return rule, err
}

// UpdateRule mengganti isi baris pesangonAturan.
func (s PesangonGormStore) UpdateRule(ctx context.Context, id string, rule models.PesangonAturan) (models.PesangonAturan, error) {
	var existing models.PesangonAturan
	if err := s.DB.WithContext(ctx).Where("id = ?", id).First(&existing).Error; err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			return models.PesangonAturan{}, ErrNotFoundPesangon
		default:
			return models.PesangonAturan{}, fmt.Errorf("read pesangonAturan: %w", err)
		}
	}
	existing.PesangonBulan = rule.PesangonBulan
	existing.PenghargaanBulan = rule.PenghargaanBulan
	existing.GantiKerugianBulan = rule.GantiKerugianBulan
	existing.SumberRegulasi = rule.SumberRegulasi
	if err := s.DB.WithContext(ctx).Save(&existing).Error; err != nil {
		return models.PesangonAturan{}, fmt.Errorf("update pesangonAturan: %w", err)
	}
	return existing, nil
}

// DeleteRule menghapus baris pesangonAturan.
func (s PesangonGormStore) DeleteRule(ctx context.Context, id string) error {
	result := s.DB.WithContext(ctx).Where("id = ?", id).Delete(&models.PesangonAturan{})
	if result.Error != nil {
		return fmt.Errorf("delete pesangonAturan: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFoundPesangon
	}
	return nil
}

// ErrNotFoundPesangon menandai aturan yang tidak ditemukan.
var ErrNotFoundPesangon = errors.New("aturan pesangon tidak ditemukan")
