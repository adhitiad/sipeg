// Aturan master pesangonAturan: faktor pesangon/uang penghargaan/ganti
// kerugian per kategori status pegawai dan per tahun masa kerja.
package domain

import "errors"

// PesangonAturanInput adalah isian untuk membuat/mengubah satu baris aturan.
type PesangonAturanInput struct {
	KategoriStatusID   string
	MasaKerjaTahun     int
	PesangonBulan      float64
	PenghargaanBulan   float64
	GantiKerugianBulan float64
	SumberRegulasi     string
}

// ErrValidation menandai isian master yang tidak sah.
var ErrValidation = errors.New("isian pesangonAturan tidak sah")

// Validate memeriksa isian; nilai negatif ditolak.
func (i PesangonAturanInput) Validate() error {
	if i.KategoriStatusID == "" {
		return ErrValidation
	}
	if i.MasaKerjaTahun < 0 {
		return ErrValidation
	}
	if i.PesangonBulan < 0 || i.PenghargaanBulan < 0 || i.GantiKerugianBulan < 0 {
		return ErrValidation
	}
	return nil
}
