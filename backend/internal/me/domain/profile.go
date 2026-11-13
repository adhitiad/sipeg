// Package domain berisi aturan self-service profil principal. Nil/kosong
// ditangani eksplisit sehingga lapisan HTTP cukup meneruskan DTO.
package domain

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// ErrValidation menandai data profil yang melanggar aturan.
var ErrValidation = errors.New("me profile input is not valid")

// ErrNotFound menandai record user principal yang tidak ditemukan. Dengan
// sesi terverifikasi kondisi ini mengisyaratkan data tidak konsisten.
var ErrNotFound = errors.New("me profile record not found")

const (
	// MaxNamaRunes membatasi panjang nama yang boleh disimpan.
	MaxNamaRunes = 200
	// MaxNoHpRunes membatasi panjang nomor HP setelah dipadatkan.
	MaxNoHpRunes = 20
)

// phonePattern menerima nomor Indonesia: awalan 0 atau 62 (opsional +),
// dilanjutkan 8 sampai 15 digit.
var phonePattern = regexp.MustCompile(`^\+?(62|0)[0-9]{8,15}$`)

// UpdateProfileInput adalah data yang boleh diubah principal pada profilnya.
// Field nil berarti tidak diubah; teks kosong pada NoHp berarti mengosongkan.
type UpdateProfileInput struct {
	Nama *string
	NoHp *string
}

// Fields melaporkan nama kolom yang dibawa input, untuk catatan audit.
func (input UpdateProfileInput) Fields() []string {
	fields := make([]string, 0, 2)
	if input.Nama != nil {
		fields = append(fields, "nama")
	}
	if input.NoHp != nil {
		fields = append(fields, "noHp")
	}
	return fields
}

// Normalize memangkas dan memvalidasi input. Mengembalikan ErrValidation bila
// kolom melanggar aturan atau tidak ada satu pun kolom yang dikirim.
func Normalize(input UpdateProfileInput) (UpdateProfileInput, error) {
	normalized := UpdateProfileInput{}

	if input.Nama != nil {
		nama := strings.Join(strings.Fields(*input.Nama), " ")
		if nama == "" {
			return UpdateProfileInput{}, fmt.Errorf("%w: nama must not be empty", ErrValidation)
		}
		if utf8.RuneCountInString(nama) > MaxNamaRunes {
			return UpdateProfileInput{}, fmt.Errorf("%w: nama exceeds %d characters", ErrValidation, MaxNamaRunes)
		}
		normalized.Nama = &nama
	}

	if input.NoHp != nil {
		compact := compactPhone(*input.NoHp)
		if compact == "" {
			normalized.NoHp = &compact
		} else {
			if utf8.RuneCountInString(compact) > MaxNoHpRunes {
				return UpdateProfileInput{}, fmt.Errorf("%w: noHp exceeds %d characters", ErrValidation, MaxNoHpRunes)
			}
			if !phonePattern.MatchString(compact) {
				return UpdateProfileInput{}, fmt.Errorf("%w: noHp must be a valid Indonesian phone number", ErrValidation)
			}
			normalized.NoHp = &compact
		}
	}

	if normalized.Nama == nil && normalized.NoHp == nil {
		return UpdateProfileInput{}, fmt.Errorf("%w: at least one field is required", ErrValidation)
	}

	return normalized, nil
}

// compactPhone membuang spasi, titik, dan tanda hubung yang umum dipakai
// penulis nomor HP.
func compactPhone(raw string) string {
	var builder strings.Builder
	for _, r := range strings.TrimSpace(raw) {
		switch r {
		case ' ', '-', '.', '(', ')':
			continue
		default:
			builder.WriteRune(r)
		}
	}
	return builder.String()
}
