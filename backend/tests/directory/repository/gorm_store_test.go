package repository

import (
	"errors"
	. "simpeg/backend/internal/directory/repository"
	"strings"
	"testing"

	"simpeg/backend/internal/crypto"
	"simpeg/backend/internal/directory/domain"

	"gorm.io/gorm"
)

func TestClassifyRecordError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want error
	}{
		{name: "not found", err: gorm.ErrRecordNotFound, want: domain.ErrNotFound},
		{name: "duplicate", err: gorm.ErrDuplicatedKey, want: domain.ErrConflict},
		{name: "other error", err: errors.New("database unavailable")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := ClassifyRecordError(test.err)
			if test.want != nil && !errors.Is(got, test.want) {
				t.Fatalf("got %v, want %v", got, test.want)
			}
			if test.want == nil && got.Error() != test.err.Error() {
				t.Fatalf("got %v, want original error %v", got, test.err)
			}
		})
	}
}

func TestEscapeLikeTreatsUserWildcardsLiterally(t *testing.T) {
	input := `50%_complete\`
	want := `50\%\_complete\\`
	if got := EscapeLike(input); got != want {
		t.Fatalf("EscapeLike(%q) = %q, want %q", input, got, want)
	}
}

func TestEmployeeColumnsMapsPatchToSchemaColumns(t *testing.T) {
	patch := domain.NewEmployeePatch(map[string]domain.Field{
		"nama":              {Present: true, Value: "  Nama Baru  "},
		"nip":               {Present: true, Value: "  12345  "},
		"kategoriPegawaiId": {Present: true, Null: true},
		"homebaseId":        {Present: true, Value: "11111111-1111-4111-8111-111111111111"},
		"jabatanStruktural": {Present: true, Value: " Dekan "},
		"jabatanFungsional": {Present: true, Null: true},
	})

	columns, err := EmployeeColumns(patch)
	if err != nil {
		t.Fatalf("EmployeeColumns: %v", err)
	}
	if columns["nama"] != "Nama Baru" {
		t.Fatalf("nama = %v", columns["nama"])
	}
	nipCiphertext, ok := columns["nip"].(string)
	if !ok || !strings.HasPrefix(nipCiphertext, crypto.CiphertextPrefix) {
		t.Fatalf("nip bukan ciphertext ber-prefix %q: %v", crypto.CiphertextPrefix, columns["nip"])
	}
	expectedHash, err := crypto.BlindIndex("12345")
	if err != nil {
		t.Fatal(err)
	}
	if columns["nipHash"] != expectedHash {
		t.Fatalf("nipHash = %v, want blind index dari 12345", columns["nipHash"])
	}
	if columns["kategoriPegawaiId"] != nil {
		t.Fatalf("kategoriPegawaiId = %v, want nil", columns["kategoriPegawaiId"])
	}
	if columns["homebaseId"] != "11111111-1111-4111-8111-111111111111" {
		t.Fatalf("homebaseId = %v", columns["homebaseId"])
	}
	if columns["jabatan_struktural"] != "Dekan" {
		t.Fatalf("jabatan_struktural = %v", columns["jabatan_struktural"])
	}
	if columns["jabatan_fungsional"] != nil {
		t.Fatalf("jabatan_fungsional = %v, want nil", columns["jabatan_fungsional"])
	}
}

func TestEmployeeColumnsKeepsAbsentFieldsUntouched(t *testing.T) {
	columns, err := EmployeeColumns(domain.NewEmployeePatch(map[string]domain.Field{
		"nama": {Present: true, Value: "Seseorang"},
	}))
	if err != nil {
		t.Fatalf("EmployeeColumns: %v", err)
	}
	if _, ok := columns["nip"]; ok {
		t.Fatal("absent fields must not appear in the update map")
	}
	if _, ok := columns["nipHash"]; ok {
		t.Fatal("absent fields must not appear in the update map")
	}
}

func TestOrganizationColumnsMapsPatchToSchemaColumns(t *testing.T) {
	patch := domain.NewOrganizationPatch(map[string]domain.Field{
		"name":        {Present: true, Value: "Fakultas"},
		"description": {Present: true, Null: true},
		"jenis":       {Present: true, Value: " fakultas "},
		"parentId":    {Present: true, Null: true},
	})

	columns := OrganizationColumns(patch)
	if columns["name"] != "Fakultas" {
		t.Fatalf("name = %v", columns["name"])
	}
	if columns["description"] != nil {
		t.Fatalf("description = %v, want nil", columns["description"])
	}
	if columns["jenis"] != "FAKULTAS" {
		t.Fatalf("jenis = %v", columns["jenis"])
	}
	if value, ok := columns["parentId"]; !ok || value != (*string)(nil) {
		t.Fatalf("parentId = %v", columns["parentId"])
	}
}
