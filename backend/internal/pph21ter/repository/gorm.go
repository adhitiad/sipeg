// Package repository menyimpan tabel tarif efektif PPh 21 TER lewat GORM.
package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"simpeg/backend/internal/audit"
	"simpeg/backend/internal/models"
	"simpeg/backend/internal/pph21ter/domain"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GormStore adalah adapter persistensi tabel tarif efektif PPh 21 TER.
type GormStore struct {
	DB *gorm.DB
}

// NewGormStore membuat adapter persistensi tabel TER.
func NewGormStore(db *gorm.DB) GormStore { return GormStore{DB: db} }

// ListTer membaca seluruh baris tarif TER.
func (s GormStore) ListTer(ctx context.Context) ([]models.Pph21TerTarif, error) {
	var rows []models.Pph21TerTarif
	if err := s.DB.WithContext(ctx).Order(`"berlakuMulai" DESC, "lapisan" ASC`).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list pph21 ter: %w", err)
	}
	return rows, nil
}

// ListTerOn mengembalikan baris tarif TER yang berlaku pada tanggal tertentu.
func (s GormStore) ListTerOn(ctx context.Context, asOf string) ([]models.Pph21TerTarif, error) {
	day, err := time.Parse("2006-01-02", strings.TrimSpace(asOf))
	if err != nil {
		return nil, domain.ErrValidation
	}
	var latest time.Time
	if err := s.DB.WithContext(ctx).Model(&models.Pph21TerTarif{}).
		Select(`MAX("berlakuMulai")`).
		Where(`"berlakuMulai" <= ?`, day).
		Scan(&latest).Error; err != nil {
		return nil, fmt.Errorf("latest pph21 ter: %w", err)
	}
	if latest.IsZero() {
		return []models.Pph21TerTarif{}, nil
	}
	var rows []models.Pph21TerTarif
	if err := s.DB.WithContext(ctx).Where(`"berlakuMulai" = ?`, latest).
		Order(`"lapisan" ASC`).
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list pph21 ter on %s: %w", latest.Format("2006-01-02"), err)
	}
	return rows, nil
}

// CreateTer menyimpan baris tarif TER baru dan mencatat jejak audit.
func (s GormStore) CreateTer(ctx context.Context, actorID string, row models.Pph21TerTarif) (models.Pph21TerTarif, error) {
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return classifyRecordError(err)
		}
		return audit.Write(ctx, tx, audit.Entry{
			ActorID:  actorID,
			Action:   audit.ActionPPh21TerCreated,
			TargetID: optionalRef(row.ID),
			Note:     optionalNote("pph21.ter.created"),
		})
	})
	return row, err
}

// UpdateTer menyimpan perubahan baris tarif TER dan mencatat jejak audit.
func (s GormStore) UpdateTer(ctx context.Context, actorID, id string, updates map[string]any) (models.Pph21TerTarif, error) {
	var row models.Pph21TerTarif
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&row).Error; err != nil {
			return classifyRecordError(err)
		}
		updates["updatedAt"] = time.Now().UTC()
		if err := tx.Model(&row).Updates(updates).Error; err != nil {
			return classifyRecordError(err)
		}
		if err := tx.First(&row, "id = ?", id).Error; err != nil {
			return fmt.Errorf("reload pph21 ter: %w", err)
		}
		return audit.Write(ctx, tx, audit.Entry{
			ActorID:  actorID,
			Action:   audit.ActionPPh21TerUpdated,
			TargetID: optionalRef(id),
			Note:     optionalNote("pph21.ter.updated fields=" + strings.Join(sortedKeys(updates), ",")),
		})
	})
	return row, err
}

// DeleteTer menghapus satu baris tarif TER dan mencatat jejak audit.
func (s GormStore) DeleteTer(ctx context.Context, actorID, id string) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row models.Pph21TerTarif
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&row).Error; err != nil {
			return classifyRecordError(err)
		}
		if err := tx.Delete(&row).Error; err != nil {
			return fmt.Errorf("delete pph21 ter: %w", err)
		}
		return audit.Write(ctx, tx, audit.Entry{
			ActorID:  actorID,
			Action:   audit.ActionPPh21TerDeleted,
			TargetID: optionalRef(id),
			Note:     optionalNote("pph21.ter.deleted"),
		})
	})
}

// classifyRecordError memetakan error GORM/PostgreSQL ke error domain.
func classifyRecordError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, gorm.ErrRecordNotFound):
		return domain.ErrNotFound
	case errors.Is(err, gorm.ErrDuplicatedKey):
		return domain.ErrConflict
	case isUniqueViolation(err):
		return domain.ErrConflict
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
