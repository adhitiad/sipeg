package domain

import (
	"strings"
	"time"

	"simpeg/backend/internal/models"
)

// Batas panjang data konfirmasi pembayaran.
const (
	MaxReferenceLength = 200
	MaxActorLength     = 200
	MaxNoteLength      = 2000
)

// PaymentInput memuat bukti pembayaran yang dicatat untuk sebuah slip gaji.
//
// Nominal tidak pernah menjadi bagian dari data ini: nominal tetap pada slip
// gaji dan hanya ditampilkan pada halaman terautentikasi.
type PaymentInput struct {
	// ReferensiTransaksi adalah nomor referensi batch atau transaksi bank.
	ReferensiTransaksi *string
	// PelakuPengesahan adalah nama atau NIP pihak yang mengonfirmasi.
	PelakuPengesahan string
	// TanggalBayar adalah tanggal efektif pembayaran.
	TanggalBayar time.Time
	// Keterangan adalah catatan tambahan konfirmasi.
	Keterangan *string
}

// Validate memeriksa aturan data konfirmasi pembayaran.
func (input PaymentInput) Validate() error {
	if strings.TrimSpace(input.PelakuPengesahan) == "" ||
		len(input.PelakuPengesahan) > MaxActorLength {
		return ErrValidation
	}
	if input.ReferensiTransaksi != nil && len(*input.ReferensiTransaksi) > MaxReferenceLength {
		return ErrValidation
	}
	if input.Keterangan != nil && len(*input.Keterangan) > MaxNoteLength {
		return ErrValidation
	}
	if input.TanggalBayar.IsZero() {
		return ErrValidation
	}
	return nil
}

// CanMarkPaid melaporkan apakah slip pada status tertentu boleh ditandai sudah
// dibayar.
//
// Slip berstatus DRAF belum final sehingga belum dapat dibayar. Slip yang
// sudah TERBAYAR tidak dapat dibayar dua kali.
func CanMarkPaid(status models.StatusSlipGaji) bool {
	return status == models.StatusSlipTerbit
}

// StatusLabel mengembalikan label status slip dalam Bahasa Indonesia.
func StatusLabel(status models.StatusSlipGaji) string {
	switch status {
	case models.StatusSlipDraf:
		return "Draf"
	case models.StatusSlipTerbit:
		return "Terbit"
	case models.StatusSlipTerbayar:
		return "Terbayar"
	default:
		return "Tidak Diketahui"
	}
}
