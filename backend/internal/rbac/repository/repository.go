// Package repository menyimpan port persistensi RBAC berbasis scoped
// assignment (`DEC-087`). Termasuk operasi baca dan tulis (CRUD) untuk
// role master, permission master, matriks role-permission, assignment
// user-role dengan scope, dan delegasi.
package repository

import (
	"context"
	"fmt"

	"simpeg/backend/internal/models"

	"gorm.io/gorm"
)

// Store adalah port persistensi RBAC.
type Store interface {
	ListRoles(ctx context.Context) ([]models.RbacRole, error)
	ListPermissions(ctx context.Context) ([]models.RbacPermission, error)
	ListRolePermissions(ctx context.Context) ([]models.RbacRolePermission, error)
	ListUserRoleScopes(ctx context.Context, userID string) ([]models.RbacUserRoleScope, error)

	CreateRole(ctx context.Context, role models.RbacRole) (models.RbacRole, error)
	UpdateRole(ctx context.Context, id uint, role models.RbacRole) (models.RbacRole, error)
	DeleteRole(ctx context.Context, id uint) error

	CreatePermission(ctx context.Context, perm models.RbacPermission) (models.RbacPermission, error)
	DeletePermission(ctx context.Context, id uint) error

	AssignPermissionToRole(ctx context.Context, roleID, permID uint) error
	RevokePermissionFromRole(ctx context.Context, roleID, permID uint) error

	CreateUserRoleScope(ctx context.Context, scope models.RbacUserRoleScope) (models.RbacUserRoleScope, error)
	DeleteUserRoleScope(ctx context.Context, id uint) error

	ListDelegasi(ctx context.Context, userID string) ([]models.Delegasi, error)
	CreateDelegasi(ctx context.Context, d models.Delegasi) (models.Delegasi, error)
	RevokeDelegasi(ctx context.Context, id uint) error
}

// GormStore adapter GORM untuk port RBAC.
type GormStore struct {
	DB *gorm.DB
}

// ListRoles mengembalikan seluruh role master berbasis tier.
func (s GormStore) ListRoles(ctx context.Context) ([]models.RbacRole, error) {
	var items []models.RbacRole
	if err := s.DB.WithContext(ctx).Order("tier_level ASC, id ASC").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("list rbac roles: %w", err)
	}
	return items, nil
}

// ListPermissions mengembalikan seluruh permission master.
func (s GormStore) ListPermissions(ctx context.Context) ([]models.RbacPermission, error) {
	var items []models.RbacPermission
	if err := s.DB.WithContext(ctx).Order("id ASC").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("list rbac permissions: %w", err)
	}
	return items, nil
}

// ListRolePermissions mengembalikan seluruh baris matriks role-permission.
func (s GormStore) ListRolePermissions(ctx context.Context) ([]models.RbacRolePermission, error) {
	var items []models.RbacRolePermission
	if err := s.DB.WithContext(ctx).Order("role_id ASC, permission_id ASC").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("list rbac role permissions: %w", err)
	}
	return items, nil
}

// ListUserRoleScopes mengembalikan assignment user-role milik satu user.
func (s GormStore) ListUserRoleScopes(ctx context.Context, userID string) ([]models.RbacUserRoleScope, error) {
	var items []models.RbacUserRoleScope
	if err := s.DB.WithContext(ctx).Where(`"user_id" = ?`, userID).Find(&items).Error; err != nil {
		return nil, fmt.Errorf("list rbac user role scopes: %w", err)
	}
	return items, nil
}

// CreateRole membuat role master baru.
func (s GormStore) CreateRole(ctx context.Context, role models.RbacRole) (models.RbacRole, error) {
	if err := s.DB.WithContext(ctx).Create(&role).Error; err != nil {
		return models.RbacRole{}, fmt.Errorf("create rbac role: %w", err)
	}
	return role, nil
}

// UpdateRole memperbarui role master.
func (s GormStore) UpdateRole(ctx context.Context, id uint, role models.RbacRole) (models.RbacRole, error) {
	role.ID = id
	if err := s.DB.WithContext(ctx).Save(&role).Error; err != nil {
		return models.RbacRole{}, fmt.Errorf("update rbac role: %w", err)
	}
	return role, nil
}

// DeleteRole menghapus role master beserta matriksnya.
func (s GormStore) DeleteRole(ctx context.Context, id uint) error {
	if err := s.DB.WithContext(ctx).Delete(&models.RbacRole{}, id).Error; err != nil {
		return fmt.Errorf("delete rbac role: %w", err)
	}
	return nil
}

// CreatePermission membuat permission master baru.
func (s GormStore) CreatePermission(ctx context.Context, perm models.RbacPermission) (models.RbacPermission, error) {
	if err := s.DB.WithContext(ctx).Create(&perm).Error; err != nil {
		return models.RbacPermission{}, fmt.Errorf("create rbac permission: %w", err)
	}
	return perm, nil
}

// DeletePermission menghapus permission master.
func (s GormStore) DeletePermission(ctx context.Context, id uint) error {
	if err := s.DB.WithContext(ctx).Delete(&models.RbacPermission{}, id).Error; err != nil {
		return fmt.Errorf("delete rbac permission: %w", err)
	}
	return nil
}

// AssignPermissionToRole menambahkan permission ke role.
func (s GormStore) AssignPermissionToRole(ctx context.Context, roleID, permID uint) error {
	row := models.RbacRolePermission{RoleID: roleID, PermissionID: permID}
	if err := s.DB.WithContext(ctx).Create(&row).Error; err != nil {
		return fmt.Errorf("assign permission to role: %w", err)
	}
	return nil
}

// RevokePermissionFromRole mencabut permission dari role.
func (s GormStore) RevokePermissionFromRole(ctx context.Context, roleID, permID uint) error {
	if err := s.DB.WithContext(ctx).Where(`"role_id" = ? AND "permission_id" = ?`, roleID, permID).Delete(&models.RbacRolePermission{}).Error; err != nil {
		return fmt.Errorf("revoke permission from role: %w", err)
	}
	return nil
}

// CreateUserRoleScope membuat assignment user-role dengan scope.
func (s GormStore) CreateUserRoleScope(ctx context.Context, scope models.RbacUserRoleScope) (models.RbacUserRoleScope, error) {
	if err := s.DB.WithContext(ctx).Create(&scope).Error; err != nil {
		return models.RbacUserRoleScope{}, fmt.Errorf("create user role scope: %w", err)
	}
	return scope, nil
}

// DeleteUserRoleScope menghapus assignment user-role.
func (s GormStore) DeleteUserRoleScope(ctx context.Context, id uint) error {
	if err := s.DB.WithContext(ctx).Delete(&models.RbacUserRoleScope{}, id).Error; err != nil {
		return fmt.Errorf("delete user role scope: %w", err)
	}
	return nil
}

// ListDelegasi mengembalikan delegasi yang diterima seorang user.
func (s GormStore) ListDelegasi(ctx context.Context, userID string) ([]models.Delegasi, error) {
	var items []models.Delegasi
	if err := s.DB.WithContext(ctx).Where(`"penerimaUserId" = ?`, userID).Find(&items).Error; err != nil {
		return nil, fmt.Errorf("list delegasi: %w", err)
	}
	return items, nil
}

// CreateDelegasi membuat delegasi baru.
func (s GormStore) CreateDelegasi(ctx context.Context, d models.Delegasi) (models.Delegasi, error) {
	if err := s.DB.WithContext(ctx).Create(&d).Error; err != nil {
		return models.Delegasi{}, fmt.Errorf("create delegasi: %w", err)
	}
	return d, nil
}

// RevokeDelegasi menonaktifkan delegasi.
func (s GormStore) RevokeDelegasi(ctx context.Context, id uint) error {
	if err := s.DB.WithContext(ctx).Model(&models.Delegasi{}).Where("id = ?", id).Update("aktif", false).Error; err != nil {
		return fmt.Errorf("revoke delegasi: %w", err)
	}
	return nil
}
