// Package models memetakan tabel RBAC berbasis scoped assignment (`DEC-087`).
// Nama kolom mengikuti migrasi `000032_rbac_scoped.sql` (snake_case); field
// GORM memakai tag eksplisit agar tidak bergantung pada konvensi penamaan.
package models

import "time"

// RbacPermission adalah permission master per modul dan aksi.
type RbacPermission struct {
	ID        uint      `gorm:"primaryKey;column:id"`
	Module    string    `gorm:"column:module;uniqueIndex:uq_rbac_permissions_module_action;not null"`
	Action    string    `gorm:"column:action;uniqueIndex:uq_rbac_permissions_module_action;not null"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (m *RbacPermission) TableName() string { return "rbac_permissions" }

// RbacRole adalah role master berbasis tier.
type RbacRole struct {
	ID        uint      `gorm:"primaryKey;column:id"`
	Code      string    `gorm:"column:code;uniqueIndex;not null"`
	Name      string    `gorm:"column:name;not null"`
	TierLevel int       `gorm:"column:tier_level;not null"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (m *RbacRole) TableName() string { return "rbac_roles" }

// RbacUserRoleScope adalah assignment user-role dengan scope organisasi
// serta daftar permission yang dilarang (blacklist).
type RbacUserRoleScope struct {
	ID                uint      `gorm:"primaryKey;column:id"`
	UserID            string    `gorm:"column:user_id;type:uuid;index:rbac_user_role_scope_user_idx"`
	RoleID            uint      `gorm:"column:role_id;not null"`
	BiroCode          *string   `gorm:"column:biro_code"`
	FakultasID        *string   `gorm:"column:fakultas_id;type:uuid"`
	JurusanID         *string   `gorm:"column:jurusan_id;type:uuid"`
	UnitID            *string   `gorm:"column:unit_id;type:uuid"`
	DeniedPermissions []byte    `gorm:"column:denied_permissions;type:jsonb"`
	CreatedAt         time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt         time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (m *RbacUserRoleScope) TableName() string { return "rbac_user_role_scopes" }

// RbacRolePermission menghubungkan role dengan permission (matriks).
type RbacRolePermission struct {
	RoleID       uint `gorm:"column:role_id;primaryKey"`
	PermissionID uint `gorm:"column:permission_id;primaryKey"`
}

func (m *RbacRolePermission) TableName() string { return "rbac_role_permissions" }
