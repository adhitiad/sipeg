package payscale_test

import (
	"strings"
	"testing"
	"time"

	dirdomain "simpeg/backend/internal/directory/domain"
	"simpeg/backend/internal/payscale/domain"
)

func TestCreateSkalaGajiValidateMenerima(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 11, 0)

	cases := []domain.CreateSkalaGaji{
		{Kode: "SKJ-2026", Nama: "Skala Gaji Yayasan 2026", BerlakuMulai: start},
		{Kode: "SKJ-2026", Nama: "Skala Gaji Yayasan 2026", BerlakuMulai: start, BerlakuSelesai: &end},
	}
	for i, input := range cases {
		if err := input.Validate(); err != nil {
			t.Fatalf("kasus %d harusnya valid: %v", i, err)
		}
	}
}

func TestCreateSkalaGajiValidateMenolak(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	before := start.AddDate(0, -1, 0)

	cases := []domain.CreateSkalaGaji{
		{Kode: "", Nama: "Skala Gaji Yayasan", BerlakuMulai: start},
		{Kode: strings.Repeat("K", 51), Nama: "Skala", BerlakuMulai: start},
		{Kode: "SKJ", Nama: "", BerlakuMulai: start},
		{Kode: "SKJ", Nama: "Skala", BerlakuMulai: time.Time{}},
		{Kode: "SKJ", Nama: "Skala", BerlakuMulai: start, BerlakuSelesai: &before},
	}
	for i, input := range cases {
		if err := input.Validate(); err == nil {
			t.Fatalf("kasus %d harusnya ditolak", i)
		}
	}
}

func TestCreateSkalaGajiDetailValidate(t *testing.T) {
	valid := domain.CreateSkalaGajiDetail{
		SkalaGajiId:      "00000000-0000-0000-0000-000000000001",
		KategoriStatusId: "d0000000-0000-0000-0000-000000000001",
		Tingkat:          4,
		Nominal:          9750000,
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("harusnya valid: %v", err)
	}

	cases := []domain.CreateSkalaGajiDetail{
		{SkalaGajiId: "bukan-uuid", KategoriStatusId: "d0000000-0000-0000-0000-000000000001", Tingkat: 0, Nominal: 1},
		{SkalaGajiId: "00000000-0000-0000-0000-000000000001", KategoriStatusId: "bukan-uuid", Tingkat: 0, Nominal: 1},
		{SkalaGajiId: "00000000-0000-0000-0000-000000000001", KategoriStatusId: "d0000000-0000-0000-0000-000000000001", Tingkat: -1, Nominal: 1},
		{SkalaGajiId: "00000000-0000-0000-0000-000000000001", KategoriStatusId: "d0000000-0000-0000-0000-000000000001", Tingkat: 0, Nominal: -1},
	}
	for i, input := range cases {
		if err := input.Validate(); err == nil {
			t.Fatalf("kasus %d harusnya ditolak", i)
		}
	}
}

func TestSkalaGajiPatchValidate(t *testing.T) {
	valid := domain.NewSkalaGajiPatch(map[string]dirdomain.Field{
		"nama":         {Value: "Skala Baru", Present: true},
		"berlakuMulai": {Value: "2027-04-01", Present: true},
	})
	if err := valid.Validate(); err != nil {
		t.Fatalf("patch harusnya valid: %v", err)
	}

	badDate := domain.NewSkalaGajiPatch(map[string]dirdomain.Field{
		"berlakuMulai": {Value: "01-04-2027", Present: true},
	})
	if err := badDate.Validate(); err == nil {
		t.Fatal("tanggal non-ISO harusnya ditolak")
	}

	badLevel := domain.NewSkalaGajiDetailPatch(map[string]dirdomain.Field{
		"tingkat": {Value: "empat", Present: true},
	})
	if err := badLevel.Validate(); err == nil {
		t.Fatal("tingkat non-numerik harusnya ditolak")
	}

	badNominal := domain.NewSkalaGajiDetailPatch(map[string]dirdomain.Field{
		"nominal": {Value: "sembilan juta", Present: true},
	})
	if err := badNominal.Validate(); err == nil {
		t.Fatal("nominal non-numerik harusnya ditolak")
	}
}
