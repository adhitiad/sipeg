// Package domain memuat aturan bisnis honor mengajar per SKS (dosen LB).
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

// CreateHonorRate memuat tarif honor per SKS baru.
type CreateHonorRate struct {
	BerlakuMulai time.Time
	PerSks       float64
}

// Validate memeriksa aturan tarif honor baru.
func (input CreateHonorRate) Validate() error {
	if input.BerlakuMulai.IsZero() {
		return ErrValidation
	}
	if input.PerSks < 0 {
		return ErrValidation
	}
	return nil
}

// HonorRatePatch memuat perubahan tarif honor per SKS.
type HonorRatePatch struct {
	BerlakuMulai domain.Field
	PerSks       domain.Field
	Aktif        domain.Field
}

var HonorRatePatchFields = map[string]bool{
	"berlakuMulai": true, "perSks": true, "aktif": true,
}

// NewHonorRatePatch menyusun patch dari peta field.
func NewHonorRatePatch(fields map[string]domain.Field) HonorRatePatch {
	return HonorRatePatch{
		BerlakuMulai: fields["berlakuMulai"],
		PerSks:       fields["perSks"],
		Aktif:        fields["aktif"],
	}
}

// Validate memeriksa aturan perubahan tarif honor.
func (patch HonorRatePatch) Validate() error {
	if patch.BerlakuMulai.Present && patch.BerlakuMulai.Null {
		return ErrValidation
	}
	if patch.BerlakuMulai.Present {
		if v := strings.TrimSpace(patch.BerlakuMulai.Value); v == "" {
			return ErrValidation
		} else if _, err := time.Parse("2006-01-02", v); err != nil {
			return ErrValidation
		}
	}
	if patch.PerSks.Present && patch.PerSks.Null {
		return ErrValidation
	}
	if patch.PerSks.Present {
		v := strings.TrimSpace(patch.PerSks.Value)
		if v == "" {
			return ErrValidation
		}
		amount, err := parseFloat(v)
		if err != nil || amount < 0 {
			return ErrValidation
		}
	}
	if patch.Aktif.Present && patch.Aktif.Null {
		return ErrValidation
	}
	return nil
}

// HasChanges melaporkan apakah patch memuat setidaknya satu perubahan.
func (patch HonorRatePatch) HasChanges() bool {
	return patch.BerlakuMulai.Present || patch.PerSks.Present || patch.Aktif.Present
}

// CreateHonorPeriode memuat beban mengajar seorang dosen LB untuk satu periode.
type CreateHonorPeriode struct {
	UserId            string
	PeriodeAkademikId string
	Sks               float64
	RatePerSks        float64
	SkId              *string
}

// Validate memeriksa aturan beban mengajar baru.
func (input CreateHonorPeriode) Validate() error {
	if !validUUID(input.UserId) || !validUUID(input.PeriodeAkademikId) {
		return ErrValidation
	}
	if input.Sks < 0 || input.Sks > 200 {
		return ErrValidation
	}
	if input.RatePerSks < 0 {
		return ErrValidation
	}
	if input.SkId != nil && !validUUID(*input.SkId) {
		return ErrValidation
	}
	return nil
}

// TotalHonor melaporkan total honor per SKS × jumlah SKS (Rp penuh).
func TotalHonor(sks, ratePerSks float64) float64 {
	return sks * ratePerSks
}

// HonorPeriodePatch memuat perubahan beban mengajar.
type HonorPeriodePatch struct {
	Sks        domain.Field
	RatePerSks domain.Field
	SkId       domain.Field
}

var HonorPeriodePatchFields = map[string]bool{
	"sks": true, "ratePerSks": true, "skId": true,
}

// NewHonorPeriodePatch menyusun patch dari peta field.
func NewHonorPeriodePatch(fields map[string]domain.Field) HonorPeriodePatch {
	return HonorPeriodePatch{
		Sks:        fields["sks"],
		RatePerSks: fields["ratePerSks"],
		SkId:       fields["skId"],
	}
}

// Validate memeriksa aturan perubahan beban mengajar.
func (patch HonorPeriodePatch) Validate() error {
	if patch.Sks.Present && patch.Sks.Null {
		return ErrValidation
	}
	if patch.Sks.Present {
		v, err := parseFloat(strings.TrimSpace(patch.Sks.Value))
		if err != nil || v < 0 || v > 200 {
			return ErrValidation
		}
	}
	if patch.RatePerSks.Present && patch.RatePerSks.Null {
		return ErrValidation
	}
	if patch.RatePerSks.Present {
		v, err := parseFloat(strings.TrimSpace(patch.RatePerSks.Value))
		if err != nil || v < 0 {
			return ErrValidation
		}
	}
	if patch.SkId.Present && patch.SkId.Value == "" {
		return ErrValidation
	}
	return nil
}

// HasChanges melaporkan apakah patch memuat setidaknya satu perubahan.
func (patch HonorPeriodePatch) HasChanges() bool {
	return patch.Sks.Present || patch.RatePerSks.Present || patch.SkId.Present
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

func parseFloat(value string) (float64, error) {
	return strconv.ParseFloat(value, 64)
}
