package calculasi_test

import (
	"math"
	"testing"

	"simpeg/backend/internal/calculasi/domain"
)

func TestPtkpToTer(t *testing.T) {
	cases := map[string]string{
		"TK/0": "A", "TK/1": "A", "TK/2": "B", "TK/3": "B",
		"K/0": "A", "K/1": "B", "K/2": "B", "K/3": "C",
		"K/I/0": "A", "K/I/1": "B", "K/I/2": "B", "K/I/3": "C",
		"unknown": "A",
	}
	for ptkp, ter := range cases {
		if got := domain.PtkpToTer(ptkp); got != ter {
			t.Fatalf("PtkpToTer(%q) = %q, want %q", ptkp, got, ter)
		}
	}
}

func TestResolveTer(t *testing.T) {
	rows := []domain.TerRow{
		{Kategori: "A", Lapisan: 1, BatasBawah: 0, BatasAtas: fptr(5400000), TarifPct: 0},
		{Kategori: "A", Lapisan: 2, BatasBawah: 5400000, BatasAtas: fptr(5650000), TarifPct: 0.25},
		{Kategori: "A", Lapisan: 3, BatasBawah: 5650000, TarifPct: 0.5},
		{Kategori: "B", Lapisan: 1, BatasBawah: 0, TarifPct: 0.5},
	}

	row, err := domain.ResolveTarif(rows, "A", 5300000)
	if err != nil || row.TarifPct != 0 {
		t.Fatalf("lapisan 1: row=%v err=%v", row, err)
	}
	row, err = domain.ResolveTarif(rows, "A", 5500000)
	if err != nil || row.TarifPct != 0.25 {
		t.Fatalf("lapisan 2: row=%v err=%v", row, err)
	}
	row, err = domain.ResolveTarif(rows, "A", 9000000)
	if err != nil || row.TarifPct != 0.5 {
		t.Fatalf("lapisan 3: row=%v err=%v", row, err)
	}
	row, err = domain.ResolveTarif(rows, "B", 9000000)
	if err != nil || row.TarifPct != 0.5 {
		t.Fatalf("kategori B: row=%v err=%v", row, err)
	}
	_, err = domain.ResolveTarif(rows, "C", 9000000)
	if err == nil {
		t.Fatal("kategori C tanpa baris harusnya ErrNotFound")
	}
}

func TestCalculateHonorAndBruto(t *testing.T) {
	result, err := domain.Calculate(domain.SlipInput{
		Periode:    "2026-06",
		GajiPokok:  9000000,
		HonorSks:   12,
		RatePerSks: 100000,
		Tunjangan:  []domain.Komponen{{Jenis: "TUNJANGAN_FUNGSIONAL", Nominal: 1500000}},
		Bpjs: []domain.BpjsRate{
			{Jenis: "KESEHATAN", IuranPegawaiPct: 1, IuranPemberiKerjaPct: 4, Dasar: "GAJI_POKOK"},
			{Jenis: "KETENAGAKERJAAN", IuranPegawaiPct: 2, IuranPemberiKerjaPct: 3.7, Dasar: "GAJI_POKOK"},
		},
		Ter: []domain.TerRow{
			{Kategori: "A", Lapisan: 1, BatasBawah: 0, TarifPct: 0},
		},
		StatusPtkp: "TK/0",
	})
	if err != nil {
		t.Fatalf("calculate: %v", err)
	}
	if math.Abs(result.HonorTotal-1200000) > 0.01 {
		t.Fatalf("honor total = %v", result.HonorTotal)
	}
	if math.Abs(result.TunjanganTotal-2700000) > 0.01 {
		t.Fatalf("tunjangan total = %v", result.TunjanganTotal)
	}
	if math.Abs(result.Bruto-11700000) > 0.01 {
		t.Fatalf("bruto = %v", result.Bruto)
	}
	if math.Abs(result.BpjsPegawaiTotal-270000) > 0.01 {
		t.Fatalf("bpjs pegawai = %v", result.BpjsPegawaiTotal)
	}
	if math.Abs(result.BpjsKesehatan-90000) > 0.01 {
		t.Fatalf("bpjs kesehatan = %v", result.BpjsKesehatan)
	}
	if math.Abs(result.BpjsKetenagakerjaan-180000) > 0.01 {
		t.Fatalf("bpjs ketenagakerjaan = %v", result.BpjsKetenagakerjaan)
	}
	if result.Pph21 != 0 {
		t.Fatalf("pph21 harus 0 untuk bruto 11.7jt (TER A lapisan 1), dapat %v", result.Pph21)
	}
	if math.Abs(result.Netto-11430000) > 0.01 {
		t.Fatalf("netto = %v", result.Netto)
	}
	if len(result.Components) == 0 {
		t.Fatal("components kosong")
	}
}

func TestCalculatePph21NonZero(t *testing.T) {
	result, err := domain.Calculate(domain.SlipInput{
		Periode:   "2026-06",
		GajiPokok: 20000000,
		Bpjs:      []domain.BpjsRate{},
		Ter: []domain.TerRow{
			{Kategori: "A", Lapisan: 1, BatasBawah: 0, BatasAtas: fptr(15000000), TarifPct: 0},
			{Kategori: "A", Lapisan: 2, BatasBawah: 15000000, TarifPct: 5},
		},
		StatusPtkp: "TK/0",
	})
	if err != nil {
		t.Fatalf("calculate: %v", err)
	}
	if math.Abs(result.Pph21-1000000) > 0.01 {
		t.Fatalf("pph21 = %v, want 1.000.000", result.Pph21)
	}
}

func TestCalculateRejectsNegativeNominal(t *testing.T) {
	_, err := domain.Calculate(domain.SlipInput{
		Periode:    "2026-06",
		GajiPokok:  1000,
		Tunjangan:  []domain.Komponen{{Jenis: "X", Nominal: -1}},
		StatusPtkp: "TK/0",
	})
	if err == nil {
		t.Fatal("nominal negatif harus ditolak")
	}
}

func fptr(value float64) *float64 {
	return &value
}
