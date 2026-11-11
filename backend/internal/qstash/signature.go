// Package qstash mempublikasikan pekerjaan asinkron dan memverifikasi callback
// dari QStash.
//
// Signature callback diverifikasi terhadap raw request body sebelum payload
// Trusted. QStash menandatangani body dengan kunci HS256 dan menyantumkan
// body asli (base64) sebagai klaim JWT, sehingga verifikasi tidak bergantung
// pada parsing ulang body.
package qstash

import (
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// HeaderSignature memuat JWT signature dari QStash.
const HeaderSignature = "Upstash-Signature"

// ErrInvalidSignature menandakan signature tidak valid atau tidak cocok.
var ErrInvalidSignature = errors.New("invalid qstash signature")

// ErrExpiredSignature menandakan signature sudah kedaluwarsa.
var ErrExpiredSignature = errors.New("qstash signature expired")

// Verifier memverifikasi signature callback QStash.
type Verifier struct {
	// SigningKeys berisi kunci penandatanganan yang sah. Kunci pertama
	// adalah kunci saat ini, kunci kedua adalah kunci berikutnya untuk
	// rotasi tanpa downtime.
	SigningKeys []string
	// ExpectedSubject adalah URL tujuan yang harus cocok dengan klaim sub.
	ExpectedSubject string
	// Clock dapat diganti pada pengujian.
	Clock func() time.Time
}

// NewVerifier membuat verifier signature QStash.
func NewVerifier(signingKeys []string, expectedSubject string) *Verifier {
	return &Verifier{SigningKeys: signingKeys, ExpectedSubject: expectedSubject, Clock: time.Now}
}

// Verify memeriksa signature pada header dan kecocokannya dengan raw body.
func (v *Verifier) Verify(header string, rawBody []byte) error {
	header = strings.TrimSpace(header)
	if header == "" {
		return ErrInvalidSignature
	}

	claims := jwt.MapClaims{}
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	for _, key := range v.SigningKeys {
		if strings.TrimSpace(key) == "" {
			continue
		}
		signingKey := []byte(key)
		token, err := parser.ParseWithClaims(header, claims, func(parsed *jwt.Token) (any, error) {
			if _, ok := parsed.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", parsed.Header["alg"])
			}
			return signingKey, nil
		})
		if err != nil || token == nil || !token.Valid {
			continue
		}
		if err := v.checkClaims(claims, rawBody); err != nil {
			return err
		}
		return nil
	}
	return ErrInvalidSignature
}

// checkClaims memastikan subjek, masa berlaku, dan body cocok.
func (v *Verifier) checkClaims(claims jwt.MapClaims, rawBody []byte) error {
	now := time.Now
	if v.Clock != nil {
		now = v.Clock
	}
	current := now()
	if expiry, ok := claims["exp"].(float64); ok {
		if current.After(time.Unix(int64(expiry), 0)) {
			return ErrExpiredSignature
		}
	}
	if notBefore, ok := claims["nbf"].(float64); ok {
		if current.Add(-time.Minute).Before(time.Unix(int64(notBefore), 0)) {
			return ErrInvalidSignature
		}
	}
	if v.ExpectedSubject != "" {
		subject, _ := claims["sub"].(string)
		if subject != v.ExpectedSubject {
			return ErrInvalidSignature
		}
	}

	bodyClaim, ok := claims["body"].(string)
	if !ok {
		return ErrInvalidSignature
	}
	decoded, err := base64.StdEncoding.DecodeString(bodyClaim)
	if err != nil {
		return ErrInvalidSignature
	}
	if subtle.ConstantTimeCompare(decoded, rawBody) != 1 {
		return ErrInvalidSignature
	}
	return nil
}

// SignatureFromRequest mengambil header signature dari permintaan.
func SignatureFromRequest(r *http.Request) string {
	return r.Header.Get(HeaderSignature)
}
