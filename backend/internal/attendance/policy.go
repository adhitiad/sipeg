package attendance

import (
	"context"

	"simpeg/backend/internal/auth"
)

// allowsAttendance selalu memakai evaluator scoped saat tersedia; grant statis
// hanya dipakai pada principal yang memang belum memiliki assignment RBAC.
func allowsAttendance(ctx context.Context, policy auth.Policy, principal auth.Principal, permission auth.Permission) bool {
	if policy == nil {
		return false
	}
	if scoped, ok := policy.(auth.ScopedPolicy); ok {
		return scoped.AllowsScoped(ctx, principal, permission)
	}
	return policy.Allows(principal.RoleName, permission)
}
