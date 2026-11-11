package domain

import "strings"

// OrganizationTypes adalah daftar jenis unit organisasi yang sah.
var OrganizationTypes = []string{"UNIT", "FAKULTAS", "JURUSAN", "BIRO", "BAGIAN"}

// ValidOrganizationType memeriksa apakah sebuah jenis unit organisasi sah.
func ValidOrganizationType(value string) bool {
	for _, candidate := range OrganizationTypes {
		if candidate == value {
			return true
		}
	}
	return false
}

// DefaultOrganizationType adalah jenis unit yang dipakai bila permintaan
// tidak menyebut jenis secara eksplisit.
const DefaultOrganizationType = "UNIT"

// NormalizeOrganizationType memangkas spasi di sekeliling nilai jenis unit dan
// mengubahnya menjadi huruf besar.
func NormalizeOrganizationType(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

// CreateOrganization memuat data minimum untuk membuat unit organisasi.
type CreateOrganization struct {
	Name        string
	Description *string
	Jenis       string
	ParentID    *string
}

// Validate memeriksa aturan unit organisasi yang baru dibuat.
func (input CreateOrganization) Validate() error {
	if strings.TrimSpace(input.Name) == "" || TextLength(input.Name) > MaxNameLength {
		return ErrValidation
	}
	if !ValidOrganizationType(input.Jenis) {
		return ErrValidation
	}
	if input.ParentID != nil && !ValidUUID(*input.ParentID) {
		return ErrValidation
	}
	if input.Description != nil && TextLength(*input.Description) > MaxDescription {
		return ErrValidation
	}
	return nil
}

// OrganizationPatchFields adalah daftar field unit organisasi yang boleh diubah.
var OrganizationPatchFields = map[string]bool{
	"name": true, "description": true, "jenis": true, "parentId": true,
}

// OrganizationPatch memuat perubahan unit organisasi yang diminta.
type OrganizationPatch struct {
	Name        Field
	Description Field
	Jenis       Field
	ParentID    Field
}

// NewOrganizationPatch menyusun patch dari peta field hasil dekode permintaan.
func NewOrganizationPatch(fields map[string]Field) OrganizationPatch {
	return OrganizationPatch{
		Name:        fields["name"],
		Description: fields["description"],
		Jenis:       fields["jenis"],
		ParentID:    fields["parentId"],
	}
}

// Validate memeriksa aturan perubahan unit organisasi.
func (patch OrganizationPatch) Validate() error {
	if patch.Name.Present {
		if patch.Name.Null || TextLength(strings.TrimSpace(patch.Name.Value)) == 0 ||
			TextLength(patch.Name.Value) > MaxNameLength {
			return ErrValidation
		}
	}
	if patch.Description.Present && !patch.Description.Null &&
		TextLength(patch.Description.Value) > MaxDescription {
		return ErrValidation
	}
	if patch.Jenis.Present {
		if patch.Jenis.Null || !ValidOrganizationType(NormalizeOrganizationType(patch.Jenis.Value)) {
			return ErrValidation
		}
	}
	if patch.ParentID.Present && !patch.ParentID.Null && !ValidUUID(patch.ParentID.Value) {
		return ErrValidation
	}
	return nil
}

// HasChanges melaporkan apakah patch memuat setidaknya satu perubahan.
func (patch OrganizationPatch) HasChanges() bool {
	return patch.Name.Present || patch.Description.Present || patch.Jenis.Present || patch.ParentID.Present
}
