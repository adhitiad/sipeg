package money

import (
	"math"
	"testing"
)

func TestFormatUsesIndonesianSeparators(t *testing.T) {
	cases := []struct {
		name   string
		amount float64
		want   string
	}{
		{name: "nol", amount: 0, want: "Rp 0"},
		{name: "kurang dari ribuan", amount: 999, want: "Rp 999"},
		{name: "tepat ribuan", amount: 1000, want: "Rp 1.000"},
		{name: "jutaan", amount: 1500000, want: "Rp 1.500.000"},
		{name: "miliar", amount: 1250000000, want: "Rp 1.250.000.000"},
		{name: "desimal", amount: 1500000.5, want: "Rp 1.500.000,50"},
		{name: "desimal satu digit", amount: 1500000.05, want: "Rp 1.500.000,05"},
		{name: "negatif", amount: -2500, want: "-Rp 2.500"},
		{name: "pembulatan ke atas", amount: 0.005, want: "Rp 0,01"},
		{name: "pembulatan ke bawah", amount: 1.999, want: "Rp 2"},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := Format(test.amount); got != test.want {
				t.Fatalf("Format(%v) = %q, want %q", test.amount, got, test.want)
			}
		})
	}
}

func TestFormatWithCodeIncludesCurrencyCode(t *testing.T) {
	if got := FormatWithCode(1500000); got != "Rp 1.500.000 IDR" {
		t.Fatalf("FormatWithCode = %q", got)
	}
}

func TestFormatHandlesNonFiniteValues(t *testing.T) {
	if got := Format(math.NaN()); got != "Rp 0" {
		t.Fatalf("Format(NaN) = %q", got)
	}
}
