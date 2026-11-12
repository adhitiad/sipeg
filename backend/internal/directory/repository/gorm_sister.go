package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"simpeg/backend/internal/audit"
	"simpeg/backend/internal/directory/domain"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s GormStore) ListSister(ctx context.Context, options listing.Options) ([]models.SisterSyncMap, int64, error) {
	query := s.DB.WithContext(ctx).Model(&models.SisterSyncMap{})
	if options.UserID != "" {
		query = query.Where("userId = ?", options.UserID)
	}
	if len(options.ScopeHomebaseIDs) > 0 {
		query = query.Where("userId IN (SELECT id FROM user WHERE homebaseId IN ?)", options.ScopeHomebaseIDs)
	}
	if options.Search != "" {
		pattern := "%" + escapeLike(options.Search) + "%"
		query = query.Where("idEksternal ILIKE ? ESCAPE '\\'", pattern)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count sister: %w", err)
	}
	var items []models.SisterSyncMap
	order := listing.OrderClause(domain.SisterSortColumns, options.Sort, options.Order, "createdAt")
	err := query.Order(order).Offset((options.Page - 1) * options.PageSize).Limit(options.PageSize).Find(&items).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list sister: %w", err)
	}
	return items, total, nil
}

func (s GormStore) GetSister(ctx context.Context, id string) (models.SisterSyncMap, error) {
	var item models.SisterSyncMap
	err := s.DB.WithContext(ctx).Where("id = ?", id).First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.SisterSyncMap{}, domain.ErrNotFound
		}
		return models.SisterSyncMap{}, fmt.Errorf("read sister: %w", err)
	}
	return item, nil
}

func (s GormStore) CreateSister(ctx context.Context, actorID string, item models.SisterSyncMap) (models.SisterSyncMap, error) {
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&item).Error; err != nil {
			return classifyRecordError(err)
		}
		return createAudit(ctx, tx, audit.Entry{ActorID: actorID, Action: audit.ActionSisterCreated, TargetID: &item.ID, UnitID: &item.UserId})
	})
	return item, err
}

func (s GormStore) UpdateSister(ctx context.Context, actorID, id string, patch domain.SisterSyncMapPatch) (models.SisterSyncMap, error) {
	updates := sisterColumns(patch)
	var item models.SisterSyncMap
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&item).Error; err != nil {
			return classifyRecordError(err)
		}
		if err := tx.Model(&item).Updates(updates).Error; err != nil {
			return classifyRecordError(err)
		}
		if err := tx.First(&item, "id = ?", id).Error; err != nil {
			return fmt.Errorf("reload sister: %w", err)
		}
		return createAudit(ctx, tx, audit.Entry{ActorID: actorID, Action: audit.ActionSisterUpdated, TargetID: optionalRef(id), UnitID: &item.UserId})
	})
	return item, err
}

func (s GormStore) ArchiveSister(ctx context.Context, actorID, id string) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var item models.SisterSyncMap
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&item).Error; err != nil {
			return classifyRecordError(err)
		}
		result := tx.Model(&models.SisterSyncMap{}).Where("id = ?").Update("aktif", false)
		if result.Error != nil {
			return fmt.Errorf("archive sister: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return domain.ErrNotFound
		}
		return createAudit(ctx, tx, audit.Entry{ActorID: actorID, Action: audit.ActionSisterArchived, TargetID: optionalRef(id), UnitID: &item.UserId})
	})
}

func (s GormStore) UpdateSisterSyncTime(ctx context.Context, actorID, id string, syncTime time.Time) (models.SisterSyncMap, error) {
	var item models.SisterSyncMap
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&item).Error; err != nil {
			return classifyRecordError(err)
		}
		if err := tx.Model(&item).Updates(map[string]any{"sinkronisasiTerakhir": syncTime, "updatedAt": syncTime}).Error; err != nil {
			return classifyRecordError(err)
		}
		note := "sync=" + syncTime.Format("2006-01-02T15:04:05Z")
		return createAudit(ctx, tx, audit.Entry{ActorID: actorID, Action: audit.ActionSisterUpdated, TargetID: optionalRef(id), UnitID: &item.UserId, Note: &note})
	})
	if err != nil {
		return models.SisterSyncMap{}, err
	}
	item.SinkronisasiTerakhir = &syncTime
	return item, nil
}

func sisterColumns(patch domain.SisterSyncMapPatch) map[string]any {
	updates := make(map[string]any)
	if patch.SistemEksternal.Present {
		if patch.SistemEksternal.Null {
			updates["sistemEksternal"] = nil
		} else {
			updates["sistemEksternal"] = patch.SistemEksternal.Value
		}
	}
	if patch.IDEksternal.Present {
		if patch.IDEksternal.Null {
			updates["idEksternal"] = nil
		} else {
			updates["idEksternal"] = patch.IDEksternal.Value
		}
	}
	if patch.SinkronisasiTerakhir.Present {
		if patch.SinkronisasiTerakhir.Null {
			updates["sinkronisasiTerakhir"] = nil
		} else {
			updates["sinkronisasiTerakhir"] = patch.SinkronisasiTerakhir.Value
		}
	}
	if patch.Status.Present {
		if patch.Status.Null {
			updates["status"] = nil
		} else {
			updates["status"] = patch.Status.Value
		}
	}
	return updates
}
