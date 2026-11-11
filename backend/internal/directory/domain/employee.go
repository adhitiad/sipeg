package domain

import "strings"

// CreateEmployee memuat dataminimum untuk membuat profil pegawai.
type CreateEmployee struct {
	Nama              string
	Email             string
	NIP               *string
	KategoriPegawaiID *string
	HomebaseID        *string
	JabatanStruktural *string
	JabatanFungsional *string
}

// Validate memeriksa aturan profil pegawai yang baru dibuat.
func (input CreateEmployee) Validate() error {
	if strings.TrimSpace(input.Nama) == "" || TextLength(input.Nama) > MaxNameLength {
		return ErrValidation
	}
	if !ValidEmail(strings.TrimSpace(input.Email)) {
		return ErrValidation
	}
	if input.NIP != nil {
		nip := strings.TrimSpace(*input.NIP)
		if nip == "" || TextLength(nip) > MaxNIPLength {
			return ErrValidation
		}
	}
	for _, id := range []*string{input.KategoriPegawaiID, input.HomebaseID} {
		if id != nil && !ValidUUID(*id) {
			return ErrValidation
		}
	}
	for _, position := range []*string{input.JabatanStruktural, input.JabatanFungsional} {
		if position != nil && TextLength(strings.TrimSpace(*position)) > MaxPositionLength {
			return ErrValidation
		}
	}
	return nil
}

// EmployeePatchFields adalah daftar field profil pegawai yang boleh diubah.
var EmployeePatchFields = map[string]bool{
	"nama": true, "nip": true, "kategoriPegawaiId": true,
	"homebaseId": true, "jabatanStruktural": true, "jabatanFungsional": true,
}

// EmployeePatch memuat perubahan profil pegawai yang diminta.
type EmployeePatch struct {
	Nama              Field
	NIP               Field
	KategoriPegawaiID Field
	HomebaseID        Field
	JabatanStruktural Field
	JabatanFungsional Field
}

// NewEmployeePatch menyusun patch dari peta field hasil dekode permintaan.
func NewEmployeePatch(fields map[string]Field) EmployeePatch {
	return EmployeePatch{
		Nama:              fields["nama"],
		NIP:               fields["nip"],
		KategoriPegawaiID: fields["kategoriPegawaiId"],
		HomebaseID:        fields["homebaseId"],
		JabatanStruktural: fields["jabatanStruktural"],
		JabatanFungsional: fields["jabatanFungsional"],
	}
}

// Validate memeriksa aturan perubahan profil pegawai.
//
// Nama tidak boleh kosong dan tidak boleh lebih panjang dari batas. NIP yang
// dikosongkan berarti menghapus NIP. Referensi harus berupa UUID.
func (patch EmployeePatch) Validate() error {
	if patch.Nama.Present {
		if patch.Nama.Null || TextLength(strings.TrimSpace(patch.Nama.Value)) == 0 ||
			TextLength(patch.Nama.Value) > MaxNameLength {
			return ErrValidation
		}
	}
	if patch.NIP.Present && !patch.NIP.Null {
		if TextLength(patch.NIP.Value) > MaxNIPLength {
			return ErrValidation
		}
	}
	for _, field := range []Field{patch.KategoriPegawaiID, patch.HomebaseID} {
		if field.Present && !field.Null && !ValidUUID(field.Value) {
			return ErrValidation
		}
	}
	for _, field := range []Field{patch.JabatanStruktural, patch.JabatanFungsional} {
		if field.Present && !field.Null && TextLength(field.Value) > MaxPositionLength {
			return ErrValidation
		}
	}
	return nil
}

// HasChanges melaporkan apakah patch memuat setidaknya satu perubahan.
func (patch EmployeePatch) HasChanges() bool {
	return patch.Nama.Present || patch.NIP.Present || patch.KategoriPegawaiID.Present ||
		patch.HomebaseID.Present || patch.JabatanStruktural.Present || patch.JabatanFungsional.Present
}
