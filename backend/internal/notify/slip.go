package notify

import (
	"encoding/json"
	"fmt"
	"strings"
)

// SlipAvailablePayload menyusun payload pemberitahuan ketersediaan slip.
//
// Payload ini tidak memuat nominal. Nominal gaji hanya tersedia pada halaman
// penggajian yang terautentikasi.
func SlipAvailablePayload(recipient, recipientName, period, userID string) ([]byte, error) {
	return json.Marshal(emailPayload{
		Kind:      KindSlipAvailable,
		Recipient: recipient,
		Name:      recipientName,
		Period:    period,
		UserID:    userID,
	})
}

// SlipAvailableNotice membangun pemberitahuan ketersediaan slip gaji.
//
// Isi pesan sengaja tidak memuat nominal dan tidak menyatakan bahwa gaji
// sudah dibayar. Pesan hanya mengarahkan pegawai ke halaman penggajian.
func SlipAvailableNotice(recipient, recipientName, period, link string) Message {
	name := strings.TrimSpace(recipientName)
	if name == "" {
		name = "Pegawai"
	}
	body := fmt.Sprintf(
		"Halo %s,\n\nSlip gaji Anda untuk periode %s sudah tersedia. Rincian gaji dapat dilihat pada halaman penggajian Anda.\n\n%s\n\n"+
			"Jika Anda tidak mengenali pesan ini, abaikan atau hubungi bagian kepegawaian.",
		name, period, link,
	)
	return Message{
		To:             recipient,
		Subject:        "Slip gaji periode " + period + " sudah tersedia",
		Body:           body,
		IdempotencyKey: "slip-available:" + period + ":" + recipient,
	}
}
