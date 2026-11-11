package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

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
	token, err := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return v.secret, nil
	})
	if err != nil {
		return claims, err
	}
	if !token.Valid {
		return claims, errors.New("invalid token")
	}
	if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
		return claims, errors.New("token expired")
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
		next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), JWTPrincipalFromClaims(claims))))
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
	// Coba JWT terlebih dahulu (format: header.payload.signature)
	parts := strings.Split(token, ".")
	if len(parts) == 3 {
		if claims, err := r.jwtVerifier.VerifyToken(token); err == nil {
			return JWTPrincipalFromClaims(claims), nil
		}
	}
	// Fallback ke resolusi sesi GORM (token opaque)
	return r.gormResolver.ResolvePrincipal(ctx, token)
}
