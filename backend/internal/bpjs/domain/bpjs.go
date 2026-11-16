// Package domain memuat aturan bisnis BPJS Kesehatan/Ketenagakerjaan.
package domain

import (
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

// CreateKonfigurasi memuat tarif iuran BPJS baru untuk periode berlaku.
type CreateKonfigurasi struct {
	Jenis                string
	IuranPegawaiPct      float64
	IuranPemberiKerjaPct float64
	Dasar                string
	BatasAtas            *float64
	BerlakuMulai         time.Time
}

var DasarSesuai = []string{"GAJI_POKOK", "TOTAL_GAJI"}

// Validate memeriksa aturan tarif iuran BPJS baru.
func (input CreateKonfigurasi) Validate() error {
	if !validJenis(input.Jenis) {
		return ErrValidation
	}
	if input.IuranPegawaiPct < 0 || input.IuranPegawaiPct > 100 {
		return ErrValidation
	}
	if input.IuranPemberiKerjaPct < 0 || input.IuranPemberiKerjaPct > 100 {
		return ErrValidation
	}
	if !validDasar(input.Dasar) {
		return ErrValidation
	}
	if input.BatasAtas != nil && *input.BatasAtas < 0 {
		return ErrValidation
	}
	if input.BerlakuMulai.IsZero() {
		return ErrValidation
	}
	return nil
}

func validJenis(value string) bool {
	switch value {
	case "KESEHATAN", "KETENAGAKERJAAN":
		return true
	}
	return false
}

func validDasar(value string) bool {
	for _, candidate := range DasarSesuai {
		if candidate == value {
			return true
		}
	}
	return false
}

// KonfigurasiPatch memuat perubahan tarif iuran BPJS.
type KonfigurasiPatch struct {
	IuranPegawaiPct      domain.Field
	IuranPemberiKerjaPct domain.Field
	Dasar                domain.Field
	BatasAtas            domain.Field
	BerlakuMulai         domain.Field
	Aktif                domain.Field
}

var KonfigurasiPatchFields = map[string]bool{
	"iuranPegawaiPct": true, "iuranPemberiKerjaPct": true, "dasar": true,
	"batasAtas": true, "berlakuMulai": true, "aktif": true,
}

// NewKonfigurasiPatch menyusun patch dari peta field.
func NewKonfigurasiPatch(fields map[string]domain.Field) KonfigurasiPatch {
	return KonfigurasiPatch{
		IuranPegawaiPct:      fields["iuranPegawaiPct"],
		IuranPemberiKerjaPct: fields["iuranPemberiKerjaPct"],
		Dasar:                fields["dasar"],
		BatasAtas:            fields["batasAtas"],
		BerlakuMulai:         fields["berlakuMulai"],
		Aktif:                fields["aktif"],
	}
}

// Validate memeriksa aturan perubahan tarif iuran BPJS.
func (patch KonfigurasiPatch) Validate() error {
	if patch.IuranPegawaiPct.Present && patch.IuranPegawaiPct.Null {
		return ErrValidation
	}
	if patch.IuranPemberiKerjaPct.Present && patch.IuranPemberiKerjaPct.Null {
		return ErrValidation
	}
	if patch.Dasar.Present && patch.Dasar.Null {
		return ErrValidation
	}
	if patch.Dasar.Present {
		if !validDasar(patch.Dasar.Value) {
			return ErrValidation
		}
	}
	if patch.BerlakuMulai.Present && patch.BerlakuMulai.Null {
		return ErrValidation
	}
	if patch.BerlakuMulai.Present {
		v := strings.TrimSpace(patch.BerlakuMulai.Value)
		if v == "" {
			return ErrValidation
		}
		if _, err := time.Parse("2006-01-02", v); err != nil {
			return ErrValidation
		}
	}
	if patch.Aktif.Present && patch.Aktif.Null {
		return ErrValidation
	}
	return nil
}

// HasChanges melaporkan apakah patch memuat setidaknya satu perubahan.
func (patch KonfigurasiPatch) HasChanges() bool {
	return patch.IuranPegawaiPct.Present || patch.IuranPemberiKerjaPct.Present || patch.Dasar.Present ||
		patch.BatasAtas.Present || patch.BerlakuMulai.Present || patch.Aktif.Present
}

// CreatePeserta memuat kepesertaan BPJS baru untuk satu pegawai.
type CreatePeserta struct {
	UserId           string
	Jenis            string
	NomorKepesertaan *string
	Mulai            time.Time
	Selesai          *time.Time
}

// Validate memeriksa aturan kepesertaan BPJS baru.
func (input CreatePeserta) Validate() error {
	if !validUUID(input.UserId) {
		return ErrValidation
	}
	if !validJenis(input.Jenis) {
		return ErrValidation
	}
	if input.Mulai.IsZero() {
		return ErrValidation
	}
	if input.Selesai != nil && input.Selesai.Before(input.Mulai) {
		return ErrValidation
	}
	return nil
}

// PesertaPatch memuat perubahan kepesertaan BPJS.
type PesertaPatch struct {
	NomorKepesertaan domain.Field
	Mulai            domain.Field
	Selesai          domain.Field
	Aktif            domain.Field
}

var PesertaPatchFields = map[string]bool{
	"nomorKepesertaan": true, "mulai": true, "selesai": true, "aktif": true,
}

// NewPesertaPatch menyusun patch dari peta field.
func NewPesertaPatch(fields map[string]domain.Field) PesertaPatch {
	return PesertaPatch{
		NomorKepesertaan: fields["nomorKepesertaan"],
		Mulai:            fields["mulai"],
		Selesai:          fields["selesai"],
		Aktif:            fields["aktif"],
	}
}

// Validate memeriksa aturan perubahan kepesertaan BPJS.
func (patch PesertaPatch) Validate() error {
	if patch.Selesai.Present && patch.Mulai.Present && !patch.Selesai.Null && !patch.Mulai.Null {
		start, err1 := time.Parse("2006-01-02", strings.TrimSpace(patch.Mulai.Value))
		end, err2 := time.Parse("2006-01-02", strings.TrimSpace(patch.Selesai.Value))
		if err1 == nil && err2 == nil && end.Before(start) {
			return ErrValidation
		}
	}
	return nil
}

// HasChanges melaporkan apakah patch memuat setidaknya satu perubahan.
func (patch PesertaPatch) HasChanges() bool {
	return patch.NomorKepesertaan.Present || patch.Mulai.Present || patch.Selesai.Present || patch.Aktif.Present
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
