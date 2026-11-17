// Package rbac menyediakan adapter policy yang menggabungkan grant statis
// (fallback) dan evaluator RBAC berbasis scoped assignment (`DEC-087`).
//
// Adapter mengimplementasikan auth.Policy (delegasi ke grant statis) dan
// auth.ScopedPolicy (mencoba evaluator RBAC terlebih dahulu; bila tidak ada
// data RBAC untuk principal, kembali ke grant statis). Migrasi bertahap:
// handler baru dapat memanggil AllowsScoped; handler lama tetap memakai
// Allows tanpa perubahan.
package rbac

import (
	"context"

	"simpeg/backend/internal/auth"
	rbacapp "simpeg/backend/internal/rbac/app"
	"simpeg/backend/internal/rbac/domain"
)

// PolicyAdapter menggabungkan grant statis dan evaluator RBAC.
type PolicyAdapter struct {
	Static auth.Policy
	RBAC   *rbacapp.Service
}

// NewPolicyAdapter membuat adapter.
func NewPolicyAdapter(static auth.Policy, rbac *rbacapp.Service) *PolicyAdapter {
	return &PolicyAdapter{Static: static, RBAC: rbac}
}

// Allows mengimplementasikan auth.Policy: delegasi ke grant statis agar
// handler lama tetap berfungsi tanpa perubahan.
func (a *PolicyAdapter) Allows(roleName string, permission auth.Permission) bool {
	if a.Static == nil {
		return false
	}
	return a.Static.Allows(roleName, permission)
}

// List melaporkan seluruh permission milik role — delegasi ke grant statis
// agar klien (/me) tetap menerima matriks efektif saat ini. Bila evaluator
// RBAC otoritatif berlaku untuk principal, daftar ini adalah superset
// (mencerminkan grant statis, bukan hasil evaluasi RBAC); ini keterbatasan
// yang disengaja hingga evaluator dapat menghitung daftar efektif per
// principal.
func (a *PolicyAdapter) List(roleName string) []auth.Permission {
	lister, ok := a.Static.(auth.PermissionLister)
	if !ok || lister == nil {
		return []auth.Permission{}
	}
	return lister.List(roleName)
}

// AllowsScoped mengimplementasikan auth.ScopedPolicy: mencoba evaluator RBAC
// terlebih dahulu. Bila RBAC otoritatif (principal memiliki assignment), hasil
// RBAC dipakai apa adanya termasuk penolakan (RBAC dapat mencabut izin statis).
// Bila RBAC tidak memiliki data assignment atau tidak dikonfigurasi, kembali ke
// grant statis untuk mendukung migrasi bertahap.
func (a *PolicyAdapter) AllowsScoped(ctx context.Context, principal auth.Principal, permission auth.Permission) bool {
	if a.RBAC != nil {
		module, action, scope := domain.ParsePermission(string(permission))
		decision, err := a.RBAC.CanAccess(ctx, principal, module, action, string(scope))
		if err != nil {
			// Error membaca assignment/matriks RBAC bukan berarti belum ada
			// assignment. Tolak akses; fallback statis hanya sah untuk
			// keputusan NoAssignment yang eksplisit.
			return false
		}
		if decision.Authoritative {
			return decision.Allowed
		}
	}

	if a.Static == nil {
		return false
	}
	return a.Static.Allows(principal.RoleName, permission)
}
