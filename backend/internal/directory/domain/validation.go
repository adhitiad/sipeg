package domain

import (
	"net/mail"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

// Batas panjang nilai teks yang diterima untuk data pegawai dan unit.
const (
	MaxNameLength      = 200
	MaxEmailLength     = 254
	MaxNIPLength       = 50
	MaxNIKLength       = 50
	MaxPositionLength  = 200
	MaxPekerjaanLength = 200
	MaxDescription     = 2000
)

// TextLength menghitung jumlah karakter unicode pada sebuah nilai.
func TextLength(value string) int {
	return utf8.RuneCountInString(value)
}

// ValidUUID memeriksa apakah sebuah nilai adalah UUID yang sah.
func ValidUUID(value string) bool {
	_, err := uuid.Parse(value)
	return err == nil
}

// ValidEmail memeriksa apakah sebuah nilai adalah alamat email yang sah.
func ValidEmail(value string) bool {
	address, err := mail.ParseAddress(strings.TrimSpace(value))
	return err == nil && address.Address == strings.TrimSpace(value) && len(value) <= MaxEmailLength
}

// CleanOptional memangkas nilai opsional dan mengubahnya menjadi nil bila
// kosong setelah dipangkas.
func CleanOptional(value *string) *string {
	if value == nil {
		return nil
	}
	cleaned := strings.TrimSpace(*value)
	if cleaned == "" {
		return nil
	}
	return &cleaned
}

// Field menyimpan perubahan satu field pada permintaan PATCH.
//
// Present menandai field ada dalam permintaan, Null menandai nilai eksplisit
// null, dan Value memuat teks yang diminta. Kombinasi ini membedakan "field
// tidak dikirim" dari "field dikosongkan".
type Field struct {
	Present bool
	Null    bool
	Value   string
}

// String mengembalikan nilai field atau string kosong bila tidak hadir.
func (f Field) String() string {
	if !f.Present || f.Null {
		return ""
	}
	return f.Value
}

// validateDate memeriksa apakah string tanggal tidak kosong dan memuat
// setidaknya tahun dan bulan (format "YYYY-MM" atau "YYYY-MM-DD").
func validateDate(value string) bool {
	return len(value) >= 7
}
