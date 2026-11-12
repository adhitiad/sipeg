// Package domain memuat aturan bisnis untuk riwayat golongan pegawai.
package domain

import "strings"

// GolonganValid melaporkan apakah string golongan tidak kosong.
func GolonganValid(value string) bool {
	return len(strings.TrimSpace(value)) > 0
}

// CreateRiwayatGolongan memuat data untuk membuat catatan riwayat golongan
// pegawai.
type CreateRiwayatGolongan struct {
	UserID      string
	Golongan    string
	Ruang       string
	TMT         string // "YYYY-MM-DD"
	AngkaKredit int
	NomorSk     *string
}

// Validate memeriksa aturan data riwayat golongan.
func (input CreateRiwayatGolongan) Validate() error {
	if !GolonganValid(input.Golongan) || TextLength(input.Golongan) > 20 {
		return ErrValidation
	}
	if !GolonganValid(input.Ruang) || TextLength(input.Ruang) > 10 {
		return ErrValidation
	}
	if !validateDate(input.TMT) {
		return ErrValidation
	}
	if input.AngkaKredit < 0 {
		return ErrValidation
	}
	if input.NomorSk != nil && TextLength(strings.TrimSpace(*input.NomorSk)) > MaxPositionLength {
		return ErrValidation
	}
	return nil
}

// GolonganPatchFields adalah daftar field yang boleh diubah.
var GolonganPatchFields = map[string]bool{
	"golongan": true, "ruang": true, "tmt": true, "angkaKredit": true, "nomorSk": true,
}

// RiwayatGolonganPatch memuat perubahan catatan riwayat golongan.
type RiwayatGolonganPatch struct {
	Golongan    Field
	Ruang       Field
	TMT         Field
	AngkaKredit Field
	NomorSk     Field
}

// NewRiwayatGolonganPatch menyusun patch dari peta field.
func NewRiwayatGolonganPatch(fields map[string]Field) RiwayatGolonganPatch {
	return RiwayatGolonganPatch{
		Golongan:    fields["golongan"],
		Ruang:       fields["ruang"],
		TMT:         fields["tmt"],
		AngkaKredit: fields["angkaKredit"],
		NomorSk:     fields["nomorSk"],
	}
}

// Validate memeriksa aturan perubahan riwayat golongan.
func (patch RiwayatGolonganPatch) Validate() error {
	if patch.Golongan.Present && !patch.Golongan.Null {
		if !GolonganValid(patch.Golongan.Value) || TextLength(patch.Golongan.Value) > 20 {
			return ErrValidation
		}
	}
	if patch.Ruang.Present && !patch.Ruang.Null {
		if !GolonganValid(patch.Ruang.Value) || TextLength(patch.Ruang.Value) > 10 {
			return ErrValidation
		}
	}
	if patch.TMT.Present && !patch.TMT.Null {
		if !validateDate(patch.TMT.Value) {
			return ErrValidation
		}
	}
	if patch.AngkaKredit.Present && !patch.AngkaKredit.Null {
		if patch.AngkaKredit.Value == "" {
			return ErrValidation
		}
	}
	if patch.NomorSk.Present && !patch.NomorSk.Null {
		if TextLength(patch.NomorSk.Value) > MaxPositionLength {
			return ErrValidation
		}
	}
	return nil
}

// HasChanges melaporkan apakah patch memuat setidaknya satu perubahan.
func (patch RiwayatGolonganPatch) HasChanges() bool {
	return patch.Golongan.Present || patch.Ruang.Present || patch.TMT.Present ||
		patch.AngkaKredit.Present || patch.NomorSk.Present
}
