// Package domain memuat aturan bisnis untuk komponen tunjangan pegawai.
package domain

import (
	"simpeg/backend/internal/directory/domain"
)

// JenisTunjanganValid melaporkan apakah jenis tunjangan valid.
func JenisTunjanganValid(value string) bool {
	switch value {
	case "STRUKTURAL", "FUNGSIONAL", "BERAS", "MAKANAN", "TRANSPORTASI":
		return true
	}
	return false
}

// CreateKomponenTunjangan memuat data untuk membuat entri komponen tunjangan.
type CreateKomponenTunjangan struct {
	UserID         string
	JenisTunjangan string
	Nilai          float64
	Aktif          bool
}

// Validate memeriksa aturan data komponen tunjangan.
func (input CreateKomponenTunjangan) Validate() error {
	if !JenisTunjanganValid(input.JenisTunjangan) {
		return domain.ErrValidation
	}
	if input.Nilai < 0 {
		return domain.ErrValidation
	}
	return nil
}

// KomponenTunjanganPatchFields adalah daftar field yang boleh diubah.
var KomponenTunjanganPatchFields = map[string]bool{
	"jenisTunjangan": true, "nilai": true, "aktif": true,
}

// KomponenTunjanganPatch memuat perubahan entri komponen tunjangan.
type KomponenTunjanganPatch struct {
	JenisTunjanging domain.Field
	Nilai           domain.Field
	Aktif           domain.Field
}

// NewKomponenTunjanganPatch menyusun patch dari peta field.
func NewKomponenTunjanganPatch(fields map[string]domain.Field) KomponenTunjanganPatch {
	return KomponenTunjanganPatch{
		JenisTunjanging: fields["jenisTunjangan"],
		Nilai:           fields["nilai"],
		Aktif:           fields["aktif"],
	}
}

// Validate memeriksa aturan perubahan komponen tunjangan.
func (patch KomponenTunjanganPatch) Validate() error {
	if patch.JenisTunjanging.Present && !patch.JenisTunjanging.Null {
		if !JenisTunjanganValid(patch.JenisTunjanging.Value) {
			return domain.ErrValidation
		}
	}
	if patch.Nilai.Present && !patch.Nilai.Null {
		if patch.Nilai.Value == "" {
			return domain.ErrValidation
		}
	}
	return nil
}

// HasChanges melaporkan apakah patch memuat setidaknya satu perubahan.
func (patch KomponenTunjanganPatch) HasChanges() bool {
	return patch.JenisTunjanging.Present || patch.Nilai.Present || patch.Aktif.Present
}
