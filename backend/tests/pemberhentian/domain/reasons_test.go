package domain

import (
	"testing"

	"simpeg/backend/internal/models"
	. "simpeg/backend/internal/pemberhentian/domain"
)

func TestValidAlasan(t *testing.T) {
	for _, a := range AlasanList() {
		if !ValidAlasan(a) {
			t.Fatalf("alasan %q harus valid", a)
		}
	}
	if ValidAlasan("ALASAN_BARU") {
		t.Fatal("alasan tak dikenal tidak boleh valid")
	}
	if !ValidAlasan("meninggal_dunia") {
		t.Fatal("ValidAlasan menormalkan case (sikap tolerant), harus tetap valid")
	}
}

func TestJenisFromAlasan(t *testing.T) {
	cases := []struct {
		alasan string
		want   models.JenisPemberhentian
	}{
		{AlasanBup, models.PemberhentianBup},
		{AlasanMengundurDiri, models.PemberhentianAtasPermintaan},
		{AlasanKesalahanBerat, models.PemberhentianTidakHormat},
		{AlasanMeninggalDunia, models.PemberhentianHormat},
		{AlasanSakitBerkelanjutan, models.PemberhentianHormat},
		{AlasanPenyederhanaan, models.PemberhentianHormat},
	}
	for _, c := range cases {
		got, err := JenisFromAlasan(c.alasan)
		if err != nil {
			t.Fatalf("%s: %v", c.alasan, err)
		}
		if got != c.want {
			t.Fatalf("%s -> %v, want %v", c.alasan, got, c.want)
		}
	}
}

func TestHonoured(t *testing.T) {
	honoured := []string{AlasanMeninggalDunia, AlasanSakitBerkelanjutan, AlasanBup, AlasanMengundurDiri, AlasanPenyederhanaan}
	for _, a := range honoured {
		if !Honoured(a) {
			t.Fatalf("%s harus hormat", a)
		}
	}
	if Honoured(AlasanKesalahanBerat) {
		t.Fatal("kesalahan berat tidak boleh hormat")
	}
}

func TestHitungCompensationTidakHormat(t *testing.T) {
	c := HitungCompensation(map[int]float64{5: 10}, 7, false)
	if c.TotalBulan != 0 || !c.Tersedia {
		t.Fatalf("tidak hormat = %+v, want 0", c)
	}
}

func TestHitungCompensationHormatTanpaAturan(t *testing.T) {
	c := HitungCompensation(nil, 7, true)
	if c.Tersedia {
		t.Fatalf("tanpa aturan harus Tersedia=false, got %+v", c)
	}
}

func TestHitungCompensationHormat(t *testing.T) {
	c := HitungCompensation(map[int]float64{0: 1, 5: 2, 10: 4}, 12, true)
	if c.PesangonBulan != 4 {
		t.Fatalf("pesangon = %v, want 4 (lantai 10)", c.PesangonBulan)
	}
	if c.TambahanBulan != 2 {
		t.Fatalf("tambahan = %v, want 2 (12/5)", c.TambahanBulan)
	}
	if c.TotalBulan != 6 {
		t.Fatalf("total = %v, want 6", c.TotalBulan)
	}
}
