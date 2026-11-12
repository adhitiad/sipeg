// Package domain memuat aturan bisnis untuk arsip dokumen pegawai.
package domain

import (
	"strings"

	"simpeg/backend/internal/models"
)

// JenisDokumenArsipValid melaporkan apakah jenis dokumen arsip dikenal.
func JenisDokumenArsipValid(value models.JenisDokumenArsip) bool {
	switch value {
	case models.DokumenKTP, models.DokumenNPWP, models.DokumenKartuKeluarga,
		models.DokumenRekeningBank, models.DokumenSKPengangkatanYayasan,
		models.DokumenSKMendikbud, models.DokumenPaktaIntegritas,
		models.DokumenLainnya, models.DokumenFoto, models.DokumenIjazahTerakhir,
		models.DokumenSKKepegawaianKontrak, models.DokumenBPJSKetenagakerjaan,
		models.DokumenBPJSkesehatan, models.DokumenSertifikatSerdos:
		return true
	}
	return false
}

// JenisDokumenWajib melaporkan apakah jenis dokumen wajib dimiliki oleh
// setiap pegawai/dosen.
func JenisDokumenWajib(value models.JenisDokumenArsip) bool {
	switch value {
	case models.DokumenFoto, models.DokumenKTP, models.DokumenKartuKeluarga,
		models.DokumenIjazahTerakhir, models.DokumenSKKepegawaianKontrak:
		return true
	}
	return false
}

// DokumenArsipRule memuat aturan validasi berkas untuk satu jenis dokumen.
type DokumenArsipRule struct {
	FormatDiizinkan []string // tipe MIME yang diizinkan (image/png, application/pdf, dst.)
	UkuranMaks      int64    // ukuran maksimal berkas dalam byte
}

// dokumenArsipRules memuat aturan validasi berkas per jenis dokumen.
//
// Berkas disimpan di Cloudinary; backend hanya menyimpan URL dan metadata,
// sehingga validasi dilakukan pada metadata (mimeType dan fileSize) yang
// dikirim bersama URL, bukan pada isi berkas itu sendiri.
var dokumenArsipRules = map[models.JenisDokumenArsip]DokumenArsipRule{
	// Wajib untuk pegawai/dosen.
	models.DokumenFoto: {
		FormatDiizinkan: []string{"image/png", "image/jpeg", "image/webp"},
		UkuranMaks:      3250585, // 3.1 MB
	},
	models.DokumenKTP: {
		FormatDiizinkan: []string{"image/png", "image/jpeg", "image/webp", "application/pdf"},
		UkuranMaks:      1153433, // 1.1 MB
	},
	models.DokumenKartuKeluarga: {
		FormatDiizinkan: []string{"image/png", "image/jpeg", "image/webp", "application/pdf"},
		UkuranMaks:      1153433, // 1.1 MB
	},
	models.DokumenIjazahTerakhir: {
		FormatDiizinkan: []string{"application/pdf"},
		UkuranMaks:      5347737, // 5.1 MB
	},
	models.DokumenSKKepegawaianKontrak: {
		FormatDiizinkan: []string{"application/pdf"},
		UkuranMaks:      7444591, // 7.1 MB
	},
	// Opsional.
	models.DokumenNPWP: {
		FormatDiizinkan: []string{"image/png", "image/jpeg", "image/webp", "application/pdf"},
		UkuranMaks:      1153433, // 1.1 MB
	},
	models.DokumenBPJSKetenagakerjaan: {
		FormatDiizinkan: []string{"image/png", "image/jpeg", "image/webp", "application/pdf"},
		UkuranMaks:      1153433, // 1.1 MB
	},
	models.DokumenBPJSkesehatan: {
		FormatDiizinkan: []string{"image/png", "image/jpeg", "image/webp", "application/pdf"},
		UkuranMaks:      1153433, // 1.1 MB
	},
	models.DokumenSertifikatSerdos: {
		FormatDiizinkan: []string{"application/pdf"},
		UkuranMaks:      5347737, // 5.1 MB
	},
	// Jenis lain (tidak memiliki batasan ketat dari kebijakan di atas).
	models.DokumenRekeningBank: {
		FormatDiizinkan: []string{"image/png", "image/jpeg", "image/webp", "application/pdf"},
		UkuranMaks:      1153433, // 1.1 MB
	},
	models.DokumenSKPengangkatanYayasan: {
		FormatDiizinkan: []string{"image/png", "image/jpeg", "image/webp", "application/pdf"},
		UkuranMaks:      5347737, // 5.1 MB
	},
	models.DokumenSKMendikbud: {
		FormatDiizinkan: []string{"image/png", "image/jpeg", "image/webp", "application/pdf"},
		UkuranMaks:      5347737, // 5.1 MB
	},
	models.DokumenPaktaIntegritas: {
		FormatDiizinkan: []string{"image/png", "image/jpeg", "image/webp", "application/pdf"},
		UkuranMaks:      1153433, // 1.1 MB
	},
	models.DokumenLainnya: {
		FormatDiizinkan: []string{"image/png", "image/jpeg", "image/webp", "application/pdf"},
		UkuranMaks:      10485760, // 10 MB
	},
}

// formatDiizinkan melaporkan apakah tipe MIME diizinkan untuk jenis dokumen.
func formatDiizinkan(jenis models.JenisDokumenArsip, mimeType string) bool {
	rule, ok := dokumenArsipRules[jenis]
	if !ok {
		return false
	}
	for _, f := range rule.FormatDiizinkan {
		if f == mimeType {
			return true
		}
	}
	return false
}

// ukuranDiizinkan melaporkan apakah ukuran berkas tidak melebihi batas untuk
// jenis dokumen.
func ukuranDiizinkan(jenis models.JenisDokumenArsip, fileSize int64) bool {
	rule, ok := dokumenArsipRules[jenis]
	if !ok {
		return false
	}
	return fileSize > 0 && fileSize <= rule.UkuranMaks
}

// ValidasiMetadataDokumen memeriksa kecukupan metadata berkas (URL, tipe MIME,
// dan ukuran) untuk jenis dokumen tertentu.
//
// Backend tidak menangani unggah berkas; berkas diunggah langsung ke
// Cloudinary oleh klien, dan backend hanya menyimpan URL beserta metadata.
func ValidasiMetadataDokumen(jenis models.JenisDokumenArsip, fileURL string, fileSize int64, mimeType string) error {
	if !JenisDokumenArsipValid(jenis) {
		return ErrValidation
	}
	if strings.TrimSpace(fileURL) == "" {
		return ErrValidation
	}
	if !formatDiizinkan(jenis, mimeType) {
		return ErrValidation
	}
	if !ukuranDiizinkan(jenis, fileSize) {
		return ErrValidation
	}
	return nil
}

// CreateArsipDokumenPegawai memuat data untuk membuat entri arsip dokumen
// pegawai.
type CreateArsipDokumenPegawai struct {
	UserID             string
	JenisDokumen       models.JenisDokumenArsip
	Judul              string
	FileURL            string
	FileSize           int64
	MimeType           string
	Keterangan         *string
	BerlakuSampai      *string // "YYYY-MM-DD"
	CloudinaryPublicId *string
}

// Validate memeriksa aturan data arsip dokumen pegawai.
func (input CreateArsipDokumenPegawai) Validate() error {
	if !JenisDokumenArsipValid(input.JenisDokumen) {
		return ErrValidation
	}
	if strings.TrimSpace(input.Judul) == "" || TextLength(input.Judul) > MaxNameLength {
		return ErrValidation
	}
	if err := ValidasiMetadataDokumen(input.JenisDokumen, input.FileURL, input.FileSize, input.MimeType); err != nil {
		return err
	}
	if input.Keterangan != nil && TextLength(*input.Keterangan) > MaxDescription {
		return ErrValidation
	}
	if input.BerlakuSampai != nil && !validateDate(*input.BerlakuSampai) {
		return ErrValidation
	}
	return nil
}

// ArsipPatchFields adalah daftar field yang boleh diubah.
var ArsipPatchFields = map[string]bool{
	"jenisDokumen": true, "judul": true, "fileUrl": true, "fileSize": true,
	"mimeType": true, "keterangan": true, "berlakuSampai": true,
	"cloudinaryPublicId": true,
}

// ArsipDokumenPatch memuat perubahan entri arsip dokumen.
type ArsipDokumenPatch struct {
	JenisDokumen       Field
	Judul              Field
	FileURL            Field
	FileSize           Field
	MimeType           Field
	Keterangan         Field
	BerlakuSampai      Field
	CloudinaryPublicId Field
}

// NewArsipDokumenPatch menyusun patch dari peta field.
func NewArsipDokumenPatch(fields map[string]Field) ArsipDokumenPatch {
	return ArsipDokumenPatch{
		JenisDokumen:       fields["jenisDokumen"],
		Judul:              fields["judul"],
		FileURL:            fields["fileUrl"],
		FileSize:           fields["fileSize"],
		MimeType:           fields["mimeType"],
		Keterangan:         fields["keterangan"],
		BerlakuSampai:      fields["berlakuSampai"],
		CloudinaryPublicId: fields["cloudinaryPublicId"],
	}
}

// Validate memeriksa aturan perubahan arsip dokumen.
func (patch ArsipDokumenPatch) Validate() error {
	if patch.JenisDokumen.Present && !patch.JenisDokumen.Null {
		if !JenisDokumenArsipValid(models.JenisDokumenArsip(patch.JenisDokumen.Value)) {
			return ErrValidation
		}
	}
	if patch.Judul.Present {
		if patch.Judul.Null || TextLength(strings.TrimSpace(patch.Judul.Value)) == 0 ||
			TextLength(patch.Judul.Value) > MaxNameLength {
			return ErrValidation
		}
	}
	if patch.FileURL.Present {
		if patch.FileURL.Null || strings.TrimSpace(patch.FileURL.Value) == "" {
			return ErrValidation
		}
	}
	if patch.MimeType.Present && !patch.MimeType.Null {
		if !formatDiizinkan(models.JenisDokumenArsip(patch.JenisDokumen.Value), patch.MimeType.Value) {
			return ErrValidation
		}
	}
	if patch.Keterangan.Present && !patch.Keterangan.Null {
		if TextLength(patch.Keterangan.Value) > MaxDescription {
			return ErrValidation
		}
	}
	if patch.BerlakuSampai.Present && !patch.BerlakuSampai.Null {
		if !validateDate(patch.BerlakuSampai.Value) {
			return ErrValidation
		}
	}
	return nil
}

// HasChanges melaporkan apakah patch memuat setidaknya satu perubahan.
func (patch ArsipDokumenPatch) HasChanges() bool {
	return patch.JenisDokumen.Present || patch.Judul.Present || patch.FileURL.Present ||
		patch.FileSize.Present || patch.MimeType.Present || patch.Keterangan.Present ||
		patch.BerlakuSampai.Present || patch.CloudinaryPublicId.Present
}
