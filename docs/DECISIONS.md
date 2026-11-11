# Log Keputusan SIMPEG

Catatan keputusan produk dan arsitektur beserta pemilik, tanggal, dan statusnya.
Decision log ini adalah daftar ringkas; **rincian lengkap ada di `docs/PRD.md` §11**
dan bukti implementasinya ada di `backend/` serta `agents/rules/`.

Status yang dipakai:

| Status | Arti |
|---|---|
| **Disetujui** | Sudah diputuskan dan tercatat di PRD §11. Perubahan memerlukan persetujuan ulang. |
| **Usulan** | Rekomendasi teknis yang sudah diimplementasikan atau siap diimplementasikan, tetapi belum disetujui pemilik. |
| **Menunggu** | Butuh keputusan pemilik sebelum dikerjakan. |
| **Batal** | Tidak lagi berlaku; alasan dicatat pada baris yang sama. |

Pemilik produk dan kelompok pilot **belum ditetapkan** — lihat `DEC-016`. Sampai
nama itu ditetapkan, setiap keputusan pada log ini tercatat atas nama "Pemilik
produk (belum ditetapkan)".

## Format entri

```
### DEC-NNN — Judul singkat

- **Tanggal:** YYYY-MM-DD
- **Pemilik:** namaInitiatorAtasan
- **Status:** Disetujui | Usulan | Menunggu | Batal
- **Rincian:** tautan ke PRD §11 atau bagian terkait
- **Bukti:** file/tes yang membuktikan keputusan diterapkan
```

---

## Keputusan produk

### DEC-001 — Target institusi Universitas Subang

- **Tanggal:** 2026-11-10
- **Pemilik:** Pemilik produk (belum ditetapkan)
- **Status:** Disetujui
- **Rincian:** `docs/PRD.md` §11 "Keputusan environment deployment"
- **Bukti:** `docs/PRD.md` §11

### DEC-002 — MVP dibatasi tahap 1

- **Tanggal:** 2026-11-10
- **Pemilik:** Pemilik produk (belum ditetapkan)
- **Status:** Disetujui
- **Rincian:** `docs/PRD.md` §11 "Usulan tahapan ruang lingkup"
- **Bukti:** `docs/PRD.md` §11

### DEC-003 — Matriks peran dan cakupan data

- **Tanggal:** 2026-11-10
- **Pemilik:** Pemilik produk dan pemilik keamanan (belum ditetapkan)
- **Status:** Disetujui
- **Rincian:** `docs/PRD.md` §11 "Matriks peran dan cakupan"
- **Bukti:** `backend/cmd/server/main.go` (matriks `auth.StaticPolicy`), `backend/internal/httpserver/scope_handlers_test.go`, `organization_scope_test.go`

### DEC-004 — Cakupan operator adalah unit penugasan beserta turunannya

- **Tanggal:** 2026-11-11
- **Pemilik:** Pemilik produk (belum ditetapkan)
- **Status:** Disetujui
- **Rincian:** `docs/PRD.md` §11 "Matriks peran dan cakupan"
- **Bukti:** `backend/internal/directory/repository/gorm_catalog.go` (recursive CTE pohon `Homebase`), `backend/internal/directory/repository/list_test.go`, `backend/internal/httpserver/scope_handlers_test.go`

### DEC-005 — Provider autentikasi Better Auth

- **Tanggal:** 2026-11-11
- **Pemilik:** Pemilik identitas/IT (belum ditetapkan)
- **Status:** Disetujui
- **Rincian:** `docs/PRD.md` §11 "Keputusan autentikasi dan kontrak sesi"
- **Bukti:** `backend/internal/auth/jwt.go`, `backend/internal/auth/gorm_resolver.go`, tabel `session`/`account`/`verification`/`twoFactor` pada skema acuan

### DEC-006 — Kontrak sesi: cookie server-side dengan proteksi CSRF

- **Tanggal:** 2026-11-11
- **Pemilik:** Pemilik identitas/IT (belum ditetapkan)
- **Status:** Disetujui
- **Rincian:** `docs/PRD.md` §11 "Keputusan autentikasi dan kontrak sesi"
- **Bukti:** `backend/internal/auth/jwt.go` (verifikasi JWT), `backend/internal/auth/revoker.go` (pencabutan sesi), `backend/internal/httpserver/auth_handlers.go` (`POST /api/v1/auth/logout`)

### DEC-007 — Migration runner Goose

- **Tanggal:** 2026-11-11
- **Pemilik:** Pemilik platform/DB (belum ditetapkan)
- **Status:** Disetujui
- **Rincian:** `docs/PRD.md` §11 "Draf migrasi, backup, audit, dan ketersediaan"
- **Bukti:** `backend/.goose.yaml`, `backend/migrations/`, `backend/internal/database/migrations_test.go`

### DEC-008 — Versi API memakai prefix path `/api/v1`

- **Tanggal:** 2026-11-11
- **Pemilik:** Pemilik arsitektur (belum ditetapkan)
- **Status:** Disetujui
- **Rincian:** `docs/PRD.md` §11 "Keputusan versi API"
- **Bukti:** `backend/internal/httpserver/version.go`, `docs/openapi.yaml`, `backend/internal/httpserver/version_test.go`

### DEC-009 — Bentuk respons error seragam

- **Tanggal:** 2026-11-11
- **Pemilik:** Pemilik arsitektur (belum ditetapkan)
- **Status:** Disetujui
- **Rincian:** `docs/PRD.md` §11 dan `agents/rules/architecture.md`
- **Bukti:** `backend/internal/apierror/`, `backend/internal/httpserver/errors.go`, `backend/internal/httpserver/health_test.go`

### DEC-010 — Jejak audit append-only dengan akses dibatasi per unit

- **Tanggal:** 2026-11-11
- **Pemilik:** Pemilik kebijakan (belum ditetapkan)
- **Status:** Disetujui
- **Rincian:** `docs/PRD.md` §11 "Kebijakan jejak audit"
- **Bukti:** `backend/internal/audit/` (termasuk tes penjaga append-only), `backend/migrations/000003_audit_log_unit.sql`, `backend/internal/httpserver/audit_handlers.go`

### DEC-011 — Environment deployment pertama Supabase production

- **Tanggal:** 2026-11-11
- **Pemilik:** Pemilik produk (belum ditetapkan)
- **Status:** Disetujui
- **Rincian:** `docs/PRD.md` §11 "Keputusan environment deployment"
- **Bukti:** `docs/RUNBOOK.md` §7 (prasyarat sebelum deploy), `backend/migrations/` (belum pernah diterapkan ke database nyata)

### DEC-012 — Stack frontend: Bun, shadcn/ui, TanStack Start/Query/Table/Store

- **Tanggal:** 2026-11-11
- **Pemilik:** Pemilik arsitektur (belum ditetapkan)
- **Status:** Disetujui
- **Rincian:** `docs/PRD.md` §11 "Keputusan stack frontend"
- **Bukti:** `agents/rules/frontend.md`

### DEC-013 — Antrean keluar memakai outbox persisten di PostgreSQL

- **Tanggal:** 2026-11-11
- **Pemilik:** Pemilik arsitektur (belum ditetapkan)
- **Status:** Disetujui
- **Rincian:** `docs/PRD.md` §11 "Keputusan integrasi layanan pendukung"
- **Bukti:** `backend/internal/outbox/`, `backend/migrations/000006_outbox_message.sql`

### DEC-021 — Mata uang tunggal: Rupiah Indonesia (IDR)

- **Tanggal:** 2026-11-11
- **Pemilik:** Pemilik produk (belum ditetapkan)
- **Status:** Disetujui
- **Rincian:** `docs/PRD.md` §11 "Keputusan mata uang, zona waktu, dan struktur backend"
- **Catatan:** sistem memakai satu mata uang tanpa konversi atau nilai tukar tersimpan. Format nominal dibentuk hanya oleh `internal/money` (`money.Format` → `Rp 1.500.000`; pemisah ribuan titik, desimal koma). Simbol atau kode mata uang lain tidak boleh muncul di API maupun dokumen.
- **Bukti:** `backend/internal/money/`, `backend/internal/httpserver/payroll_handlers.go`

### DEC-022 — Zona waktu operasional Asia/Jakarta, penyimpanan UTC

- **Tanggal:** 2026-11-11
- **Pemilik:** Pemilik platform/DB (belum ditetapkan)
- **Status:** Disetujui
- **Rincian:** `docs/PRD.md` §11 "Keputusan mata uang, zona waktu, dan struktur backend"
- **Catatan:** tampilan dan aturan kalender memakai Asia/Jakarta (WIB) lewat `internal/timeutil`; timestamp disimpan UTC pada kolom `timestamp without time zone` dengan DSN yang dipaksa `timezone=UTC`; kolom `date` ditulis sebagai tengah malam UTC agar tanggal yang dibaca kembali tetap sama.
- **Bukti:** `backend/internal/timeutil/`, `backend/internal/database/postgres.go`, `backend/internal/database/postgres_test.go`

### DEC-023 — Struktur backend empat lapis per area bisnis

- **Tanggal:** 2026-11-11
- **Pemilik:** Pemilik arsitektur (belum ditetapkan)
- **Status:** Disetujui
- **Rincian:** `docs/PRD.md` §11 "Keputusan mata uang, zona waktu, dan struktur backend"
- **Catatan:** tiap area bisnis dibedakan menjadi `domain` (aturan tanpa I/O), `app` (use case dengan pemeriksaan permission dan cakupan), `repository` (port persistensi + adapter GORM, satu-satunya yang menulis SQL), dan `httpserver` (transport, DTO, middleware). Kontrak daftar bersama ada di `internal/listing`. Handler tidak memuat aturan bisnis.
- **Bukti:** `backend/internal/directory/{domain,app,repository}`, `backend/internal/payroll/{domain,app,repository}`, `backend/internal/listing/`

### DEC-025 — Modul penggajian dan konfirmasi pembayaran

- **Tanggal:** 2026-11-11
- **Pemilik:** Pemilik produk dan pemilik keuangan (belum ditetapkan)
- **Status:** Disetujui
- **Rincian:** `docs/PRD.md` §11 "Keputusan modul penggajian dan konfirmasi pembayaran"
- **Catatan:** permission `payroll:read:any` (admin), `payroll:read:homebase` (operator, slip pegawai unitnya beserta turunannya), `payroll:read:self` (pegawai), dan `payroll:manage` (hanya admin). Slip hanya dapat dibayar satu kali dan hanya dari status `TERBIT`; baris `pembayaran`, status slip `TERBAYAR`, antrean notifikasi gaji, dan jejak audit `payroll.payment.confirmed` ditulis dalam satu transaksi. Email gaji kini dapat terkirim karena sumber konfirmasi pembayaran sudah ada, dan tetap tidak memuat nominal.
- **Bukti:** `backend/internal/payroll/{domain,app,repository}`, `backend/internal/httpserver/payroll_handlers.go`, `backend/migrations/000008_payment_confirmation.sql`, `prisma/schema.prisma`

### DEC-026 — Konfirmasi pembayaran manual pada MVP, adapter bank disiapkan

- **Tanggal:** 2026-11-11
- **Pemilik:** Pemilik produk dan pemilik keuangan (belum ditetapkan)
- **Status:** Disetujui
- **Rincian:** `docs/PRD.md` §11 "Keputusan modul penggajian dan konfirmasi pembayaran"
- **Catatan:** integrasi langsung ke bank sulit dilakukan sekarang, maka pencatatan pembayaran pada MVP dilakukan manual oleh admin: referensi transaksi/rekapan bank, waktu, dan pengesah diisi manusia ke tabel `pembayaran`. Integrasi otomatis ke bank (REST dan websocket) disiapkan sebagai adapter lanjutan yang menulis bukti pembayaran lewat jalur transaksional yang sama dengan idempotensi; implementasinya menunggu kontrak API vendor bank dan tidak dibuat sebelum benar-benar dipakai.
- **Bukti:** `backend/internal/payroll/domain/payment.go`, `backend/internal/payroll/repository/gorm.go`, `backend/internal/httpserver/payroll_handlers.go`

### DEC-027 — Jadwal gajian per unit dan pemberitahuan slip H+1

- **Tanggal:** 2026-11-11
- **Pemilik:** Pemilik produk (belum ditetapkan)
- **Status:** Disetujui
- **Rincian:** `docs/PRD.md` §11 "Keputusan jadwal gajian, pemberitahuan slip H+1, dan kalender hari libur"
- **Catatan:** tanggal gajian bersifat tetap (1–28) per unit organisasi, diinput lewat `PUT /api/v1/payroll/schedules/{unitId}` dengan permission `payroll:schedule` (admin, seluruh unit) atau `payroll:schedule:homebase` (operator, unitnya beserta turunannya); nilai disimpan di tabel `jadwalGaji` (satu baris per unit). Pemberitahuan ketersediaan slip dikirim H+1 pada **tanggal kalender tetap** — tidak digeser karena akhir pekan atau hari libur nasional, dan tidak maju selama periode berjalan belum lewat jadwalnya. Jenis pesan outbox `email.slip-tersedia` dengan kunci idempotensi `slip-available:{slipId}`; pekerjaan dijalankan lewat job QStash `payroll.notify` (`5 19 * * *` = 02:05 WIB). Isi pesan hanya nama, periode, dan tautan — **tidak memuat nominal gaji dan tidak menyatakan gaji sudah dibayar**; itu pemberitahuan ketersediaan slip, bukan notifikasi gaji terbayar, yang tetap mensyaratkan konfirmasi pembayaran tersimpan.
- **Bukti:** `backend/internal/payroll/domain/schedule.go`, `backend/internal/payroll/app/schedule.go`, `backend/internal/payroll/repository/schedule.go`, `backend/internal/notify/slip.go`, `backend/internal/httpserver/payroll_schedule_handlers.go`, `backend/migrations/000009_payroll_schedule.sql`, `prisma/schema.prisma`

### DEC-028 — Integrasi kalender hari libur nasional (api.co.id)

- **Tanggal:** 2026-11-11
- **Pemilik:** Pemilik platform (belum ditetapkan)
- **Status:** Disetujui
- **Rincian:** `docs/PRD.md` §11 "Keputusan jadwal gajian, pemberitahuan slip H+1, dan kalender hari libur"
- **Catatan:** kalender hari libur nasional diisi dari penyedia **api.co.id** (`GET /holidays/indonesia`, header `x-api-co-id`; sebelumnya dievaluasi date.nager.at dan tidak dipakai). Sinkronisasi hanya berjalan bila `HOLIDAY_API_URL` dan `HOLIDAY_API_KEY` terisi; dipicu admin lewat `POST /api/v1/holidays/{year}/refresh` (permission `holiday:manage`) dan disimpan dengan mengganti seluruh entri tahun tersebut dalam satu transaksi bersama jejak audit `holiday.refreshed`. Entri disimpan untuk `type` `Public Holiday`, `National Holiday`, dan `Joint Holiday`; `Observance` dilewati; tanggal di luar tahun diminta dan tanggal ganda dibuang; entri tanpa nama atau tanggal tidak sah menolak seluruh penyegaran. Pembacaan (`GET /api/v1/holidays?year=`, permission `holiday:read`) selalu dari tabel `HariLibur` lokal — tidak pernah bergantung pada penyedia — dan di-cache di Redis (`holiday:{year}`, TTL 24 jam). Kalender tidak mengubah tanggal gajian maupun pemberitahuan H+1. `HOLIDAY_API_KEY` saat ini kosong di `.env` sehingga penyedia nonaktif sampai kunci dari dashboard api.co.id diisi.
- **Bukti:** `backend/internal/holiday/`, `backend/internal/httpserver/holiday_handlers.go`, `backend/cmd/server/main.go`, `backend/migrations/000010_holiday.sql`, `.env.example`, `.env`

---

## Keputusan integrasi

### DEC-014 — Provider email SMTP

- **Tanggal:** 2026-11-11
- **Pemilik:** Pemilik komunikasi/IT (belum ditetapkan)
- **Status:** Disetujui
- **Rincian:** `docs/PRD.md` §11 "Keputusan integrasi layanan pendukung"
- **Catatan:** Mekanisme SMTP sudah diimplementasikan (`backend/internal/notify`) dan konsisten dengan konfigurasi pada `.env.example` dan `README.md`. Schema menyebut Resend sebagai kandidat lain; Resend juga menyediakan API SMTP sehingga migrasi ke sana tidak memerlukan perubahan kode. **Alamat pengirim ditetapkan: `no-reply@simpeg.unisubang.ac.id`**, dengan origin aplikasi `https://simpeg.unisubang.ac.id`. Backend memvalidasi bentuk alamat pengirim saat start.
- **Bukti:** `backend/internal/notify/smtp.go`, `backend/internal/notify/notify_test.go`, `backend/internal/notify/deliverer.go`, `backend/internal/config/config_test.go`, `.env.example`

### DEC-015 — Use case Redis: cache daftar kategori pegawai

- **Tanggal:** 2026-11-11
- **Pemilik:** Pemilik platform (belum ditetapkan)
- **Status:** Disetujui
- **Rincian:** `docs/PRD.md` §11 "Keputusan integrasi layanan pendukung"
- **Catatan:** use case yang spesifik adalah cache daftar kategori pegawai: data referensi dengan kardinalitas rendah, dibaca setiap kali formulir pegawai dibuka, dan tidak termasuk data sensitif. TTL 5 menit, satu namespace `simpeg`, dan `ReadThrough` memuat ulang dari PostgreSQL saat cache miss atau Redis gagal. **Queue tidak wajib untuk alur kritis** karena durability dijamin outbox di PostgreSQL, bukan oleh Redis. Kunci cache memuat seluruh parameter daftar (page, size, sort, order, pencarian) supaya tiap kombinasi filter punya entri sendiri. Cache sudah disambungkan ke `GET /api/v1/categories`; tanpa Redis konfigurasi pembacaan tetap langsung ke PostgreSQL.
- **Bukti:** `backend/internal/directory/app/catalog.go`, `backend/internal/directory/app/catalog_test.go`, `backend/internal/cache/`

### DEC-016 — S3-compatible storage di luar scope

- **Tanggal:** 2026-11-11
- **Pemilik:** Pemilik produk (belum ditetapkan)
- **Status:** Disetujui
- **Rincian:** `docs/PRD.md` §11 "Keputusan integrasi layanan pendukung"
- **Catatan:** mekanismenya sengaja tidak dibuat agar tidak ada kode tanpa pemakaian.

### DEC-024 — Callback QStash dan pembersihan audit terjadwal

- **Tanggal:** 2026-11-11
- **Pemilik:** Pemilik platform (belum ditetapkan)
- **Status:** Disetujui
- **Rincian:** `docs/PRD.md` §11 "Keputusan integrasi layanan pendukung" dan "Kebijakan jejak audit"
- **Catatan:** QStash dipakai sebagai penjadwal, bukan sumber pekerjaan. Callback `POST /api/v1/internal/qstash/audit/cleanup` tidak memakai sesi bearer; pengesahnya signature HS256 atas raw request body dengan subjek yang cocok dengan URL tujuan. Pekerjaannya memangkas jejak audit lebih tua dari `AUDIT_RETENTION_DAYS` hari (default 90, `0` = tanpa pemangkasan) lewat `audit.Cleaner`, bersifat idempoten, dan menulis jejak `audit.cleaned`. Jadwal harian `0 19 * * *` (19:00 UTC = 02:00 WIB) didaftarkan lewat `internal/jobs.EnsureSchedule` yang idempoten, dengan 3 percobaan ulang.
- **Bukti:** `backend/internal/httpserver/qstash_handlers.go`, `backend/internal/qstash/signature.go`, `backend/internal/qstash/schedule.go`, `backend/internal/jobs/jobs.go`, `backend/internal/audit/cleaner.go`, `backend/cmd/server/main.go`

---

## Keputusan yang masih menunggu

### DEC-017 — Pemilik produk, kelompok pilot, dan pembagian peran approval

- **Tanggal:** 2026-11-11
- **Pemilik:** Rektor Universitas Subang
- **Status:** Disetujui
- **Rincian:** `docs/PRD.md` §11 "Keputusan environment deployment"
- **Catatan:**
  - **Pemilik produk:** Rektor Universitas Subang.
  - **Kelompok pengguna pilot:** admin sistem dan operator kepegawaian dari satu fakultas dan satu biro.
  - **Pembagian peran approval:** matriks peran disetujui bersama Rektor dan Kabiro.
  - Nama perorangan untuk admin sistem, operator, dan anggota Kabiro belum ditetapkan; yang dibutuhkan agar pilot dapat berjalan adalah daftar akun dengan peran (`admin`, `operator`) beserta unit penugasannya.
- **Bukti:** matriks peran di `docs/PRD.md` §11, grant di `backend/cmd/server/main.go`

### DEC-018 — Arsip dan ekspor jejak audit

- **Tanggal:** —
- **Pemilik:** Pemilik kebijakan (belum ditetapkan)
- **Status:** Menunggu
- **Rincian:** `docs/PRD.md` §11 "Kebijakan jejak audit"
- **Catatan:** masa retensi otomatis (90 hari, `AUDIT_RETENTION_DAYS`) beserta pekerjaan pembersihan terjadwalnya sudah disetujui terpisah — lihat `DEC-024`. Yang masih menunggu adalah kewajiban arsip ke media lain, pihak yang berhak mengekspor jejak, dan prosedur pemangkasan lain di luar retensi otomatis.
- **Bukti:** `backend/internal/audit/audit_test.go` (penjaga append-only)

### DEC-019 — Paparan Supabase Data API dan kebijakan RLS

- **Tanggal:** —
- **Pemilik:** Pemilik platform/DB (belum ditetapkan)
- **Status:** Menunggu
- **Rincian:** `docs/PRD.md` §11 "Keputusan dan penanggung jawab yang masih diperlukan" butir 5
- **Catatan:** RLS default-deny sudah aktif pada seluruh tabel aplikasi, jadi tabel yang belum diputuskan belum menciptakan kebocoran. Policy eksplisit baru bisa dibuat setelah keputusan ini.
- **Bukti:** `backend/migrations/000004_row_level_security.sql`, `backend/internal/database/migrations_test.go`

### DEC-020 — Target ketersediaan, RPO, RTO, dan SLO

- **Tanggal:** —
- **Pemilik:** Pemilik platform/DB dan pemilik operasi (belum ditetapkan)
- **Status:** Menunggu
- **Rincian:** `docs/PRD.md` §11 "Draf migrasi, backup, audit, dan ketersediaan"
- **Catatan:** default `RPO_HOURS` (24 jam), `RTO_HOURS` (4 jam), dan `SLO_TARGET` (`99.9`) sudah dapat dikonfigurasi lewat environment dan dibaca `config.Load()`, tetapi angka final tetap menunggu persetujuan pemilik platform/DB dan pemilik operasi sesuai tier layanan dan anggaran.
- **Bukti:** `backend/internal/config/`, `backend/internal/config/config_test.go`
