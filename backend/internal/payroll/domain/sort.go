package domain

// Allowlist kolom sortable slip gaji. Kolom SQL hanya boleh berasal dari
// allowlist ini.
var SlipSortColumns = map[string]string{
	"periode":     `"periode"`,
	"tanggalGaji": `"tanggalGaji"`,
	"createdAt":   `"createdAt"`,
}

// SlipSortFields mengembalikan kunci sort yang valid untuk daftar slip gaji.
func SlipSortFields() []string {
	return []string{"createdAt", "periode", "tanggalGaji"}
}

// Notification adalah pesan yang harus diantrekan ketika pembayaran tercatat.
//
// Pesan disusun oleh lapisan aplikasi dan ditulis ke antrean keluar di dalam
// transaksi yang sama dengan pencatatan pembayaran, sehingga notifikasi tidak
// pernah ada tanpa bukti pembayaran.
type Notification struct {
	// Kind adalah jenis pesan outbox.
	Kind string
	// DedupeKey mencegah satu peristiwa pembayaran menghasilkan dua pesan.
	DedupeKey string
	// Payload adalah isi pesan dalam bentuk JSON.
	Payload string
}
