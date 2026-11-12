package domain

import (
	"simpeg/backend/internal/listing"
)

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

var TunjanganSortColumns = map[string]string{
	"jenisTunjangan": `"jenisTunjangan"`,
	"nilai":          `"nilai"`,
	"createdAt":      `"createdAt"`,
	"updatedAt":      `"updatedAt"`,
}

var PotonganSortColumns = map[string]string{
	"jenisPotongan": `"jenisPotongan"`,
	"nominal":       `"nominal"`,
	"createdAt":     `"createdAt"`,
	"updatedAt":     `"updatedAt"`,
}

// TunjanganSortFields mengembalikan kunci sort yang valid untuk daftar komponen tunjangan.
func TunjanganSortFields() []string {
	return listing.SortKeys(TunjanganSortColumns)
}

// PotonganSortFields mengembalikan kunci sort yang valid untuk daftar komponen potongan.
func PotonganSortFields() []string {
	return listing.SortKeys(PotonganSortColumns)
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
