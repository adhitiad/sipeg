package domain

import (
	"testing"
	"time"

	. "simpeg/backend/internal/thr/domain"
)

func TestValidBesaran(t *testing.T) {
	for _, b := range BesaranList() {
		if !ValidBesaran(b) {
			t.Fatalf("besaran %q harus valid", b)
		}
	}
	if ValidBesaran("SETENGGAH_GAJI") {
		t.Fatal("besaran tak dikenal tidak boleh valid")
	}
}

func TestNominalSatuGajiPokok(t *testing.T) {
	n, err := Nominal(BesaranSatuGajiPokok, nil, Basis{GajiPokok: 9_500_000, TotalGaji: 11_000_000})
	if err != nil || n != 9_500_000 {
		t.Fatalf("= (%v, %v), want 9_500_000", n, err)
	}
}

func TestNominalGajiKotor(t *testing.T) {
	n, err := Nominal(BesaranGajiKotor, nil, Basis{GajiPokok: 9_500_000, TotalGaji: 11_000_000})
	if err != nil || n != 11_000_000 {
		t.Fatalf("= (%v, %v), want 11_000_000", n, err)
	}
}

func TestNominalPersen(t *testing.T) {
	persen := 100.0
	n, err := Nominal(BesaranPersen, &persen, Basis{GajiPokok: 9_500_000, TotalGaji: 11_000_000})
	if err != nil || n != 9_500_000 {
		t.Fatalf("= (%v, %v), want 9_500_000", n, err)
	}
	half := 50.0
	n, err = Nominal(BesaranPersen, &half, Basis{GajiPokok: 9_500_000, TotalGaji: 11_000_000})
	if err != nil || n != 4_750_000 {
		t.Fatalf("= (%v, %v), want 4_750_000", n, err)
	}
	n, err = Nominal(BesaranPersen, nil, Basis{GajiPokok: 9_500_000, TotalGaji: 11_000_000})
	if err == nil {
		t.Fatalf("persen tanpa nilai harus error, got %v", n)
	}
}

func TestNominalPersenTakeHome(t *testing.T) {
	// DEC-086: THR 2028 sebesar 56% take-home pay pada gaji bersih Rp 6.500.000
	// menghasilkan nominal Rp 3.640.000.
	persen := 56.0
	n, err := Nominal(BesaranPersenTakeHome, &persen, Basis{GajiBersih: 6_500_000})
	if err != nil || n != 3_640_000 {
		t.Fatalf("= (%v, %v), want 3_640_000", n, err)
	}
	if err := ValidateConfig(2028, BesaranPersenTakeHome, &persen, 3, time.Date(2028, 3, 9, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("config PERSEN_TAKE_HOME valid ditolak: %v", err)
	}
	if err := ValidateConfig(2028, BesaranPersenTakeHome, nil, 3, time.Date(2028, 3, 9, 0, 0, 0, 0, time.UTC)); err == nil {
		t.Fatal("PERSEN_TAKE_HOME tanpa persen harus ditolak")
	}
}

func TestValidateConfig(t *testing.T) {
	berlaku := time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC)

	if err := ValidateConfig(2027, BesaranGajiKotor, nil, 3, berlaku); err != nil {
		t.Fatalf("config valid ditolak: %v", err)
	}
	persen := 100.0
	if err := ValidateConfig(2027, BesaranPersen, &persen, 3, berlaku); err != nil {
		t.Fatalf("config PERSEN valid ditolak: %v", err)
	}
	if err := ValidateConfig(2025, BesaranGajiKotor, nil, 3, berlaku); err == nil {
		t.Fatal("tahun sebelum 2026 harus ditolak")
	}
	if err := ValidateConfig(2027, BesaranPersen, nil, 3, berlaku); err == nil {
		t.Fatal("PERSEN tanpa persen harus ditolak")
	}
	if err := ValidateConfig(2028, BesaranPersenTakeHome, &persen, 3, berlaku); err != nil {
		t.Fatalf("config PERSEN_TAKE_HOME valid ditolak: %v", err)
	}
	if err := ValidateConfig(2028, BesaranPersenTakeHome, nil, 3, berlaku); err == nil {
		t.Fatal("PERSEN_TAKE_HOME tanpa persen harus ditolak")
	}
	if err := ValidateConfig(2027, "SETENGAH_GAJI", nil, 3, berlaku); err == nil {
		t.Fatal("besaran tak dikenal harus ditolak")
	}
	if err := ValidateConfig(2027, BesaranGajiKotor, &persen, 3, berlaku); err == nil {
		t.Fatal("GAJI_KOTOR dengan persen harus ditolak")
	}
	if err := ValidateConfig(2027, BesaranGajiKotor, nil, -1, berlaku); err == nil {
		t.Fatal("syarat negatif harus ditolak")
	}
	if err := ValidateConfig(2027, BesaranGajiKotor, nil, 61, berlaku); err == nil {
		t.Fatal("syarat > 60 harus ditolak")
	}
	if err := ValidateConfig(2027, BesaranGajiKotor, nil, 3, time.Time{}); err == nil {
		t.Fatal("berlakunya nol harus ditolak")
	}
}

func TestCutoffFor(t *testing.T) {
	berlaku := time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC)
	cutoff := CutoffFor(berlaku, 3)
	want := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	if !cutoff.Equal(want) {
		t.Fatalf("cutoff = %v, want %v", cutoff, want)
	}
}
