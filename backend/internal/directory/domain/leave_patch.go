// Perubahan pengajuan cuti yang masih MENUNGGU. Pemisahan berkas ini
// menjaga setiap berkas di bawah batas ukuran sumber (387 baris).
package domain

import (
	"strconv"
	"strings"

	"simpeg/backend/internal/models"
)

// CutiPatchFields adalah daftar field yang boleh diubah pada pengajuan
// cuti yang masih MENUNGGU.
var CutiPatchFields = map[string]bool{
	"jenis": true, "tanggalMulai": true, "tanggalSelesai": true,
	"alamatCuti": true, "alasan": true, "subJenis": true, "subJenisBersalin": true,
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
	SubJenisBersalin   Field
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
		SubJenisBersalin:   fields["subJenisBersalin"],
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
	if patch.SubJenisBersalin.Present && !patch.SubJenisBersalin.Null {
		if !validSubJenisBersalin(models.SubJenisBersalin(patch.SubJenisBersalin.Value)) {
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
		patch.SubJenisBersalin.Present ||
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
	if patch.SubJenisBersalin.Present {
		if patch.SubJenisBersalin.Null {
			merged.SubJenisBersalin = nil
		} else {
			v := models.SubJenisBersalin(patch.SubJenisBersalin.Value)
			merged.SubJenisBersalin = &v
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
