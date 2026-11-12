// Package cloudinary menyediakan utilitas tanda tangan unggah berkas ke
// Cloudinary.
//
// Backend tidak menangani unggah berkas itu sendiri. Alur yang dijalankan:
//  1. Klien meminta tanda tangan unggah ke backend (timestamp + parameter
//     lainnya).
//  2. Backend menghasilkan tanda tangan dengan CLOUDINARY_API_SECRET.
//  3. Klien mengunggah berkas langsung ke Cloudinary menggunakan tanda
//     tangan, apiKey, cloudName, dan timestamp yang diterima.
//  4. Backend hanya menyimpan URL dan metadata berkas (ukuran, tipe MIME,
//     public ID) pada tabel arsipDokumenPegawai.
//
// Tanda tangan Cloudinary dihitung dengan:
//  1. Parameter yang akan ditandatangani diurutkan menurut kunci secara
//     alfabetik.
//  2. Disatukan sebagai "key1=value1&key2=value2...".
//  3. API secret disisipkan di akhir string.
//  4. Dihasilkan hash SHA-1.
package cloudinary

import (
	"crypto/sha1"
	"fmt"
	"sort"
	"strconv"
	"time"
)

// Config adalah konfigurasi Cloudinary yang dibaca dari environment.
type Config struct {
	CloudName string
	APIKey    string
	APISecret string
}

// Enabled melaporkan apakah Cloudinary sudah dikonfigurasi lengkap.
func (c Config) Enabled() bool {
	return c.CloudName != "" && c.APIKey != "" && c.APISecret != ""
}

// UploadSignature memuat tanda tangan dan parameter yang dibutuhkan klien
// untuk mengunggah berkas langsung ke Cloudinary.
type UploadSignature struct {
	CloudName    string `json:"cloudName"`
	APIKey       string `json:"apiKey"`
	Timestamp    int64  `json:"timestamp"`
	Signature    string `json:"signature"`
	Folder       string `json:"folder,omitempty"`
	ResourceType string `json:"resourceType,omitempty"`
}

// Service menghasilkan tanda tangan unggah Cloudinary.
type Service struct {
	cfg Config
}

// New membuat Service baru dari konfigurasi.
func New(cfg Config) *Service {
	return &Service{cfg: cfg}
}

// Enabled melaporkan apakah Cloudinary sudah dikonfigurasi lengkap.
func (s *Service) Enabled() bool {
	return s.cfg.Enabled()
}

// SignUpload menghasilkan tanda tangan unggah Cloudinary untuk parameter yang
// diberikan. Parameter tambahan (mis. folder, public_id) dapat disertakan dan
// akan ikut ditandatangani.
func (s *Service) SignUpload(extra map[string]string) (UploadSignature, error) {
	if !s.cfg.Enabled() {
		return UploadSignature{}, fmt.Errorf("cloudinary belum dikonfigurasi: CLOUDINARY_CLOUD_NAME, CLOUDINARY_API_KEY, dan CLOUDINARY_API_SECRET wajib diisi")
	}
	timestamp := time.Now().Unix()

	params := make(map[string]string, len(extra)+1)
	for k, v := range extra {
		params[k] = v
	}
	params["timestamp"] = strconv.FormatInt(timestamp, 10)

	// Urutkan kunci secara alfabetik.
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Gabungkan sebagai "key=value&..." lalu sisipkan API secret di akhir.
	toSign := ""
	for i, k := range keys {
		if i > 0 {
			toSign += "&"
		}
		toSign += k + "=" + params[k]
	}
	toSign += s.cfg.APISecret

	sig := sha1Hash(toSign)

	return UploadSignature{
		CloudName:    s.cfg.CloudName,
		APIKey:       s.cfg.APIKey,
		Timestamp:    timestamp,
		Signature:    sig,
		Folder:       extra["folder"],
		ResourceType: extra["resource_type"],
	}, nil
}

// sha1Hash menghitung hash SHA-1 dari string input dan mengembalikannya dalam
// bentuk heksadesimal.
func sha1Hash(s string) string {
	h := sha1.New()
	h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum(nil))
}
