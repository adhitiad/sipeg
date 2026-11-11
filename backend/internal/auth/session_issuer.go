package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"simpeg/backend/internal/models"
	"simpeg/backend/internal/timeutil"
)

// ErrSessionNotFound sudah didefinisikan di revoker.go dan dipakai kembali di sini
// untuk pengguna yang tidak ditemukan atau tidak aktif saat penerbitan sesi.

// SessionIssuer membuat sesi baru untuk sebuah pengguna dan menyimpannya ke
// database. Implementasi inti adalah GORM (tabel session); nanti dapat
// digantikan atau didampingi oleh runtime Better Auth saat flow login tersedia.
//
// Role dan homebase yang disimpan di sesi diambil dari data server-side
// (tabel user/role), bukan dari input panggilan, sehingga endpoint ini tidak
// dapat digunakan untuk naikkan role secara sembarang.
type SessionIssuer interface {
	// IssueSession membuat sesi untuk userID. Mengembalikan token sesi dan
	// waktu kedaluwarsa.
	IssueSession(ctx context.Context, userID string) (token string, expiresAt time.Time, err error)
}

var errIssuerDB = errors.New("session issuer database error")

// GormSessionIssuer menyimpan sesi baru ke tabel session melalui GORM.
type GormSessionIssuer struct {
	DB       *gorm.DB
	Lifetime time.Duration
}

// NewGormSessionIssuer membuat issuer sesi GORM dengan lifetime yang dikonfigurasi.
func NewGormSessionIssuer(db *gorm.DB, lifetime time.Duration) *GormSessionIssuer {
	return &GormSessionIssuer{DB: db, Lifetime: lifetime}
}

// IssueSession membuat sesi untuk userID. Role dan homebase diambil dari data
// server-side. Mengembalikan error ErrSessionNotFound bila user tidak ditemukan
// atau tidak aktif.
func (i *GormSessionIssuer) IssueSession(ctx context.Context, userID string) (string, time.Time, error) {
	if i.DB == nil {
		return "", time.Time{}, errIssuerDB
	}

	var user models.User
	if err := i.DB.WithContext(ctx).First(&user, "id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", time.Time{}, ErrSessionNotFound
		}
		return "", time.Time{}, fmt.Errorf("find user: %w", err)
	}
	if !user.Aktif {
		return "", time.Time{}, ErrSessionNotFound
	}

	if user.RoleId != nil && *user.RoleId != "" {
		var role models.Role
		if err := i.DB.WithContext(ctx).First(&role, "id = ?", *user.RoleId).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return "", time.Time{}, ErrSessionNotFound
			}
			return "", time.Time{}, fmt.Errorf("find role: %w", err)
		}
		// Role divalidasi ada saat penerbitan. Nama role dibaca ulang oleh
		// resolver pada setiap permintaan dari tabel Role, bukan disalin ke sesi.
		_ = role.ID
	}

	token := newSessionToken()
	expiresAt := timeutil.NowUTC().Add(i.Lifetime)

	session := models.Session{
		UserId:    user.ID,
		ExpiresAt: expiresAt,
		Token:     token,
	}
	if err := i.DB.WithContext(ctx).Create(&session).Error; err != nil {
		return "", time.Time{}, fmt.Errorf("create session: %w", err)
	}

	return token, expiresAt, nil
}

// newSessionToken menghasilkan token acak 32 byte (64 karakter heksadesimal).
func newSessionToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("session token generation failed: %v", err))
	}
	return hex.EncodeToString(b)
}
