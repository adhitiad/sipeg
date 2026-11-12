// Package domain memuat aturan bisnis untuk riwayat pelatihan pegawai.
package domain

import (
	"strings"

	"simpeg/backend/internal/models"
)

// JenisPelatihanValid melaporkan apakah jenis pelatihan valid.
func JenisPelatihanValid(value models.JenisPelatihan) bool {
	switch value {
	case models.JenisPelatihanDiklat, models.JenisPelatihanLokakarya, models.JenisPelatihanSoftSkills, models.JenisPelatihanLainnya:
		return true
	}
	return false
}

// CreateRiwayatPelatihan memuat data untuk membuat catatan riwayat pelatihan
// pegawai.
type CreateRiwayatPelatihan struct {
	UserID             string
	NamaPelatihan      string
	JenisPelatihan     models.JenisPelatihan
	Penyelenggara      string
	TanggalMulai       string // "YYYY-MM-DD"
	TanggalSelesai     string // "YYYY-MM-DD"
	DurasiJam          int
	MemilikiSertifikat bool
}

// Validate memeriksa aturan data riwayat pelatihan.
func (input CreateRiwayatPelatihan) Validate() error {
	if strings.TrimSpace(input.NamaPelatihan) == "" || TextLength(input.NamaPelatihan) > MaxNameLength {
		return ErrValidation
	}
	if !JenisPelatihanValid(input.JenisPelatihan) {
		return ErrValidation
	}
	if strings.TrimSpace(input.Penyelenggara) == "" || TextLength(input.Penyelenggara) > MaxNameLength {
		return ErrValidation
	}
	if !validateDate(input.TanggalMulai) {
		return ErrValidation
	}
	if !validateDate(input.TanggalSelesai) {
		return ErrValidation
	}
	if input.DurasiJam < 0 {
		return ErrValidation
	}
	return nil
}

// PelatihanPatchFields adalah daftar field yang boleh diubah.
var PelatihanPatchFields = map[string]bool{
	"namaPelatihan": true, "jenisPelatihan": true, "penyelenggara": true,
	"tanggalMulai": true, "tanggalSelesai": true, "durasiJam": true,
	"memilikiSertifikat": true,
}

// RiwayatPelatihanPatch memuat perubahan catatan riwayat pelatihan.
type RiwayatPelatihanPatch struct {
	NamaPelatihan      Field
	JenisPelatihan     Field
	Penyelenggara      Field
	TanggalMulai       Field
	TanggalSelesai     Field
	DurasiJam          Field
	MemilikiSertifikat Field
}

// NewRiwayatPelatihanPatch menyusun patch dari peta field.
func NewRiwayatPelatihanPatch(fields map[string]Field) RiwayatPelatihanPatch {
	return RiwayatPelatihanPatch{
		NamaPelatihan:      fields["namaPelatihan"],
		JenisPelatihan:     fields["jenisPelatihan"],
		Penyelenggara:      fields["penyelenggara"],
		TanggalMulai:       fields["tanggalMulai"],
		TanggalSelesai:     fields["tanggalSelesai"],
		DurasiJam:          fields["durasiJam"],
		MemilikiSertifikat: fields["memilikiSertifikat"],
	}
}

// Validate memeriksa aturan perubahan riwayat pelatihan.
func (patch RiwayatPelatihanPatch) Validate() error {
	if patch.NamaPelatihan.Present {
		if patch.NamaPelatihan.Null || TextLength(strings.TrimSpace(patch.NamaPelatihan.Value)) == 0 ||
			TextLength(patch.NamaPelatihan.Value) > MaxNameLength {
			return ErrValidation
		}
	}
	if patch.JenisPelatihan.Present && !patch.JenisPelatihan.Null {
		if !JenisPelatihanValid(models.JenisPelatihan(patch.JenisPelatihan.Value)) {
			return ErrValidation
		}
	}
	if patch.Penyelenggara.Present && !patch.Penyelenggara.Null {
		if TextLength(patch.Penyelenggara.Value) > MaxNameLength {
			return ErrValidation
		}
	}
	if patch.TanggalMulai.Present && !patch.TanggalMulai.Null {
		if !validateDate(patch.TanggalMulai.Value) {
			return ErrValidation
		}
	}
	if patch.TanggalSelesai.Present && !patch.TanggalSelesai.Null {
		if !validateDate(patch.TanggalSelesai.Value) {
			return ErrValidation
		}
	}
	if patch.DurasiJam.Present && !patch.DurasiJam.Null {
		if patch.DurasiJam.Value == "" {
			return ErrValidation
		}
	}
	return nil
}

// HasChanges melaporkan apakah patch memuat setidaknya satu perubahan.
func (patch RiwayatPelatihanPatch) HasChanges() bool {
	return patch.NamaPelatihan.Present || patch.JenisPelatihan.Present ||
		patch.Penyelenggara.Present || patch.TanggalMulai.Present ||
		patch.TanggalSelesai.Present || patch.DurasiJam.Present ||
		patch.MemilikiSertifikat.Present
}
