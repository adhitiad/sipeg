// Package repository menyimpan tarif iuran BPJS dan kepesertaannya lewat GORM.
package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"simpeg/backend/internal/audit"
	bpjsdomain "simpeg/backend/internal/bpjs/domain"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GormStore adalah adapter persistensi BPJS.
type GormStore struct {
	DB *gorm.DB
}

// NewGormStore membuat adapter persistensi BPJS.
func NewGormStore(db *gorm.DB) GormStore { return GormStore{DB: db} }

// ListKonfigurasi membaca seluruh tarif iuran BPJS.
func (s GormStore) ListKonfigurasi(ctx context.Context) ([]models.BpjsKonfigurasi, error) {
	var rows []models.BpjsKonfigurasi
	if err := s.DB.WithContext(ctx).Order(`"berlakuMulai" DESC, "jenis" ASC`).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list bpjs konfigurasi: %w", err)
	}
	return rows, nil
}

// GetKonfigurasi membaca satu tarif iuran BPJS.
func (s GormStore) GetKonfigurasi(ctx context.Context, id string) (models.BpjsKonfigurasi, error) {
	var row models.BpjsKonfigurasi
	if err := s.DB.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return models.BpjsKonfigurasi{}, classifyRecordError(err)
	}
	return row, nil
}

// CreateKonfigurasi menyimpan tarif iuran BPJS baru dan mencatat jejak audit.
func (s GormStore) CreateKonfigurasi(ctx context.Context, actorID string, row models.BpjsKonfigurasi) (models.BpjsKonfigurasi, error) {
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return classifyRecordError(err)
		}
		return audit.Write(ctx, tx, audit.Entry{
			ActorID:  actorID,
			Action:   audit.ActionBpjsKonfigurasiCreated,
			TargetID: optionalRef(row.ID),
			Note:     optionalNote("bpjs.konfigurasi.created"),
		})
	})
	return row, err
}

// UpdateKonfigurasi menyimpan perubahan tarif iuran BPJS dan mencatat jejak audit.
func (s GormStore) UpdateKonfigurasi(ctx context.Context, actorID, id string, updates map[string]any) (models.BpjsKonfigurasi, error) {
	var row models.BpjsKonfigurasi
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&row).Error; err != nil {
			return classifyRecordError(err)
		}
		updates["updatedAt"] = time.Now().UTC()
		if err := tx.Model(&row).Updates(updates).Error; err != nil {
			return classifyRecordError(err)
		}
		if err := tx.First(&row, "id = ?", id).Error; err != nil {
			return fmt.Errorf("reload bpjs konfigurasi: %w", err)
		}
		return audit.Write(ctx, tx, audit.Entry{
			ActorID:  actorID,
			Action:   audit.ActionBpjsKonfigurasiUpdated,
			TargetID: optionalRef(id),
			Note:     optionalNote("bpjs.konfigurasi.updated fields=" + strings.Join(sortedKeys(updates), ",")),
		})
	})
	return row, err
}

// ListPeserta membaca seluruh baris kepesertaan BPJS sesuai filter dan pagination.
func (s GormStore) ListPeserta(ctx context.Context, options listing.Options) ([]models.BpjsPeserta, int64, error) {
	query := s.DB.WithContext(ctx).Model(&models.BpjsPeserta{})
	if options.UserID != "" {
		query = query.Where(`"userId" = ?`, options.UserID)
	}
	if len(options.ScopeHomebaseIDs) > 0 {
		query = query.Where(`"user"."homebaseId" IN ?`, options.ScopeHomebaseIDs).
			Joins(`JOIN "user" ON "user"."id" = "bpjsPeserta"."userId"`)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count bpjs peserta: %w", err)
	}
	var rows []models.BpjsPeserta
	if err := query.Order(`"jenis" ASC, "userId" ASC`).
		Offset((options.Page - 1) * options.PageSize).
		Limit(options.PageSize).
		Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("list bpjs peserta: %w", err)
	}
	return rows, total, nil
}

// GetPeserta membaca satu baris kepesertaan BPJS.
func (s GormStore) GetPeserta(ctx context.Context, id string) (models.BpjsPeserta, error) {
	var row models.BpjsPeserta
	if err := s.DB.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return models.BpjsPeserta{}, classifyRecordError(err)
	}
	return row, nil
}

// CreatePeserta menyimpan kepesertaan BPJS baru dan mencatat jejak audit.
func (s GormStore) CreatePeserta(ctx context.Context, actorID string, row models.BpjsPeserta) (models.BpjsPeserta, error) {
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return classifyRecordError(err)
		}
		return audit.Write(ctx, tx, audit.Entry{
			ActorID:  actorID,
			Action:   audit.ActionBpjsPesertaCreated,
			TargetID: optionalRef(row.ID),
			Note:     optionalNote("bpjs.peserta.created"),
		})
	})
	return row, err
}

// UpdatePeserta menyimpan perubahan kepesertaan BPJS dan mencatat jejak audit.
func (s GormStore) UpdatePeserta(ctx context.Context, actorID, id string, updates map[string]any) (models.BpjsPeserta, error) {
	var row models.BpjsPeserta
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&row).Error; err != nil {
			return classifyRecordError(err)
		}
		updates["updatedAt"] = time.Now().UTC()
		if err := tx.Model(&row).Updates(updates).Error; err != nil {
			return classifyRecordError(err)
		}
		if err := tx.First(&row, "id = ?", id).Error; err != nil {
			return fmt.Errorf("reload bpjs peserta: %w", err)
		}
		return audit.Write(ctx, tx, audit.Entry{
			ActorID:  actorID,
			Action:   audit.ActionBpjsPesertaUpdated,
			TargetID: optionalRef(id),
			Note:     optionalNote("bpjs.peserta.updated fields=" + strings.Join(sortedKeys(updates), ",")),
		})
	})
	return row, err
}

// classifyRecordError memetakan error GORM/PostgreSQL ke error domain.
func classifyRecordError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, gorm.ErrRecordNotFound):
		return bpjsdomain.ErrNotFound
	case errors.Is(err, gorm.ErrDuplicatedKey):
		return bpjsdomain.ErrConflict
	case isUniqueViolation(err):
		return bpjsdomain.ErrConflict
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
