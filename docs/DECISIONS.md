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

### DEC-029 — Groundwork penerbitan sesi dan masa berlaku sesi (backend)

- **Tanggal:** 2026-11-11
- **Pemilik:** Pemilik identitas/IT (belum ditetapkan)
- **Status:** Disetujui
- **Rincian:** `docs/PRD.md` §11 "Keputusan autentikasi dan kontrak sesi" serta §8 (Backend). Backend disiapkan groundwork untuk penerbitan sesi sebelum runtime Better Auth tersedia: `auth.SessionIssuer` dengan implementasi GORM (`GormSessionIssuer`) yang menulis baris baru ke tabel `session` dengan token acak dan `expiresAt`; role/homebase diambil dari data server-side (tabel user/role) sehingga tidak dapat digunakan untuk naikkan role. Endpoint bootstrap `POST /api/v1/auth/session` didaftarkan bila resolver dan issuer tersedia, memerlukan sesi yang valid dan permission `auth:session:issue` (saat ini hanya admin). Masa berlaku sesi dikonfigurasi lewat `SESSION_LIFETIME` env (`config.SessionLifetime`, default 23 jam). Nilai final ditetapkan 23 jam oleh pemilik, efektif sejak 2026-11-11.
- **Catatan:** Endpoint ini akan digantikan oleh flow login OAuth/password milik runtime Better Auth. Penerbitan token JWT tetap menjadi tanggung jawab runtime autentikasi, bukan backend Go.
- **Bukti:** `backend/internal/auth/session_issuer.go`, `backend/internal/auth/session_issuer_test.go`, `backend/internal/httpserver/auth_handlers.go`, `backend/internal/httpserver/auth_handlers_test.go`, `backend/internal/httpserver/server.go` (`MuxOptions.SessionIssuer`), `backend/cmd/server/main.go`, `backend/internal/config/config.go` (`SESSION_LIFETIME`), `.env.example`

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
- **Bukti:** `backend/internal/payroll/{domain,app,repository}`, `backend/internal/httpserver/payroll_handlers.go`, `backend/migrations/000008_payment_confirmation.sql`

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
- **Bukti:** `backend/internal/payroll/domain/schedule.go`, `backend/internal/payroll/app/schedule.go`, `backend/internal/payroll/repository/schedule.go`, `backend/internal/notify/slip.go`, `backend/internal/httpserver/payroll_schedule_handlers.go`, `backend/migrations/000009_payroll_schedule.sql`

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

### DEC-030 — Autentikasi perangkat untuk ingestion presensi

- **Tanggal:** 2026-11-11
- **Pemilik:** Pemilik platform/DB (belum ditetapkan)
- **Status:** Disetujui
- **Rincian:** `docs/PRD.md` §11 "Keputusan fondasi presensi mesin". Endpoint `POST /api/v1/attendance/events` mendukung dua jalur autentikasi: (1) sesi pengguna dengan permission `attendance:event:ingest` (hanya `admin` melalui sesi API), dan (2) autentikasi perangkat melalui header `X-Device-ID` + `Authorization: Bearer <device-token>` yang diverifikasi terhadap tabel `machineCredential` (hash, bukan plaintext). Autentikasi perangkat terpisah dari sesi pengguna dan tidak dapat digunakan untuk mengakses data non-presensi.
- **Catatan:** Tabel `machineCredential` ditambahkan via migrasi `000011_machine_credential.sql`. Role dan grant untuk autentikasi perangkat masih menunggu kontrak API/SDK vendor (Fingerspot Revo W-230N) — dokumentasi vendor belum tersedia, jadi adapter vendor dan format kredensial produksi belum dibuat.
- **Bukti:** `backend/internal/attendance/device_auth.go`, `backend/internal/attendance/device_auth_test.go`, `backend/internal/httpserver/attendance_handlers.go`, `backend/internal/httpserver/attendance_handlers_test.go`, `backend/migrations/000011_machine_credential.sql`, `backend/internal/models/models_operations.go`

### DEC-031 — Kebijakan idempotency key event presensi

- **Tanggal:** 2026-11-11
- **Pemilik:** Pemilik platform/DB (belum ditetapkan)
- **Status:** Disetujui
- **Rincian:** `docs/PRD.md` §11 "Keputusan fondasi presensi mesin". Idempotency key harus unik per pasangan `(machineId, tap event)`. Key dipanggil oleh adapter vendor dan dikirim dalam body request. Jika key sama digunakan dengan fakta tap yang berbeda (machinePin atau occurredAt berubah), server menolak dengan `409 Conflict`. Jika key sama dengan fakta tap yang sama, server mengembalikan event yang sama (`duplicate: true`) tanpa duplikasi.
- **Catatan:** Sumber idempotency key yang dijamin vendor belum ditetapkan; menunggu kontrak API/SDK vendor. Format yang disarankan: UUID v4.
- **Bukti:** `backend/internal/attendance/store.go`, `backend/internal/attendance/store_test.go`, `backend/internal/httpserver/attendance_handlers.go`, `backend/migrations/000002_attendance_events.sql` (unique index `(machineId, idempotencyKey)`)

### DEC-032 — Retensi event presensi

- **Tanggal:** 2026-11-11
- **Pemilik:** Pemilik platform/DB (belum ditetapkan)
- **Status:** Disetujui
- **Rincian:** Event presensi (`AttendanceEvent`) disimpan sebagai fakta tap mentah tanpa menentukan status harian. Masa retensi disetapkan sama dengan jejak audit: `AUDIT_RETENTION_DAYS` (default 90 hari, `0` = tanpa pemangkasan otomatis). Pembersihan event lewat mekanisme yang sama dengan `audit.Cleaner` (QStash callback) dapat ditambahkan saat kebutuhan operasional dikonfirmasi.
- **Catatan:** Aturan retensi dan mekanisme pembersihan event masih menunggu konfirmasi pemilik. Event tidak mengubah tabel `Presensi` (satu status per orang/hari).
- **Bukti:** `backend/internal/attendance/`, `backend/internal/audit/cleaner.go`, `.env.example` (`AUDIT_RETENTION_DAYS`)

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

- **Tanggal:** 12 November 2026
- **Pemilik:** Pemilik kebijakan (Rektor Universitas Subang, belum ditetapkan perorangan)
- **Status:** Disetujui
- **Rincian:** `docs/PRD.md` §11 "Kebijakan jejak audit"
- **Catatan:** Ekspor CSV disediakan melalui `GET /api/v1/audit-logs/export` dengan permission `audit:read` (admin, seluruh jejak) atau `audit:read:homebase` (operator, jejak unit penugasannya beserta turunannya). Filter sama dengan endpoint daftar (hanya `aksi` dan `search`); sort dan pagination diabaikan, urutan kronologis naik (`createdAt` ASC, `id` ASC), tanpa batas jumlah baris (streaming per batch via keyset pagination). Berkas dikirim dengan `text/csv; charset=utf-8` dan BOM UTF-8; setiap ekspor yang berhasil dicatat sebagai aksi `audit.exported` dengan jumlah baris. Kewajiban arsip ke media lain dan prosedur pemangkasan di luar retensi otomatis tetap belum ditetapkan. Masa retensi otomatis (90 hari, `AUDIT_RETENTION_DAYS`) dan pekerjaan pembersihan terjadwalnya sudah disetujui terpisah — lihat `DEC-024`.
- **Bukti:** `backend/internal/audit/audit_test.go` (penjaga append-only), `backend/internal/httpserver/audit_handlers.go` (`exportAuditLogs`), `docs/openapi.yaml`

### DEC-019 — Paparan Supabase Data API dan kebijakan RLS

- **Tanggal:** 2026-11-11
- **Pemilik:** Pemilik platform/DB (belum ditetapkan)
- **Status:** Disetujui
- **Rincian:** `docs/PRD.md` §8 (Backend dan Frontend): aplikasi mengakses database hanya melalui GORM di backend (koneksi server-side dengan TLS, role database berhak minimum); frontend berkomunikasi dengan backend lewat kontrak HTTP API, **bukan** langsung ke database/Supabase Data API.
- **Catatan:** Keputusan final: **GORM-only, Supabase Data API tidak diekspos** untuk tabel aplikasi, sehingga tidak diperlukan policy RLS per tabel pada waktu ini. RLS default-deny tetap aktif pada seluruh 45 tabel aplikasi sebagai pertahanan dalam-dalam (mengakapi kebocoran bila kebijakan berubah di masa depan). Jika Data API diekspos di masa depan, policy per tabel wajib dibuat dan diuji satu per tabel sebelum tabel diakses melalui Data API.
- **Bukti:** `docs/PRD.md` §8, `backend/migrations/000004_row_level_security.sql`, `backend/internal/database/migrations_test.go` (`TestRowLevelSecurityMigrationCoversEveryTable`)`

### DEC-020 — Target ketersediaan, RPO, RTO, dan SLO

- **Tanggal:** —
- **Pemilik:** Pemilik platform/DB dan pemilik operasi (belum ditetapkan)
- **Status:** Menunggu
- **Rincian:** `docs/PRD.md` §11 "Draf migrasi, backup, audit, dan ketersediaan"
- **Catatan:** default `RPO_HOURS` (24 jam), `RTO_HOURS` (4 jam), dan `SLO_TARGET` (`99.9`) sudah dapat dikonfigurasi lewat environment dan dibaca `config.Load()`, tetapi angka final tetap menunggu persetujuan pemilik platform/DB dan pemilik operasi sesuai tier layanan dan anggaran.
- **Bukti:** `backend/internal/config/`, `backend/internal/config/config_test.go`

### DEC-033 — Workflow deployment migrasi Goose

- **Tanggal:** 2026-11-11
- **Pemilik:** Pemilik platform/DB (belum ditetapkan)
- **Status:** Disetujui
- **Rincian:** `docs/PRD.md` §8 (Backend) dan `docs/RUNBOOK.md` §7 (Pemulihan setelah penerapan migrasi). Workflow GitHub Actions `.github/workflows/deploy-migrations.yml` dibuat untuk menjalankan migrasi Goose secara terkendali. Workflow dipicu secara manual (`workflow_dispatch`) dengan pilihan environment `staging` atau `production`, bukan berjalan otomatis pada setiap push. Setiap jalankan memverifikasi format, vet, tes, paritas skema, nomor migrasi urut, dan header migrasi sebelum menjalankan `goose status` lalu `goose up` terhadap database target. `DATABASE_URL` disimpan sebagai secret GitHub, tidak ada di repo.
- **Catatan:** Workflow siap digunakan, tetapi migrasi masih belum pernah diterapkan pada database nyata. Sebelum menjalankan di production, operator wajib mencoba di staging dengan data sintetis dan memastikan backup/restore sudah diuji (lihat `docs/RUNBOOK.md` §7).
- **Bukti:** `.github/workflows/deploy-migrations.yml`, `backend/.goose.yaml`, `backend/migrations/`, `backend/internal/database/migrations_test.go`, `docs/RUNBOOK.md` §7

### DEC-034 — Kebijakan akses Google OAuth, allowlist domain, undangan, dan MFA

- **Tanggal:** 2026-11-11
- **Pemilik:** Pemilik produk (belum ditetapkan nama perorangan)
- **Status:** Disetujui
- **Rincian:** `docs/PRD.md` §4 FR-01 Akses pengguna dan §11 Keputusan integrasi yang masih terbuka. Autentikasi SIMPEG menggunakan Google OAuth via runtime Better Auth. Kebijakan akses ditetapkan sebagai berikut:
  1. Eligibility ditentukan oleh kombinasi **domain email yang diizinkan** (`ALLOWED_EMAIL_DOMAINS`) dan **undangan**. Kedua mekanisme aktif secara bersamaan (keduanya).
  2. Masa berlaku sesi (opaque session backend) ditetapkan final **23 jam** (`SESSION_LIFETIME=23h`), menggantikan groundwork 24 jam.
  3. MFA bersifat opsional dan tidak diwajibkan untuk semua peran. Aktivasi MFA hanya untuk pengguna yang menginginkan keamanan ekstra; tidak ada kewajiban peran berisiko.
- **Catatan:** Implementasi backend menyediakan konfigurasi `GoogleClientID`, `GoogleClientSecret`, `AllowedEmailDomains`, dan `InviteOnly` via environment. Validasi konfigurasi menolak kombinasi allowlist/undangan tanpa Google OAuth terkonfigurasi. Runtime Better Auth pada frontend/layer autentikasi yang akan menegakkan allowlist dan undangan. Backend tidak mengimplementasikan alur OAuth secara langsung; backend hanya memverifikasi JWT/sesi Better Auth.
- **Bukti:** `backend/internal/config/config.go` (field Google/allowlist/invite), `.env.example` (`GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `ALLOWED_EMAIL_DOMAINS`, `INVITE_ONLY`, `SESSION_LIFETIME=23h`), `docs/PRD.md` §4 FR-01, `docs/DECISIONS.md` DEC-029

### DEC-035 — Versi Go minimum dan versi alat CI yang dikunci

- **Tanggal:** 2026-11-11
- **Pemilik:** Pemilik platform (belum ditetapkan)
- **Status:** Disetujui
- **Rincian:** `backend/go.mod` mengunci versi Go minimum modul (`go 1.26.11`), dan `.github/workflows/backend.yml` menjalankan `staticcheck@v0.8.2` serta `govulncheck@latest`. CI memakai `go-version-file: backend/go.mod`, sehingga versi Go yang diuji CI sama persis dengan versi minimum yang dinyatakan modul.
- **Catatan:** Versi Go dikunci pada rilis patch yang sudah menambal advisori `GO-2026-6859` (CVE-2026-97054,FP race pada `net/http.MaxBytesReader` yang dapat membuat HTTP/1 server panic). Rilis `go 1.26.0` yang sebelumnya dipakai masih rentan, dan `govulncheck` di CI akan gagal pada versi itu. `staticcheck@2025.1.1` yang sebelumnya dipin tidak lagi membaca export data Go 1.26 ("export data version 5 is greater than maximum supported version 2"), sehingga CI tidak dapat menyelesaikan pemeriksaan statis; `v0.8.2` membaca export data Go 1.26 dengan benar dan melaporkan nol temuan pada kode saat ini. `golang.org/x/crypto` dinaikkan ke `v0.58.1-0.20261009181718-2f1f834330b2` untuk menutup advisori `GO-2026-5932` pada paket `openpgp` yang tidak lagi dipelihara. Kenaikan versi Go tidak mengubah kode sumber dan tidak mengubah kontrak API.
- **Bukti:** `backend/go.mod`, `backend/go.sum`, `.github/workflows/backend.yml`, `docs/PERFORMANCE-BASELINE.md` §1 (angka benchmark diukur memakai Go 1.26.11), `docs/IMPLEMENTATION-PLAN.md` §8

### DEC-036 — Perkakas pengukuran performa dan database pengukuran terpisah

- **Tanggal:** 2026-11-12
- **Pemilik:** Pemilik platform (belum ditetapkan)
- **Status:** Disetujui
- **Rincian:** Perkakas pengukuran performa dibuat sebagai `backend/cmd/perfbench` dengan subcommand `seed`, `schema`, `explain`, `latency`, `e2e`, dan `deps`. Perkakas measuring menjalankan binari `cmd/server` asli sebagai proses anak, sehingga pengukuran memakai rantai handler produksi tanpa duplikasi. SQL yang dianalisis direkam pada level connection pool GORM, sehingga `EXPLAIN` dijalankan atas statement yang benar-benar diterima PostgreSQL.
- **Catatan:** Database pengukuran dibuat sebagai database terpisah (`simpeg_perf`) pada proyek Supabase yang sama. Database `postgres` yang dipakai production tidak dipakai untuk seeding, dan datanya tetap nol baris. Pemeriksaan keselarasan skema (`perfbench schema`) membandingkan kolom yang dinyatakan model GORM dengan kolom yang benar-benar ada pada database target, karena keberadaan migrasi tidak membuktikan migrasi sudah diterapkan maupun model sudah selaras dengan tabel. Perkakas hanya membaca kredensial dari environment; tidak ada nilai rahasia yang ditulis di dalam repo.
- **Bukti:** `backend/cmd/perfbench/`, `backend/migrations/000016_payment_updated_at.sql`, `docs/PERFORMANCE-BASELINE.md` §2–§4, `docs/RUNBOOK.md` §8

### DEC-037 — Perbaikan jalur konfirmasi pembayaran gaji pada skema nyata

- **Tanggal:** 2026-11-12
- **Pemilik:** Pemilik platform (belum ditetapkan)
- **Status:** Disetujui
- **Rincian:** Model `models.Pembayaran` memakai `BaseModel`, sehingga GORM menulis kolom `updatedAt` pada setiap `INSERT`. Tabel `pembayaran` yang dibuat migrasi `000008_payment_confirmation.sql` hanya memiliki `createdAt`. Akibatnya setiap konfirmasi pembayaran gaji pada `POST /api/v1/payroll/slips/{payrollId}/payment` gagal dengan `SQLSTATE 42703`.
- **Catatan:** Perbaikan dibuat sebagai migrasi `000016_payment_updated_at.sql`, bukan dengan mengubah migrasi lama, agar jejak versi tetap utuh. Migrasi menambahkan kolom yang sudah dinyatakan model sehingga tidak ada perubahan perilaku aplikasi. Keselarasan model dan skema diverifikasi ulang oleh `perfbench schema` terhadap seluruh 67 model. Temuan ini sekaligus membatalkananggapan bahwa migrasi belum pernah diterapkan pada database nyata: `goose_db_version` pada database `postgres` sudah memuat 16 versi sejak 2026-11-11.
- **Bukti:** `backend/migrations/000016_payment_updated_at.sql`, `backend/internal/models/models_payroll.go`, `backend/internal/payroll/repository/gorm.go`, `docs/RUNBOOK.md` §8

### DEC-038 — Perbaikan endpoint QStash dan jadwal regional

- **Tanggal:** 2026-11-12
- **Pemilik:** Pemilik platform (belum ditetapkan)
- **Status:** Disetujui
- **Rincian:** `qstash.Publisher` menempelkan URL tujuan langsung ke `BaseURL`. Nilai `QSTASH_URL` pada environment adalah root API regional (`https://qstash-us-east-1.upstash.io`), bukan endpoint lengkap, sehingga URL tujuan menjadi `https://qstash-us-east-1.upstash.io/<tujuan>` dan QStash menjawab `404`. Endpoint jadwal juga jatuh ke region global karena `QSTASH_SCHEDULES_URL` kosong.
- **Catatan:** Publisher kini menambah jalur `/v2/publish` bila `BaseURL` belum memuatnya, dan endpoint jadwal diturunkan dari `BaseURL` yang sama sehingga token, publikasi, dan jadwal berada pada satu region. Nilai `QSTASH_SCHEDULES_URL` yang diisi eksplisit tetap dipakai apa adanya. Pemeriksaan nyata menunjukkan format URL tujuan harus berupa URL mentah pada jalur, bukan ter-encode: QStash menolak bentuk ter-encode dengan "endpoint has invalid scheme".
- **Bukti:** `backend/internal/qstash/publisher.go`, `backend/internal/qstash/schedule.go`, `backend/internal/qstash/publisher_endpoint_test.go`, `docs/PERFORMANCE-BASELINE.md` §5

### DEC-039 — Dukungan STARTTLS pada transport SMTP

- **Tanggal:** 2026-11-12
- **Pemilik:** Pemilik platform (belum ditetapkan)
- **Status:** Disetujui
- **Rincian:** `notify.smtpTransport` hanya knewa dua mode: TLS langsung (`UseTLS`) dan koneksi polos tanpa peningkatan. Server submission pada porta 587 menerima koneksi polos lalu menawarkan `STARTTLS`. Dengan `SMTP_HOST=smtp.gmail.com` dan `SMTP_PORT=587`, mode TLS langsung gagal dengan `tls: first record does not look like a TLS handshake`, sehingga tidak ada email yang pernah terkirim.
- **Catatan:** Transport kini menaikkan koneksi ke TLS otomatis bila server menawarkan `STARTTLS`, dan peningkatan dilakukan sebelum autentikasi karena kredensial tidak boleh dikirim lewat koneksi polos. Penentuan mode dipindah ke `config.SMTPConfig`: `SMTP_SECURE=true` berarti TLS langsung, `SMTP_SECURE=false` berarti koneksi polos yang dinaikkan lewat STARTTLS, dan bila `SMTP_SECURE` tidak disetel, mode ditebak dari porta (465 memakai TLS langsung, porta lain memakai STARTTLS). Nilai bawaan lama `SMTP_SECURE=true` tidak lagi berlaku karena tidak sesuai dengan porta submission.
- **Bukti:** `backend/internal/notify/smtp.go`, `backend/internal/config/smtp.go`, `backend/internal/notify/smtp_starttls_test.go`, `docs/PERFORMANCE-BASELINE.md` §5

### DEC-040 — Penyimpanan metrik SLO di PostgreSQL dan alert lewat antrean keluar

- **Tanggal:** 2026-11-12
- **Pemilik:** Pemilik platform dan pemilik operasi (belum ditetapkan)
- **Status:** Disetujui
- **Rincian:** Alert SLO dijalankan sebagai pekerjaan terjadwal QStash (`slo.evaluate`, cron `*/5 * * * *`) yang memanggil `POST /api/v1/internal/qstash/slo/evaluate`. Callback memakai autentikasi signature QStash, sama seperti callback lain. Metrik dikumpulkan pengoleki di dalam proses backend dan disimpan ke tabel `sloMetric` di PostgreSQL.
- **Catatan:** Metrik disimpan di PostgreSQL, bukan Redis, karena Redis pada repo ini diperuntukkan hanya untuk cache data referensi dan ditolak untuk data historis maupun status. Alert tidak dikirim langsung melainkan diantrekan ke antrean keluar persisten dengan kunci idempotensi yang memuat awal jendela, sehingga percobaan ulang callback QStash tidak menghasilkan email ganda. Persentil dihitung dari histogram bucket; keputusan pelanggaran memakai batas atas bucket sehingga tidak pernah lolos karena perkiraan yang terlalu kecil, dan ambang 800 ms sengaja ada sebagai batas bucket karena angka itu target SLO bawaan.
- **Bukti:** `backend/internal/slo/`, `backend/internal/httpserver/slo_handlers.go`, `backend/migrations/000017_slo_metric.sql`, `backend/internal/jobs/jobs.go`, `docs/RUNBOOK.md` §8

### DEC-041 — Bentuk alamat email pengirim dan pemisahan pengirim amplop

- **Tanggal:** 2026-11-12
- **Pemilik:** Pemilik platform (belum ditetapkan)
- **Status:** Disetujui
- **Rincian:** Bentuk baku RFC 5322 `"Nama Tampilan" <alamat@domain>` ditolak oleh validasi `SMTP_FROM` yang hanya menerima alamat polos. Konsekuensinya backend menolak start dengan `config: SMTP_FROM must be a valid email address`, sehingga layanan tidak dapat dijalankan sama sekali dengan konfigurasi email yang lazim.
- **Catatan:** Pemisahan alamat kini berada di `notify.SplitAddress`: bentuk polos dikembalikan apa adanya, bentuk nama tampilan menghasilkan alamat polos di dalam kurung sudut. Header `From` memakai nilai yang dikonfigurasi apa adanya, sedangkan perintah SMTP `MAIL FROM` memakai alamat polos karena perintah itu tidak menerima bentuk nama tampilan. Validasi dipindah ke `notify.ValidateAddress` dan menerima kedua bentuk. Bentuk polos tetap menjadi default pada `.env.example`.
- **Bukti:** `backend/internal/notify/address.go`, `backend/internal/notify/smtp.go`, `backend/internal/config/config.go`, `backend/internal/notify/address_test.go`, `docs/PERFORMANCE-BASELINE.md` §8

### DEC-042 — RabbitMQ menggantikan QStash sebagai pengangkut pekerjaan terjadwal pada deployment self-hosted

- **Tanggal:** 2026-11-12
- **Pemilik:** Pemilik platform (belum ditetapkan)
- **Status:** Disetujui (arah pengguna), rincian ditetapkan pada pengerjaan ini
- **Rincian:** Deployment docker/podman memakai RabbitMQ self-hosted untuk pekerjaan terjadwal. Penjadwal `cmd/jobsched` (container terpisah) menerbitkan pesan `{"kind": ...}` ke exchange `simpeg.jobs` sesuai cron UTC yang sama dengan jadwal QStash; server mengonsumsi antrean `simpeg.jobs` dan menjalankan logika yang sama dengan callback QStash, lalu mengakui pesan hanya setelah sukses. Pesan gagal pindah ke `simpeg.jobs.dead` untuk diperiksa operator; tick cron berikutnya memicu eksekusi baru.
- **Catatan:** Bentuk payload dan kind identik dengan callback QStash, sehingga kedua transport setara dan pekerjaan tetap idempoten. Mode QStash tetap tersedia sebagai alternatif cloud (bila `RABBITMQ_URL` kosong dan kredensial QStash terisi); bila `RABBITMQ_URL` diisi, pendaftaran jadwal QStash dilewati. Pemeriksaan SLO hanya diterbitkan bila pengoleki metrik aktif (`SLO_EVALUATE_ENABLED` penjadwal = `SLO_ENABLED` backend) agar pesan tidak menumpuk di dead-letter pada deployment tanpa SLO.
- **Bukti:** `backend/internal/mq/`, `backend/cmd/jobsched/`, `backend/cmd/server/workers.go`, `docker-compose.yml`, `rabbitmq/docker-compose.yml`, `rabbitmq/README.md`, dan verifikasi end-to-end pada Docker lokal (publish `audit.cleanup` → `mq_job_done`; publish kind tanpa handler → antrean `simpeg.jobs.dead` berisi 1 pesan). Broker dipisah menjadi stack tersendiri pada `rabbitmq/docker-compose.yml` atas arah pengguna agar pengembangan broker tidak terikat siklus hidup stack backend; stack backend bergabung ke jaringan `simpeg-broker` sebagai external network.

### DEC-043 — Kenaikan versi Go modul ke 1.27.3

- **Tanggal:** 2026-11-12
- **Pemilik:** Pemilik platform (belum ditetapkan)
- **Status:** Disetujui (arah pengguna)
- **Rincian:** Direktif `go` pada `backend/go.mod` naik dari `1.26.11` ke `1.27.3`, mengikuti toolchain yang tersedia pada mesin pengembang. Versi 1.27.3 lebih baru daripada 1.26.11 sehingga tetap memuat tambalan advisori `GO-2026-6859` yang menjadi alasan penguncian sebelumnya (DEC-039). CI memakai `go-version-file: backend/go.mod` sehingga otomatis menguji 1.27.3; `staticcheck@v0.8.2` di CI terbukti tetap dapat membaca export data Go 1.27.3. Image builder `backend/Dockerfile` memakai `golang:1.27.3-alpine` dengan `GOTOOLCHAIN=local`.
- **Bukti:** `backend/go.mod`, `backend/Dockerfile`, dan verifikasi lokal pada Go 1.27.3: `gofmt -l` bersih, `go vet`, `go test`, `go build`, `go mod verify`, dan `go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./...` tanpa temuan.

**Koreksi (12 November 2026):** klaim pada entri ini bahwa "1.27.3 tetap memuat tambalan advisori `GO-2026-6859`" sudah tidak benar. Advisori `GO-2026-6859` (`net/http` race `MaxBytesReader`, CVE-2026-97054) diperbarui pada 11 November 2026 dengan tambalan pada `go1.27.4`; `go1.27.3` rentan karena kode backend memakai `http.MaxBytesReader`. Modul dinaikkan ke `go1.27.4` dan image dibangun ulang tanpa `gosu` — lihat `DEC-045`.


### DEC-044 — Aturan eviden pengajuan cuti

- **Tanggal:** 2026-11-12
- **Pemilik:** Pemilik produk (belum ditetapkan)
- **Status:** Disetujui (arah pengguna)
- **Rincian:** Cuti melahirkan (`BERSALIN`), cuti sakit (`SAKIT` dan `SAKIT_BERKEPANJANGAN`), dan cuti harian (`IZIN_PENTING`) wajib menyertakan eviden saat pengajuan dibuat; pengajuan tanpa eviden ditolak `422`, dan eviden dicek ulang saat persetujuan. Bentuk eviden: `SURAT_DOKTER` (berkas, nomor, tanggal surat dokter wajib) hanya untuk cuti sakit dan melahirkan; `FOTO` (berkas dan waktu tangkap wajib) atau `LAINNYA` (waktu dan keterangan wajib) untuk cuti harian. Metadata disimpan pada kolom baru tabel `Cuti`: `jenisEviden` (enum `SURAT_DOKTER`/`FOTO`/`LAINNYA`), `fileEvidenUrl`, `waktuEviden`, `catatanEviden`. Kolom lama `fileSuratUrl` di-`RENAME` menjadi `fileEvidenUrl` (bukan drop) lewat migrasi `000019_cuti_eviden.sql`.
- **Bukti:** `backend/migrations/000019_cuti_eviden.sql`, `backend/internal/directory/domain/leave_eviden.go`, `backend/internal/directory/app/leave.go`, `backend/internal/httpserver/hr_leave_handlers.go`, dan tes di `hr_leave_handlers_test.go` serta `leave_eviden_test.go`; kontrak di `docs/openapi.yaml`; kebijakan di `docs/PRD.md` §11.


### DEC-045 — Go dinaikkan ke 1.27.4; image container tidak lagi mengandung gosu

- **Tanggal:** 2026-11-12
- **Pemilik:** Pemilik platform (belum ditetapkan)
- **Status:** Disetujui (arah pengguna)
- **Rincian:** Direktif `go` pada `backend/go.mod` dan `ARG GO_VERSION` pada `backend/Dockerfile` naik dari `1.27.3` ke `1.27.4`. Alasan: advisori `GO-2026-6859` (race `MaxBytesReader` pada `net/http`, CVE-2026-97054, ditemukan pada `go1.27.3`) diperbaiki pada `go1.27.4`; backend memakai `http.MaxBytesReader` di 3 jalur handler. Hasil: `govulncheck@latest` melaporkan "No vulnerabilities found." Semua CVE standar Go yang dilaporkan sebelumnya (CVE-2025-68121, CVE-2026-39821, CVE-2026-39822; CVE-2026-56857 dan CVE-2026-78663 dengan rentang <1.26.9) tidak memengaruhi `go1.27.4` karena tambalan sudah ada pada cabang 1.27 yang lebih lama atau rentangnya tidak mencakup 1.27.x. Image runtime adalah `gcr.io/distroless/static-debian12:nonroot` yang hanya memuat 3 binary statis (`/bin/server`, `/bin/jobsched`, `/bin/migrate`); `usr/sbin/` kosong dan tidak ada `gosu` — temuan CVE lama pada `/usr/sbin/gosu` berasal dari image Alpine-lawas yang sudah tidak dibangun.
- **Bukti:** `backend/go.mod`, `backend/Dockerfile`, `go version -m` binary image (`go1.27.4`), `docker export` image (tanpa `/usr/sbin/gosu`), `govulncheck` bersih, `gofmt`/`vet`/`test`/`build` hijau.


### DEC-046 — Workflow migrasi Goose diperkuat (belum dijalankan di GitHub)

- **Tanggal:** 2026-11-12
- **Pemilik:** Pemilik platform (belum ditetapkan)
- **Status:** Disetujui (arah pengguna), file workflow diperkuat di repo dan urutan perintah diuji manual terhadap DB uji `simpeg_perf`
- **Rincian:** Workflow `.github/workflows/deploy-migrations.yml` belum pernah ada di repo remote (belum pernah di-commit/push), GitHub environment `staging`/`production` belum dibuat, dan secret `DATABASE_URL` belum ada, sehingga workflow belum pernah dijalankan. Workflow diperkuat: pakai runner migrasi milik repo (`backend/cmd/migrate` dengan goose v3.30.0 ter-pinning lewat `backend/go.mod`, bukan `go install ...@latest`), gagal cepat bila `DATABASE_URL` kosong, `timeout-minutes: 30`, grup concurrency, `migrate status` sebelum dan sesudah `migrate up`. Environment dan secret harus dibuat di GitHub sebelum workflow dapat dijalankan; penerapan harus dicoba di `staging` (DB uji) terlebih dahulu sebelum `production`.
- **Bukti:** `.github/workflows/deploy-migrations.yml`, `backend/cmd/migrate/main.go`, dan verifikasi urutan perintah manual terhadap `simpeg_perf` (status → up no-op versi 19 → status).


### DEC-047 — Manajemen dokumen pegawai/dosen melalui arsip dokumen + Cloudinary

- **Tanggal:** 2026-11-12
- **Pemilik:** Pemilik produk (belum ditetapkan)
- **Status:** Disetujui (arah pengguna)
- **Rincian:** Modul arsip dokumen pegawai (`ArsipDokumenPegawai`) sudah ada dan diperluas untuk menampung 11 jenis dokumen pegawai/dosen (KTP, NPWP, Kartu Keluarga, Rekening Bank, SK Pengangkatan Yayasan, SK Mendikbud, Pakta Integritas, Lainnya, Foto, Ijazah Terakhir, SK Kepegawaian/Kontrak, BPJS Ketenagakerjaan, BPJS Kesehatan, Sertifikat Serdos). Setiap jenis memiliki aturan format MIME dan ukuran maksimal yang divalidasi pada metadata berkas (backend tidak menangani unggah berkas; berkas diunggah langsung ke Cloudinary oleh klien, backend hanya menyimpan URL + metadata). Migrasi `000020_arsip_dokumen_tambahan.sql` menambah nilai enum baru pada `JenisDokumenArsip` dan kolom `fileSize`, `mimeType`, `cloudinaryPublicId` pada tabel `arsipDokumenPegawai`. Endpoint: `GET/POST/PATCH/DELETE /api/v1/arsip{...}`. Konfigurasi Cloudinary dibaca dari `CLOUDINARY_CLOUD_NAME`, `CLOUDINARY_API_KEY`, `CLOUDINARY_API_SECRET` (`.env.example`). `no_rek` dan `nama_rek` sudah tersedia sebagai `BankRekening` dan `BankAtasNama` pada tabel `user`. Timestamps `createdAt`/`updatedAt` disimpan UTC di database dan ditampilkan Asia/Jakarta.
- **Bukti:** `backend/migrations/000020_arsip_dokumen_tambahan.sql`, `backend/internal/models/models_hr.go`, `backend/internal/models/enums.go`, `backend/internal/directory/domain/arsip.go`, `backend/internal/directory/app/arsip.go`, `backend/internal/directory/repository/gorm_arsip.go`, `backend/internal/httpserver/hr_arsip_handlers.go`, `backend/internal/config/config.go` (`CloudinaryConfig`), `.env.example`, `docs/openapi.yaml`.


