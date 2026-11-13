// Package crypto menyediakan enkripsi aplikasi (application-level encryption)
// untuk kolom berisi PII: user.nik, user.nip, user.npwp,
// RiwayatPendidikan.noIjazah, dan KeluargaPegawai.nik.
//
// Desain (DEC-053, diperluas DEC-054):
//
//   - Nilai disimpan sebagai 'enc1:' + base64url(nonce (12 byte) ‖
//     AES-256-GCM(ciphertext ‖ tag)). Nonce acak per enkripsi; kesamaan
//     plaintext TIDAK terlihat di database.
//   - Setiap kolom plaintext berpasangan dengan kolom hash (blind index)
//     berisi HMAC-SHA256 atas bentuk ternormalisasi dari nilai (trim, huruf
//     besar, tanpa spasi/titik/strip/garis bawah). Hash dipakai untuk
//     unique index parsial dan query kesetaraan; enkripsi itu sendiri tidak
//     dapat dipakai untuk pencarian atau pengurutan.
//   - Kunci berasal dari environment PII_ENCRYPTION_KEY (base64, tepat 32
//     byte / AES-256). Kunci blind index diturunkan dari kunci yang sama
//     (HMAC dengan label domain), sehingga hanya satu rahasia yang perlu
//     dikelola. Rotasi kunci belum didukung — perubahan kunci harus
//     disertai backfill ulang seluruh kolom PII.
//   - Tanpa kunci yang dikonfigurasi seluruh operasi enkripsi gagal dengan
//     galat eksplisit; tidak ada nilai bawaan rahasia di kode.
//
// Nilai Encrypted di memori selalu plaintext; enkripsi terjadi di Value()
// (saat tulis ke database) dan dekripsi di Scan() (saat baca). Kode yang
// mengisi Encrypted dengan ciphertext mentah ditolak.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql/driver"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
)

const (
	// CiphertextPrefix menandai nilai terenkripsi (versi format 1).
	CiphertextPrefix = "enc1:"
	keySize          = 32
	nonceSize        = 12
)

var (
	encKey [keySize]byte
	idxKey [keySize]byte
	ready  atomic.Bool
)

var errNotConfigured = errors.New("crypto: PII_ENCRYPTION_KEY belum dikonfigurasi")

// Configure menyetel kunci enkripsi PII (tepat 32 byte) dan menurunkan kunci
// blind index. Aman dipanggil sekali saat startup.
func Configure(key []byte) error {
	if len(key) != keySize {
		return fmt.Errorf("crypto: kunci enkripsi PII harus tepat %d byte (PII_ENCRYPTION_KEY base64, %d byte ter-decode), dapat %d byte", keySize, keySize, len(key))
	}
	copy(encKey[:], key)
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte("simpeg/pii-blind-index/v1"))
	copy(idxKey[:], mac.Sum(nil))
	ready.Store(true)
	return nil
}

// Ready melapor apakah kunci sudah dikonfigurasi.
func Ready() bool { return ready.Load() }

// Encrypt mengenkripsi plaintext dengan nonce acak dan mengembalikan
// ciphertext ber-prefix CiphertextPrefix.
func Encrypt(plaintext string) (string, error) {
	if !ready.Load() {
		return "", errNotConfigured
	}
	if plaintext == "" {
		return "", errors.New("crypto: teks kosong tidak dapat dienkripsi")
	}
	gcm, err := gcmFor(encKey[:])
	if err != nil {
		return "", err
	}
	nonce := make([]byte, nonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("crypto: gagal membangkitkan nonce: %w", err)
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return CiphertextPrefix + base64.RawURLEncoding.EncodeToString(sealed), nil
}

// Encrypted adalah string yang disimpan terenkripsi pada kolom database.
// Di memori nilai selalu plaintext; Value() mengenkripsi saat tulis,
// Scan() mendekripsi saat baca.
type Encrypted string

// Value mengimplementasikan driver.Valuer: menyimpan nilai terenkripsi.
func (e Encrypted) Value() (driver.Value, error) {
	if e == "" {
		return nil, nil
	}
	if strings.HasPrefix(string(e), CiphertextPrefix) {
		return nil, errors.New("crypto: Encrypted berisi ciphertext; isi selalu plaintext di memori")
	}
	return Encrypt(string(e))
}

// Scan mengimplementasikan sql.Scanner: mendekripsi nilai dari database.
// Nilai NULL mengisi Encrypted kosong. Nilai non-'enc1:' ditolak — data yang
// tidak pernah dimigrasikan tidak boleh tampak seperti plaintext valid.
func (e *Encrypted) Scan(src any) error {
	switch value := src.(type) {
	case nil:
		*e = ""
		return nil
	case string:
		return e.decrypt(value)
	case []byte:
		return e.decrypt(string(value))
	default:
		return fmt.Errorf("crypto: tipe tak didukung untuk kolom PII: %T", src)
	}
}

func (e *Encrypted) decrypt(s string) error {
	if !ready.Load() {
		return errNotConfigured
	}
	if !strings.HasPrefix(s, CiphertextPrefix) {
		return fmt.Errorf("crypto: nilai PII di database tidak terenkripsi (prefix %q); terapkan migrasi enkripsi", CiphertextPrefix)
	}
	raw, err := base64.RawURLEncoding.DecodeString(s[len(CiphertextPrefix):])
	if err != nil {
		return fmt.Errorf("crypto: ciphertext PII tidak valid (base64): %w", err)
	}
	if len(raw) <= nonceSize {
		return fmt.Errorf("crypto: ciphertext PII terlalu pendek")
	}
	gcm, err := gcmFor(encKey[:])
	if err != nil {
		return err
	}
	plaintext, err := gcm.Open(nil, raw[:nonceSize], raw[nonceSize:], nil)
	if err != nil {
		return fmt.Errorf("crypto: gagal mendekripsi PII (kunci salah atau data korup)")
	}
	*e = Encrypted(plaintext)
	return nil
}

func gcmFor(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("crypto: gagal membuat cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: gagal membuat GCM: %w", err)
	}
	return gcm, nil
}

// NormalizeIdentity menormalkan identitas untuk blind index: buang spasi,
// titik, strip, dan garis bawah, lalu huruf besar. NIK/NIP/NPWP semuanya
// digit sehingga variasi format (mis. '01.234.567.8-901.000') cocok.
func NormalizeIdentity(s string) string {
	return strings.ToUpper(stripper.Replace(strings.TrimSpace(s)))
}

var stripper = strings.NewReplacer(
	" ", "", "\t", "", ".", "", "-", "", "_", "",
)

// BlindIndex menghitung hash HMAC-SHA256 untuk bentuk ternormalisasi dari
// plaintext. Deterministik: nilai yang sama menghasilkan hash yang sama.
// Hash tidak dapat dibalik menjadi plaintext.
func BlindIndex(plaintext string) (string, error) {
	if !ready.Load() {
		return "", errNotConfigured
	}
	normalized := NormalizeIdentity(plaintext)
	if normalized == "" {
		return "", errors.New("crypto: blind index tidak dapat dihitung dari nilai kosong")
	}
	mac := hmac.New(sha256.New, idxKey[:])
	_, _ = mac.Write([]byte(normalized))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

// BlindIndexPtr menghitung blind index untuk field *Encrypted (nil-aman).
func BlindIndexPtr(e *Encrypted) (*string, error) {
	if e == nil || *e == "" {
		return nil, nil
	}
	hash, err := BlindIndex(string(*e))
	if err != nil {
		return nil, err
	}
	return &hash, nil
}

// Deref mengembalikan plaintext dari *Encrypted ("" untuk nil/kosong).
func Deref(e *Encrypted) string {
	if e == nil {
		return ""
	}
	return string(*e)
}

// PtrString mengekstrak plaintext sebagai *string untuk DTO (nil-aman).
func PtrString(e *Encrypted) *string {
	if e == nil || *e == "" {
		return nil
	}
	s := string(*e)
	return &s
}

// PtrEncrypted membungkus *string sebagai *Encrypted plaintext (nil-aman).
// Enkripsi terjadi di Value() saat GORM menulis.
func PtrEncrypted(s *string) *Encrypted {
	if s == nil || strings.TrimSpace(*s) == "" {
		return nil
	}
	e := Encrypted(*s)
	return &e
}
