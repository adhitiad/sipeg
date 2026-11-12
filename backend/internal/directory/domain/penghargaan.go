// Package domain memuat aturan bisnis untuk riwayat penghargaan pegawai.
package domain

import (
	"strings"

	"simpeg/backend/internal/models"
)

// JenisPenghargaanValid melaporkan apakah jenis penghargaan valid.
func JenisPenghargaanValid(value models.JenisPenghargaan) bool {
	switch value {
	case models.PenghargaanDosenTeladan, models.PenghargaanSatyalancana, models.PenghargaanLainnya:
		return true
	}
	return false
}

// CreateRiwayatPenghargaan memuat data untuk membuat catatan riwayat
// penghargaan pegawai.
type CreateRiwayatPenghargaan struct {
	UserID           string
	NamaPenghargaan  string
	JenisPenghargaan models.JenisPenghargaan
	Penyelenggara    string
	TanggalTerima    string // "YYYY-MM-DD"
	NomorSk          *string
}

// Validate memeriksa aturan data riwayat penghargaan.
func (input CreateRiwayatPenghargaan) Validate() error {
	if strings.TrimSpace(input.NamaPenghargaan) == "" || TextLength(input.NamaPenghargaan) > MaxNameLength {
		return ErrValidation
	}
	if !JenisPenghargaanValid(input.JenisPenghargaan) {
		return ErrValidation
	}
	if strings.TrimSpace(input.Penyelenggara) == "" || TextLength(input.Penyelenggara) > MaxNameLength {
		return ErrValidation
	}
	if !validateDate(input.TanggalTerima) {
		return ErrValidation
	}
	if input.NomorSk != nil && TextLength(strings.TrimSpace(*input.NomorSk)) > MaxPositionLength {
		return ErrValidation
	}
	return nil
}

// PenghargaanPatchFields adalah daftar field yang boleh diubah.
var PenghargaanPatchFields = map[string]bool{
	"namaPenghargaan": true, "jenisPenghargaan": true, "penyelenggara": true,
	"tanggalTerima": true, "nomorSk": true,
}

// RiwayatPenghargaanPatch memuat perubahan catatan riwayat penghargaan.
type RiwayatPenghargaanPatch struct {
	NamaPenghargaan  Field
	JenisPenghargaan Field
	Penyelenggara    Field
	TanggalTerima    Field
	NomorSk          Field
}

// NewRiwayatPenghargaanPatch menyusun patch dari peta field.
func NewRiwayatPenghargaanPatch(fields map[string]Field) RiwayatPenghargaanPatch {
	return RiwayatPenghargaanPatch{
		NamaPenghargaan:  fields["namaPenghargaan"],
		JenisPenghargaan: fields["jenisPenghargaan"],
		Penyelenggara:    fields["penyelenggara"],
		TanggalTerima:    fields["tanggalTerima"],
		NomorSk:          fields["nomorSk"],
	}
}

// Validate memeriksa aturan perubahan riwayat penghargaan.
func (patch RiwayatPenghargaanPatch) Validate() error {
	if patch.NamaPenghargaan.Present {
		if patch.NamaPenghargaan.Null || TextLength(strings.TrimSpace(patch.NamaPenghargaan.Value)) == 0 ||
			TextLength(patch.NamaPenghargaan.Value) > MaxNameLength {
			return ErrValidation
		}
	}
	if patch.JenisPenghargaan.Present && !patch.JenisPenghargaan.Null {
		if !JenisPenghargaanValid(models.JenisPenghargaan(patch.JenisPenghargaan.Value)) {
			return ErrValidation
		}
	}
	if patch.Penyelenggara.Present && !patch.Penyelenggara.Null {
		if TextLength(patch.Penyelenggara.Value) > MaxNameLength {
			return ErrValidation
		}
	}
	if patch.TanggalTerima.Present && !patch.TanggalTerima.Null {
		if !validateDate(patch.TanggalTerima.Value) {
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
func (patch RiwayatPenghargaanPatch) HasChanges() bool {
	return patch.NamaPenghargaan.Present || patch.JenisPenghargaan.Present ||
		patch.Penyelenggara.Present || patch.TanggalTerima.Present ||
		patch.NomorSk.Present
}
