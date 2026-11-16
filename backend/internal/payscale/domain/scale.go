// Package domain memuat aturan bisnis master skala gaji pokok yayasan.
//
// Lapisan ini tidak melakukan akses database maupun HTTP. Aturan di sini
// menjadi satu-satunya sumber kebenaran atas validitas data yang dipakai
// oleh lapisan aplikasi (logika use case) dan lapisan transport.
package domain

import (
	"strconv"
	"strings"
	"time"

	"simpeg/backend/internal/directory/domain"
)

var (
	ErrNotFound   = domain.ErrNotFound
	ErrConflict   = domain.ErrConflict
	ErrValidation = domain.ErrValidation
	ErrForbidden  = domain.ErrForbidden
)

// SkalaSortFields adalah allowlist kolom sort untuk daftar skala gaji.
var SkalaSortFields = []string{"kode", "nama", "berlakuMulai", "berlakuSelesai", "aktif", "createdAt", "updatedAt"}

// SkalaSortColumns memetakan kunci sort ke kolom SQL (hanya allowlist).
var SkalaSortColumns = map[string]string{
	"kode":           `"kode"`,
	"nama":           `"nama"`,
	"berlakuMulai":   `"berlakuMulai"`,
	"berlakuSelesai": `"berlakuSelesai"`,
	"aktif":          `"aktif"`,
	"createdAt":      `"createdAt"`,
	"updatedAt":      `"updatedAt"`,
}

// CreateSkalaGaji memuat data untuk membuat skala gaji baru.
type CreateSkalaGaji struct {
	Kode           string
	Nama           string
	BerlakuMulai   time.Time
	BerlakuSelesai *time.Time
}

// Validate memeriksa aturan data skala gaji baru.
func (input CreateSkalaGaji) Validate() error {
	input.Kode = strings.TrimSpace(input.Kode)
	input.Nama = strings.TrimSpace(input.Nama)
	if input.Kode == "" || len(input.Kode) > 50 {
		return ErrValidation
	}
	if input.Nama == "" || len(input.Nama) > 200 {
		return ErrValidation
	}
	if input.BerlakuMulai.IsZero() {
		return ErrValidation
	}
	if input.BerlakuSelesai != nil && input.BerlakuSelesai.Before(input.BerlakuMulai) {
		return ErrValidation
	}
	return nil
}

// SkalaGajiPatch memuat perubahan skala gaji.
type SkalaGajiPatch struct {
	Kode           domain.Field
	Nama           domain.Field
	BerlakuMulai   domain.Field
	BerlakuSelesai domain.Field
	Aktif          domain.Field
}

// SkalaGajiPatchFields adalah daftar field yang boleh diubah.
var SkalaGajiPatchFields = map[string]bool{
	"kode": true, "nama": true, "berlakuMulai": true, "berlakuSelesai": true, "aktif": true,
}

// NewSkalaGajiPatch menyusun patch dari peta field.
func NewSkalaGajiPatch(fields map[string]domain.Field) SkalaGajiPatch {
	return SkalaGajiPatch{
		Kode:           fields["kode"],
		Nama:           fields["nama"],
		BerlakuMulai:   fields["berlakuMulai"],
		BerlakuSelesai: fields["berlakuSelesai"],
		Aktif:          fields["aktif"],
	}
}

// Validate memeriksa aturan perubahan skala gaji.
func (patch SkalaGajiPatch) Validate() error {
	if patch.Kode.Present && !patch.Kode.Null {
		v := strings.TrimSpace(patch.Kode.Value)
		if v == "" || len(v) > 50 {
			return ErrValidation
		}
	}
	if patch.Nama.Present && !patch.Nama.Null {
		v := strings.TrimSpace(patch.Nama.Value)
		if v == "" || len(v) > 200 {
			return ErrValidation
		}
	}
	if patch.BerlakuMulai.Present && patch.BerlakuMulai.Null {
		return ErrValidation
	}
	if patch.BerlakuSelesai.Present && patch.BerlakuSelesai.Null {
		// nilai kosong eksplisit diperbolehkan untuk membersihkan tanggal akhir
	}
	if !validDateField(patch.BerlakuMulai) || !validDateField(patch.BerlakuSelesai) {
		return ErrValidation
	}
	if patch.Aktif.Present && patch.Aktif.Null {
		return ErrValidation
	}
	return nil
}

// validDateField melaporkan apakah nilai tanggal sah (`YYYY-MM-DD` atau
// nilai null untuk berlakuSelesai, atau field tidak diisi sama sekali).
func validDateField(field domain.Field) bool {
	if !field.Present || field.Null {
		return true
	}
	v := strings.TrimSpace(field.Value)
	if v == "" {
		return true
	}
	_, err := time.Parse("2006-01-02", v)
	return err == nil
}

// HasChanges melaporkan apakah patch memuat setidaknya satu perubahan.
func (patch SkalaGajiPatch) HasChanges() bool {
	return patch.Kode.Present || patch.Nama.Present || patch.BerlakuMulai.Present || patch.BerlakuSelesai.Present || patch.Aktif.Present
}

// CreateSkalaGajiDetail memuat data untuk membuat baris nominal.
type CreateSkalaGajiDetail struct {
	SkalaGajiId      string
	KategoriStatusId string
	Tingkat          int
	Nominal          float64
}

// Validate memeriksa aturan data baris nominal baru.
func (input CreateSkalaGajiDetail) Validate() error {
	if !validUUID(input.SkalaGajiId) || !validUUID(input.KategoriStatusId) {
		return ErrValidation
	}
	if input.Tingkat < 0 || input.Tingkat > 99 {
		return ErrValidation
	}
	if input.Nominal < 0 {
		return ErrValidation
	}
	return nil
}

// SkalaGajiDetailPatch memuat perubahan baris nominal.
type SkalaGajiDetailPatch struct {
	KategoriStatusId domain.Field
	Tingkat          domain.Field
	Nominal          domain.Field
}

var SkalaGajiDetailPatchFields = map[string]bool{
	"KategoriStatusId": true, "tingkat": true, "nominal": true,
}

// NewSkalaGajiDetailPatch menyusun patch dari peta field.
func NewSkalaGajiDetailPatch(fields map[string]domain.Field) SkalaGajiDetailPatch {
	return SkalaGajiDetailPatch{
		KategoriStatusId: fields["KategoriStatusId"],
		Tingkat:          fields["tingkat"],
		Nominal:          fields["nominal"],
	}
}

// Validate memeriksa aturan perubahan baris nominal.
func (patch SkalaGajiDetailPatch) Validate() error {
	if patch.KategoriStatusId.Present && patch.KategoriStatusId.Null {
		return ErrValidation
	}
	if patch.KategoriStatusId.Present {
		if !validUUID(patch.KategoriStatusId.Value) {
			return ErrValidation
		}
	}
	if patch.Tingkat.Present && patch.Tingkat.Null {
		return ErrValidation
	}
	if patch.Tingkat.Present {
		v, err := strconv.Atoi(strings.TrimSpace(patch.Tingkat.Value))
		if err != nil || v < 0 || v > 99 {
			return ErrValidation
		}
	}
	if patch.Nominal.Present && patch.Nominal.Null {
		return ErrValidation
	}
	if patch.Nominal.Present {
		v, err := strconv.ParseFloat(strings.TrimSpace(patch.Nominal.Value), 64)
		if err != nil || v < 0 {
			return ErrValidation
		}
	}
	return nil
}

// HasChanges melaporkan apakah patch memuat setidaknya satu perubahan.
func (patch SkalaGajiDetailPatch) HasChanges() bool {
	return patch.KategoriStatusId.Present || patch.Tingkat.Present || patch.Nominal.Present
}

// SkalaGajiWithDetails memuat skala gaji beserta baris nominalnya.
type SkalaGajiWithDetails struct {
	Skala   SkalaGajiRecord
	Details []SkalaGajiDetailRecord
}

// SkalaGajiRecord adalah bentuk baca skala gaji.
type SkalaGajiRecord struct {
	ID             string
	Kode           string
	Nama           string
	BerlakuMulai   time.Time
	BerlakuSelesai *time.Time
	Aktif          bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// SkalaGajiDetailRecord adalah bentuk baca baris nominal.
type SkalaGajiDetailRecord struct {
	ID               string
	SkalaGajiId      string
	KategoriStatusId string
	KategoriKode     string
	KategoriNama     string
	Tingkat          int
	Nominal          float64
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func validUUID(value string) bool {
	if len(value) != 36 {
		return false
	}
	for i, r := range value {
		switch {
		case i == 8 || i == 13 || i == 18 || i == 23:
			if r != '-' {
				return false
			}
		default:
			isHex := (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
			if !isHex {
				return false
			}
		}
	}
	return true
}
