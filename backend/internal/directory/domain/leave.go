// Package domain memuat aturan bisnis untuk pengajuan cuti pegawai.
//
// Aturan yang diatur di sini (disetujui produk, `DEC-079`, 2026-11-14 —
// selaras Pedoman Hak Cuti, Kesejahteraan, dan Perlindungan Kerja Universitas):
//   - Hitungan hari kerja: akhir pekan dan hari libur nasional tidak dihitung
//     untuk cuti tahunan dan cuti panjang.
//   - Cuti tahunan: kuota 12 hari kerja per tahun; maksimal 6 hari sisa yang
//     dapat dibawa ke tahun berikutnya; sisa carry hangus 30 Juni tahun
//     berjalan (penghitungan carry bersifat lazy).
//   - Cuti panjang: 2 bulan; syarat masa kerja 6 tahun berturut-turut atau
//     haji; tidak memblokir cuti tahunan pada tahun berjalan.
//   - Cuti bersalin: 3 bulan (MELAHIRKAN); keguguran: 1,5 bulan (45 hari);
//     sub-jenis wajib dipilih.
//   - Izin penting: durasi maksimum per sub-jenis dalam hari kalender
//     (menikah 7, menikahkan anak 4, istri melahirkan/keguguran 4, duka
//     keluarga inti 2, duka keluarga lainnya 1).
//   - Cuti sakit berkepanjangan: persen gaji pokok berjalan 100/75/50/25%
//     (bulan 1–4/5–8/9–12/13–16), dibuat otomatis saat persetujuan (`DEC-079`).
//   - Saldo dipotong saat pengajuan cuti disetujui (`DEC-079`).
//   - Tidak boleh ada cuti disetujui yang tumpang tindih periode per pegawai.
//   - Approver berjenjang: atasan langsung → kepala unit → HRD (TingkatTerusan).
//   - Status: MENUNGGU → DITERIMA / DITOLAK; disetujuiBy diisi saat disetujui.
//   - Cuti melahirkan, cuti sakit, dan cuti harian wajib menyertakan eviden.
package domain

import (
	"strings"
	"time"

	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
)

// JenisCutiPakaiSaldo menandai jenis cuti yang mengurangi saldo cuti.
var JenisCutiPakaiSaldo = map[models.JenisCuti]bool{
	models.JenisCutiTahunan: true,
}

// CutiSortColumns adalah allowlist kolom sortable untuk daftar cuti.
var CutiSortColumns = map[string]string{
	"userId":         `"userId"`,
	"jenis":          `"jenis"`,
	"tanggalMulai":   `"tanggalMulai"`,
	"tanggalSelesai": `"tanggalSelesai"`,
	"status":         `"status"`,
	"createdAt":      `"createdAt"`,
	"updatedAt":      `"updatedAt"`,
}

// CutiSortFields mengembalikan kunci sort yang valid untuk daftar cuti.
func CutiSortFields() []string { return listing.SortKeys(CutiSortColumns) }

// CreateCuti memuat data untuk pengajuan cuti baru.
type CreateCuti struct {
	UserID             string                      `json:"userId"`
	Jenis              models.JenisCuti            `json:"jenis"`
	TanggalMulai       time.Time                   `json:"tanggalMulai"`
	TanggalSelesai     time.Time                   `json:"tanggalSelesai"`
	AlamatCuti         *string                     `json:"alamatCuti"`
	Alasan             *string                     `json:"alasan"`
	SubJenis           *models.SubJenisIzinPenting `json:"subJenis"`
	SubJenisBersalin   *models.SubJenisBersalin    `json:"subJenisBersalin"`
	NomorSurat         *string                     `json:"nomorSurat"`
	TanggalSuratDokter *time.Time                  `json:"tanggalSuratDokter"`
	JenisEviden        *models.JenisEviden         `json:"jenisEviden"`
	FileEvidenUrl      *string                     `json:"fileEvidenUrl"`
	WaktuEviden        *time.Time                  `json:"waktuEviden"`
	CatatanEviden      *string                     `json:"catatanEviden"`
	PersenGaji         float64                     `json:"persenGaji"`
	UntukHaji          bool                        `json:"untukHaji"`
}

// Record memuat data pengajuan cuti dalam bentuk catatan lengkap.
func (input CreateCuti) Record() models.Cuti {
	return models.Cuti{
		UserId:             input.UserID,
		Jenis:              input.Jenis,
		TanggalMulai:       input.TanggalMulai,
		TanggalSelesai:     input.TanggalSelesai,
		AlamatCuti:         input.AlamatCuti,
		Alasan:             input.Alasan,
		SubJenis:           input.SubJenis,
		SubJenisBersalin:   input.SubJenisBersalin,
		NomorSurat:         input.NomorSurat,
		TanggalSuratDokter: input.TanggalSuratDokter,
		JenisEviden:        input.JenisEviden,
		FileEvidenUrl:      input.FileEvidenUrl,
		WaktuEviden:        input.WaktuEviden,
		CatatanEviden:      input.CatatanEviden,
		PersenGaji:         input.PersenGaji,
		UntukHaji:          input.UntukHaji,
	}
}

// Validate memeriksa aturan data pengajuan cuti.
func (input CreateCuti) Validate() error {
	return ValidasiCuti(input.Record())
}

// ValidasiCuti memeriksa aturan pengajuan cuti pada bentuk catatan lengkap.
//
// Pemeriksaan ini dipakai setelah perubahan digabung dengan pengajuan yang
// sudah ada, sehingga evidentiary dan jenis cuti selalu dinilai bersama.
func ValidasiCuti(item models.Cuti) error {
	if !validJenisCuti(item.Jenis) {
		return ErrValidation
	}
	if item.TanggalMulai.IsZero() || item.TanggalSelesai.IsZero() {
		return ErrValidation
	}
	if item.TanggalSelesai.Before(item.TanggalMulai) {
		return ErrValidation
	}
	if item.Alasan != nil && strings.TrimSpace(*item.Alasan) == "" {
		return ErrValidation
	}
	if item.PersenGaji < 0 || item.PersenGaji > 100 {
		return ErrValidation
	}
	if item.Jenis == models.JenisCutiIzinPenting && item.SubJenis == nil {
		return ErrValidation
	}
	if item.JenisEviden != nil && !ValidJenisEviden(*item.JenisEviden) {
		return ErrValidation
	}
	if err := ValidasiDurasiCuti(item); err != nil {
		return err
	}
	return ValidasiEviden(item)
}

// validJenisCuti melaporkan apakah jenis cuti dikenal.
func validJenisCuti(j models.JenisCuti) bool {
	switch j {
	case models.JenisCutiTahunan, models.JenisCutiPanjang, models.JenisCutiBersalin,
		models.JenisCutiSakit, models.JenisCutiSakitBerkepanjangan,
		models.JenisCutiIzinPenting, models.JenisCutiPolitik:
		return true
	}
	return false
}

// validSubJenisBersalin melaporkan apakah sub-jenis bersalin dikenal.
func validSubJenisBersalin(j models.SubJenisBersalin) bool {
	return j == models.SubJenisMelahirkan || j == models.SubJenisKeguguran
}

// teksPatch mengubah nilai field permintaan menjadi pointer teks, atau nil
// bila field dikosongkan secara eksplisit.
func teksPatch(field Field) *string {
	if field.Null {
		return nil
	}
	value := strings.TrimSpace(field.Value)
	if value == "" {
		return nil
	}
	return &value
}

// parseTanggal menguraikan tanggal kalender "YYYY-MM-DD" atau timestamp
// RFC3339. Tanggal kalender dibaca sebagai tengah malam UTC agar tanggal yang
// dibaca kembali tetap sama.
func parseTanggal(value string) (time.Time, bool) {
	trimmed := strings.TrimSpace(value)
	if parsed, err := time.Parse("2006-01-02", trimmed); err == nil {
		return parsed, true
	}
	if parsed, err := time.Parse(time.RFC3339, trimmed); err == nil {
		return parsed, true
	}
	return time.Time{}, false
}

// parseWaktu menguraikan timestamp RFC3339 yang memuat jam.
func parseWaktu(value string) (time.Time, bool) {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, false
	}
	return parsed, true
}

// HariKerjaAntara menghitung jumlah hari kerja antara dua tanggal (inklusif),
// mengecualikan akhir pekan (Sabtu, Minggu). Hari libur nasional tidak
// dapat dihitung karena data kalender libur belum tersedia; pemanggil harus
// mengurangi sendiri bila diperlukan.
func HariKerjaAntara(mulai, selesai time.Time) int {
	if selesai.Before(mulai) {
		return 0
	}
	count := 0
	for d := mulai; !d.After(selesai); d = d.AddDate(0, 0, 1) {
		weekday := d.Weekday()
		if weekday != time.Saturday && weekday != time.Sunday {
			count++
		}
	}
	return count
}
