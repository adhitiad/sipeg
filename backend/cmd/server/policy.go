package main

import (
	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/policymatrix"
)

// RolePermissionMatrix mengembalikan matriks grant permission per role.
// Deny-by-default: role yang tidak tercantum atau izin yang tidak disebutkan
// ditolak. Matriks ini disetujui dalam docs/PRD.md §11 dan diterapkan di
// backend sebagai auth.StaticPolicy; definisi tunggalnya berada di paket
// `internal/policymatrix` (dipakai juga oleh perkakas seed RBAC).
func RolePermissionMatrix() auth.StaticPolicy {
	matrix := policymatrix.Matrix()
	policy := auth.StaticPolicy{}
	for roleName, grants := range matrix {
		byPermission := make(map[auth.Permission]bool, len(grants))
		for permission, granted := range grants {
			byPermission[auth.Permission(permission)] = granted
		}
		policy[roleName] = byPermission
	}
	return policy
}
