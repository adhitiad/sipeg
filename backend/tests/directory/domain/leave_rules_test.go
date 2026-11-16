package domain

import (
	"errors"
	. "simpeg/backend/internal/directory/domain"
	"testing"
	"time"

	"simpeg/backend/internal/models"
)

func bersalin(sub *models.SubJenisBersalin, mulai, selesai time.Time) models.Cuti {
	return models.Cuti{
		Jenis:            models.JenisCutiBersalin,
		SubJenisBersalin: sub,
		TanggalMulai:     mulai,
		TanggalSelesai:   selesai,
	}
}

func ptrJenisBersalin(v models.SubJenisBersalin) *models.SubJenisBersalin { return &v }

func TestValidasiDurasiCutiBersalin(t *testing.T) {
	mulai := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	if err := ValidasiDurasiCuti(bersalin(nil, mulai, mulai.AddDate(0, 0, 27))); !errors.Is(err, ErrValidation) {
		t.Fatalf("BERSALIN tanpa subJenis harus ditolak, err = %v", err)
	}
	mel := ptrJenisBersalin(models.SubJenisMelahirkan)
	if err := ValidasiDurasiCuti(bersalin(mel, mulai, mulai.AddDate(0, 3, 0))); err != nil {
		t.Fatalf("MELAHIRKAN tepat 3 bulan harus lolos, err = %v", err)
	}
	if err := ValidasiDurasiCuti(bersalin(mel, mulai, mulai.AddDate(0, 3, 1))); !errors.Is(err, ErrValidation) {
		t.Fatalf("MELAHIRKAN melebihi 3 bulan harus ditolak, err = %v", err)
	}
	keg := ptrJenisBersalin(models.SubJenisKeguguran)
	if err := ValidasiDurasiCuti(bersalin(keg, mulai, mulai.AddDate(0, 0, 44))); err != nil {
		t.Fatalf("KEGUGURAN 45 hari harus lolos, err = %v", err)
	}
	if err := ValidasiDurasiCuti(bersalin(keg, mulai, mulai.AddDate(0, 0, 45))); err != nil {
		t.Fatalf("KEGUGURAN tepat 45 hari harus lolos, err = %v", err)
	}
	if err := ValidasiDurasiCuti(bersalin(keg, mulai, mulai.AddDate(0, 0, 46))); !errors.Is(err, ErrValidation) {
		t.Fatalf("KEGUGURAN melebihi 45 hari harus ditolak, err = %v", err)
	}
	if err := ValidasiDurasiCuti(bersalin(ptrJenisBersalin("LAINNYA"), mulai, mulai)); !errors.Is(err, ErrValidation) {
		t.Fatalf("subJenis tak dikenal harus ditolak, err = %v", err)
	}
}

func TestValidasiDurasiIzinPenting(t *testing.T) {
	mulai := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		sub     models.SubJenisIzinPenting
		maks    int
		lolos   int
		ditolak int
	}{
		{models.SubJenisMenikah, 7, 7, 8},
		{models.SubJenisMenikahkanAnak, 4, 4, 5},
		{models.SubJenisMelahirkanIstri, 4, 4, 5},
		{models.SubJenisKeluargaInti, 2, 2, 3},
		{models.SubJenisKeluargaLain, 1, 1, 2},
	}
	for _, c := range cases {
		item := models.Cuti{Jenis: models.JenisCutiIzinPenting, SubJenis: &c.sub, TanggalMulai: mulai, TanggalSelesai: mulai.AddDate(0, 0, c.lolos-1)}
		if err := ValidasiDurasiCuti(item); err != nil {
			t.Fatalf("%s %d hari harus lolos, err = %v", c.sub, c.lolos, err)
		}
		item.TanggalSelesai = mulai.AddDate(0, 0, c.ditolak-1)
		if err := ValidasiDurasiCuti(item); !errors.Is(err, ErrValidation) {
			t.Fatalf("%s %d hari harus ditolak, err = %v", c.sub, c.ditolak, err)
		}
	}
}

func TestValidasiDurasiCutiPanjang(t *testing.T) {
	mulai := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	item := models.Cuti{Jenis: models.JenisCutiPanjang, TanggalMulai: mulai, TanggalSelesai: mulai.AddDate(0, 2, 0)}
	if err := ValidasiDurasiCuti(item); err != nil {
		t.Fatalf("PANJANG tepat 2 bulan harus lolos, err = %v", err)
	}
	item.TanggalSelesai = mulai.AddDate(0, 2, 1)
	if err := ValidasiDurasiCuti(item); !errors.Is(err, ErrValidation) {
		t.Fatalf("PANJANG melebihi 2 bulan harus ditolak, err = %v", err)
	}
}

func TestSisaCutiHangus(t *testing.T) {
	row := models.SaldoCuti{KuotaTahunan: 12, Terpakai: 3, SisaCarry: 6, Hangus: false}
	if got := SisaCuti(row); got != 15 {
		t.Fatalf("sisa sebelum hangus = %d, want 15", got)
	}
	row.Hangus = true
	if got := SisaCuti(row); got != 9 {
		t.Fatalf("sisa setelah hangus = %d, want 9", got)
	}
	row = models.SaldoCuti{KuotaTahunan: 12, Terpakai: 13, SisaCarry: 6, Hangus: true}
	if got := SisaCuti(row); got != 0 {
		t.Fatalf("sisa tidak pernah negatif, got = %d", got)
	}
}

func TestPersenSakitBerkepanjangan(t *testing.T) {
	cases := []struct {
		bulan int
		want  float64
	}{
		{1, 100}, {4, 100}, {5, 75}, {8, 75}, {9, 50}, {12, 50}, {13, 25}, {16, 25}, {17, 0}, {24, 0},
	}
	for _, c := range cases {
		if got := PersenSakitBerkepanjangan(c.bulan); got != c.want {
			t.Fatalf("bulan ke-%d = %.0f, want %.0f", c.bulan, got, c.want)
		}
	}
}

func TestKalenderHariAntara(t *testing.T) {
	mulai := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	if got := KalenderHariAntara(mulai, mulai.AddDate(0, 0, 6)); got != 7 {
		t.Fatalf("KalenderHariAntara = %d, want 7", got)
	}
	if got := KalenderHariAntara(mulai, mulai); got != 1 {
		t.Fatalf("KalenderHariAntara satu hari = %d, want 1", got)
	}
}
