// Package repository menyimpan honor mengajar per SKS (dosen LB) lewat GORM.
package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"simpeg/backend/internal/audit"
	honordomain "simpeg/backend/internal/honorsks/domain"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GormStore adalah adapter persistensi honor mengajar per SKS.
type GormStore struct {
	DB *gorm.DB
}

// NewGormStore membuat adapter persistensi honor mengajar per SKS.
func NewGormStore(db *gorm.DB) GormStore { return GormStore{DB: db} }

// ListRates membaca seluruh tarif honor per SKS.
func (s GormStore) ListRates(ctx context.Context) ([]models.HonorSksRate, error) {
	var rows []models.HonorSksRate
	if err := s.DB.WithContext(ctx).Order(`"berlakuMulai" DESC`).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list honor sks rate: %w", err)
	}
	return rows, nil
}

// GetRate membaca satu tarif honor per SKS.
func (s GormStore) GetRate(ctx context.Context, id string) (models.HonorSksRate, error) {
	var row models.HonorSksRate
	if err := s.DB.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return models.HonorSksRate{}, classifyRecordError(err)
	}
	return row, nil
}

// LatestRateOn mengembalikan tarif berlaku terakhir pada tanggal tertentu.
func (s GormStore) LatestRateOn(ctx context.Context, asOf string) (models.HonorSksRate, error) {
	var row models.HonorSksRate
	day, err := time.Parse("2006-01-02", strings.TrimSpace(asOf))
	if err != nil {
		return models.HonorSksRate{}, honordomain.ErrValidation
	}
	if err := s.DB.WithContext(ctx).
		Where(`"berlakuMulai" <= ? AND "aktif" = ?`, day, true).
		Order(`"berlakuMulai" DESC`).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.HonorSksRate{}, honordomain.ErrNotFound
		}
		return models.HonorSksRate{}, fmt.Errorf("latest honor sks rate: %w", err)
	}
	return row, nil
}

// CreateRate menyimpan tarif honor per SKS baru dan mencatat jejak audit.
func (s GormStore) CreateRate(ctx context.Context, actorID string, row models.HonorSksRate) (models.HonorSksRate, error) {
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return classifyRecordError(err)
		}
		return audit.Write(ctx, tx, audit.Entry{
			ActorID:  actorID,
			Action:   audit.ActionHonorRateCreated,
			TargetID: optionalRef(row.ID),
			Note:     optionalNote("honor.rate.created"),
		})
	})
	return row, err
}

// UpdateRate menyimpan perubahan tarif honor per SKS dan mencatat jejak audit.
func (s GormStore) UpdateRate(ctx context.Context, actorID, id string, updates map[string]any) (models.HonorSksRate, error) {
	var row models.HonorSksRate
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&row).Error; err != nil {
			return classifyRecordError(err)
		}
		updates["updatedAt"] = time.Now().UTC()
		if err := tx.Model(&row).Updates(updates).Error; err != nil {
			return classifyRecordError(err)
		}
		if err := tx.First(&row, "id = ?", id).Error; err != nil {
			return fmt.Errorf("reload honor rate: %w", err)
		}
		return audit.Write(ctx, tx, audit.Entry{
			ActorID:  actorID,
			Action:   audit.ActionHonorRateUpdated,
			TargetID: optionalRef(id),
			Note:     optionalNote("honor.rate.updated fields=" + strings.Join(sortedKeys(updates), ",")),
		})
	})
	return row, err
}

// ListPeriode membaca seluruh baris beban mengajar sesuai filter dan pagination.
func (s GormStore) ListPeriode(ctx context.Context, options listing.Options) ([]models.HonorSksPeriode, int64, error) {
	query := s.DB.WithContext(ctx).Model(&models.HonorSksPeriode{})
	if options.UserID != "" {
		query = query.Where(`"userId" = ?`, options.UserID)
	}
	if len(options.ScopeHomebaseIDs) > 0 {
		query = query.Where(`"user"."homebaseId" IN ?`, options.ScopeHomebaseIDs).
			Joins(`JOIN "user" ON "user"."id" = "honorSksPeriode"."userId"`)
	}
	if options.Periode != "" {
		query = query.Where(`"periodeAkademikId" = ?`, options.Periode)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count honor periode: %w", err)
	}
	var rows []models.HonorSksPeriode
	if err := query.Order(`"userId" ASC, "periodeAkademikId" ASC`).
		Offset((options.Page - 1) * options.PageSize).
		Limit(options.PageSize).
		Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("list honor periode: %w", err)
	}
	return rows, total, nil
}

// GetPeriode membaca satu baris beban mengajar.
func (s GormStore) GetPeriode(ctx context.Context, id string) (models.HonorSksPeriode, error) {
	var row models.HonorSksPeriode
	if err := s.DB.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return models.HonorSksPeriode{}, classifyRecordError(err)
	}
	return row, nil
}

// CreatePeriode menyimpan beban mengajar baru dan mencatat jejak audit.
func (s GormStore) CreatePeriode(ctx context.Context, actorID string, row models.HonorSksPeriode) (models.HonorSksPeriode, error) {
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var periode models.PeriodeAkademik
		if err := tx.Where("id = ?", row.PeriodeAkademikId).First(&periode).Error; err != nil {
			return classifyRecordError(err)
		}
		if err := tx.Create(&row).Error; err != nil {
			return classifyRecordError(err)
		}
		return audit.Write(ctx, tx, audit.Entry{
			ActorID:  actorID,
			Action:   audit.ActionHonorPeriodeAssigned,
			TargetID: optionalRef(row.ID),
			Note:     optionalNote("honor.periode.assigned"),
		})
	})
	return row, err
}

// UpdatePeriode menyimpan perubahan beban mengajar dan mencatat jejak audit.
func (s GormStore) UpdatePeriode(ctx context.Context, actorID, id string, updates map[string]any) (models.HonorSksPeriode, error) {
	var row models.HonorSksPeriode
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&row).Error; err != nil {
			return classifyRecordError(err)
		}
		updates["updatedAt"] = time.Now().UTC()
		if totalSks, ok := updates["sks"]; ok {
			rate := row.RatePerSks
			if v, ok := updates["ratePerSks"]; ok {
				rate = v.(float64)
			}
			updates["totalHonor"] = totalSks.(float64) * rate
		} else if rate, ok := updates["ratePerSks"]; ok {
			updates["totalHonor"] = row.Sks * rate.(float64)
		}
		if err := tx.Model(&row).Updates(updates).Error; err != nil {
			return classifyRecordError(err)
		}
		if err := tx.First(&row, "id = ?", id).Error; err != nil {
			return fmt.Errorf("reload honor periode: %w", err)
		}
		return audit.Write(ctx, tx, audit.Entry{
			ActorID:  actorID,
			Action:   audit.ActionHonorPeriodeUpdated,
			TargetID: optionalRef(id),
			Note:     optionalNote("honor.periode.updated fields=" + strings.Join(sortedKeys(updates), ",")),
		})
	})
	return row, err
}

// DeletePeriode menghapus beban mengajar dan mencatat jejak audit.
func (s GormStore) DeletePeriode(ctx context.Context, actorID, id string) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row models.HonorSksPeriode
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&row).Error; err != nil {
			return classifyRecordError(err)
		}
		if err := tx.Delete(&row).Error; err != nil {
			return fmt.Errorf("delete honor periode: %w", err)
		}
		return audit.Write(ctx, tx, audit.Entry{
			ActorID:  actorID,
			Action:   audit.ActionHonorPeriodeRemoved,
			TargetID: optionalRef(id),
			Note:     optionalNote("honor.periode.removed"),
		})
	})
}

// classifyRecordError memetakan error GORM/PostgreSQL ke error domain.
func classifyRecordError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, gorm.ErrRecordNotFound):
		return honordomain.ErrNotFound
	case errors.Is(err, gorm.ErrDuplicatedKey):
		return honordomain.ErrConflict
	case isUniqueViolation(err):
		return honordomain.ErrConflict
	default:
		return fmt.Errorf("database operation failed: %w", err)
	}
}

func isUniqueViolation(err error) bool {
	var pgErr interface{ GetSQLState() string }
	if errors.As(err, &pgErr) {
		return pgErr.GetSQLState() == "23505"
	}
	msg := err.Error()
	return strings.Contains(msg, "23505") || strings.Contains(msg, "duplicate key")
}

func optionalRef(id string) *string {
	ref := id
	return &ref
}

func optionalNote(note string) *string {
	return &note
}

func sortedKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}
