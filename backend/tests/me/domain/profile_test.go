package domain

import (
	"errors"
	"strings"
	"testing"

	"simpeg/backend/internal/me/domain"
)

func ptr(value string) *string { return &value }

func TestNormalizeAcceptsValidProfile(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		input    domain.UpdateProfileInput
		wantNama string
		wantNoHp string
	}{
		{name: "nama dipangkas dan spasi digandakan dirapikan", input: domain.UpdateProfileInput{Nama: ptr("  Budi   Santoso ")}, wantNama: "Budi Santoso"},
		{name: "nomor dengan awalan 0", input: domain.UpdateProfileInput{NoHp: ptr("081234567890")}, wantNoHp: "081234567890"},
		{name: "nomor dengan awalan +62", input: domain.UpdateProfileInput{NoHp: ptr("+6281234567890")}, wantNoHp: "+6281234567890"},
		{name: "nomor dengan awalan 62", input: domain.UpdateProfileInput{NoHp: ptr("6281234567890")}, wantNoHp: "6281234567890"},
		{name: "nomor dipangkas dari spasi dan tanda hubung", input: domain.UpdateProfileInput{NoHp: ptr(" 0812-3456-7890 ")}, wantNoHp: "081234567890"},
		{name: "nomor kosong mengosongkan nilai tersimpan", input: domain.UpdateProfileInput{NoHp: ptr("")}, wantNoHp: ""},
		{name: "nama dan nomor sekaligus", input: domain.UpdateProfileInput{Nama: ptr("Budi"), NoHp: ptr("+62 812-3456-7890")}, wantNama: "Budi", wantNoHp: "+6281234567890"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got, err := domain.Normalize(testCase.input)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if testCase.wantNama != "" {
				if got.Nama == nil || *got.Nama != testCase.wantNama {
					t.Fatalf("expected nama %q, got %v", testCase.wantNama, got.Nama)
				}
			}
			if testCase.input.NoHp != nil {
				if got.NoHp == nil || *got.NoHp != testCase.wantNoHp {
					t.Fatalf("expected noHp %q, got %v", testCase.wantNoHp, got.NoHp)
				}
			}
		})
	}
}

func TestNormalizeRejectsInvalidProfile(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input domain.UpdateProfileInput
	}{
		{name: "nama kosong", input: domain.UpdateProfileInput{Nama: ptr("   ")}},
		{name: "nama terlalu panjang", input: domain.UpdateProfileInput{Nama: ptr(strings.Repeat("a", 201))}},
		{name: "nomor dengan huruf", input: domain.UpdateProfileInput{NoHp: ptr("08123abc5678")}},
		{name: "nomor dengan awalan tidak dikenal", input: domain.UpdateProfileInput{NoHp: ptr("+15551234567")}},
		{name: "nomor terlalu pendek", input: domain.UpdateProfileInput{NoHp: ptr("0812")}},
		{name: "nomor terlalu panjang", input: domain.UpdateProfileInput{NoHp: ptr("08123456789012345678")}},
		{name: "tanpa satu pun kolom", input: domain.UpdateProfileInput{}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			_, err := domain.Normalize(testCase.input)
			if !errors.Is(err, domain.ErrValidation) {
				t.Fatalf("expected ErrValidation, got %v", err)
			}
		})
	}
}

func TestNormalizeReportsFields(t *testing.T) {
	t.Parallel()

	got, err := domain.Normalize(domain.UpdateProfileInput{Nama: ptr("Budi"), NoHp: ptr("081234567890")})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	fields := got.Fields()
	if len(fields) != 2 || fields[0] != "nama" || fields[1] != "noHp" {
		t.Fatalf("expected fields [nama noHp], got %v", fields)
	}
}
