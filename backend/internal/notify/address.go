package notify

import (
	"fmt"
	"strings"
)

// SplitAddress memisahkan alamat email menjadi bentuk polos.
//
// Bentuk baku RFC 5322 "Nama Tampilan <alamat@domain>" lazim dipakai pada
// header From. Bentuk itu dipakai apa adanya pada header email, sedangkan
// perintah SMTP MAIL FROM memerlukan alamat polos. Fungsi ini mengembalikan
// alamat polos dan ok=false bila nilai tidak dapat diurai.
func SplitAddress(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", false
	}
	open := strings.LastIndex(trimmed, "<")
	if open < 0 {
		return trimmed, true
	}
	close := strings.LastIndex(trimmed, ">")
	if close <= open {
		return "", false
	}
	address := strings.TrimSpace(trimmed[open+1 : close])
	if address == "" {
		return "", false
	}
	return address, true
}

// EnvelopeSender mengembalikan alamat polos untuk perintah MAIL FROM.
//
// Nilai dikembalikan apa adanya bila tidak dapat diurai, karena pemanggil
// sudah memvalidasi bentuknya lebih dulu.
func EnvelopeSender(from string) string {
	address, ok := SplitAddress(from)
	if !ok {
		return strings.TrimSpace(from)
	}
	return address
}

// ValidateAddress memeriksa bentuk alamat email sederhana.
//
// Bentuk polos dan bentuk "Nama Tampilan <alamat@domain>" sama-sama
// diterima. Pemeriksaan lengkap dilakukan oleh pustaka email saat
// pengiriman.
func ValidateAddress(value string) error {
	address, ok := SplitAddress(value)
	if !ok {
		return fmt.Errorf("alamat email tidak valid: %s", value)
	}
	at := strings.Index(address, "@")
	if at <= 0 || at == len(address)-1 {
		return fmt.Errorf("alamat email tidak valid: %s", value)
	}
	local, domain := address[:at], address[at+1:]
	if strings.ContainsAny(local, " \t\r\n") || strings.ContainsAny(domain, " \t\r\n") {
		return fmt.Errorf("alamat email tidak valid: %s", value)
	}
	if !strings.Contains(domain, ".") || strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return fmt.Errorf("alamat email tidak valid: %s", value)
	}
	return nil
}
