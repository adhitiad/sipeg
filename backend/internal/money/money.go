// Package money formats monetary values in Indonesian Rupiah.
//
// SIMPEG memakai satu mata uang saja: Rupiah Indonesia (IDR). Tidak ada
// konversi antar-mata uang dan tidak ada nilai tukar yang disimpan di sistem.
// Nilai uang pada database adalah angka desimal dalam Rupiah penuh; paket ini
// hanya menyediakan representasi teks untuk ditampilkan ke pengguna.
package money

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Code adalah kode mata uang ISO 4217 yang dipakai SIMPEG.
const Code = "IDR"

// Symbol adalah simbol mata uang yang dipakai pada teks rupiah.
const Symbol = "Rp"

const (
	// decimals adalah jumlah desimal yang dipertahankan saat pembulatan.
	decimals = 2
	// decimalScale adalah pengali untuk mengubah nilai uang ke desimal penuh.
	decimalScale = 100
)

// Format mengubah nilai uang menjadi teks rupiah dengan konvensi Indonesia:
// pemisah ribuan memakai titik dan pemisah desimal memakai koma.
//
// Contoh: 1500000 -> "Rp 1.500.000"; 1500000.5 -> "Rp 1.500.000,50".
func Format(amount float64) string {
	if math.IsNaN(amount) || math.IsInf(amount, 0) {
		return Symbol + " 0"
	}

	negative := amount < 0
	value := math.Abs(math.Round(amount*decimalScale)) / decimalScale
	whole := int64(value)
	fraction := int64(math.Round((value - float64(whole)) * decimalScale))

	text := groupThousands(whole)
	if fraction > 0 {
		text += "," + twoDigits(fraction)
	}
	if negative {
		return "-" + Symbol + " " + text
	}
	return Symbol + " " + text
}

// FormatWithCode mengubah nilai uang menjadi teks rupiah yang mencantumkan
// kode mata uang, misalnya "Rp 1.500.000 IDR".
func FormatWithCode(amount float64) string {
	return Format(amount) + " " + Code
}

// groupThousands mengelompokkan digit ribuan memakai titik.
func groupThousands(value int64) string {
	digits := strconv.FormatInt(value, 10)
	if len(digits) <= 3 {
		return digits
	}

	var builder strings.Builder
	head := len(digits) % 3
	if head > 0 {
		builder.WriteString(digits[:head])
	}
	for index := head; index < len(digits); index += 3 {
		if builder.Len() > 0 {
			builder.WriteByte('.')
		}
		builder.WriteString(digits[index : index+3])
	}
	return builder.String()
}

// twoDigits memformat bagian desimal dengan nol di depan bila perlu.
func twoDigits(value int64) string {
	return fmt.Sprintf("%0*d", decimals, value)
}
