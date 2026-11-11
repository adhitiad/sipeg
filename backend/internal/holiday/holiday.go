// Package holiday memuat kalender hari libur nasional Indonesia.
//
// Kalender diisi dari penyedia hari libur pihak ketiga lewat sinkronisasi
// tahunan yang dipicu admin. Data disimpan pada tabel `HariLibur` dan dibaca
// sebagai data referensi; pembacaan tidak pernah bergantung pada penyedia.
// Perubahan kalender tidak memicu pergeseran tanggal gajian maupun
// pemberitahuan slip gaji H+1, yang memakai tanggal kalender tetap.
package holiday

import (
	"errors"
	"fmt"
	"time"
)

// Batas tahun yang diterima untuk sinkronisasi dan pembacaan kalender.
const (
	MinYear = 2000
	MaxYear = 2100
)

// ErrInvalidYear menandai tahun di luar rentang yang diterima.
var ErrInvalidYear = errors.New("holiday year is not valid")

// ErrProviderNotConfigured menandai penyedia hari libur belum dikonfigurasi.
var ErrProviderNotConfigured = errors.New("holiday provider is not configured")

// ErrProvider menandai kegagalan penyedia hari libur.
type ErrProvider struct {
	// Detail memuat alasan ringkas tanpa kunci API.
	Detail string
}

func (e ErrProvider) Error() string { return "holiday provider: " + e.Detail }

// Entry adalah satu hari libur pada kalender.
type Entry struct {
	Tanggal time.Time `json:"tanggal"`
	Nama    string    `json:"nama"`
}

// ValidateYear memeriksa rentang tahun kalender.
func ValidateYear(year int) error {
	if year < MinYear || year > MaxYear {
		return fmt.Errorf("%w: tahun harus %d-%d", ErrInvalidYear, MinYear, MaxYear)
	}
	return nil
}
