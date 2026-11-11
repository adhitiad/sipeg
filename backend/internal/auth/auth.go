package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"simpeg/backend/internal/apierror"
)

type Permission string

type Principal struct {
	UserID     string
	Email      string
	RoleName   string
	HomebaseID string
}

type SessionResolver interface {
	ResolvePrincipal(context.Context, string) (Principal, error)
}

type Policy interface {
	Allows(roleName string, permission Permission) bool
}

type StaticPolicy map[string]map[Permission]bool

func (p StaticPolicy) Allows(roleName string, permission Permission) bool {
	return p[roleName][permission]
}

type contextKey struct{}

func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, contextKey{}, principal)
}

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(contextKey{}).(Principal)
	return principal, ok
}

type sessionTokenKey struct{}

// WithSessionToken menyimpan token sesi terverifikasi agar handler dapat mencabut sesi.
func WithSessionToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, sessionTokenKey{}, token)
}

// SessionTokenFromContext mengembalikan token sesi terverifikasi dari context.
func SessionTokenFromContext(ctx context.Context) (string, bool) {
	token, ok := ctx.Value(sessionTokenKey{}).(string)
	return token, ok
}

func Authenticate(resolver SessionResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := bearerToken(r.Header.Get("Authorization"))
			if token == "" {
				writeError(w, http.StatusUnauthorized, apierror.CodeUnauthorized)
				return
			}
			if resolver == nil {
				writeError(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
				return
			}

			principal, err := resolver.ResolvePrincipal(r.Context(), token)
			if errors.Is(err, ErrInvalidSession) || (err == nil && principal.UserID == "") {
				writeError(w, http.StatusUnauthorized, apierror.CodeUnauthorized)
				return
			}
			if err != nil {
				writeError(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
				return
			}

			next.ServeHTTP(w, r.WithContext(WithPrincipal(WithSessionToken(r.Context(), token), principal)))
		})
	}
}

func RequirePermission(policy Policy, permission Permission, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := PrincipalFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, apierror.CodeUnauthorized)
			return
		}
		if policy == nil || !policy.Allows(principal.RoleName, permission) {
			writeError(w, http.StatusForbidden, apierror.CodeForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireAnyPermission mengizinkan akses jika salah satu dari daftar permission diberikan kepada role principal.
func RequireAnyPermission(policy Policy, permissions []Permission, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := PrincipalFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, apierror.CodeUnauthorized)
			return
		}
		if policy == nil {
			writeError(w, http.StatusForbidden, apierror.CodeForbidden)
			return
		}
		for _, p := range permissions {
			if policy.Allows(principal.RoleName, p) {
				next.ServeHTTP(w, r)
				return
			}
		}
		writeError(w, http.StatusForbidden, apierror.CodeForbidden)
	})
}

// UnitScope menyatakan unit-unit yang boleh dijangkau principal di bawah
// unit penugasannya, beserta seluruh turunannya pada pohon organisasi.
type UnitScope struct {
	RootID  string
	UnitIDs map[string]bool
	list    []string
}

// NewUnitScope membuat UnitScope dari daftar unit hasil resolusi.
func NewUnitScope(rootID string, unitIDs []string) UnitScope {
	set := make(map[string]bool, len(unitIDs))
	for _, id := range unitIDs {
		set[id] = true
	}
	return UnitScope{RootID: rootID, UnitIDs: set, list: unitIDs}
}

// Contains memeriksa apakah sebuah unit berada dalam cakupan.
func (s UnitScope) Contains(unitID string) bool {
	if unitID == "" {
		return false
	}
	return s.UnitIDs[unitID]
}

// Empty melaporkan apakah cakupan tidak berisi unit apa pun.
func (s UnitScope) Empty() bool { return len(s.UnitIDs) == 0 }

// List mengembalikan unit-unit dalam cakupan.
func (s UnitScope) List() []string { return s.list }

// CanAccessUserInScope memeriksa akses ke record pegawai terhadap cakupan unit.
// Permission global ("...:any") mengizinkan seluruh organisasi; permission
// homebase hanya mengizinkan record pada unit yang berada dalam cakupan.
func CanAccessUserInScope(principal Principal, targetUserID, targetHomebaseID string, permission Permission, policy Policy, scope UnitScope) bool {
	if policy == nil || !policy.Allows(principal.RoleName, permission) {
		return false
	}
	if principal.UserID != "" && principal.UserID == targetUserID {
		return true
	}
	switch permission {
	case "employee:read:any", "employee:update:any", "employee:create", "employee:import", "employee:archive":
		return true
	case "employee:read:homebase", "employee:update:homebase", "employee:create:homebase",
		"employee:import:homebase", "employee:archive:homebase":
		return scope.Contains(targetHomebaseID)
	default:
		return false
	}
}

func bearerToken(header string) string {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}

func writeError(w http.ResponseWriter, status int, code string) {
	apierror.Write(w, status, code)
}
