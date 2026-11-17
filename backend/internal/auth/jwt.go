package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

// BetterAuthClaims adalah klaim JWT yang diterbitkan Better Auth session.
type BetterAuthClaims struct {
	UserID     string `json:"userId"`
	Email      string `json:"email"`
	RoleName   string `json:"role"`
	HomebaseID string `json:"homebaseId"`
	jwt.RegisteredClaims
}

// JWTPrincipalFromClaims mengubah klaim JWT menjadi Principal.
func JWTPrincipalFromClaims(claims BetterAuthClaims) Principal {
	return Principal{
		UserID:     claims.UserID,
		Email:      claims.Email,
		RoleName:   claims.RoleName,
		HomebaseID: claims.HomebaseID,
	}
}

// JWTVerifier memverifikasi token JWT yang diterbitkan Better Auth.
type JWTVerifier struct {
	secret []byte
}

// NewJWTVerifier membuat verifikasi JWT dengan shared secret Better Auth.
func NewJWTVerifier(secret string) *JWTVerifier {
	return &JWTVerifier{secret: []byte(secret)}
}

// VerifyToken memverifikasi dan mengurai JWT. Mengembalikan error jika token tidak valid.
func (v *JWTVerifier) VerifyToken(tokenString string) (BetterAuthClaims, error) {
	claims := BetterAuthClaims{}
	if v == nil || len(v.secret) == 0 {
		return claims, errors.New("JWT secret is not configured")
	}
	token, err := jwt.ParseWithClaims(
		tokenString,
		&claims,
		func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return v.secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return claims, err
	}
	if !token.Valid {
		return claims, errors.New("invalid token")
	}
	if _, err := uuid.Parse(strings.TrimSpace(claims.UserID)); err != nil {
		return claims, errors.New("JWT userId is missing or invalid")
	}
	return claims, nil
}

// JWTAuthenticate adalah middleware HTTP yang mengekstrak Bearer JWT, memverifikasinya,
// dan menyimpan principal ke context. Digunakan sebagai alternatif atau pelengkap
// GormSessionResolver untuk integrasi Better Auth.
func JWTAuthenticate(verifier *JWTVerifier, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r.Header.Get("Authorization"))
		if token == "" {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		claims, err := verifier.VerifyToken(token)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		ctx := contextWithJWTClaims(r.Context(), claims)
		next.ServeHTTP(w, r.WithContext(WithPrincipal(ctx, JWTPrincipalFromClaims(claims))))
	})
}

// JWTFromContext mengambil klaim JWT dari context (jika disimpan oleh middleware custom).
func JWTFromContext(ctx context.Context) (BetterAuthClaims, bool) {
	claims, ok := ctx.Value(jwtClaimsKey{}).(BetterAuthClaims)
	return claims, ok
}

type jwtClaimsKey struct{}

func contextWithJWTClaims(ctx context.Context, claims BetterAuthClaims) context.Context {
	return context.WithValue(ctx, jwtClaimsKey{}, claims)
}

// CombinedSessionResolver mencoba verifikasi JWT (Better Auth) terlebih dahulu,
// lalu fallback ke pencarian sesi GORM untuk token opaque. Digunakan saat backend
// harus mendukung baik sesi lama (database token) maupun sesi baru (JWT Better Auth).
type CombinedSessionResolver struct {
	jwtVerifier  *JWTVerifier
	gormResolver GormSessionResolver
}

// NewCombinedSessionResolver membuat resolver gabungan JWT + GORM session.
func NewCombinedSessionResolver(jwtVerifier *JWTVerifier, db *gorm.DB) CombinedSessionResolver {
	return CombinedSessionResolver{
		jwtVerifier:  jwtVerifier,
		gormResolver: GormSessionResolver{DB: db},
	}
}

func (r CombinedSessionResolver) ResolvePrincipal(ctx context.Context, token string) (Principal, error) {
	// Token berformat JWT tidak boleh turun kelas menjadi sesi opaque ketika
	// signature/expiry tidak valid. Ini mencegah authentication downgrade.
	if strings.Count(token, ".") == 2 {
		if r.jwtVerifier == nil {
			return Principal{}, ErrInvalidSession
		}
		claims, err := r.jwtVerifier.VerifyToken(token)
		if err != nil {
			return Principal{}, ErrInvalidSession
		}
		// Role, email, status aktif, dan homebase selalu dimuat ulang dari DB;
		// klaim JWT lama tidak dapat mempertahankan hak akses yang telah dicabut.
		return r.gormResolver.ResolveUserPrincipal(ctx, claims.UserID)
	}
	return r.gormResolver.ResolvePrincipal(ctx, token)
}
