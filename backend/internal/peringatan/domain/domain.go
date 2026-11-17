// Package domain memuat aturan penerbitan SP1 (peringatan) dari alpa
// berulang (DEC-077): validasi tingkat peringatan, alasan, tanggal, dan
// penerbit. SP1 hanya pencatatan peringatan dan tidak memengaruhi slip gaji.
package domain

import (
	"errors"
	"strings"
	"time"

	"simpeg/backend/internal/models"
)

// ErrInvalidTingkat menandai tingkat peringatan yang tidak dikenal.
var ErrInvalidTingkat = errors.New("tingkat peringatan tidak dikenal")

// ErrInvalidAlasan menandai alasan yang kosong.
var ErrInvalidAlasan = errors.New("alasan wajib diisi")

// ErrInvalidDate menandai tanggal yang kosong/tidak sah.
var ErrInvalidDate = errors.New("tanggal wajib diisi")

// ErrInvalidUserID menandai userId yang kosong.
var ErrInvalidUserID = errors.New("userId wajib diisi")

// ErrInvalidPenerbit menandai penerbit yang kosong.
var ErrInvalidPenerbit = errors.New("penerbit wajib diisi")

// ErrInvalidSanksi menandai sanksi yang tidak dikenal.
var ErrInvalidSanksi = errors.New("sanksi tidak dikenal")

// ErrForbidden menandai penolakan karena permission/cakupan.
var ErrForbidden = errors.New("tidak berwenang menerbitkan peringatan")

// ErrNotFound menandai peringatan yang tidak ditemukan.
var ErrNotFound = errors.New("peringatan tidak ditemukan")

// ValidTingkat melaporkan apakah tingkat peringatan dikenal.
func ValidTingkat(t models.TingkatPeringatan) bool {
	switch t {
	case models.PeringatanLisan, models.PeringatanTertulisI, models.PeringatanTertulisII, models.PeringatanTertulisIII:
		return true
	}
	return false
}

// NormalizeTingkat menormalkan tingkat peringatan (trim + uppercase).
func NormalizeTingkat(raw string) models.TingkatPeringatan {
	return models.TingkatPeringatan(strings.ToUpper(strings.TrimSpace(raw)))
}

// ValidSanksi melaporkan apakah sanksi dikenal (kosong berarti tidak ada).
func ValidSanksi(s *models.JenisSanksi) bool {
	if s == nil || strings.TrimSpace(string(*s)) == "" {
		return true
	}
	switch *s {
	case models.SanksiTeguran, models.SanksiPenundaanKGB, models.SanksiPotongTunjangan:
		return true
	}
	return false
}

// ThresholdAlpaHari adalah ambang alpa tanpa keterangan (14 hari) dalam
// rentang 2 bulan (DEC-077, keputusan pemilik 2026-11-17).
const ThresholdAlpaHari = 14

// ThresholdAlpaBulan adalah rentang bulan untuk ambang alpa.
const ThresholdAlpaBulan = 2

// ValidateInput memeriksa field wajib penerbitan SP1.
func ValidateInput(userID, alasan, penerbit string, tingkat models.TingkatPeringatan, tanggal time.Time, sanksi *models.JenisSanksi) error {
	if strings.TrimSpace(userID) == "" {
		return ErrInvalidUserID
	}
	if strings.TrimSpace(alasan) == "" {
		return ErrInvalidAlasan
	}
	if !ValidTingkat(tingkat) {
		return ErrInvalidTingkat
	}
	if tanggal.IsZero() {
		return ErrInvalidDate
	}
	if strings.TrimSpace(penerbit) == "" {
		return ErrInvalidPenerbit
	}
	if !ValidSanksi(sanksi) {
		return ErrInvalidSanksi
	}
	return nil
}
