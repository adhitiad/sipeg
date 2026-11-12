// Package domain memuat aturan bisnis untuk pemetaan ID eksternal SISTER/PDDIKTI.
package domain

import (
	"strings"

	"simpeg/backend/internal/models"
)

// SistemEksternalValid melaporkan apakah sistem eksternal valid.
func SistemEksternalValid(value models.SistemEksternal) bool {
	switch value {
	case models.SistemSISTER, models.SistemPDDIKTI, models.SistemLainnya:
		return true
	}
	return false
}

// StatusSinkronisasiValid melaporkan apakah status sinkronisasi valid.
func StatusSinkronisasiValid(value models.StatusSinkronisasi) bool {
	switch value {
	case models.SinkronisasiAktif, models.SinkronisasiNonaktif:
		return true
	}
	return false
}

// CreateSisterSyncMap memuat data untuk membuat entri pemetaan ID eksternal.
type CreateSisterSyncMap struct {
	UserID          string
	SistemEksternal models.SistemEksternal
	IDEksternal     string
	Status          models.StatusSinkronisasi
}

// Validate memeriksa aturan data pemetaan ID eksternal.
func (input CreateSisterSyncMap) Validate() error {
	if !SistemEksternalValid(input.SistemEksternal) {
		return ErrValidation
	}
	if strings.TrimSpace(input.IDEksternal) == "" || TextLength(input.IDEksternal) > 100 {
		return ErrValidation
	}
	if !StatusSinkronisasiValid(input.Status) {
		return ErrValidation
	}
	return nil
}

// SisterSyncMapPatchFields adalah daftar field yang boleh diubah.
var SisterSyncMapPatchFields = map[string]bool{
	"sistemEksternal": true, "idEksternal": true, "sinkronisasiTerakhir": true, "status": true,
}

// SisterSyncMapPatch memuat perubahan entri pemetaan ID eksternal.
type SisterSyncMapPatch struct {
	SistemEksternal      Field
	IDEksternal          Field
	SinkronisasiTerakhir Field
	Status               Field
}

// NewSisterSyncMapPatch menyusun patch dari peta field.
func NewSisterSyncMapPatch(fields map[string]Field) SisterSyncMapPatch {
	return SisterSyncMapPatch{
		SistemEksternal:      fields["sistemEksternal"],
		IDEksternal:          fields["idEksternal"],
		SinkronisasiTerakhir: fields["sinkronisasiTerakhir"],
		Status:               fields["status"],
	}
}

// Validate memeriksa aturan perubahan pemetaan ID eksternal.
func (patch SisterSyncMapPatch) Validate() error {
	if patch.SistemEksternal.Present && !patch.SistemEksternal.Null {
		if !SistemEksternalValid(models.SistemEksternal(patch.SistemEksternal.Value)) {
			return ErrValidation
		}
	}
	if patch.IDEksternal.Present {
		if patch.IDEksternal.Null || strings.TrimSpace(patch.IDEksternal.Value) == "" ||
			TextLength(patch.IDEksternal.Value) > 100 {
			return ErrValidation
		}
	}
	if patch.SinkronisasiTerakhir.Present && !patch.SinkronisasiTerakhir.Null {
		if !validateDate(patch.SinkronisasiTerakhir.Value) {
			return ErrValidation
		}
	}
	if patch.Status.Present && !patch.Status.Null {
		if !StatusSinkronisasiValid(models.StatusSinkronisasi(patch.Status.Value)) {
			return ErrValidation
		}
	}
	return nil
}

// HasChanges melaporkan apakah patch memuat setidaknya satu perubahan.
func (patch SisterSyncMapPatch) HasChanges() bool {
	return patch.SistemEksternal.Present || patch.IDEksternal.Present ||
		patch.SinkronisasiTerakhir.Present || patch.Status.Present
}
