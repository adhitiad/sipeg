// Package domain memuat aturan bisnis untuk pengajuan cuti pegawai.
//
// Aturan yang diatur di sini (disetujui produk):
//   - Hitungan hari kerja: akhir pekan dan hari libur nasional tidak dihitung
//     untuk cuti tahunan dan cuti besar.
//   - Cuti tahunan: kuota 12 hari kerja per tahun.
//   - Cuti besar: hanya bagi pegawai yang telah bekerja 3 tahun berturut-turut
//     tanpa mengambil cuti tahunan; maksimal 90 hari kerja per 3 tahun.
//   - Cuti melahirkan dan cuti sakit tidak memakai saldo.
//   - Saldo dicadangkan saat pengajuan disetujui, baru terpotong saat cuti dimulai.
//   - Tidak boleh ada cuti disetujui yang tumpang tindih periode per pegawai.
//   - Approver berjenjang: atasan langsung → kepala unit → HRD (TingkatTerusan).
//   - Status: MENUNGGU → DITERIMA / DITOLAK; disetujuiBy diisi saat disetujui.
//   - Cuti melahirkan, cuti sakit, dan cuti harian wajib menyertakan eviden.
package domain

import (
	"strconv"
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

// CutiPatchFields adalah daftar field yang boleh diubah pada pengajuan
// cuti yang masih MENUNGGU.
var CutiPatchFields = map[string]bool{
	"jenis": true, "tanggalMulai": true, "tanggalSelesai": true,
	"alamatCuti": true, "alasan": true, "subJenis": true,
	"nomorSurat": true, "tanggalSuratDokter": true,
	"jenisEviden": true, "fileEvidenUrl": true,
	"waktuEviden": true, "catatanEviden": true,
	"persenGaji": true, "untukHaji": true,
}

// CutiPatch memuat perubahan pengajuan cuti.
type CutiPatch struct {
	Jenis              Field
	TanggalMulai       Field
	TanggalSelesai     Field
	AlamatCuti         Field
	Alasan             Field
	SubJenis           Field
	NomorSurat         Field
	TanggalSuratDokter Field
	JenisEviden        Field
	FileEvidenUrl      Field
	WaktuEviden        Field
	CatatanEviden      Field
	PersenGaji         Field
	UntukHaji          Field
}

// NewCutiPatch menyusun patch dari peta field.
func NewCutiPatch(fields map[string]Field) CutiPatch {
	return CutiPatch{
		Jenis:              fields["jenis"],
		TanggalMulai:       fields["tanggalMulai"],
		TanggalSelesai:     fields["tanggalSelesai"],
		AlamatCuti:         fields["alamatCuti"],
		Alasan:             fields["alasan"],
		SubJenis:           fields["subJenis"],
		NomorSurat:         fields["nomorSurat"],
		TanggalSuratDokter: fields["tanggalSuratDokter"],
		JenisEviden:        fields["jenisEviden"],
		FileEvidenUrl:      fields["fileEvidenUrl"],
		WaktuEviden:        fields["waktuEviden"],
		CatatanEviden:      fields["catatanEviden"],
		PersenGaji:         fields["persenGaji"],
		UntukHaji:          fields["untukHaji"],
	}
}

// Validate memeriksa aturan perubahan pengajuan cuti.
func (patch CutiPatch) Validate() error {
	if patch.Jenis.Present && !patch.Jenis.Null && !validJenisCuti(models.JenisCuti(patch.Jenis.Value)) {
		return ErrValidation
	}
	if patch.JenisEviden.Present && !patch.JenisEviden.Null && !ValidJenisEviden(models.JenisEviden(patch.JenisEviden.Value)) {
		return ErrValidation
	}
	if patch.PersenGaji.Present && !patch.PersenGaji.Null {
		v, err := strconv.ParseFloat(strings.ReplaceAll(patch.PersenGaji.Value, ",", "."), 64)
		if err != nil || v < 0 || v > 100 {
			return ErrValidation
		}
	}
	if patch.TanggalMulai.Present && !patch.TanggalMulai.Null {
		if _, ok := parseTanggal(patch.TanggalMulai.Value); !ok {
			return ErrValidation
		}
	}
	if patch.TanggalSelesai.Present && !patch.TanggalSelesai.Null {
		if _, ok := parseTanggal(patch.TanggalSelesai.Value); !ok {
			return ErrValidation
		}
	}
	if patch.TanggalSuratDokter.Present && !patch.TanggalSuratDokter.Null {
		if _, ok := parseTanggal(patch.TanggalSuratDokter.Value); !ok {
			return ErrValidation
		}
	}
	if patch.WaktuEviden.Present && !patch.WaktuEviden.Null {
		if _, ok := parseWaktu(patch.WaktuEviden.Value); !ok {
			return ErrValidation
		}
	}
	if patch.UntukHaji.Present && !patch.UntukHaji.Null {
		if _, err := strconv.ParseBool(patch.UntukHaji.Value); err != nil {
			return ErrValidation
		}
	}
	return nil
}

// HasChanges melaporkan apakah patch memuat setidaknya satu perubahan.
func (patch CutiPatch) HasChanges() bool {
	return patch.Jenis.Present || patch.TanggalMulai.Present ||
		patch.TanggalSelesai.Present || patch.AlamatCuti.Present ||
		patch.Alasan.Present || patch.SubJenis.Present ||
		patch.NomorSurat.Present || patch.TanggalSuratDokter.Present ||
		patch.JenisEviden.Present || patch.FileEvidenUrl.Present ||
		patch.WaktuEviden.Present || patch.CatatanEviden.Present ||
		patch.PersenGaji.Present || patch.UntukHaji.Present
}

// ApplyCutiPatch menggabungkan perubahan permintaan pada pengajuan cuti yang
// sudah ada. Field yang tidak dikirim tetap memakai nilai lama sehingga
// pemeriksaan aturan selalu menilai keadaan akhir, bukan keadaan sebagian.
func ApplyCutiPatch(item models.Cuti, patch CutiPatch) (models.Cuti, error) {
	merged := item
	if patch.Jenis.Present {
		if patch.Jenis.Null {
			return models.Cuti{}, ErrValidation
		}
		merged.Jenis = models.JenisCuti(patch.Jenis.Value)
	}
	if patch.TanggalMulai.Present {
		t, ok := parseTanggal(patch.TanggalMulai.Value)
		if !ok || patch.TanggalMulai.Null {
			return models.Cuti{}, ErrValidation
		}
		merged.TanggalMulai = t
	}
	if patch.TanggalSelesai.Present {
		t, ok := parseTanggal(patch.TanggalSelesai.Value)
		if !ok || patch.TanggalSelesai.Null {
			return models.Cuti{}, ErrValidation
		}
		merged.TanggalSelesai = t
	}
	if patch.AlamatCuti.Present {
		merged.AlamatCuti = teksPatch(patch.AlamatCuti)
	}
	if patch.Alasan.Present {
		merged.Alasan = teksPatch(patch.Alasan)
	}
	if patch.SubJenis.Present {
		if patch.SubJenis.Null {
			merged.SubJenis = nil
		} else {
			jenis := models.SubJenisIzinPenting(patch.SubJenis.Value)
			merged.SubJenis = &jenis
		}
	}
	if patch.NomorSurat.Present {
		merged.NomorSurat = teksPatch(patch.NomorSurat)
	}
	if patch.TanggalSuratDokter.Present {
		if patch.TanggalSuratDokter.Null {
			merged.TanggalSuratDokter = nil
		} else {
			t, ok := parseTanggal(patch.TanggalSuratDokter.Value)
			if !ok {
				return models.Cuti{}, ErrValidation
			}
			merged.TanggalSuratDokter = &t
		}
	}
	if patch.JenisEviden.Present {
		if patch.JenisEviden.Null {
			merged.JenisEviden = nil
		} else {
			jenis := models.JenisEviden(patch.JenisEviden.Value)
			merged.JenisEviden = &jenis
		}
	}
	if patch.FileEvidenUrl.Present {
		merged.FileEvidenUrl = teksPatch(patch.FileEvidenUrl)
	}
	if patch.WaktuEviden.Present {
		if patch.WaktuEviden.Null {
			merged.WaktuEviden = nil
		} else {
			t, ok := parseWaktu(patch.WaktuEviden.Value)
			if !ok {
				return models.Cuti{}, ErrValidation
			}
			merged.WaktuEviden = &t
		}
	}
	if patch.CatatanEviden.Present {
		merged.CatatanEviden = teksPatch(patch.CatatanEviden)
	}
	if patch.PersenGaji.Present {
		if patch.PersenGaji.Null {
			return models.Cuti{}, ErrValidation
		}
		v, err := strconv.ParseFloat(strings.ReplaceAll(patch.PersenGaji.Value, ",", "."), 64)
		if err != nil {
			return models.Cuti{}, ErrValidation
		}
		merged.PersenGaji = v
	}
	if patch.UntukHaji.Present {
		if patch.UntukHaji.Null {
			return models.Cuti{}, ErrValidation
		}
		v, err := strconv.ParseBool(patch.UntukHaji.Value)
		if err != nil {
			return models.Cuti{}, ErrValidation
		}
		merged.UntukHaji = v
	}
	return merged, nil
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
