// Package repository menyimpan master skala gaji pokok yayasan lewat GORM.
// Perubahan dan jejak audit ditulis dalam satu transaksi.
package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"simpeg/backend/internal/audit"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
	"simpeg/backend/internal/payscale/domain"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GormStore adalah adapter persistensi skala gaji pokok yayasan.
type GormStore struct {
	DB *gorm.DB
}

// NewGormStore membuat adapter persistensi skala gaji pokok yayasan.
func NewGormStore(db *gorm.DB) GormStore { return GormStore{DB: db} }

// ListSkala membaca daftar skala gaji sesuai filter dan pagination.
func (s GormStore) ListSkala(ctx context.Context, options listing.Options) ([]models.SkalaGaji, int64, error) {
	query := s.DB.WithContext(ctx).Model(&models.SkalaGaji{})
	if options.Search != "" {
		pattern := "%" + escapeLike(options.Search) + "%"
		query = query.Where("kode ILIKE ? ESCAPE '\\' OR nama ILIKE ? ESCAPE '\\'", pattern, pattern)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count skala gaji: %w", err)
	}
	var rows []models.SkalaGaji
	order := listing.OrderClause(domain.SkalaSortColumns, options.Sort, options.Order, "kode")
	err := query.Order(order).Offset((options.Page - 1) * options.PageSize).Limit(options.PageSize).Find(&rows).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list skala gaji: %w", err)
	}
	return rows, total, nil
}

// GetSkala membaca satu skala gaji (aktif maupun tidak).
func (s GormStore) GetSkala(ctx context.Context, id string) (models.SkalaGaji, error) {
	var skala models.SkalaGaji
	if err := s.DB.WithContext(ctx).Where("id = ?", id).First(&skala).Error; err != nil {
		return models.SkalaGaji{}, classifyRecordError(err)
	}
	return skala, nil
}

// GetSkalaWithDetails membaca skala gaji beserta seluruh baris nominalnya
// (terurut berdasarkan tingkat lalu kategori).
func (s GormStore) GetSkalaWithDetails(ctx context.Context, id string) (models.SkalaGaji, []models.SkalaGajiDetail, error) {
	var skala models.SkalaGaji
	if err := s.DB.WithContext(ctx).Where("id = ?", id).First(&skala).Error; err != nil {
		return models.SkalaGaji{}, nil, classifyRecordError(err)
	}
	var details []models.SkalaGajiDetail
	if err := s.DB.WithContext(ctx).
		Where("\"skalaGajiId\" = ?", id).
		Order(`"tingkat" ASC, "kategoriStatusId" ASC`).
		Find(&details).Error; err != nil {
		return models.SkalaGaji{}, nil, fmt.Errorf("list detail skala: %w", err)
	}
	return skala, details, nil
}

// CreateSkala menyimpan skala gaji baru dan mencatat jejak audit.
func (s GormStore) CreateSkala(ctx context.Context, actorID string, skala models.SkalaGaji) (models.SkalaGaji, error) {
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&skala).Error; err != nil {
			return classifyRecordError(err)
		}
		return audit.Write(ctx, tx, audit.Entry{
			ActorID:  actorID,
			Action:   audit.ActionSkalaCreated,
			TargetID: optionalRef(skala.ID),
			Note:     optionalNote("skala.created"),
		})
	})
	return skala, err
}

// UpdateSkala menyimpan perubahan skala gaji dan mencatat jejak audit.
func (s GormStore) UpdateSkala(ctx context.Context, actorID, id string, updates map[string]any) (models.SkalaGaji, error) {
	var skala models.SkalaGaji
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&skala).Error; err != nil {
			return classifyRecordError(err)
		}
		updates["updatedAt"] = time.Now().UTC()
		if err := tx.Model(&skala).Updates(updates).Error; err != nil {
			return classifyRecordError(err)
		}
		if err := tx.First(&skala, "id = ?", id).Error; err != nil {
			return fmt.Errorf("reload skala: %w", err)
		}
		return audit.Write(ctx, tx, audit.Entry{
			ActorID:  actorID,
			Action:   audit.ActionSkalaUpdated,
			TargetID: optionalRef(id),
			Note:     optionalNote("skala.updated fields=" + strings.Join(sortedKeys(updates), ",")),
		})
	})
	return skala, err
}

// GetDetail membaca satu baris nominal.
func (s GormStore) GetDetail(ctx context.Context, id string) (models.SkalaGajiDetail, error) {
	var detail models.SkalaGajiDetail
	if err := s.DB.WithContext(ctx).Where("id = ?", id).First(&detail).Error; err != nil {
		return models.SkalaGajiDetail{}, classifyRecordError(err)
	}
	return detail, nil
}

// ListDetails membaca seluruh baris nominal pada satu skala gaji.
func (s GormStore) ListDetails(ctx context.Context, skalaID string) ([]models.SkalaGajiDetail, error) {
	var details []models.SkalaGajiDetail
	if err := s.DB.WithContext(ctx).
		Where("\"skalaGajiId\" = ?", skalaID).
		Order(`"tingkat" ASC, "kategoriStatusId" ASC`).
		Find(&details).Error; err != nil {
		return nil, fmt.Errorf("list detail skala: %w", err)
	}
	return details, nil
}

// CreateDetail menyimpan baris nominal baru dan mencatat jejak audit.
func (s GormStore) CreateDetail(ctx context.Context, actorID string, detail models.SkalaGajiDetail) (models.SkalaGajiDetail, error) {
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var skala models.SkalaGaji
		if err := tx.Where("id = ?", detail.SkalaGajiId).First(&skala).Error; err != nil {
			return classifyRecordError(err)
		}
		var kategori models.KategoriStatusPegawai
		if err := tx.Where("id = ?", detail.KategoriStatusId).First(&kategori).Error; err != nil {
			return classifyRecordError(err)
		}
		if err := tx.Create(&detail).Error; err != nil {
			return classifyRecordError(err)
		}
		return audit.Write(ctx, tx, audit.Entry{
			ActorID:  actorID,
			Action:   audit.ActionSkalaDetailCreated,
			TargetID: optionalRef(detail.ID),
			Note:     optionalNote("skala.detail.created"),
		})
	})
	return detail, err
}

// UpdateDetail menyimpan perubahan baris nominal dan mencatat jejak audit.
func (s GormStore) UpdateDetail(ctx context.Context, actorID, id string, updates map[string]any) (models.SkalaGajiDetail, error) {
	var detail models.SkalaGajiDetail
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&detail).Error; err != nil {
			return classifyRecordError(err)
		}
		if _, ok := updates["kategoriStatusId"]; ok {
			var kategori models.KategoriStatusPegawai
			if err := tx.Where("id = ?", updates["kategoriStatusId"]).First(&kategori).Error; err != nil {
				return classifyRecordError(err)
			}
		}
		updates["updatedAt"] = time.Now().UTC()
		if err := tx.Model(&detail).Updates(updates).Error; err != nil {
			return classifyRecordError(err)
		}
		if err := tx.First(&detail, "id = ?", id).Error; err != nil {
			return fmt.Errorf("reload detail: %w", err)
		}
		return audit.Write(ctx, tx, audit.Entry{
			ActorID:  actorID,
			Action:   audit.ActionSkalaDetailUpdated,
			TargetID: optionalRef(id),
			Note:     optionalNote("skala.detail.updated fields=" + strings.Join(sortedKeys(updates), ",")),
		})
	})
	return detail, err
}

// DeleteDetail menghapus baris nominal dan mencatat jejak audit.
func (s GormStore) DeleteDetail(ctx context.Context, actorID, id string) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var detail models.SkalaGajiDetail
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&detail).Error; err != nil {
			return classifyRecordError(err)
		}
		if err := tx.Delete(&detail).Error; err != nil {
			return fmt.Errorf("delete detail skala: %w", err)
		}
		return audit.Write(ctx, tx, audit.Entry{
			ActorID:  actorID,
			Action:   audit.ActionSkalaDetailDeleted,
			TargetID: optionalRef(id),
			Note:     optionalNote("skala.detail.deleted"),
		})
	})
}

// ListKategoriStatus membaca seluruh taxonomy kategori status pegawai.
func (s GormStore) ListKategoriStatus(ctx context.Context) ([]models.KategoriStatusPegawai, error) {
	var rows []models.KategoriStatusPegawai
	if err := s.DB.WithContext(ctx).Order(`"kode" ASC`).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list kategori status pegawai: %w", err)
	}
	return rows, nil
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

// isUniqueViolation melaporkan apakah error adalah pelanggaran unik
// PostgreSQL (SQLSTATE 23505), dengan membungkus error GORM bila perlu.
func isUniqueViolation(err error) bool {
	var pgErr interface{ GetSQLState() string }
	if errors.As(err, &pgErr) {
		return pgErr.GetSQLState() == "23505"
	}
	msg := err.Error()
	return strings.Contains(msg, "23505") || strings.Contains(msg, "duplicate key")
}

func escapeLike(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(value)
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
