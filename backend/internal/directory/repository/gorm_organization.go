package repository

import (
	"context"
	"errors"
	"fmt"

	"simpeg/backend/internal/audit"
	"simpeg/backend/internal/directory/domain"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ListOrganizations membaca daftar unit aktif sesuai filter dan pagination.
func (s GormStore) ListOrganizations(ctx context.Context, options listing.Options) ([]models.Homebase, int64, error) {
	query := s.DB.WithContext(ctx).Model(&models.Homebase{}).Where("aktif = ?", true)
	if options.Jenis != "" {
		query = query.Where("jenis = ?", options.Jenis)
	}
	if options.ParentID != "" {
		query = query.Where(`"parentId" = ?`, options.ParentID)
	}
	if options.Search != "" {
		pattern := "%" + escapeLike(options.Search) + "%"
		query = query.Where("name ILIKE ? ESCAPE '\\'", pattern)
	}
	if len(options.ScopeHomebaseIDs) > 0 {
		query = query.Where("id IN ?", options.ScopeHomebaseIDs)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count organizations: %w", err)
	}
	var organizations []models.Homebase
	order := listing.OrderClause(domain.OrganizationSortColumns, options.Sort, options.Order, "name")
	err := query.Order(order).Offset((options.Page - 1) * options.PageSize).Limit(options.PageSize).Find(&organizations).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list organizations: %w", err)
	}
	return organizations, total, nil
}

// GetOrganization membaca satu unit organisasi aktif.
func (s GormStore) GetOrganization(ctx context.Context, id string) (models.Homebase, error) {
	var organization models.Homebase
	err := s.DB.WithContext(ctx).Where("id = ? AND aktif = ?", id, true).First(&organization).Error
	return organization, classifyRecordError(err)
}

// CreateOrganization membuat satu unit organisasi dan mencatat jejak audit.
func (s GormStore) CreateOrganization(ctx context.Context, actorID string, organization models.Homebase) (models.Homebase, error) {
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := validateOrganizationParent(tx, "", organization.ParentId); err != nil {
			return err
		}
		if err := tx.Create(&organization).Error; err != nil {
			return classifyRecordError(err)
		}
		return createAudit(ctx, tx, audit.Entry{ActorID: actorID, Action: audit.ActionOrganizationCreated, TargetID: &organization.ID, UnitID: &organization.ID})
	})
	return organization, err
}

// UpdateOrganization memperbarui unit organisasi dan mencatat jejak audit.
func (s GormStore) UpdateOrganization(ctx context.Context, actorID, id string, patch domain.OrganizationPatch) (models.Homebase, error) {
	updates := organizationColumns(patch)
	var organization models.Homebase
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND aktif = ?", id, true).First(&organization).Error; err != nil {
			return classifyRecordError(err)
		}
		if parentID, ok := updates["parentId"]; ok {
			value, _ := parentID.(*string)
			if err := validateOrganizationParent(tx, id, value); err != nil {
				return err
			}
		}
		if err := tx.Model(&organization).Updates(updates).Error; err != nil {
			return classifyRecordError(err)
		}
		if err := tx.First(&organization, "id = ?", id).Error; err != nil {
			return fmt.Errorf("reload organization: %w", err)
		}
		return createAudit(ctx, tx, audit.Entry{ActorID: actorID, Action: audit.ActionOrganizationUpdated, TargetID: optionalRef(id), UnitID: &id})
	})
	return organization, err
}

// ArchiveOrganization menonaktifkan unit yang tidak lagi memiliki anggota
// maupun unit anak aktif.
func (s GormStore) ArchiveOrganization(ctx context.Context, actorID, id string) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var organization models.Homebase
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND aktif = ?", id, true).First(&organization).Error; err != nil {
			return classifyRecordError(err)
		}
		var count int64
		if err := tx.Model(&models.User{}).Where(`"homebaseId" = ? AND aktif = ?`, id, true).Count(&count).Error; err != nil {
			return fmt.Errorf("check organization employees: %w", err)
		}
		var childCount int64
		if err := tx.Model(&models.Homebase{}).Where(`"parentId" = ? AND aktif = ?`, id, true).Count(&childCount).Error; err != nil {
			return fmt.Errorf("check organization children: %w", err)
		}
		if count > 0 || childCount > 0 {
			return domain.ErrHasActiveMember
		}
		result := tx.Model(&models.Homebase{}).Where("id = ? AND aktif = ?", id, true).Update("aktif", false)
		if result.Error != nil {
			return fmt.Errorf("archive organization: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return domain.ErrNotFound
		}
		return createAudit(ctx, tx, audit.Entry{ActorID: actorID, Action: audit.ActionOrganizationArchived, TargetID: optionalRef(id), UnitID: &id})
	})
}

// organizationColumns memetakan patch domain menjadi kolom database.
func organizationColumns(patch domain.OrganizationPatch) map[string]any {
	updates := make(map[string]any)
	if patch.Name.Present {
		updates["name"] = patch.Name.Value
	}
	if patch.Description.Present {
		if patch.Description.Null {
			updates["description"] = nil
		} else {
			updates["description"] = patch.Description.Value
		}
	}
	if patch.Jenis.Present {
		updates["jenis"] = domain.NormalizeOrganizationType(patch.Jenis.Value)
	}
	if patch.ParentID.Present {
		if patch.ParentID.Null {
			updates["parentId"] = (*string)(nil)
		} else {
			parentID := patch.ParentID.Value
			updates["parentId"] = &parentID
		}
	}
	return updates
}

// validateOrganizationParent memastikan induk unit ada, aktif, dan tidak
// menimbulkan siklus pada pohon organisasi.
func validateOrganizationParent(tx *gorm.DB, organizationID string, parentID *string) error {
	if parentID == nil {
		return nil
	}
	if *parentID == organizationID {
		return domain.ErrInvalidParent
	}
	currentID := *parentID
	seen := map[string]bool{}
	for currentID != "" {
		if currentID == organizationID || seen[currentID] {
			return domain.ErrInvalidParent
		}
		seen[currentID] = true
		var parent models.Homebase
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND aktif = ?", currentID, true).First(&parent).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrInvalidParent
			}
			return fmt.Errorf("validate organization parent: %w", err)
		}
		if parent.ParentId == nil {
			break
		}
		currentID = *parent.ParentId
	}
	return nil
}
