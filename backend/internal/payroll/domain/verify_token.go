package domain

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
)

// NewSlipVerificationToken mengembalikan token verifikasi slip kriptografis.
//
// Token 128-bit dienkode base64url tanpa padding (22 karakter) dan hanya
// disimpan pada slip (kolom KodeVerifikasi). Halaman verifikasi publik
// mencocokkan token ini untuk menampilkan nama, periode, dan status slip —
// tanpa nominal dan tanpa data sensitif lain.
func NewSlipVerificationToken() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// ErrInvalidSlipToken menandai token yang bukan base64url 22 karakter.
var ErrInvalidSlipToken = errors.New("slip verification token is invalid")

// ValidSlipVerificationToken melaporkan apakah token sesuai bentuk yang dihasilkan.
func ValidSlipVerificationToken(token string) bool {
	if len(token) != 22 {
		return false
	}
	for _, r := range token {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			continue
		}
		return false
	}
	return true
}
