// Package domain memuat aturan bisnis PPh 21 skema TER (PMK 168/2023).
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

// CreateTer memuat satu baris tarif TER baru.
type CreateTer struct {
	Kategori     string
	Lapisan      int
	BatasBawah   float64
	BatasAtas    *float64
	TarifPct     float64
	BerlakuMulai time.Time
}

// Validate memeriksa aturan baris tarif TER baru.
func (input CreateTer) Validate() error {
	if !validKategori(input.Kategori) {
		return ErrValidation
	}
	if input.Lapisan < 1 || input.Lapisan > 999 {
		return ErrValidation
	}
	if input.BatasBawah < 0 {
		return ErrValidation
	}
	if input.BatasAtas != nil && *input.BatasAtas <= input.BatasBawah {
		return ErrValidation
	}
	if input.TarifPct < 0 || input.TarifPct > 100 {
		return ErrValidation
	}
	if input.BerlakuMulai.IsZero() {
		return ErrValidation
	}
	return nil
}

func validKategori(value string) bool {
	switch value {
	case "A", "B", "C":
		return true
	}
	return false
}

// TerPatch memuat perubahan satu baris tarif TER.
type TerPatch struct {
	Kategori     domain.Field
	Lapisan      domain.Field
	BatasBawah   domain.Field
	BatasAtas    domain.Field
	TarifPct     domain.Field
	BerlakuMulai domain.Field
}

var TerPatchFields = map[string]bool{
	"kategori": true, "lapisan": true, "batasBawah": true,
	"batasAtas": true, "tarifPct": true, "berlakuMulai": true,
}

// NewTerPatch menyusun patch dari peta field.
func NewTerPatch(fields map[string]domain.Field) TerPatch {
	return TerPatch{
		Kategori:     fields["kategori"],
		Lapisan:      fields["lapisan"],
		BatasBawah:   fields["batasBawah"],
		BatasAtas:    fields["batasAtas"],
		TarifPct:     fields["tarifPct"],
		BerlakuMulai: fields["berlakuMulai"],
	}
}

// Validate memeriksa aturan perubahan baris tarif TER.
func (patch TerPatch) Validate() error {
	if patch.Kategori.Present {
		if !validKategori(patch.Kategori.Value) {
			return ErrValidation
		}
	}
	if patch.Lapisan.Present {
		if v, err := parseInt(patch.Lapisan.Value); err != nil || v < 1 || v > 999 {
			return ErrValidation
		}
	}
	if patch.BatasBawah.Present {
		v, err := parseFloat(patch.BatasBawah.Value)
		if err != nil || v < 0 {
			return ErrValidation
		}
	}
	if patch.TarifPct.Present {
		v, err := parseFloat(patch.TarifPct.Value)
		if err != nil || v < 0 || v > 100 {
			return ErrValidation
		}
	}
	if patch.BerlakuMulai.Present {
		if _, err := time.Parse("2006-01-02", strings.TrimSpace(patch.BerlakuMulai.Value)); err != nil {
			return ErrValidation
		}
	}
	return nil
}

// HasChanges melaporkan apakah patch memuat setidaknya satu perubahan.
func (patch TerPatch) HasChanges() bool {
	return patch.Kategori.Present || patch.Lapisan.Present || patch.BatasBawah.Present ||
		patch.BatasAtas.Present || patch.TarifPct.Present || patch.BerlakuMulai.Present
}

// TerRecord adalah bentuk baca baris tarif TER.
type TerRecord struct {
	ID           string
	Kategori     string
	Lapisan      int
	BatasBawah   float64
	BatasAtas    *float64
	TarifPct     float64
	BerlakuMulai time.Time
}

// ResolveTarif mengembalikan tarif efektif (dalam persen) untuk kategori dan
// penghasilan bruto bulanan tertentu. Mengembalikan ErrNotFound bila tidak ada
// lapisan yang cocok.
func ResolveTarif(rows []TerRecord, kategori string, grossMonthly float64) (TerRecord, error) {
	if !validKategori(kategori) {
		return TerRecord{}, ErrValidation
	}
	var latest *TerRecord
	for i := range rows {
		row := rows[i]
		if row.Kategori != kategori {
			continue
		}
		if grossMonthly < row.BatasBawah {
			continue
		}
		if row.BatasAtas != nil && grossMonthly >= *row.BatasAtas {
			continue
		}
		if latest == nil || row.BerlakuMulai.After(latest.BerlakuMulai) {
			latest = &rows[i]
		} else if latest != nil && row.BerlakuMulai.Equal(latest.BerlakuMulai) && row.Lapisan > latest.Lapisan {
			latest = &rows[i]
		}
	}
	if latest == nil {
		return TerRecord{}, ErrNotFound
	}
	return *latest, nil
}

func parseFloat(value string) (float64, error) {
	return strconv.ParseFloat(strings.TrimSpace(value), 64)
}

func parseInt(value string) (int, error) {
	return strconv.Atoi(strings.TrimSpace(value))
}
