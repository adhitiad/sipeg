// Aturan durasi cuti menurut Pedoman Hak Cuti, Kesejahteraan, dan
// Perlindungan Kerja Universitas (diimplementasikan 2026-11-14, `DEC-079`).
//
// Aturan yang diatur di sini bersifat murni (tanpa I/O): durasi maksimum per
// jenis/sub-jenis dan sub-jenis wajib. Syarat masa kerja dan saldo tahunan
// diperiksa di lapisan aplikasi karena membutuhkan data pegawai.
package domain

import (
	"time"

	"simpeg/backend/internal/models"
)

const (
	// MasaKerjaCutiTahunan adalah masa kerja minimum untuk cuti tahunan:
	// 1 tahun secara terus-menerus.
	MasaKerjaCutiTahunan = time.Hour * 24 * 365
	// MasaKerjaCutiPanjang adalah masa kerja minimum untuk cuti panjang:
	// 6 tahun berturut-turut (alternatif: haji, lewat UntukHaji).
	MasaKerjaCutiPanjang = time.Hour * 24 * 365 * 6
	// CutiPanjangBulan adalah durasi maksimum cuti panjang (2 bulan).
	CutiPanjangBulan = 2
	// CutiBersalinBulan adalah durasi maksimum cuti melahirkan (3 bulan).
	CutiBersalinBulan = 3
	// CutiKeguguranHari adalah durasi maksimum istirahat pascakeguguran
	// (1,5 bulan, 45 hari).
	CutiKeguguranHari = 45
	// CarryCutiTahunanMaks adalah maksimal sisa cuti tahunan yang dapat
	// dibawa ke tahun berikutnya.
	CarryCutiTahunanMaks = 6
	// SisaCarryHangusBulan/Hari adalah batas hangus sisa carry: 30 Juni
	// tahun berjalan.
	SisaCarryHangusBulan = time.June
	SisaCarryHangusHari  = 30
)

// izinPentingMaksHariKalender memuat batas durasi izin penting per sub-jenis
// (hari kalender berturut-turut sejak tanggal mulai).
var izinPentingMaksHariKalender = map[models.SubJenisIzinPenting]int{
	models.SubJenisMenikah:         7,
	models.SubJenisMenikahkanAnak:  4,
	models.SubJenisMelahirkanIstri: 4,
	models.SubJenisKeluargaInti:    2,
	models.SubJenisKeluargaLain:    1,
}

// KalenderHariAntara menghitung jumlah hari kalender antara dua tanggal
// (inklusif).
func KalenderHariAntara(mulai, selesai time.Time) int {
	if selesai.Before(mulai) {
		return 0
	}
	return int(selesai.Sub(mulai).Hours()/24) + 1
}

// SisaCuti mengembalikan sisa hari cuti yang masih boleh dipakai pada sebuah
// baris saldo. Bila carry sudah hangus (on/after 30 Juni), hanya kuota utama
// yang dihitung.
func SisaCuti(row models.SaldoCuti) int {
	remaining := row.KuotaTahunan - row.Terpakai
	if !row.Hangus {
		remaining += row.SisaCarry
	}
	if remaining < 0 {
		return 0
	}
	return remaining
}

// PersenSakitBerkepanjangan mengembalikan persen gaji pokok yang dibayarkan
// pada bulan ke-n cuti sakit berkepanjangan. Bulan 1–4: 100%; 5–8: 75%;
// 9–12: 50%; 13–16: 25%; sesudah 16: 0 (proteksi berakhir).
func PersenSakitBerkepanjangan(bulanKe int) float64 {
	switch {
	case bulanKe <= 4:
		return 100
	case bulanKe <= 8:
		return 75
	case bulanKe <= 12:
		return 50
	case bulanKe <= 16:
		return 25
	default:
		return 0
	}
}

// ValidasiDurasiCuti memeriksa sub-jenis wajib dan durasi maksimum menurut
// aturan Universitas. Pemeriksaan masa kerja dan saldo dilakukan di lapisan
// aplikasi (butuh data pegawai).
func ValidasiDurasiCuti(item models.Cuti) error {
	switch item.Jenis {
	case models.JenisCutiPanjang:
		if item.TanggalSelesai.After(item.TanggalMulai.AddDate(0, CutiPanjangBulan, 0)) {
			return ErrValidation
		}
	case models.JenisCutiBersalin:
		if item.SubJenisBersalin == nil {
			return ErrValidation
		}
		var limit time.Time
		switch *item.SubJenisBersalin {
		case models.SubJenisMelahirkan:
			limit = item.TanggalMulai.AddDate(0, CutiBersalinBulan, 0)
		case models.SubJenisKeguguran:
			limit = item.TanggalMulai.AddDate(0, 0, CutiKeguguranHari)
		default:
			return ErrValidation
		}
		if item.TanggalSelesai.After(limit) {
			return ErrValidation
		}
	case models.JenisCutiIzinPenting:
		if item.SubJenis == nil {
			return ErrValidation
		}
		maks, ok := izinPentingMaksHariKalender[*item.SubJenis]
		if !ok {
			return ErrValidation
		}
		if KalenderHariAntara(item.TanggalMulai, item.TanggalSelesai) > maks {
			return ErrValidation
		}
	}
	return nil
}

// MasaKerjaCukup melaporkan apakah tanggal mulai cuti sudah memenuhi masa
// kerja minimum (tanggalMasuk + masaKerja sejak awal tahun berjalan untuk
// cuti tahunan tidak dipakai; syarat dinilai pada tanggal mulai cuti).
func MasaKerjaCukup(tanggalMasuk *time.Time, masaKerja time.Duration, tanggalCuti time.Time) bool {
	if tanggalMasuk == nil {
		// Data tanggal masuk tidak tersedia; syarat dilewati dan
		// dicatat sebagai asumsi kerja (DEC-079).
		return true
	}
	return !tanggalCuti.Before(tanggalMasuk.Add(masaKerja))
}

// KategoriStatusMembatasiCuti melaporkan apakah kategori status pegawai
// membatasi hak cuti tahunan/panjang/bersalin (DEC-092). Kategori dengan
// kode DOSEN_LB (honor) dan DOSEN_DPK (dosen DPK via LLDikti) tidak mendapat
// cuti tahunan, panjang, dan bersalin. Kode kosong (kategori belum ditetapkan
// atau migrasi 000025 belum diterapkan) dianggap tidak membatasi (asumsi
// kerja, backward compatible).
func KategoriStatusMembatasiCuti(kode string) bool {
	switch kode {
	case "DOSEN_LB", "DOSEN_DPK":
		return true
	}
	return false
}

// JenisCutiDiizinkanUntukKategori melaporkan apakah jenis cuti tertentu
// diizinkan untuk kategori status dengan kode yang diberikan (DEC-092).
// Untuk kategori yang membatasi, hanya sakit (biasa & berkepanjangan), izin
// penting, dan politik yang diizinkan; tahunan, panjang, dan bersalin
// ditolak. Kode kosong tidak membatasi (semua jenis diizinkan).
func JenisCutiDiizinkanUntukKategori(jenis models.JenisCuti, kode string) bool {
	if !KategoriStatusMembatasiCuti(kode) {
		return true
	}
	switch jenis {
	case models.JenisCutiTahunan, models.JenisCutiPanjang, models.JenisCutiBersalin:
		return false
	}
	return true
}

// MasaKerjaPanjangSufficient melaporkan apakah pegawai memenuhi syarat cuti
// panjang lewat masa kerja 6 tahun berturut-turut.
func MasaKerjaPanjangSufficient(tanggalMasuk *time.Time, tanggalCuti time.Time) bool {
	if tanggalMasuk == nil {
		return false // syarat haji tetap tersedia
	}
	return !tanggalCuti.Before(tanggalMasuk.Add(MasaKerjaCutiPanjang))
}
