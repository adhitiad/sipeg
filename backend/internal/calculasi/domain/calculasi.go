// Package domain memuat aturan perhitungan slip gaji: resolusi skala, bruto,
// BPJS, PPh 21 TER (PMK 168/2023), dan gaji bersih.
//
// Seluruh nominal dalam Rupiah Indonesia. Lapisan ini tidak melakukan akses
// database maupun HTTP; seluruh data masuk lewat SlipInput.
package domain

import "errors"

// ErrNotFound menandai data skala/tarif yang tidak tersedia untuk periode.
var ErrNotFound = errors.New("scale or rate not found")

// ErrValidation menandai input perhitungan yang tidak sah.
var ErrValidation = errors.New("payroll calculation value is not valid")

// Komponen adalah satu komponen tunjangan/potongan untuk satu periode.
type Komponen struct {
	Jenis   string
	Nominal float64
}

// BpjsRate adalah konfigurasi iuran BPJS berlaku untuk satu jenis.
type BpjsRate struct {
	Jenis                string
	IuranPegawaiPct      float64
	IuranPemberiKerjaPct float64
	Dasar                string
	BatasAtas            *float64
}

// TerRow adalah satu baris tabel tarif efektif PPh 21 TER.
type TerRow struct {
	Kategori   string
	Lapisan    int
	BatasBawah float64
	BatasAtas  *float64
	TarifPct   float64
}

// SlipInput memuat seluruh data yang dibutuhkan untuk menghitung satu slip.
type SlipInput struct {
	Periode    string
	GajiPokok  float64
	HonorSks   float64
	RatePerSks float64
	Tunjangan  []Komponen
	Potongan   []Komponen
	Bpjs       []BpjsRate
	Ter        []TerRow
	StatusPtkp string
}

// SlipComponent adalah satu baris yang disimpan pada slip (penerimaan/poton).
type SlipComponent struct {
	Jenis   string
	Nominal float64
}

// SlipResult adalah hasil perhitungan slip gaji.
type SlipResult struct {
	GajiPokok           float64
	HonorTotal          float64
	TunjanganNonHonor   float64
	TunjanganTotal      float64
	Bruto               float64
	BpjsKesehatan       float64
	BpjsKetenagakerjaan float64
	BpjsPegawaiTotal    float64
	BpjsPemberiKerja    float64
	Pph21               float64
	PotonganLain        float64
	Netto               float64
	PtkpKategori        string
	Components          []SlipComponent
}

// BasisBpjs mengembalikan dasar pengenaan iuran BPJS sesuai konfigurasi:
// GAJI_POKOK memakai gaji pokok, TOTAL_GAJI memakai bruto (dihitung setelah
// seluruh penerimaan).
func BasisBpjs(rate BpjsRate, gajiPokok, bruto float64) float64 {
	switch rate.Dasar {
	case "TOTAL_GAJI":
		return bruto
	default:
		return gajiPokok
	}
}

// ComputeBpjs menghitung iuran BPJS pegawai dan pemberi kerja untuk satu jenis,
// dengan batas atas bila dikonfigurasi.
func ComputeBpjs(rate BpjsRate, gajiPokok, bruto float64) (pegawai, pemberiKerja float64) {
	basis := BasisBpjs(rate, gajiPokok, bruto)
	if rate.BatasAtas != nil && basis > *rate.BatasAtas {
		basis = *rate.BatasAtas
	}
	pegawai = rate.IuranPegawaiPct / 100 * basis
	pemberiKerja = rate.IuranPemberiKerjaPct / 100 * basis
	return pegawai, pemberiKerja
}

// PtkpToTer memetakan status PTKP ke kategori TER (PMK 168/2023):
// A = TK/0, TK/1, TK/2, TK/3
// B = K/0, K/1, K/2, K/3
// C = K/I/0, K/I/1, K/I/2, K/I/3
func PtkpToTer(ptkp string) string {
	switch ptkp {
	case "TK/0", "TK/1", "TK/2", "TK/3":
		return "A"
	case "K/0", "K/1", "K/2", "K/3":
		return "B"
	case "K/I/0", "K/I/1", "K/I/2", "K/I/3":
		return "C"
	default:
		return "A"
	}
}

// ResolveTer mencari tarif efektif PPh 21 TER untuk penghasilan bruto bulanan
// tertentu pada tabel yang berlaku. Mengembalikan ErrNotFound bila tidak ada
// lapisan yang cocok.
func ResolveTer(rows []TerRow, kategori string, grossMonthly float64) (TerRow, error) {
	return resolveTer(rows, kategori, grossMonthly)
}

// ResolveTarif adalah alias ResolveTer untuk pemakaian lapisan slip.
func ResolveTarif(rows []TerRow, kategori string, grossMonthly float64) (TerRow, error) {
	return resolveTer(rows, kategori, grossMonthly)
}

func resolveTer(rows []TerRow, kategori string, grossMonthly float64) (TerRow, error) {
	var found *TerRow
	for i := range rows {
		row := rows[i]
		if row.Kategori != kategori {
			continue
		}
		if grossMonthly < row.BatasBawah {
			continue
		}
		if row.BatasAtas != nil && grossMonthly >= *row.BatasAtas {
			continue
		}
		if found == nil {
			found = &rows[i]
		}
	}
	if found == nil {
		return TerRow{}, ErrNotFound
	}
	return *found, nil
}

// Calculate menghitung satu slip gaji dari SlipInput.
func Calculate(input SlipInput) (SlipResult, error) {
	if input.Periode == "" {
		return SlipResult{}, ErrValidation
	}
	if input.GajiPokok < 0 || input.HonorSks < 0 || input.RatePerSks < 0 {
		return SlipResult{}, ErrValidation
	}

	honorTotal := input.HonorSks * input.RatePerSks
	tunjanganNonHonor := 0.0
	components := []SlipComponent{
		{Jenis: "GAJI_POKOK", Nominal: input.GajiPokok},
	}
	if honorTotal > 0 {
		components = append(components, SlipComponent{Jenis: "HONOR_SKS", Nominal: honorTotal})
	}
	for _, komponen := range input.Tunjangan {
		if komponen.Nominal < 0 {
			return SlipResult{}, ErrValidation
		}
		tunjanganNonHonor += komponen.Nominal
		components = append(components, SlipComponent(komponen))
	}
	tunjanganTotal := tunjanganNonHonor + honorTotal
	bruto := input.GajiPokok + tunjanganTotal

	bpjsPegawaiTotal := 0.0
	bpjsPemberiKerja := 0.0
	var bpjsKesehatan, bpjsKetenagakerjaan float64
	for _, rate := range input.Bpjs {
		pegawai, pemberiKerja := ComputeBpjs(rate, input.GajiPokok, bruto)
		bpjsPegawaiTotal += pegawai
		bpjsPemberiKerja += pemberiKerja
		switch rate.Jenis {
		case "KESEHATAN":
			bpjsKesehatan = pegawai
			components = append(components, SlipComponent{Jenis: "BPJS_KESEHATAN", Nominal: pegawai})
		case "KETENAGAKERJAAN":
			bpjsKetenagakerjaan = pegawai
			components = append(components, SlipComponent{Jenis: "BPJS_KETENAGAKERJAAN", Nominal: pegawai})
		}
	}

	pph21 := 0.0
	if bruto > 0 {
		kategori := PtkpToTer(input.StatusPtkp)
		row, err := ResolveTer(input.Ter, kategori, bruto)
		if err != nil {
			return SlipResult{}, err
		}
		pph21 = row.TarifPct / 100 * bruto
		if pph21 > 0 {
			components = append(components, SlipComponent{Jenis: "PPH21", Nominal: pph21})
		}
	}

	potonganLain := 0.0
	for _, komponen := range input.Potongan {
		if komponen.Nominal < 0 {
			return SlipResult{}, ErrValidation
		}
		potonganLain += komponen.Nominal
		components = append(components, SlipComponent(komponen))
	}

	netto := bruto - bpjsPegawaiTotal - pph21 - potonganLain

	return SlipResult{
		GajiPokok:           input.GajiPokok,
		HonorTotal:          honorTotal,
		TunjanganNonHonor:   tunjanganNonHonor,
		TunjanganTotal:      tunjanganTotal,
		Bruto:               bruto,
		BpjsKesehatan:       bpjsKesehatan,
		BpjsKetenagakerjaan: bpjsKetenagakerjaan,
		BpjsPegawaiTotal:    bpjsPegawaiTotal,
		BpjsPemberiKerja:    bpjsPemberiKerja,
		Pph21:               pph21,
		PotonganLain:        potonganLain,
		Netto:               netto,
		PtkpKategori:        PtkpToTer(input.StatusPtkp),
		Components:          components,
	}, nil
}
