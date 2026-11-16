package domain

// Aturan potongan alpa (tanpa keterangan) pada penggajian inti — DEC-076.
//
// Pegawai yang alpa tanpa keterangan dipotong Rp 25.000 per hari alpa pada
// bulan berjalan, dengan batas maksimal Rp 750.000 per bulan. Aturan berlaku
// untuk dosen tetap yayasan, tendik tetap, dan tendik kontrak. Dosen LB
// (honor per SKS) dan dosen DPK (dibayar negara) tidak tercakup.

const (
	// AlpaDailyRateIDR adalah potongan per hari alpa dalam Rupiah.
	AlpaDailyRateIDR = 25000.0
	// AlpaMonthlyCapIDR adalah batas maksimal potongan alpa per bulan.
	AlpaMonthlyCapIDR = 750000.0

	// JenisPotonganAlpa adalah jenis komponen potongan pada slip.
	JenisPotonganAlpa = "ALPA"
)

// EligibleKategoriAlpa memuat kategori status pegawai yang tercakup aturan
// potongan alpa.
var EligibleKategoriAlpa = map[string]bool{
	"DOSEN_TETAP_YAYASAN": true,
	"TENDIK_TETAP":        true,
	"TENDIK_KONTRAK":      true,
}

// EligibleForAlpaDeduction melaporkan apakah kategori status pegawai
// tercakup aturan potongan alpa.
func EligibleForAlpaDeduction(kategoriKode string) bool {
	return EligibleKategoriAlpa[kategoriKode]
}

// AlpaDeduction menghitung potongan alpa untuk sejumlah hari alpa:
// Rp 25.000 per hari dengan batas maksimal Rp 750.000 per bulan.
func AlpaDeduction(days int64) float64 {
	if days <= 0 {
		return 0
	}
	value := float64(days) * AlpaDailyRateIDR
	if value > AlpaMonthlyCapIDR {
		return AlpaMonthlyCapIDR
	}
	return value
}
