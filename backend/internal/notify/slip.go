// Package notify memuat pembangun pemberitahuan email slip gaji.
//
// SlipAvailableNotice membangun pemberitahuan H-1 sebelum tanggal gajian:
// nama, periode, tautan halaman verifikasi publik slip, dan QR menuju
// halaman yang sama. Isi pesan tidak memuat nominal dan tidak menyatakan
// bahwa gaji sudah dibayar.
package notify

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/skip2/go-qrcode"
)

// SlipAvailablePayload menyusun payload pemberitahuan ketersediaan slip.
//
// Payload ini tidak memuat nominal. Nominal gaji hanya tersedia pada halaman
// penggajian yang terautentikasi. Token verifikasi dibawa agar pengirim dapat
// menyusun tautan halaman verifikasi publik slip.
func SlipAvailablePayload(recipient, recipientName, period, userID, token string) ([]byte, error) {
	return json.Marshal(emailPayload{
		Kind:      KindSlipAvailable,
		Recipient: recipient,
		Name:      recipientName,
		Period:    period,
		UserID:    userID,
		Token:     token,
	})
}

// SlipAvailableNotice membangun pemberitahuan ketersediaan slip gaji H-1.
//
// Pesan memuat nama, periode, dan tautan verifikasi publik slip beserta QR
// yang menunjuk ke URL yang sama. Tidak ada nominal dan tidak ada klaim
// bahwa gaji sudah dibayar.
func SlipAvailableNotice(recipient, recipientName, period, link string) (Message, error) {
	name := strings.TrimSpace(recipientName)
	if name == "" {
		name = "Pegawai"
	}
	link = strings.TrimSpace(link)
	if link == "" {
		link = "#"
	}

	plain := fmt.Sprintf(
		"Halo %s,\n\nSlip gaji Anda untuk periode %s sudah tersedia. Verifikasi slip melalui tautan berikut:\n%s\n\n"+
			"Jika Anda tidak mengenali pesan ini, abaikan atau hubungi bagian kepegawaian.",
		name, period, link,
	)

	html := fmt.Sprintf(
		`<!DOCTYPE html>
<html lang="id">
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"></head>
<body style="margin:0;padding:24px;background:#f4f6f2;font-family:Georgia,'Times New Roman',serif;color:#173a40">
  <table role="presentation" cellpadding="0" cellspacing="0" style="max-width:480px;margin:0 auto;background:#ffffff;border-collapse:collapse">
    <tr>
      <td style="background:#173a40;padding:18px 24px">
        <div style="letter-spacing:2px;text-transform:uppercase;font-size:11px;color:#c9a227;font-family:Georgia,serif">SIMPEG</div>
        <div style="color:#ffffff;font-size:22px;margin-top:6px;font-family:Georgia,serif">Slip Gaji %s</div>
      </td>
    </tr>
    <tr>
      <td style="padding:22px 24px">
        <p style="margin:0 0 14px 0;font-size:15px;line-height:1.5">Halo %s,</p>
        <p style="margin:0 0 14px 0;font-size:15px;line-height:1.5">Slip gaji Anda untuk periode <strong>%s</strong> sudah tersedia. Silakan verifikasi slip melalui tautan berikut atau pindai QR di bawah:</p>
        <p style="margin:0 0 16px 0"><a href="%s" style="color:#2f6a4a;font-weight:bold;font-size:14px;text-decoration:underline">%s</a></p>
        <table role="presentation" cellpadding="0" cellspacing="0" style="margin:0 auto"><tr><td style="padding:8px;background:#ffffff;border:1px solid #d5d9d1"><img src="cid:slip-qr" alt="QR verifikasi slip" width="160" height="160" style="display:block"></td></tr></table>
        <p style="margin:16px 0 0 0;font-size:12px;line-height:1.5;color:#416166">Verifikasi hanya menampilkan nama, periode, dan status pencairan — bukan nominal. QR ini khusus untuk pemilik slip.</p>
      </td>
    </tr>
    <tr>
      <td style="padding:12px 24px;background:#e8ece3;font-size:11px;color:#416166;line-height:1.5">SIMPEG — Sistem Informasi Kepegawaian Universitas Subang<br>Jika Anda tidak mengenali pesan ini, abaikan atau hubungi bagian kepegawaian.</td>
    </tr>
  </table>
</body>
</html>`, period, name, period, link, link,
	)

	qrPNG, err := qrcode.Encode(link, qrcode.Medium, 256)
	if err != nil {
		return Message{}, fmt.Errorf("generate qr: %w", err)
	}

	return Message{
		To:             recipient,
		Subject:        "Slip gaji periode " + period + " sudah tersedia",
		Body:           plain,
		HTMLBody:       html,
		InlineImages:   []InlineImage{{Filename: "slip-qr.png", ContentID: "slip-qr", ContentType: "image/png", Content: qrPNG}},
		IdempotencyKey: "slip-available:" + period + ":" + recipient,
	}, nil
}

// buildMultipartRelated membangun email mentah multipart/related berisi
// multipart/alternative (text/plain + text/html) dan gambar inline.
func buildMultipartRelated(from string, message Message) ([]byte, error) {
	boundaryAlt := "alt-" + boundaryToken(message.IdempotencyKey+"-alt")
	boundaryRel := "rel-" + boundaryToken(message.IdempotencyKey+"-rel")

	var builder strings.Builder
	writeHeader := func(name, value string) {
		builder.WriteString(name)
		builder.WriteString(": ")
		builder.WriteString(value)
		builder.WriteString("\r\n")
	}

	writeHeader("From", from)
	writeHeader("To", message.To)
	writeHeader("Subject", message.Subject)
	writeHeader("MIME-Version", "1.0")
	writeHeader("Content-Type", "multipart/related; boundary=\""+boundaryRel+"\"")
	builder.WriteString("\r\n")

	builder.WriteString("--" + boundaryRel + "\r\n")
	builder.WriteString("Content-Type: multipart/alternative; boundary=\"" + boundaryAlt + "\"\r\n\r\n")

	builder.WriteString("--" + boundaryAlt + "\r\n")
	builder.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	builder.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
	builder.WriteString(base64EncodeLines(message.Body))
	builder.WriteString("\r\n")

	builder.WriteString("--" + boundaryAlt + "\r\n")
	builder.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	builder.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
	builder.WriteString(base64EncodeLines(message.HTMLBody))
	builder.WriteString("\r\n")
	builder.WriteString("--" + boundaryAlt + "--\r\n")

	for _, image := range message.InlineImages {
		builder.WriteString("--" + boundaryRel + "\r\n")
		builder.WriteString("Content-Type: " + image.ContentType + "; name=\"" + image.Filename + "\"\r\n")
		builder.WriteString("Content-Transfer-Encoding: base64\r\n")
		builder.WriteString("Content-ID: <" + image.ContentID + ">\r\n")
		builder.WriteString("Content-Disposition: inline; filename=\"" + image.Filename + "\"\r\n\r\n")
		builder.WriteString(base64EncodeBytes(image.Content))
		builder.WriteString("\r\n")
	}
	builder.WriteString("--" + boundaryRel + "--\r\n")

	return []byte(builder.String()), nil
}

// boundaryToken menyaring karakter pembatas MIME menjadi hanya alfanumerik dan tanda hubung.
func boundaryToken(value string) string {
	var builder strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			builder.WriteRune(r)
		default:
			builder.WriteRune('-')
		}
	}
	return builder.String()
}

func base64EncodeLines(text string) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(text))
	return wrap76(encoded)
}

func base64EncodeBytes(content []byte) string {
	encoded := base64.StdEncoding.EncodeToString(content)
	return wrap76(encoded)
}

func wrap76(encoded string) string {
	var builder strings.Builder
	for len(encoded) > 76 {
		builder.WriteString(encoded[:76])
		builder.WriteString("\r\n")
		encoded = encoded[76:]
	}
	builder.WriteString(encoded)
	builder.WriteString("\r\n")
	return builder.String()
}
