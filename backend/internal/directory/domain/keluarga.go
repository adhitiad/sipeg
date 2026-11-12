// Package domain memuat aturan bisnis untuk data keluarga dan tanggungan
// pegawai. Lapisan ini tidak melakukan akses database maupun HTTP.
package domain

import (
	"strings"

	"simpeg/backend/internal/models"
)

// HubunganKeluargaValid melaporkan apakah nilai hubungan keluarga valid.
func HubunganKeluargaValid(value models.JenisHubunganKeluarga) bool {
	switch value {
	case models.HubunganSuamiIstri, models.HubunganAnak:
		return true
	}
	return false
}

// StatusTanggunganValid melaporkan apakah nilai status tanggungan valid.
func StatusTanggunganValid(value models.StatusTanggungan) bool {
	switch value {
	case models.TanggunganYa, models.TanggunganTidak:
		return true
	}
	return false
}

// CreateKeluargaPegawai memuat data minimum untuk membuat catatan keluarga
// pegawai.
type CreateKeluargaPegawai struct {
	UserID           string
	Hubungan         models.JenisHubunganKeluarga
	Nama             string
	NIK              string
	TanggalLahir     string // "YYYY-MM-DD"
	Pekerjaan        *string
	StatusTanggungan models.StatusTanggungan
}

// Validate memeriksa aturan data keluarga pegawai.
func (input CreateKeluargaPegawai) Validate() error {
	if !HubunganKeluargaValid(input.Hubungan) {
		return ErrValidation
	}
	if strings.TrimSpace(input.Nama) == "" || TextLength(input.Nama) > MaxNameLength {
		return ErrValidation
	}
	if strings.TrimSpace(input.NIK) == "" || TextLength(input.NIK) > MaxNIKLength {
		return ErrValidation
	}
	if !validateDate(input.TanggalLahir) {
		return ErrValidation
	}
	if input.Pekerjaan != nil && TextLength(strings.TrimSpace(*input.Pekerjaan)) > MaxPekerjaanLength {
		return ErrValidation
	}
	if !StatusTanggunganValid(input.StatusTanggungan) {
		return ErrValidation
	}
	return nil
}

// KeluargaPatchFields adalah daftar field yang boleh diubah pada catatan
// keluarga pegawai.
var KeluargaPatchFields = map[string]bool{
	"hubungan": true, "nama": true, "nik": true, "tanggalLahir": true,
	"pekerjaan": true, "statusTanggungan": true,
}

// KeluargaPatch memuat perubahan catatan keluarga pegawai yang diminta.
type KeluargaPatch struct {
	Hubungan         Field
	Nama             Field
	NIK              Field
	TanggalLahir     Field
	Pekerjaan        Field
	StatusTanggungan Field
}

// NewKeluargaPatch menyusun patch dari peta field hasil dekode permintaan.
func NewKeluargaPatch(fields map[string]Field) KeluargaPatch {
	return KeluargaPatch{
		Hubungan:         fields["hubungan"],
		Nama:             fields["nama"],
		NIK:              fields["nik"],
		TanggalLahir:     fields["tanggalLahir"],
		Pekerjaan:        fields["pekerjaan"],
		StatusTanggungan: fields["statusTanggungan"],
	}
}

// Validate memeriksa aturan perubahan catatan keluarga pegawai.
func (patch KeluargaPatch) Validate() error {
	if patch.Hubungan.Present && !patch.Hubungan.Null {
		if !HubunganKeluargaValid(models.JenisHubunganKeluarga(patch.Hubungan.Value)) {
			return ErrValidation
		}
	}
	if patch.Nama.Present {
		if patch.Nama.Null || TextLength(strings.TrimSpace(patch.Nama.Value)) == 0 ||
			TextLength(patch.Nama.Value) > MaxNameLength {
			return ErrValidation
		}
	}
	if patch.NIK.Present {
		if patch.NIK.Null || TextLength(strings.TrimSpace(patch.NIK.Value)) == 0 ||
			TextLength(patch.NIK.Value) > MaxNIKLength {
			return ErrValidation
		}
	}
	if patch.TanggalLahir.Present && !patch.TanggalLahir.Null {
		if !validateDate(patch.TanggalLahir.Value) {
			return ErrValidation
		}
	}
	if patch.Pekerjaan.Present && !patch.Pekerjaan.Null {
		if TextLength(patch.Pekerjaan.Value) > MaxPekerjaanLength {
			return ErrValidation
		}
	}
	if patch.StatusTanggungan.Present && !patch.StatusTanggungan.Null {
		if !StatusTanggunganValid(models.StatusTanggungan(patch.StatusTanggungan.Value)) {
			return ErrValidation
		}
	}
	return nil
}

// HasChanges melaporkan apakah patch memuat setidaknya satu perubahan.
func (patch KeluargaPatch) HasChanges() bool {
	return patch.Hubungan.Present || patch.Nama.Present || patch.NIK.Present ||
		patch.TanggalLahir.Present || patch.Pekerjaan.Present || patch.StatusTanggungan.Present
}
