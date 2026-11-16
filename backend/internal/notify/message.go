// Package notify mengirim pemberitahuan ke pengguna.
//
// Isi pemberitahuan gaji dibatasi oleh konstruktor: notifikasi gaji hanya
// menerima nama, periode, dan tautan, sehingga nominal gaji tidak pernah
// dapat ikut terbawa pada Email. Nominal hanya tersedia di halaman
// terautentikasi.
package notify

import (
	"errors"
	"fmt"
	"strings"
)

// ErrNoRecipients menandakan pesan tidak memiliki penerima.
var ErrNoRecipients = errors.New("notification has no recipients")

// ErrMissingIdempotencyKey menandakan pesan tanpa kunci idempotensi.
var ErrMissingIdempotencyKey = errors.New("notification has no idempotency key")

// Message adalah satu pemberitahuan yang siap dikirim.
type Message struct {
	To      string
	Subject string
	Body    string
	// IdempotencyKey mencegah pengiriman ganda untuk satu peristiwa logis.
	IdempotencyKey string
	// HTMLBody berisi versi HTML dari Body; bila diisi bersama InlineImages,
	// pengirim menyusun multipart/related (alternative + gambar inline).
	HTMLBody string
	// InlineImages memuat gambar inline (CID) yang dirujuk dari HTMLBody.
	InlineImages []InlineImage
}

// InlineImage adalah satu gambar yang disisipkan inline (CID) pada email HTML.
type InlineImage struct {
	// Filename tampil sebagai nama berkas pada bagian gambar.
	Filename string
	// ContentID adalah id CID tanpa kurung siku sudut (<...>).
	ContentID string
	// ContentType misalnya "image/png".
	ContentType string
	// Content adalah isi berkas gambar.
	Content []byte
}

// Validate memeriksa pesan sebelum dikirim.
func (m Message) Validate() error {
	if strings.TrimSpace(m.To) == "" {
		return ErrNoRecipients
	}
	if strings.TrimSpace(m.IdempotencyKey) == "" {
		return ErrMissingIdempotencyKey
	}
	return nil
}

// SalaryPaidNotice membangun pemberitahuan gaji yang disetujui.
//
// nominally sengaja tidak tersedia sebagai parameter: nominal gaji hanya boleh
// ditampilkan pada halaman terautentikasi, bukan pada Email.
func SalaryPaidNotice(recipient, recipientName, period, link string) Message {
	name := strings.TrimSpace(recipientName)
	if name == "" {
		name = "Pegawai"
	}
	body := fmt.Sprintf(
		"Halo %s,\n\nPenggajian untuk periode %s sudah diproses. Rincian gaji dapat dilihat pada halaman penggajian Anda.\n\n%s\n\n"+
			"Jika Anda tidak mengenali pesan ini, abaikan atau hubungi bagian kepegawaian.",
		name, period, link,
	)
	return Message{
		To:             recipient,
		Subject:        "Penggajian Anda sudah diproses",
		Body:           body,
		IdempotencyKey: "salary-paid:" + period + ":" + recipient,
	}
}

// GenericNotice membangun pemberitahuan umum tanpa data keuangan.
func GenericNotice(recipient, subject, body, idempotencyKey string) Message {
	return Message{
		To:             strings.TrimSpace(recipient),
		Subject:        subject,
		Body:           body,
		IdempotencyKey: idempotencyKey,
	}
}
