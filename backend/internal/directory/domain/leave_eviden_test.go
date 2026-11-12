package domain

import (
	"errors"
	"testing"
	"time"

	"simpeg/backend/internal/models"
)

func TestValidasiEvidenJenisTidakDiperlukan(t *testing.T) {
	item := models.Cuti{Jenis: models.JenisCutiTahunan}
	if err := ValidasiEviden(item); err != nil {
		t.Fatalf("jenis tanpa eviden harus lolos, err = %v", err)
	}
}

func TestValidasiEvidenSakitDenganSuratDokterLengkap(t *testing.T) {
	item := models.Cuti{
		Jenis:              models.JenisCutiSakit,
		JenisEviden:        ptr(models.JenisEvidenSuratDokter),
		FileEvidenUrl:      ptr("https://berkas/sd.pdf"),
		NomorSurat:         ptr("SD-2026-001"),
		TanggalSuratDokter: ptrTime(time.Date(2026, 10, 31, 8, 0, 0, 0, time.UTC)),
	}
	if err := ValidasiEviden(item); err != nil {
		t.Fatalf("sakit dengan surat dokter lengkap harus lolos, err = %v", err)
	}
}

func TestValidasiEvidenSakitTanpaEviden(t *testing.T) {
	item := models.Cuti{Jenis: models.JenisCutiSakit}
	if err := ValidasiEviden(item); !errors.Is(err, ErrValidation) {
		t.Fatalf("sakit tanpa eviden harus ditolak, err = %v", err)
	}
}

func TestValidasiEvidenSakitDenganFotoDitolak(t *testing.T) {
	item := models.Cuti{
		Jenis:         models.JenisCutiSakit,
		JenisEviden:   ptr(models.JenisEvidenFoto),
		FileEvidenUrl: ptr("https://berkas/foto.jpg"),
		WaktuEviden:   ptrTime(time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC)),
	}
	if err := ValidasiEviden(item); !errors.Is(err, ErrValidation) {
		t.Fatalf("sakit dengan foto harus ditolak, err = %v", err)
	}
}

func TestValidasiEvidenHarianDenganSuratDokterDitolak(t *testing.T) {
	item := models.Cuti{
		Jenis:         models.JenisCutiIzinPenting,
		JenisEviden:   ptr(models.JenisEvidenSuratDokter),
		FileEvidenUrl: ptr("https://berkas/sd.pdf"),
		NomorSurat:    ptr("SD-2026-001"),
	}
	if err := ValidasiEviden(item); !errors.Is(err, ErrValidation) {
		t.Fatalf("harian dengan surat dokter harus ditolak, err = %v", err)
	}
}

func TestValidasiEvidenHarianDenganFoto(t *testing.T) {
	item := models.Cuti{
		Jenis:         models.JenisCutiIzinPenting,
		JenisEviden:   ptr(models.JenisEvidenFoto),
		FileEvidenUrl: ptr("https://berkas/foto.jpg"),
		WaktuEviden:   ptrTime(time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC)),
	}
	if err := ValidasiEviden(item); err != nil {
		t.Fatalf("harian dengan foto harus lolos, err = %v", err)
	}
}

func TestValidasiEvidenHarianDenganLainnya(t *testing.T) {
	item := models.Cuti{
		Jenis:         models.JenisCutiIzinPenting,
		JenisEviden:   ptr(models.JenisEvidenLainnya),
		WaktuEviden:   ptrTime(time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC)),
		CatatanEviden: ptr("foto tangkapan layar dari aplikasi presensi"),
	}
	if err := ValidasiEviden(item); err != nil {
		t.Fatalf("harian dengan bukti lain harus lolos, err = %v", err)
	}
}

func TestValidasiEvidenSuratDokterTanpaNomorDitolak(t *testing.T) {
	item := models.Cuti{
		Jenis:         models.JenisCutiBersalin,
		JenisEviden:   ptr(models.JenisEvidenSuratDokter),
		FileEvidenUrl: ptr("https://berkas/sd.pdf"),
	}
	if err := ValidasiEviden(item); !errors.Is(err, ErrValidation) {
		t.Fatalf("surat dokter tanpa nomor harus ditolak, err = %v", err)
	}
}

func TestValidasiEvidenFotoTanpaWaktuDitolak(t *testing.T) {
	item := models.Cuti{
		Jenis:         models.JenisCutiIzinPenting,
		JenisEviden:   ptr(models.JenisEvidenFoto),
		FileEvidenUrl: ptr("https://berkas/foto.jpg"),
	}
	if err := ValidasiEviden(item); !errors.Is(err, ErrValidation) {
		t.Fatalf("foto tanpa waktu harus ditolak, err = %v", err)
	}
}

func TestValidJenisEviden(t *testing.T) {
	for _, jenis := range []models.JenisEviden{
		models.JenisEvidenSuratDokter, models.JenisEvidenFoto, models.JenisEvidenLainnya,
	} {
		if !ValidJenisEviden(jenis) {
			t.Fatalf("jenis %s harus valid", jenis)
		}
	}
	if ValidJenisEviden("BUKAN_EVIDEN") {
		t.Fatal("jenis tidak dikenal harus tidak valid")
	}
}

func TestApplyCutiPatchMempertahankanNilaiLama(t *testing.T) {
	item := models.Cuti{
		IDModel:       models.IDModel{ID: "cuti-1"},
		Jenis:         models.JenisCutiTahunan,
		JenisEviden:   ptr(models.JenisEvidenFoto),
		FileEvidenUrl: ptr("https://berkas/foto.jpg"),
		PersenGaji:    100,
	}
	patch := CutiPatch{JenisEviden: Field{Present: true, Null: true}}
	merged, err := ApplyCutiPatch(item, patch)
	if err != nil {
		t.Fatalf("apply patch tidak boleh error, err = %v", err)
	}
	if merged.JenisEviden != nil {
		t.Fatalf("jenisEviden harus dikosongkan, baru = %v", merged.JenisEviden)
	}
	if merged.FileEvidenUrl == nil || *merged.FileEvidenUrl != "https://berkas/foto.jpg" {
		t.Fatalf("fileEvidenUrl harus dipertahankan bila tidak diubah, baru = %v", merged.FileEvidenUrl)
	}
	if merged.PersenGaji != 100 {
		t.Fatalf("persenGaji harus dipertahankan, baru = %v", merged.PersenGaji)
	}
}

func TestApplyCutiPatchMenggantiSemuaNilai(t *testing.T) {
	item := models.Cuti{Jenis: models.JenisCutiTahunan}
	patch := CutiPatch{
		Jenis:          Field{Present: true, Value: "SAKIT"},
		TanggalMulai:   Field{Present: true, Value: "2026-11-01"},
		TanggalSelesai: Field{Present: true, Value: "2026-11-05"},
		PersenGaji:     Field{Present: true, Value: "80,5"},
		UntukHaji:      Field{Present: true, Value: "true"},
	}
	merged, err := ApplyCutiPatch(item, patch)
	if err != nil {
		t.Fatalf("apply patch tidak boleh error, err = %v", err)
	}
	if merged.Jenis != models.JenisCutiSakit {
		t.Fatalf("jenis harus berubah, baru = %v", merged.Jenis)
	}
	if !merged.TanggalMulai.Equal(time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("tanggalMulai salah, baru = %v", merged.TanggalMulai)
	}
	if merged.PersenGaji != 80.5 {
		t.Fatalf("persenGaji salah, baru = %v", merged.PersenGaji)
	}
	if !merged.UntukHaji {
		t.Fatal("untukHaji harus true")
	}
}

func ptr[T any](v T) *T { return &v }

func ptrTime(t time.Time) *time.Time { return &t }
