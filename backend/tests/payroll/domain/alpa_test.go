package domain_test

import (
	"testing"

	"simpeg/backend/internal/payroll/domain"
)

func TestAlpaDeduction(t *testing.T) {
	cases := []struct {
		days int64
		want float64
	}{
		{-1, 0},
		{0, 0},
		{1, 25000},
		{10, 250000},
		{29, 725000},
		{30, 750000},
		{31, 750000},
		{100, 750000},
	}
	for _, tc := range cases {
		if got := domain.AlpaDeduction(tc.days); got != tc.want {
			t.Fatalf("AlpaDeduction(%d) = %v, want %v", tc.days, got, tc.want)
		}
	}
}

func TestEligibleForAlpaDeduction(t *testing.T) {
	for _, kode := range []string{"DOSEN_TETAP_YAYASAN", "TENDIK_TETAP", "TENDIK_KONTRAK"} {
		if !domain.EligibleForAlpaDeduction(kode) {
			t.Fatalf("%q harus tercakup aturan potongan alpa", kode)
		}
	}
	for _, kode := range []string{"DOSEN_DPK", "DOSEN_LB", "", "LAINNYA"} {
		if domain.EligibleForAlpaDeduction(kode) {
			t.Fatalf("%q tidak boleh tercakup aturan potongan alpa", kode)
		}
	}
}
