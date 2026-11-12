// Package domain memuat aturan bisnis untuk komponen potongan pegawai.
package domain

import (
	"simpeg/backend/internal/directory/domain"
)

// JenisPotonganValid melaporkan apakah jenis potongan valid.
func JenisPotonganValid(value string) bool {
	switch value {
	case "PPH21", "BPJS_KESEHATAN", "BPJS_KETENAGAKERJAAN", "KAS BON", "KOPERASI":
		return true
	}
	return false
}

// CreateKomponenPotongan memuat data untuk membuat entri komponen potongan.
type CreateKomponenPotongan struct {
	UserID         string
	JenisPotongan  string
	Persentase     *float64
	Nominal        float64
	PorsiInstitusi *float64
	PorsiPegawai   *float64
	Aktif          bool
}

// Validate memeriksa aturan data komponen potongan.
func (input CreateKomponenPotongan) Validate() error {
	if !JenisPotonganValid(input.JenisPotongan) {
		return domain.ErrValidation
	}
	if input.Nominal < 0 {
		return domain.ErrValidation
	}
	if input.Persentase != nil && (*input.Persentase < 0 || *input.Persentase > 100) {
		return domain.ErrValidation
	}
	if input.PorsiInstitusi != nil && *input.PorsiInstitusi < 0 {
		return domain.ErrValidation
	}
	if input.PorsiPegawai != nil && *input.PorsiPegawai < 0 {
		return domain.ErrValidation
	}
	return nil
}

// KomponenPotonganPatchFields adalah daftar field yang boleh diubah.
var KomponenPotonganPatchFields = map[string]bool{
	"jenisPotongan": true, "persentase": true, "nominal": true,
	"porsiInstitusi": true, "porsiPegawai": true, "aktif": true,
}

// KomponenPotonganPatch memuat perubahan entri komponen potongan.
type KomponenPotonganPatch struct {
	JenisPotongan  domain.Field
	Persentase     domain.Field
	Nominal        domain.Field
	PorsiInstitusi domain.Field
	PorsiPegawai   domain.Field
	Aktif          domain.Field
}

// NewKomponenPotonganPatch menyusun patch dari peta field.
func NewKomponenPotonganPatch(fields map[string]domain.Field) KomponenPotonganPatch {
	return KomponenPotonganPatch{
		JenisPotongan:  fields["jenisPotongan"],
		Persentase:     fields["persentase"],
		Nominal:        fields["nominal"],
		PorsiInstitusi: fields["porsiInstitusi"],
		PorsiPegawai:   fields["porsiPegawai"],
		Aktif:          fields["aktif"],
	}
}

// Validate memeriksa aturan perubahan komponen potongan.
func (patch KomponenPotonganPatch) Validate() error {
	if patch.JenisPotongan.Present && !patch.JenisPotongan.Null {
		if !JenisPotonganValid(patch.JenisPotongan.Value) {
			return domain.ErrValidation
		}
	}
	if patch.Persentase.Present && !patch.Persentase.Null {
		if patch.Persentase.Value == "" {
			return domain.ErrValidation
		}
	}
	if patch.Nominal.Present && !patch.Nominal.Null {
		if patch.Nominal.Value == "" {
			return domain.ErrValidation
		}
	}
	return nil
}

// HasChanges melaporkan apakah patch memuat setidaknya satu perubahan.
func (patch KomponenPotonganPatch) HasChanges() bool {
	return patch.JenisPotongan.Present || patch.Persentase.Present ||
		patch.Nominal.Present || patch.PorsiInstitusi.Present ||
		patch.PorsiPegawai.Present || patch.Aktif.Present
}
