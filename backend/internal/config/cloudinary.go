// Package config memuat konfigurasi runtime backend SIMPEG.
//
// Berkas ini berisi konfigurasi Cloudinary untuk penyimpanan berkas dokumen
// pegawai. Berkas diunggah langsung ke Cloudinary oleh klien frontend;
// backend hanya menyimpan URL dan metadata berkas pada tabel
// arsipDokumenPegawai.
package config

// CloudinaryConfig adalah konfigurasi penyimpanan berkas di Cloudinary.
type CloudinaryConfig struct {
	CloudName string
	APIKey    string
	APISecret string
}

// CloudinaryEnabled melapor apakah Cloudinary sudah dikonfigurasi.
func (c CloudinaryConfig) Enabled() bool {
	return c.CloudName != "" && c.APIKey != "" && c.APISecret != ""
}
