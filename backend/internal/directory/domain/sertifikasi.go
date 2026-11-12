// Package domain memuat aturan bisnis untuk riwayat sertifikasi pegawai.
package domain

import (
	"strings"

	"simpeg/backend/internal/models"
)

// JenisSertifikasiValid melaporkan apakah jenis sertifikasi valid.
func JenisSertifikasiValid(value models.JenisSertifikasi) bool {
	switch value {
	case models.SertifikasiSerdos, models.SertifikasiProfesi, models.SertifikasiKompetensi, models.SertifikasiLainnya:
		return true
	}
	return false
}

// StatusSertifikasiValid melaporkan apakah status sertifikasi valid.
func StatusSertifikasiValid(value models.StatusSertifikasi) bool {
	switch value {
	case models.SertifikasiAktif, models.SertifikasiKedaluarsa:
		return true
	}
	return false
}

// CreateRiwayatSertifikasi memuat data untuk membuat catatan riwayat
// sertifikasi pegawai.
type CreateRiwayatSertifikasi struct {
	UserID           string
	JenisSertifikasi models.JenisSertifikasi
	NamaSertifikat   string
	NomorRegistrasi  *string
	Penyelenggara    string
	TanggalTerbit    string  // "YYYY-MM-DD"
	BerlakuSampai    *string // "YYYY-MM-DD"
	Status           models.StatusSertifikasi
}

// Validate memeriksa aturan data riwayat sertifikasi.
func (input CreateRiwayatSertifikasi) Validate() error {
	if !JenisSertifikasiValid(input.JenisSertifikasi) {
		return ErrValidation
	}
	if strings.TrimSpace(input.NamaSertifikat) == "" || TextLength(input.NamaSertifikat) > MaxNameLength {
		return ErrValidation
	}
	if input.NomorRegistrasi != nil && TextLength(strings.TrimSpace(*input.NomorRegistrasi)) > MaxNIPLength {
		return ErrValidation
	}
	if strings.TrimSpace(input.Penyelenggara) == "" || TextLength(input.Penyelenggara) > MaxNameLength {
		return ErrValidation
	}
	if !validateDate(input.TanggalTerbit) {
		return ErrValidation
	}
	if input.BerlakuSampai != nil && !validateDate(*input.BerlakuSampai) {
		return ErrValidation
	}
	if !StatusSertifikasiValid(input.Status) {
		return ErrValidation
	}
	return nil
}

// SertifikasiPatchFields adalah daftar field yang boleh diubah.
var SertifikasiPatchFields = map[string]bool{
	"jenisSertifikasi": true, "namaSertifikat": true, "nomorRegistrasi": true,
	"penyelenggara": true, "tanggalTerbit": true, "berlakuSampai": true, "status": true,
}

// RiwayatSertifikasiPatch memuat perubahan catatan riwayat sertifikasi.
type RiwayatSertifikasiPatch struct {
	JenisSertifikasi Field
	NamaSertifikat   Field
	NomorRegistrasi  Field
	Penyelenggara    Field
	TanggalTerbit    Field
	BerlakuSampai    Field
	Status           Field
}

// NewRiwayatSertifikasiPatch menyusun patch dari peta field.
func NewRiwayatSertifikasiPatch(fields map[string]Field) RiwayatSertifikasiPatch {
	return RiwayatSertifikasiPatch{
		JenisSertifikasi: fields["jenisSertifikasi"],
		NamaSertifikat:   fields["namaSertifikat"],
		NomorRegistrasi:  fields["nomorRegistrasi"],
		Penyelenggara:    fields["penyelenggara"],
		TanggalTerbit:    fields["tanggalTerbit"],
		BerlakuSampai:    fields["berlakuSampai"],
		Status:           fields["status"],
	}
}

// Validate memeriksa aturan perubahan riwayat sertifikasi.
func (patch RiwayatSertifikasiPatch) Validate() error {
	if patch.JenisSertifikasi.Present && !patch.JenisSertifikasi.Null {
		if !JenisSertifikasiValid(models.JenisSertifikasi(patch.JenisSertifikasi.Value)) {
			return ErrValidation
		}
	}
	if patch.NamaSertifikat.Present {
		if patch.NamaSertifikat.Null || TextLength(strings.TrimSpace(patch.NamaSertifikat.Value)) == 0 ||
			TextLength(patch.NamaSertifikat.Value) > MaxNameLength {
			return ErrValidation
		}
	}
	if patch.NomorRegistrasi.Present && !patch.NomorRegistrasi.Null {
		if TextLength(patch.NomorRegistrasi.Value) > MaxNIPLength {
			return ErrValidation
		}
	}
	if patch.Penyelenggara.Present && !patch.Penyelenggara.Null {
		if TextLength(patch.Penyelenggara.Value) > MaxNameLength {
			return ErrValidation
		}
	}
	if patch.TanggalTerbit.Present && !patch.TanggalTerbit.Null {
		if !validateDate(patch.TanggalTerbit.Value) {
			return ErrValidation
		}
	}
	if patch.BerlakuSampai.Present && !patch.BerlakuSampai.Null {
		if !validateDate(patch.BerlakuSampai.Value) {
			return ErrValidation
		}
	}
	if patch.Status.Present && !patch.Status.Null {
		if !StatusSertifikasiValid(models.StatusSertifikasi(patch.Status.Value)) {
			return ErrValidation
		}
	}
	return nil
}

// HasChanges melaporkan apakah patch memuat setidaknya satu perubahan.
func (patch RiwayatSertifikasiPatch) HasChanges() bool {
	return patch.JenisSertifikasi.Present || patch.NamaSertifikat.Present ||
		patch.NomorRegistrasi.Present || patch.Penyelenggara.Present ||
		patch.TanggalTerbit.Present || patch.BerlakuSampai.Present || patch.Status.Present
}
