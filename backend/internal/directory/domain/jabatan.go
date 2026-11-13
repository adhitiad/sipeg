package domain

import "strings"

// JabatanTypes adalah daftar jenis jabatan yang sah.
var JabatanTypes = []string{"STRUKTURAL", "FUNGSIONAL"}

// ValidJabatanType memeriksa apakah sebuah jenis jabatan sah.
func ValidJabatanType(value string) bool {
	for _, candidate := range JabatanTypes {
		if candidate == value {
			return true
		}
	}
	return false
}

// DefaultJabatanType adalah jenis jabatan yang dipakai bila permintaan tidak
// menyebut jenis secara eksplisit.
const DefaultJabatanType = "STRUKTURAL"

// NormalizeJabatanType memangkas spasi dan mengubah jenis menjadi huruf besar.
func NormalizeJabatanType(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

// CreateJabatan memuat data minimum untuk membuat master jabatan.
type CreateJabatan struct {
	Nama  string
	Jenis string
}

// Validate memeriksa aturan jabatan yang baru dibuat.
func (input CreateJabatan) Validate() error {
	if strings.TrimSpace(input.Nama) == "" || TextLength(input.Nama) > MaxNameLength {
		return ErrValidation
	}
	if !ValidJabatanType(NormalizeJabatanType(input.Jenis)) {
		return ErrValidation
	}
	return nil
}

// JabatanPatchFields adalah daftar field jabatan yang boleh diubah.
var JabatanPatchFields = map[string]bool{
	"nama": true, "jenis": true,
}

// JabatanPatch memuat perubahan master jabatan yang diminta.
type JabatanPatch struct {
	Nama  Field
	Jenis Field
}

// NewJabatanPatch menyusun patch dari peta field hasil dekode permintaan.
func NewJabatanPatch(fields map[string]Field) JabatanPatch {
	return JabatanPatch{
		Nama:  fields["nama"],
		Jenis: fields["jenis"],
	}
}

// Validate memeriksa aturan perubahan jabatan.
func (patch JabatanPatch) Validate() error {
	if patch.Nama.Present {
		if patch.Nama.Null || TextLength(strings.TrimSpace(patch.Nama.Value)) == 0 ||
			TextLength(patch.Nama.Value) > MaxNameLength {
			return ErrValidation
		}
	}
	if patch.Jenis.Present {
		if patch.Jenis.Null || !ValidJabatanType(NormalizeJabatanType(patch.Jenis.Value)) {
			return ErrValidation
		}
	}
	return nil
}

// HasChanges melaporkan apakah patch memuat setidaknya satu perubahan.
func (patch JabatanPatch) HasChanges() bool {
	return patch.Nama.Present || patch.Jenis.Present
}
