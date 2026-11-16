// Package domain memuat aturan RBAC berbasis scoped assignment
// (`DEC-087`): komposisi permission efektif, tier role, dan blacklist.
package domain

import (
	"errors"
	"strings"
)

// TierRole adalah tingkat hierarki role master (sesuai desain RBAC Scoped).
type TierRole int

const (
	// TierSuperadmin adalah peringkat global (Rektor + pembuat aplikasi).
	TierSuperadmin TierRole = 1
	// TierOperatorInduk adalah peringkat induk lintas seluruh organisasi.
	TierOperatorInduk TierRole = 2
	// TierAdmin adalah peringkat admin pada tingkat biro/fakultas/jurusan/unit.
	TierAdmin TierRole = 3
	// TierJurusan adalah peringkat admin/operator pada tingkat jurusan.
	TierJurusan TierRole = 4
)

// ScopePermission adalah cakupan akses yang diminta.
type ScopePermission string

const (
	ScopeAny      ScopePermission = "any"
	ScopeHomebase ScopePermission = "homebase"
	ScopeSelf     ScopePermission = "self"
)

// ErrForbidden menandai penolakan akses.
var ErrForbidden = errors.New("akses ditolak")

// ErrScopeViolation menandai izin ada tetapi di luar cakupan.
var ErrScopeViolation = errors.New("akses di luar cakupan")

// ErrBlacklisted menandai permission yang dilarang (blacklist).
var ErrBlacklisted = errors.New("akses dilarang oleh blacklist")

// EffectivePermission adalah permission efektif pada satu aksi lintas modul:akses.
type EffectivePermission struct {
	Module string
	Action string
	Scope  ScopePermission
	Value  string
}

// NormalizePermission membangun nilai permission flat SIMPEG
// (`<module>:<action>[:<scope>]`) dari komponen modul, aksi, dan cakupan.
func NormalizePermission(module, action string, scope ScopePermission) EffectivePermission {
	value := strings.ToLower(module) + ":" + strings.ToLower(action)
	if scope != "" && scope != ScopeAny {
		value += ":" + strings.ToLower(string(scope))
	}
	return EffectivePermission{
		Module: strings.TrimSpace(module),
		Action: strings.TrimSpace(action),
		Scope:  scope,
		Value:  value,
	}
}

// IsGlobal reports apakah cakupan permission bersifat lintas organisasi.
func (p EffectivePermission) IsGlobal() bool { return p.Scope == ScopeAny }

// ParsePermission memecah permission flat SIMPEG menjadi module, action,
// dan scope. Segmen terakhir menjadi scope bila bernilai any/homebase/self;
// selain itu scope bernilai any. Segmen pertama menjadi module; seluruh
// segmen sisanya digabung kembali dengan ":" menjadi action, sehingga
// permission tiga segmen aksi seperti `skp:periode:read:any` dipetakan ke
// module `skp`, action `periode:read`, scope `any`.
func ParsePermission(p string) (module, action string, scope ScopePermission) {
	segments := strings.Split(strings.ToLower(strings.TrimSpace(p)), ":")
	for i, s := range segments {
		segments[i] = strings.TrimSpace(s)
	}
	scope = ScopeAny
	if n := len(segments); n > 1 {
		if last := ScopePermission(segments[n-1]); last == ScopeAny || last == ScopeHomebase || last == ScopeSelf {
			scope = last
			segments = segments[:n-1]
		}
	}
	if len(segments) >= 1 {
		module = segments[0]
	}
	if len(segments) >= 2 {
		action = strings.Join(segments[1:], ":")
	}
	return module, action, scope
}

// Decision adalah hasil evaluasi akses RBAC.
// Authoritative menandakan bahwa RBAC memiliki data assignment untuk principal
// sehingga keputusan ini (termasuk penolakan) harus dipakai tanpa fallback ke
// grant statis. Bila false, RBAC belum memiliki data dan pemanggil boleh
// kembali ke grant statis.
type Decision struct {
	Allowed       bool
	Reason        string
	Authoritative bool
}

// Allow membuat keputusan akses yang diizinkan dan otoritatif.
func Allow() Decision { return Decision{Allowed: true, Authoritative: true} }

// Deny membuat keputusan penolakan otoritatif dengan alasan.
func Deny(reason string) Decision {
	return Decision{Allowed: false, Reason: reason, Authoritative: true}
}

// NoAssignment menandakan tidak ada data RBAC untuk principal sehingga
// pemanggil harus kembali ke grant statis.
func NoAssignment() Decision {
	return Decision{Allowed: false, Reason: "tidak ada assignment RBAC", Authoritative: false}
}
