# Desain — Slip Gaji Digital SIMPEG

- **Tanggal:** 2026-11-14
- **Status:** Draft desain (mock), `DEC-071`. Bukan keputusan final branding; perlu persetujuan pemilik produk sebelum rilis.
- **Berkas:** `slip-gaji.html` (mock interaktif, dibuka di browser / dicetak A5), `slip-gaji-mock.png` (pratinjau PNG), `slip-qr.png` (contoh QR verifikasi).
- **Ukuran:** ½ kertas A4 = **A5 (148 × 210 mm), potret**, margin 0 (tepi penuh). Cocok untuk layar (unduhan PDF) dan cetak.
- **Palet:** khas SIMPEG — `sea-ink` (#173a40) header, `lagoon`/`palm` (#2f6a4a) aksen Penerimaan, `gold` (#c9a227) garis aksen masthead, `foam`/`sand` kertas. Font Manrope (teks) & Fraunces (judul/nominal) — keduanya sudah dipakai aplikasi.
- **Isi:** masthead yayasan→universitas→fakultas/prodi · identitas pegawai (nama, NIP, kategori, jabatan, unit, golongan/IV, masa kerja, rekening masked) · dua kolom rincian (A Penerimaan / B Potongan) · **Penerimaan bersih** menonjol (`Rp 12.605.000` pada contoh) + tanggal bayar + status `TERBAYAR` · strip verifikasi: QR + token `SLIP-2026-10-000123` + teks "nominal tidak pernah ditampilkan di halaman verifikasi" · fineprint (otomatis, tanpa ttd, ref. pembayaran).
- **Prinsip:** dokumen resmi yang indah — bukan formulir generik; tidak menampilkan PII berlebihan (rekening masked); nominal hanya di slip terautentikasi; verifikasi publik menampilkan minimal (nama, periode, status).
- **Catatan teknis:** nilai pada mock adalah data sintetis (`simpeg-perf`). Produksi: nama file `SLIP-<periode>-<urut>.pdf`, dihasilkan dari template yang sama (HTML→PDF di worker), token verifikasi disimpan per slip (`kodeVerifikasi`/`tokenVerifikasi` — lihat catatan skema `000025_payroll_core.sql`, kolom `tokenVerifikasi` belum ada di usulan — perlu penambahan saat implementasi).
