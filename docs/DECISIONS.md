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
- **Bukti:** `backend/cmd/server/main.go` (matriks `auth.StaticPolicy`), `backend/tests/httpserver/scope_handlers_test.go`, `backend/tests/httpserver/organization_scope_test.go`

### DEC-004 — Cakupan operator adalah unit penugasan beserta turunannya

- **Tanggal:** 2026-11-11
- **Pemilik:** Pemilik produk (belum ditetapkan)
- **Status:** Disetujui
- **Rincian:** `docs/PRD.md` §11 "Matriks peran dan cakupan"
- **Bukti:** `backend/internal/directory/repository/gorm_catalog.go` (recursive CTE pohon `Homebase`), `backend/tests/directory/repository/list_test.go`, `backend/tests/httpserver/scope_handlers_test.go`

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
- **Bukti:** `backend/internal/auth/session_issuer.go`, `backend/tests/auth/session_issuer_test.go`, `backend/internal/httpserver/auth_handlers.go`, `backend/tests/httpserver/auth_handlers_test.go`, `backend/internal/httpserver/server.go` (`MuxOptions.SessionIssuer`), `backend/cmd/server/main.go`, `backend/internal/config/config.go` (`SESSION_LIFETIME`), `.env.example`

### DEC-007 — Migration runner Goose

- **Tanggal:** 2026-11-11
- **Pemilik:** Pemilik platform/DB (belum ditetapkan)
- **Status:** Disetujui
- **Rincian:** `docs/PRD.md` §11 "Draf migrasi, backup, audit, dan ketersediaan"
- **Bukti:** `backend/.goose.yaml`, `backend/migrations/`, `backend/tests/database/migrations_test.go`

### DEC-008 — Versi API memakai prefix path `/api/v1`

- **Tanggal:** 2026-11-11
- **Pemilik:** Pemilik arsitektur (belum ditetapkan)
- **Status:** Disetujui
- **Rincian:** `docs/PRD.md` §11 "Keputusan versi API"
- **Bukti:** `backend/internal/httpserver/version.go`, `docs/openapi.yaml`, `backend/tests/httpserver/version_test.go`

### DEC-009 — Bentuk respons error seragam

- **Tanggal:** 2026-11-11
- **Pemilik:** Pemilik arsitektur (belum ditetapkan)
- **Status:** Disetujui
- **Rincian:** `docs/PRD.md` §11 dan `agents/rules/architecture.md`
- **Bukti:** `backend/internal/apierror/`, `backend/internal/httpserver/errors.go`, `backend/tests/httpserver/health_test.go`

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
- **Bukti:** `backend/internal/timeutil/`, `backend/internal/database/postgres.go`, `backend/tests/database/postgres_test.go`

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
- **Catatan:** tanggal gajian bersifat tetap (1–28) per unit organisasi, diinput lewat `PUT /api/v1/payroll/schedules/{unitId}` dengan permission `payroll:schedule` (admin, seluruh unit) atau `payroll:schedule:homebase` (operator, unitnya beserta turunannya); nilai disimpan di tabel `jadwalGaji` (satu baris per unit). Pemberitahuan ketersediaan slip dikirim H+1 pada **tanggal kalender tetap** — tidak digeser karena akhir pekan atau hari libur nasional, dan tidak maju selama periode berjalan belum lewat jadwalnya. Jenis pesan outbox `email.slip-tersedia` dengan kunci idempotensi `slip-available:{slipId}`; pekerjaan dijalankan lewat job `payroll.notify` (`5 19 * * *` = 02:05 WIB) melalui RabbitMQ (self-hosted) atau QStash (cloud alternatif). Isi pesan hanya nama, periode, dan tautan — **tidak memuat nominal gaji dan tidak menyatakan gaji sudah dibayar**; itu pemberitahuan ketersediaan slip, bukan notifikasi gaji terbayar, yang tetap mensyaratkan konfirmasi pembayaran tersimpan.
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
- **Bukti:** `backend/internal/notify/smtp.go`, `backend/tests/notify/notify_test.go`, `backend/internal/notify/deliverer.go`, `backend/tests/config/config_test.go`, `.env.example`

### DEC-015 — Use case Redis: cache daftar kategori pegawai

- **Tanggal:** 2026-11-11
- **Pemilik:** Pemilik platform (belum ditetapkan)
- **Status:** Disetujui
- **Rincian:** `docs/PRD.md` §11 "Keputusan integrasi layanan pendukung"
- **Catatan:** use case yang spesifik adalah cache daftar kategori pegawai: data referensi dengan kardinalitas rendah, dibaca setiap kali formulir pegawai dibuka, dan tidak termasuk data sensitif. TTL 5 menit, satu namespace `simpeg`, dan `ReadThrough` memuat ulang dari PostgreSQL saat cache miss atau Redis gagal. **Queue tidak wajib untuk alur kritis** karena durability dijamin outbox di PostgreSQL, bukan oleh Redis. Kunci cache memuat seluruh parameter daftar (page, size, sort, order, pencarian) supaya tiap kombinasi filter punya entri sendiri. Cache sudah disambungkan ke `GET /api/v1/categories`; tanpa Redis konfigurasi pembacaan tetap langsung ke PostgreSQL.
- **Bukti:** `backend/internal/directory/app/catalog.go`, `backend/tests/directory/app/catalog_test.go`, `backend/internal/cache/`

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
- **Bukti:** `backend/internal/attendance/device_auth.go`, `backend/tests/attendance/device_auth_test.go`, `backend/internal/httpserver/attendance_handlers.go`, `backend/tests/httpserver/attendance_handlers_test.go`, `backend/migrations/000011_machine_credential.sql`, `backend/internal/models/models_operations.go`

### DEC-031 — Kebijakan idempotency key event presensi

- **Tanggal:** 2026-11-11
- **Pemilik:** Pemilik platform/DB (belum ditetapkan)
- **Status:** Disetujui
- **Rincian:** `docs/PRD.md` §11 "Keputusan fondasi presensi mesin". Idempotency key harus unik per pasangan `(machineId, tap event)`. Key dipanggil oleh adapter vendor dan dikirim dalam body request. Jika key sama digunakan dengan fakta tap yang berbeda (machinePin atau occurredAt berubah), server menolak dengan `409 Conflict`. Jika key sama dengan fakta tap yang sama, server mengembalikan event yang sama (`duplicate: true`) tanpa duplikasi.
- **Catatan:** Sumber idempotency key yang dijamin vendor belum ditetapkan; menunggu kontrak API/SDK vendor. Format yang disarankan: UUID v4.
- **Bukti:** `backend/internal/attendance/store.go`, `backend/tests/attendance/store_test.go`, `backend/internal/httpserver/attendance_handlers.go`, `backend/migrations/000002_attendance_events.sql` (unique index `(machineId, idempotencyKey)`)

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
- **Bukti:** `backend/tests/audit/audit_test.go` (penjaga append-only), `backend/internal/httpserver/audit_handlers.go` (`exportAuditLogs`), `docs/openapi.yaml`

### DEC-019 — Paparan Supabase Data API dan kebijakan RLS

- **Tanggal:** 2026-11-11
- **Pemilik:** Pemilik platform/DB (belum ditetapkan)
- **Status:** Disetujui
- **Rincian:** `docs/PRD.md` §8 (Backend dan Frontend): aplikasi mengakses database hanya melalui GORM di backend (koneksi server-side dengan TLS, role database berhak minimum); frontend berkomunikasi dengan backend lewat kontrak HTTP API, **bukan** langsung ke database/Supabase Data API.
- **Catatan:** Keputusan final: **GORM-only, Supabase Data API tidak diekspos** untuk tabel aplikasi, sehingga tidak diperlukan policy RLS per tabel pada waktu ini. RLS default-deny tetap aktif pada seluruh 45 tabel aplikasi sebagai pertahanan dalam-dalam (mengakapi kebocoran bila kebijakan berubah di masa depan). Jika Data API diekspos di masa depan, policy per tabel wajib dibuat dan diuji satu per tabel sebelum tabel diakses melalui Data API.
- **Bukti:** `docs/PRD.md` §8, `backend/migrations/000004_row_level_security.sql`, `backend/tests/database/migrations_test.go` (`TestRowLevelSecurityMigrationCoversEveryTable`)`

### DEC-020 — Target ketersediaan, RPO, RTO, dan SLO

- **Tanggal:** —
- **Pemilik:** Pemilik platform/DB dan pemilik operasi (belum ditetapkan)
- **Status:** Menunggu
- **Rincian:** `docs/PRD.md` §11 "Draf migrasi, backup, audit, dan ketersediaan"
- **Catatan:** default `RPO_HOURS` (24 jam), `RTO_HOURS` (4 jam), dan `SLO_TARGET` (`99.9`) sudah dapat dikonfigurasi lewat environment dan dibaca `config.Load()`, tetapi angka final tetap menunggu persetujuan pemilik platform/DB dan pemilik operasi sesuai tier layanan dan anggaran. Nilai awal 24 jam / 4 jam telah diratifikasi pengguna pada 13 November 2026 (`DEC-050`) — itu ratifikasi nilai awal, bukan penetapan target produksi; final tetap menunggu.
- **Bukti:** `backend/internal/config/`, `backend/tests/config/config_test.go`

### DEC-033 — Workflow deployment migrasi Goose

- **Tanggal:** 2026-11-11
- **Pemilik:** Pemilik platform/DB (belum ditetapkan)
- **Status:** Disetujui
- **Rincian:** `docs/PRD.md` §8 (Backend) dan `docs/RUNBOOK.md` §7 (Pemulihan setelah penerapan migrasi). Workflow GitHub Actions `.github/workflows/deploy-migrations.yml` dibuat untuk menjalankan migrasi Goose secara terkendali. Workflow dipicu secara manual (`workflow_dispatch`) dengan pilihan environment `staging` atau `production`, bukan berjalan otomatis pada setiap push. Setiap jalankan memverifikasi format, vet, tes, paritas skema, nomor migrasi urut, dan header migrasi sebelum menjalankan `goose status` lalu `goose up` terhadap database target. `DATABASE_URL` disimpan sebagai secret GitHub, tidak ada di repo.
- **Catatan:** Workflow siap digunakan, tetapi migrasi masih belum pernah diterapkan pada database nyata. Sebelum menjalankan di production, operator wajib mencoba di staging dengan data sintetis dan memastikan backup/restore sudah diuji (lihat `docs/RUNBOOK.md` §7).
- **Bukti:** `.github/workflows/deploy-migrations.yml`, `backend/.goose.yaml`, `backend/migrations/`, `backend/tests/database/migrations_test.go`, `docs/RUNBOOK.md` §7

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
- **Bukti:** `backend/internal/qstash/publisher.go`, `backend/internal/qstash/schedule.go`, `backend/tests/qstash/publisher_endpoint_test.go`, `docs/PERFORMANCE-BASELINE.md` §5

### DEC-039 — Dukungan STARTTLS pada transport SMTP

- **Tanggal:** 2026-11-12
- **Pemilik:** Pemilik platform (belum ditetapkan)
- **Status:** Disetujui
- **Rincian:** `notify.smtpTransport` hanya knewa dua mode: TLS langsung (`UseTLS`) dan koneksi polos tanpa peningkatan. Server submission pada porta 587 menerima koneksi polos lalu menawarkan `STARTTLS`. Dengan `SMTP_HOST=smtp.gmail.com` dan `SMTP_PORT=587`, mode TLS langsung gagal dengan `tls: first record does not look like a TLS handshake`, sehingga tidak ada email yang pernah terkirim.
- **Catatan:** Transport kini menaikkan koneksi ke TLS otomatis bila server menawarkan `STARTTLS`, dan peningkatan dilakukan sebelum autentikasi karena kredensial tidak boleh dikirim lewat koneksi polos. Penentuan mode dipindah ke `config.SMTPConfig`: `SMTP_SECURE=true` berarti TLS langsung, `SMTP_SECURE=false` berarti koneksi polos yang dinaikkan lewat STARTTLS, dan bila `SMTP_SECURE` tidak disetel, mode ditebak dari porta (465 memakai TLS langsung, porta lain memakai STARTTLS). Nilai bawaan lama `SMTP_SECURE=true` tidak lagi berlaku karena tidak sesuai dengan porta submission.
- **Bukti:** `backend/internal/notify/smtp.go`, `backend/internal/config/smtp.go`, `backend/tests/notify/smtp_starttls_test.go`, `docs/PERFORMANCE-BASELINE.md` §5

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
- **Bukti:** `backend/internal/notify/address.go`, `backend/internal/notify/smtp.go`, `backend/internal/config/config.go`, `backend/tests/notify/address_test.go`, `docs/PERFORMANCE-BASELINE.md` §8

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
- **Bukti:** `backend/migrations/000019_cuti_eviden.sql`, `backend/internal/directory/domain/leave_eviden.go`, `backend/internal/directory/app/leave.go`, `backend/internal/httpserver/hr_leave_handlers.go`, dan tes di `backend/tests/httpserver/hr_leave_handlers_test.go` serta `backend/tests/directory/domain/leave_eviden_test.go`; kontrak di `docs/openapi.yaml`; kebijakan di `docs/PRD.md` §11.


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


### DEC-048 — Fondasi frontend: sesi token di memori, satu lockfile Bun, dan gerbang mutu frontend

- **Tanggal:** 2026-11-12
- **Pemilik:** Pemilik platform (belum ditetapkan)
- **Status:** Disetujui (arah pengguna)
- **Rincian:** Frontend SIMPEG dibuat di `frontend/` memakai TanStack Start + shadcn/ui dan diintegrasikan ke repo utama (repo Git nested yang kosong di `frontend/` dihapus). Tiga keputusan diambil. (1) Package manager hanya Bun: `frontend/bun.lock` menjadi satu-satunya lockfile; `frontend/package-lock.json` yang tertinggal dari scaffolding awal dihapus karena aturan repo melarang lockfile lain. (2) Sesi dibawa lewat header `Authorization: Bearer <token>` sesuai skema `bearerSession` pada `docs/openapi.yaml`. Token hanya hidup di memori tab (`frontend/src/lib/auth-token.ts`, TanStack Store) dan tidak pernah ditulis ke `localStorage` atau `sessionStorage`, sesuai PRD §11. Request browser ke backend memakai `credentials: 'omit'` karena middleware CORS backend hanya mengirim `Access-Control-Allow-Origin` tanpa `Access-Control-Allow-Credentials`, sehingga request lintas origin dengan cookie akan ditolak browser. Bila deployment nanti memakai cookie lintas origin, backend harus menambah `Access-Control-Allow-Credentials` dan frontend harus beralih ke `credentials: 'include'`. (3) Backend Go tidak menambah endpoint pencatatan masuk; pencatatan masuk tetap milik runtime Better Auth dan dipanggil frontend lewat `VITE_AUTH_BASE_URL`. Halaman masuk menonaktifkan formulir bila runtime belum dikonfigurasi, bukan mengirim permintaan yang pasti gagal. Quality tooling dipasang sejak awal (Vitest + jsdom + Testing Library, ESLint flat config, Prettier, Playwright, Zod) dan dijalankan sebagai gerbang CI pada `.github/workflows/frontend.yml`. Keterbatasan lingkungan dicatat: Vitest 5 menjalankan worker jsdom dengan runtime Node karena runtime Bun 1.4.2 belum dapat menjalankannya (galat `EventTarget` lintas realm). Judul halaman memakai elemen heading semantik karena `CardTitle` shadcn merender `div`.
- **Bukti:** `frontend/package.json`, `frontend/bun.lock`, `frontend/src/lib/api.ts`, `frontend/src/lib/auth-token.ts`, `frontend/src/lib/session.ts`, `frontend/src/lib/require-auth.tsx`, `frontend/src/features/auth/login-form.tsx`, `frontend/vite.config.ts`, `frontend/eslint.config.js`, `frontend/vitest.config.ts`, `frontend/playwright.config.ts`, `frontend/README.md`, `.github/workflows/frontend.yml`, `backend/internal/httpserver/security.go` (CORS), `docs/openapi.yaml` (`bearerSession`).

### DEC-049 — Runtime autentikasi tertanam di frontend dan pengalaman publik SIMPEG

- **Tanggal:** 2026-11-12
- **Pemilik:** Pemilik platform (belum ditetapkan)
- **Status:** Disetujui (arah pengguna)
- **Rincian:** Runtime Better Auth dipasang tertanam di server TanStack Start pada `/api/auth/*` (`frontend/src/lib/server/better-auth.ts`, catch-all `frontend/src/routes/api/auth/$.tsx`). Konfigurasi: `pg.Pool` langsung sebagai database adapter (Kysely tidak dipakai), `advanced.database.generateId: () => crypto.randomUUID()` karena tabel auth tidak memiliki DEFAULT pada primary key, `disableSignUp: true` (pendaftaran publik dimatikan; akun dibuat administrator lewat `frontend/scripts/provision-accounts.ts` terhadap database uji sekali pakai di Docker), sesi 7 hari dengan rolling refresh 1 hari, `user.fields.name` dipetakan ke kolom `nama`, `sameSite: 'lax'`/`httpOnly`/`secure` + `useSecureCookies` di produksi, `trustedOrigins` dari `TRUSTED_ORIGINS`, dan plugin `tanstackStartCookies()`. Browser menyimpan token sesi hanya di memori tab dan mengirimnya ke Go API sebagai `Authorization: Bearer <token>`; Go memverifikasi lewat resolver GORM `session`. `/api/v1/me` tidak dipanggil bila tidak ada token di memori (hasil pasti 401 dan hanya membisingkan konsol). `POST /api/auth/sign-out` Better Auth menolak permintaan tanpa `Content-Type: application/json` dan body JSON (`415`); pemanggilan kini mengirim `{}` dengan `Content-Type: application/json`. Pengembangan harian memakai proksi Vite same-origin: `VITE_API_BASE_URL=/api/v1` dan `VITE_API_PROXY_TARGET=http://localhost:8080` pada environment proses dev (`server.proxy` di `frontend/vite.config.ts`), sehingga browser tidak membutuhkan CORS lintas origin. Pengalaman publik dibangun di atasnya: halaman depan publik, header (tombol Masuk saat keluar; avatar inisial + dropdown dengan Ruang kerja/Keluar saat masuk), footer tiga kolom, saklar bahasa Indonesia/English (default `id`) berbasis kamus, tema light bawaan dengan toggle dark (`useSyncExternalStore`), sonner untuk umpan balik masuk/keluar, dan halaman About, Privacy Policy, Terms of Service, Terms of Reference, serta komponen error/loading/not-found global. Seluruh gerbang mutu frontend hijau (format, lint nol galat nol peringatan, `tsc --noEmit`, 27 tes Vitest, build produksi, 12 tes Playwright, dan asap browser langsung: pegawai dan admin masuk, mendarat di `/workspace`, lalu keluar tanpa error konsol). `DESIGN.md` disusun sebagai draf yang **menunggu persetujuan pemilik produk**. Verifikasi memakai database disposable `simpeg-verify-pg` (postgres:16-alpine, porta 55432) dengan migrasi Goose 000001–000020; bukan Supabase production.
- **Catatan:** Skrip `provision-accounts.ts` mencetak kata sandi sekali pakai dan hanya boleh dijalankan terhadap database uji yang memang sekali pakai; kredensial verifikasi tidak pernah masuk repo dan tidak dicatat di sini. Akun produksi tetap dibuat administrator lewat proses yang disepakati, bukan lewat pendaftaran publik. Klaim "seluruh karyawan dapat masuk" belum terbukti di produksi — yang terverifikasi adalah bahwa jalur masuk/keluar dan kontrak sesi bekerja untuk keempat role pada database uji. Audit visual antislop masih menunggu (lihat `IMPLEMENTATION-PLAN.md`).
- **Bukti:** `frontend/src/lib/server/better-auth.ts`, `frontend/src/routes/api/auth/$.tsx`, `frontend/scripts/provision-accounts.ts`, `frontend/src/lib/session.ts`, `frontend/src/lib/auth-token.ts`, `frontend/vite.config.ts`, `frontend/.env.example`, `frontend/src/routes/index.tsx`, `frontend/src/routes/{about,privacy,terms,terms-of-reference}.tsx`, `frontend/src/components/{Header,Footer,ThemeToggle,AppToaster,GlobalError,GlobalLoading,GlobalNotFound}.tsx`, `frontend/src/i18n/`, `frontend/src/test/`, `frontend/e2e/app.spec.ts`, `DESIGN.md`, `agents/rules/{project,architecture,backend,frontend}.md`.

### DEC-050 — RPO/RTO awal (24 jam / 4 jam), drill restore lokal, dasbor SLO internal, dan log auth_failed

- **Tanggal:** 2026-11-13
- **Pemilik:** Pemilik produk/operasi (ratifikasi awal oleh pengguna, belum ada penetapan resmi)
- **Status:** Disetujui (arah pengguna)
- **Rincian:** Empat keputusan observability/pemulihan. (1) **RPO/RTO awal diratifikasi 24 jam / 4 jam**, sejalan dengan bawaan `RPO_HOURS`/`RTO_HOURS` (dicocokkan pada 13 November 2026); keduanya baru nilai awal, belum target produksi yang diukur. (2) **Drill backup/restore dijalankan pada database disposable lokal** (`simpeg-verify-pg`, postgres:16-alpine, bukan Supabase production): dump 134.982 byte dalam 0,35 detik; restore ke database `simpeg_restore_drill` dalam 1,36 detik tanpa galat; perbandingan skema hasil restore vs sumber identik (Goose 20 versi, 69 tabel, 181 indeks, 167 constraint, 68 tabel dengan RLS aktif, `user`=4, `session`=8, `account`=4, `auditLog`=8); database drill dihapus setelah verifikasi. PITR (point-in-time recovery) dan restore pada database target produksi **belum pernah diuji**. (3) **Dasbor SLO internal** dibangun sebagai halaman admin di frontend (`/admin/slo`) dengan sumber data `sloMetric` lewat endpoint baru read-only `GET /api/v1/admin/slo/metrics` (permission baru `slo:read`, admin saja, deny-by-default); kontrak di `docs/openapi.yaml`. Cakupan: laju galat dan latensi (rata-rata, p95, p99, maks) per rute + agregat, pada jendela 24 jam atau 7 hari. **Tidak** mencakup saturasi, health dependency, keterlambatan antrean, atau kegagalan notifikasi; **aturan alert belum terpasang pada infrastruktur mana pun** — bagian 7 dan A09 tetap terbuka untuk itu. Selama pengoleki metrik (`SLO_ENABLED`) belum menulis data, halaman menampilkan tabel kosong (respons 200 dengan total nol), bukan klaim "tidak ada lalu lintas". (4) **Login gagal pada lapisan runtime Better Auth (frontend) kini dapat diamati**: hook `after` pada konfigurasi Better Auth menulis satu baris JSON `{ event: 'auth_failed', path, reason, ts }` ke stdout; `reason` hanya kode yang dikenali (huruf/angka/`_.-`, maksimum 64 karakter) atau `http_<status>`; tidak pernah berisi body formulir, email, atau kata sandi. Kegagalan sign-in email terverifikasi pada database uji: respons publik tetap seragam (`401 INVALID_EMAIL_OR_PASSWORD`), sedangkan log internal mencatat alasan spesifik (`reason=INVALID_EMAIL` untuk email tidak dikenal, `reason=INVALID_PASSWORD` untuk kata sandi salah). Ini **bukan** alert dan **bukan** pengganti SIEM; eskalasi otomatis belum ada.
- **Bukti:** `backend/internal/slo/app/dashboard.go`, `backend/internal/httpserver/slo_dashboard_handlers.go` (+`backend/tests/httpserver/slo_dashboard_handlers_test.go`), `backend/cmd/server/main.go`, `backend/cmd/server/policy.go`, `frontend/src/components/SloMetricsPage.tsx`, `frontend/src/lib/server/better-auth.ts` (`logAuthFailure`), `frontend/src/test/auth-failure-logger.test.ts`, `frontend/src/test/slo-metrics-page.test.tsx`, `frontend/e2e/app.spec.ts`, `docs/RUNBOOK.md` §8E/§9, `docs/openapi.yaml`, `docs/PRD.md` §11.

### DEC-051 — Atasan melihat kehadiran timnya (agregat + detail per pegawai); `note` terlihat dalam cakupan

- **Tanggal:** 2026-11-13
- **Pemilik:** Pemilik produk (arah langsung dari pengguna, 13 November 2026)
- **Status:** Disetujui (arah pengguna)
- **Rincian:** Atasan (`atasan`, termasuk rektor/pimpinan yang ditugaskan pada akar pohon `Homebase`) dan operator melihat kehadiran timnya dalam dua kedalaman, dibatasi cakupan yang sama dengan `attendance:read:homebase` (unit penugasan beserta seluruh turunannya); admin memakai `attendance:read:any` (lintas organisasi): (1) agregat per tanggal (`H/I/S/A/C/LD`) lewat `GET /api/v1/attendance/unit-summary` — sudah ada; (2) detail per pegawai untuk satu tanggal lewat `GET /api/v1/attendance/unit-records?date=YYYY-MM-DD` — semua pegawai **aktif** dalam cakupan ditampilkan (LEFT JOIN ke `Presensi`); pegawai tanpa catatan pada tanggal itu tampil dengan `status` null; artinya "belum tercatat", **bukan** alpa — aturan penurunan status harian belum disetujui. Keterangan (`note`/`keterangan`) presensi **boleh terlihat** oleh atasan yang memegang cakupan (mis. alasan sakit/izin untuk keputusan operasional); karena bisa memuat informasi kesehatan/pribadi, `note` tidak pernah terlihat di luar cakupan atasan (lintas unit hanya lewat `attendance:read:any`, admin). Endpoint **tidak** menambah permission baru; memakai `attendance:read:any` atau `attendance:read:homebase` yang sudah ada. Endpoint tidak menurunkan status dari `AttendanceEvent`; PIN mesin belum dipetakan ke pegawai.
- **Bukti:** `backend/internal/attendance/{store.go, read.go, gorm_read.go}`, `backend/tests/attendance/read_test.go`, `backend/internal/httpserver/attendance_records_handlers.go` (+`backend/tests/httpserver/attendance_records_handlers_test.go`), `docs/PRD.md` §11, `docs/openapi.yaml`, `frontend/src/routes/_app/attendance/unit-records.tsx`, `frontend/src/components/TeamAttendanceTable.tsx`.

### DEC-052 — NIK, NPWP, nomor ijazah, NIP, NIDN/NIDK unik di level database

- **Tanggal:** 2026-11-13
- **Pemilik:** Pemilik produk (arah langsung dari pengguna, 13 November 2026)
- **Status:** Disetujui (arah pengguna)
- **Rincian:** NIK, NIP, NIDN, dan NIDK sudah unik (`000001_directory_crud.sql`; NIDN/NIDK di migrasi `000012_nidn_nidk_user.sql`), masing-masing indeks unik parsial `WHERE … IS NOT NULL` bila nullable. Migrasi `000021_npwp_no_ijazah_unique.sql` melengkapi: (1) `user.npwp` dijadikan unik lewat indeks parsial `user_npwp_idx`; string kosong/spasi dinormalkan ke NULL lebih dulu agar tidak menghalangi pembuatan indeks (nilai blank tidak membawa informasi; normalisasi tidak dapat dibalik pada `Down`). (2) Nomor ijazah ditambahkan pada `RiwayatPendidikan."noIjazah"` (kolom baru, `text`, nullable) dengan indeks unik parsial `riwayat_pendidikan_no_ijazah_idx` — uniknya di level **baris**, bukan per pegawai, karena satu pegawai bisa punya beberapa ijazah (S1/S2/S3). Penegakan ada di level database/Goose; API dan DTO saat ini belum mengekspos `npwp` maupun `noIjazah`, jadi tidak ada perubahan kontrak HTTP. Diterapkan pada database Supabase `postgres` dan `simpeg_perf` (13 November 2026, versi Goose 21); pada `simpeg_perf` migrasi diterapkan setelah pra-pemeriksaan read-only (5000 baris user seed sintetis; NPWP 0 terisi/kosong/duplikat, 0 baris `RiwayatPendidikan`), dan verifikasi pasca-migrasi mengonfirmasi indeks `user_npwp_idx`, indeks `riwayat_pendidikan_no_ijazah_idx`, serta kolom `RiwayatPendidikan."noIjazah"` terpasang; `perfbench schema` menyatakan 68 model selaras pada kedua DB. Bila data target nanti berisi NPWP terduplikasi, migrasi gagal dan duplikat harus diselesaikan pemilik data — nilai yang benar tidak boleh ditebak.
- **Bukti:** `backend/migrations/000021_npwp_no_ijazah_unique.sql`, `backend/internal/models/models.go`, `backend/tests/database/migration_schema_test.go`, `docs/IMPLEMENTATION-PLAN.md`.

### DEC-053 — Kolom identitas PII dienkripsi pada level aplikasi (AES-256-GCM + blind index HMAC)

- **Tanggal:** 2026-11-13
- **Pemilik:** Pemilik produk (arah langsung dari pengguna, 13 November 2026)
- **Status:** Disetujui (arah pengguna)
- **Rincian:** Kolom `user.nik`, `user.nip`, `user.npwp`, dan `RiwayatPendidikan.noIjazah` disimpan terenkripsi pada level aplikasi, **bukan** pada level database/transparent encryption. Format nilai: `enc1:` + base64url(nonce 12 byte ‖ AES-256-GCM(ciphertext ‖ tag)); nonce dibangkitkan acak pada setiap enkripsi sehingga kesamaan plaintext tidak terlihat di database. Untuk keunikan dan kesetaraan (pencarian), setiap kolom plaintext berpasangan dengan kolom blind index `nikHash`/`nipHash`/`npwpHash` (`user`) dan `noIjazahHash` (`RiwayatPendidikan`) berisi HMAC-SHA256 atas bentuk ternormalisasi dari nilai (trim, huruf besar, tanpa spasi/tab/titik/strip/garis bawah). Indeks unik parsial `WHERE "...Hash" IS NOT NULL` pada kolom-kolom itu menggantikan unique index plaintext lama yang dijatuhkan migrasi `000022_pii_identity_encrypt.sql` (`user_nik_key`, `user_nip_key`, `user_npwp_idx`, `riwayat_pendidikan_no_ijazah_idx`). Pencarian pegawai pada NIP menjadi **kesetaraan persis** lewat blind index (bukan substring `ILIKE`); substring `ILIKE` hanya berlaku untuk `nama`. Pengurutan kolom `nip` dihapus dari allowlist (`EMPLOYEE_SORT_FIELDS` di `frontend/src/lib/employees.ts`, `domain.EmployeeSortColumns` di `backend/internal/directory/domain/list.go`); sort `nama`/`createdAt`/`updatedAt` tetap. Kunci berasal dari environment `PII_ENCRYPTION_KEY` (base64, tepat 32 byte / AES-256); kunci blind index diturunkan dari kunci yang sama (HMAC dengan label domain `simpeg/pii-blind-index/v1`), sehingga hanya satu rahasia yang perlu dikelola. Production menolak start bila `PII_ENCRYPTION_KEY` kosong atau tidak valid; di luar production, jalur PII gagal dengan galat eksplisit. `cmd/perfbench seed` mewajibkan kunci yang sama. Enkripsi dan dekripsi terjadi di driver.Valuer/Scanner (`backend/internal/crypto`), sehingga kontrak HTTP tidak berubah: backend mendekripsi sebelum mengirim respons dan mengenkripsi saat menyimpan; plaintext tidak pernah ditulis ke log. Nilai pra-migrasi (plaintext) akan ditolak saat dibaca (prefix wajib `enc1:`) — fail-closed, sengaja. Migrasi `000022` tidak mengenkripsi data lama: database produksi `postgres` kosong; database uji `simpeg_perf` dikosongkan dan diisi ulang (`perfbench seed`: 5000 pegawai sintetis, 2500 dengan NIP, semuanya ciphertext, `nipHash` unik semua). Kehilangan `PII_ENCRYPTION_KEY` berarti data PII tidak dapat dibaca; salinan cadangan wajib dibuat sebelum kunci dipakai untuk data nyata, dan rotasi kunci belum didukung (perubahan kunci harus disertai backfill ulang seluruh kolom PII). Blind index membocorkan **kesetaraan** nilai kepada siapa pun yang memegang database (itu tujuannya) dan tidak melindungi dari penebakan berbasis kamus untuk ruang nilai kecil — keterbatasan inheren blind index, bukan cacat implementasi; pembatasan akses baca kolom PII tetap lewat matriks permission `docs/PRD.md` §11. `KeluargaPegawai.nik` (identitas anggota keluarga/tanggungan) awalnya di luar cakupan; cakupannya diperluas pada `DEC-054` (13 November 2026). Diterapkan 13 November 2026: kedua database pada Goose 22; `perfbench schema` menyatakan 68 model selaras pada keduanya; pada `simpeg_perf` 0 plaintext tersisa (2500 ciphertext, 2500 `nipHash` unik).
- **Bukti:** `backend/internal/crypto/pii.go` (+`backend/tests/crypto/pii_test.go`), `backend/internal/models/models.go`, `backend/migrations/000022_pii_identity_encrypt.sql`, `backend/internal/config/pii.go`, `backend/internal/config/config.go`, `backend/cmd/server/main.go`, `backend/cmd/perfbench/seed.go` (+`seed_data.go`), `backend/internal/directory/{domain/list.go, app/employee.go, repository/{gorm_employee.go, gorm_import.go, identity_hash.go}}`, `backend/tests/directory/repository/{main_test.go, gorm_store_test.go, gorm_import_test.go, list_test.go}`, `backend/internal/attendance/{store.go, gorm_read.go}`, `backend/internal/httpserver/{directory_types.go, employee_import_parse.go, attendance_records_handlers.go}`, `frontend/src/lib/employees.ts`, `frontend/src/components/EmployeeTable.tsx`, `docs/PRD.md`, `docs/IMPLEMENTATION-PLAN.md`.

### DEC-054 — KeluargaPegawai.nik ikut dienkripsi (memperluas DEC-053)

- **Tanggal:** 2026-11-13
- **Pemilik:** Pemilik produk (arah langsung dari pengguna, 13 November 2026)
- **Status:** Disetujui (arah pengguna)
- **Rincian:** Kolom `KeluargaPegawai.nik` (identitas suami/istri/anak yang ditanggung pegawai, dipakai administrasi BPJS dan tunjangan keluarga) sebelumnya di luar cakupan DEC-053 karena permintaan awal hanya menyebut empat kolom. Atas penegasan pengguna 13 November 2026, kolom ini dienkripsi dengan mekanisme yang sama: nilai tersimpan `enc1:` + base64url(nonce acak 12 byte ‖ AES-256-GCM(ciphertext ‖ tag)), berpasangan dengan blind index HMAC-SHA256 `keluargaPegawai."nikHash"` untuk pencarian **kesetaraan persis** (setelah normalisasi); tidak ada keunikan yang ditegakkan (NIK anggota keluarga tidak diwajibkan unik lintas keluarga), sehingga indeks `nikHash` bersifat non-unik (`keluarga_pegawai_nik_hash_idx`) menggantikan indeks plaintext `keluarga_pegawai_nik_idx` yang dijatuhkan migrasi `000023_keluarga_nik_encrypt.sql`. Pencarian daftar keluarga pada NIK menjadi kesetaraan persis; substring `ILIKE` hanya berlaku untuk `nama`; sort `nik` dihapus dari `domain.KeluargaSortColumns`. NIK keluarga bersifat **wajib** pada formulir (domain `CreateKeluargaPegawai.Validate` menolak kosong/null), sehingga kolom `nik` tetap `NOT NULL` dan patch tidak dapat mengosongkannya (`KeluargaPatch.NIK.Null` ditolak validasi). Enkripsi/dekripsi terjadi di driver.Valuer/Scanner (`backend/internal/crypto`); kontrak HTTP `/api/v1/keluarga` tidak berubah (backend mendekripsi sebelum mengirim). Kedua database (`postgres`, `simpeg_perf`) diverifikasi tidak memiliki baris keluarga saat migrasi diterapkan, sehingga tidak ada data plaintext tertinggal dan tidak ada backfill; migrasi tidak mengenkripsi data lama. Perbaikan kunci sama seperti DEC-053 (`PII_ENCRYPTION_KEY`, wajib pada production); rotasi kunci belum didukung. Diterapkan 13 November 2026: kedua database pada Goose 23; `perfbench schema` menyatakan 68 model selaras pada keduanya.
- **Bukti:** `backend/migrations/000023_keluarga_nik_encrypt.sql`, `backend/internal/models/models_hr.go`, `backend/internal/directory/{domain/list.go, app/keluarga.go, repository/{gorm_keluarga.go, identity_hash.go}}`, `backend/internal/httpserver/hr_keluarga_handlers.go`, `docs/PRD.md`, `docs/IMPLEMENTATION-PLAN.md`.

### DEC-055 — Endpoint agregat ringkasan pegawai `GET /api/v1/employees/summary`

- **Tanggal:** 2026-11-13
- **Pemilik:** Pemilik produk (arah pengguna 13 November 2026 menyetujui penambahan endpoint; definisi ringkasan belum diratifikasi secara tertulis)
- **Status:** Disetujui (arah pengguna)
- **Rincian:** PRD FR-05 meminta ringkasan pegawai (jumlah per status dan unit organisasi), sepanjang definisi status serta izin akses telah disepakati. Ringkasan sebelumnya dihitung client-side dengan membaca seluruh halaman daftar (`pageSize=100` berulang), yang tidak cukup untuk dataset besar. Endpoint read-only baru `GET /api/v1/employees/summary` mengembalikan `{total, active, inactive, byUnit:[{unitId,count}]}` dengan cakupan identik daftar pegawai: `employee:read:any` (lintas organisasi), `employee:read:homebase` (unit penugasan beserta turunannya), atau `employee:read:self` (hanya record sendiri). Cakupan dipaksa di lapisan aplikasi (`EmployeeService.Summary`), bukan di transport; deny-by-default untuk permission lain. **Populasi yang diringkas adalah seluruh baris pegawai dalam cakupan, termasuk baris nonaktif** (`user.aktif = false`) — perlu diratifikasi pemilik produk sebagai definisi "status"; bila definisi disepakati lain (mis. hanya aktif), endpoint disesuaikan. Pegawai tanpa unit dikelompokkan dengan `unitId` null. `unitName` sengaja tidak dikemas di endpoint; frontend memetakan `unitId` ke nama lewat daftar organisasi. Kontrak di `docs/openapi.yaml` (`EmployeeSummary`). Sisi frontend, `/dashboard` membaca endpoint ini (`lib/dashboard.ts`). Endpoint tidak menambah permission baru; memakai permission baca yang sudah ada.
- **Bukti:** `backend/internal/directory/{domain/summary.go, app/employee.go, repository/{repository.go, gorm_employee.go}}`, `backend/internal/httpserver/{directory_api.go, employee_handlers.go, directory_types.go}`, `backend/tests/httpserver/{directory_handlers_test.go, load_test.go}`, `docs/openapi.yaml`, `docs/IMPLEMENTATION-PLAN.md`, `frontend/src/lib/dashboard.ts`, `frontend/src/validations/schemas.ts`, `frontend/src/components/DashboardPage.tsx`.

### DEC-056 — Perbaikan temuan keamanan F-03 sampai F-08 dan sinkronisasi kontrak OpenAPI

- **Tanggal:** 2026-11-13
- **Pemilik:** Pemilik produk/platform (arah pengguna 13 November 2026 meminta penutupan F-03 sampai F-08, terutama untuk target MVP)
- **Status:** Disetujui (arah pengguna)
- **Rincian:** (1) **F-03:** sembilan rute terdokumentasi menjawab `404` karena struct `directoryAPI` tidak diisi field `keluarga`, `golongan`, `arsip`, `sertifikasi`, `pelatihan`, `sister`, `penghargaan`, `tunjangan`, `potongan` di `RegisterDirectoryRoutes`; seluruh field kini diisi sehingga seluruh rute terdaftar. (2) **F-04:** server dev dan preview frontend mengirim `Content-Security-Policy` enforcing (`default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'`), `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, `Referrer-Policy: no-referrer`; CSP ditegakkan pada dev/preview, sedangkan header pada lapisan deployment produksi menunggu keputusan deployment. (3) **F-05:** kontrak diperbaiki (referensi `CutiDTO`/`ArsipDTO` menunjuk `Cuti`/`Arsip`; parameter `UserID` dan `ArsipId` dideklarasikan; path `DELETE /api/v1/arsip/{arsipId}/archive` digabung menjadi `DELETE /api/v1/arsip/{arsipId}` sesuai pola `DELETE /employees/{employeeId}`); seluruh `$ref` resolve dan tidak ada operasi terdokumentasi yang tidak terdaftar. Sisa drift: 94 penanganan rute terdaftar (modul SKP, tridharma, sebagian absensi internal) belum terdokumentasi; ini didokumentasikan, bukan disembunyikan. (4) **F-06:** setiap respons API mengirim `Content-Security-Policy: default-src 'none'; frame-ancestors 'none'; base-uri 'none'`. (5) **F-08:** validasi identifier asing di lapisan transport (`requireUUID`/`requireOptionalUUID`) menolak identifier bukan UUID dengan `422`; mencakup `tridharma` (5), `skp` (3), modul `hr` (10), pegawai (create/patch), organisasi (create/patch), dan cuti. Verifikasi: `go build`, `go vet`, `go test ./...` lulus (termasuk `backend/tests/httpserver/foreign_id_validation_test.go` dan tes CSP; lihat `backend/tests/httpserver/security_test.go`); tidak diverifikasi terhadap database produksi dan tidak diuji ulang dinamis dengan ZAP.
- **Bukti:** `backend/internal/httpserver/{directory_api.go, directory_types.go, employee_handlers.go, organization_handlers.go, security.go}`, `backend/tests/httpserver/{security_test.go, foreign_id_validation_test.go}`, `frontend/vite.config.ts`, `docs/openapi.yaml`, `docs/IMPLEMENTATION-PLAN.md`, `docs/SECURITY-TEST-OWASP-2025.md` §11, PRD §11.

### DEC-057 — Master jabatan sebagai entitas (`Jabatan`)

- **Tanggal:** 2026-11-13
- **Pemilik:** Pemilik produk (arah pengguna 13 November 2026 menyetujui entitas baru beserta permission `jabatan:read` untuk semua role dan `jabatan:manage` untuk admin)
- **Status:** Disetujui (arah pengguna)
- **Rincian:** PRD FR-03 meminta data jabatan dapat dikelola dan penempatan pegawai hanya memakai jabatan yang valid. Entitas baru `Jabatan`: `id` uuid, `nama` unik (indeks unik), `jenis` enum `STRUKTURAL`/`FUNGSIONAL` (default `STRUKTURAL`), `aktif` boolean, `createdAt`/`updatedAt`. Migrasi Goose `000024_jabatan.sql` (type `JabatanJenis`); RLS default deny tanpa policy (jabatan adalah data referensi; backend sebagai pemilik tabel tetap dapat mengakses). Model GORM `models.Jabatan` terdaftar di `models.All()` dan `postgresTableNames`. Endpoint: `GET /api/v1/jabatan` (pencarian/sort `nama|jenis|createdAt|updatedAt`, filter `jenis`, paginasi), `POST /api/v1/jabatan`, `GET/PATCH/DELETE /api/v1/jabatan/{jabatanId}` — baca memakai `jabatan:read` (semua role terautentikasi), tulis memakai `jabatan:manage` (admin saja); arsip bersifat satu arah (konsisten dengan kategori/organisasi). Penempatan pegawai pada jabatan **belum** dipindahkan ke referensi ini: `user.jabatanStruktural/jabatanFungsional` tetap teks bebas sampai ada keputusan produk. Kontrak di `docs/openapi.yaml` (`JabatanPage`). Frontend: halaman `/jabatan` (daftar, pencarian, filter jenis, sort, page size 10/25/50/100, paginasi; dialog tambah/ubah/arsip khusus admin), item sidebar untuk semua role. Migrasi `000024_jabatan.sql` sudah diterapkan pada database disposable lokal `simpeg_verify` (migrasi 000001–000024, 2026-11-13) — `GET /api/v1/jabatan` menjawab 200 dengan daftar kosong. Pada database produksi migrasi **belum** diterapkan (23 versi Goose per 2026-11-13); workflow `.github/workflows/deploy-migrations.yml` menerapkannya setelah persetujuan pemilik (`DEC-033`); verifikasi skema (`perfbench schema`) belum dijalankan.
- **Bukti:** `backend/migrations/000024_jabatan.sql`, `backend/internal/models/{models.go, enums.go, registry.go}`, `backend/internal/database/postgres.go`, `backend/internal/directory/{domain/jabatan.go, domain/list.go, app/jabatan.go, repository/{repository.go, gorm_jabatan.go}}`, `backend/internal/httpserver/{directory_api.go, directory_types.go, jabatan_handlers.go}`, `backend/tests/httpserver/directory_handlers_test.go`, `backend/cmd/server/{main.go, policy.go}`, `frontend/src/lib/jabatan.ts`, `frontend/src/components/{JabatanPage, JabatanTable, JabatanDialogs}.tsx`, `frontend/src/routes/_app/jabatan.tsx`, `frontend/src/components/Sidebar.tsx`, `frontend/src/validations/schemas.ts`, `frontend/src/i18n/messages/ui.{id,en}.ts`, `docs/openapi.yaml`, `docs/PRD.md` §11.

### DEC-058 — Google OAuth dan MFA TOTP di runtime Better Auth (frontend)

- **Tanggal:** 2026-11-13
- **Pemilik:** Pemilik produk (arah pengguna 13 November 2026 menyetujui implementasi nomor 7 dari daftar MVP yang tersisa: Google OAuth + MFA; MFA opsional per akun; tampilan URI dan QR)
- **Status:** Disetujui (arah pengguna)
- **Rincian:** Runtime Better Auth (`frontend/src/lib/server/better-auth.ts`, `DEC-049`) kini memuat: (1) `socialProviders.google` yang hanya aktif bila `GOOGLE_CLIENT_ID` dan `GOOGLE_CLIENT_SECRET` diisi — kosong berarti tombol Google tetap terlihat tetapi menjawab `PROVIDER_NOT_FOUND` dan antarmuka menampilkan "Masuk dengan Google belum dikonfigurasi oleh administrator." (2) plugin `twoFactor({ issuer: 'SIMPEG' })`; tabel `twoFactor` dan kolom `user.twoFactorEnabled` sudah ada sejak migrasi `000001` (verifikasi `2026-11-13`: skrip enable→verify TOTP, kode cadangan, penguncian akun tersedia di runtime; tidak ada skema baru). Antarmuka: tombol "Masuk dengan Google" pada halaman masuk (memulai `POST /api/auth/sign-in/social` dan mengalihkan ke URL otorisasi Google), halaman tantangan `/mfa` (kode 6 digit TOTP atau kode cadangan), halaman `/account/mfa` (enrollment: scan QR dari URI yang dirender lokal via paket `qrcode`, daftar kode cadangan, verifikasi kode; atau penonaktifan dengan kata sandi), item "Keamanan akun" pada menu akun di header. MFA bersifat **opsional per akun**; tidak ada role yang dipaksa mendaftar. **Belum diverifikasi langsung terhadap penyedia Google nyata maupun terhadap database produksi** — `BETTER_AUTH_SECRET`, `GOOGLE_CLIENT_ID`, dan `GOOGLE_CLIENT_SECRET` pada `.env.example` dibiarkan sebagai placeholder kosong; pada `.env` akar repo ketiganya juga kosong pada saat dicatat. Verifikasi yang sudah dilakukan: build, lint (0 galat), `tsc --noEmit`, 54 tes Vitest, e2e 30/30 (termasuk `e2e/mfa.spec.ts`: tantangan 2FA → verifikasi TOTP/kode cadangan → ruang kerja), dan pengujian header keamanan F-04 pada `bun run preview`. Verifikasi live menunggu pemilik produk mengisi kredensial pada lingkungan yang memakai database disposable dan mendaftarkan redirect URI `<origin>/api/auth/callback/google` di Google Cloud Console.
- **Bukti:** `frontend/src/lib/server/better-auth.ts`, `frontend/src/lib/session.ts`, `frontend/src/validations/schemas.ts`, `frontend/src/features/auth/login-form.tsx`, `frontend/src/routes/_auth/mfa.tsx`, `frontend/src/routes/_app/account/mfa.tsx`, `frontend/src/routes/_app/account/mfa.tsx`, `frontend/src/components/Header.tsx`, `frontend/e2e/mfa.spec.ts`, `frontend/.env.example`, `frontend/package.json` (dependency `qrcode`, devDependency `@types/qrcode`), `docs/PRD.md` §11, `docs/IMPLEMENTATION-PLAN.md`, `docs/openapi.yaml` (tidak ada perubahan kontrak backend).


### DEC-059 — Pembagian commit besar (enam chunk) dan fitur tambahan

- **Tanggal:** 2026-11-13
- **Pemilik:** Pemilik produk (arah pengguna)
- **Status:** Disetujui (arah pengguna)
- **Rincian:** Working tree berisi ~211 berkas yang belum di-commit. Disetujui pembagian menjadi enam chunk dalam urutan:
  - **(1) tes backend** — pindah semua `backend/internal/**/*_test.go` ke `backend/tests/` (93 berkas, tanpa perubahan isi).
  - **(6) backend** — perbaikan keamanan (F-03/F-06/F-08), master data (jabatan, impor massal 250 baris), SLO, PII (`internal/crypto/`, `config/pii.go`, `config/validate.go`, `config/config.go`), sumber payroll (`gorm_potongan.go`, `gorm_tunjangan.go`, `schedule.go`), migrasi `000021`–`000024`, `cmd/perfbench/seed*.go`, dan **Setting backend** (`internal/me/` domain/app/repository, `PATCH /api/v1/me`, jejak audit `profile.updated`, `ProfileResponse` dilengkapi `nama`/`noHp`).
  - **(3) Zod relokasi** — `frontend/src/validations/schemas.ts` saja (tidak ada sisa impor ke `@/lib/schemas`).
  - **(2) frontend core** — seluruh `frontend/` kecuali 8 berkas payroll dan `validations/schemas.ts`, termasuk berkas bersama (`Sidebar.tsx`, `i18n/messages/ui.{id,en}.ts`, `routeTree.gen.ts`, `vite.config.ts`, tabel shadcn, `LoadingBars`/`LoadingRadar`, `package.json` dan seluruh tooling) serta **Setting frontend** (`/settings`, `ProfileForm`, `ChangePasswordForm`, perubahan `Sidebar`, i18n, `lib/session`).
  - **(4) UI penggajian** — 8 berkas payroll saja.
  - **(5) dokumen + root/infra** — `docs/*.md`, `docs/openapi.yaml`, `.env.example`, `.gitignore`, dependabot, `frontend.yml`, `agents/rules/`, `DESIGN.md`, `SECURITY-TEST-OWASP-2025.md`.
  - Setting adalah fitur di luar rencana enam chunk; **frontend-nya masuk (2)**, **backend-nya masuk (6)**.
- **Bukti:** 7 commit di master (6 chunk + 1 cuti frontend), tidak ada push; `zap-report-frontend.md` dan `.serena/` sengaja tetap untracked.

### DEC-060 — Fitur Setting (tema, bahasa, profil, ganti kata sandi, pintasan 2FA)

- **Tanggal:** 2026-11-13
- **Pemilik:** Pemilik produk (arah pengguna: "setting mulai di front dan jika perlu tambahan pada back maka tambahkan"; ruang lingkup dipilih "Standar + edit profil sendiri")
- **Status:** Disetujui (arah pengguna)
- **Rincian:** Halaman baru `/settings` (`frontend/src/routes/_app/settings.tsx`) dengan lima tab: Tampilan (tema terang/gelap via `useThemeMode`/`setThemeMode`, persist `localStorage 'theme'`), Bahasa (id/en via `useLocale`/`setLocale`, persist `localStorage 'simpeg:locale'`), Akun (read-only: email, peran, unit dari `GET /api/v1/me`), Profil saya (ubah `nama` dan `noHp` via **baru** `PATCH /api/v1/me`; backend `internal/me/` domain/app/repository; jejak audit `profile.updated` tanpa mencatat nilai), Keamanan (ganti kata sandi via runtime Better Auth `POST /api/auth/change-password` tanpa perubahan backend Go, pintasan ke `/account/mfa`). Entri sidebar `nav.settings` ditambahkan untuk semua pengguna terautentikasi. Pesan i18n blok `settings.*` ditambahkan ke `messages/ui.{id,en}.ts`; `en` memenuhi `Record<keyof typeof uiMessagesId, string>`. Tidak ada perubahan OpenAPI untuk tema/bahasa (client-side); OpenAPI diperbarui untuk `PATCH /api/v1/me` dan `CurrentAccount` dilengkapi `nama`/`noHp`.
- **Bukti:** `frontend/src/routes/_app/settings.tsx`, `frontend/src/components/ProfileForm.tsx`, `frontend/src/components/ChangePasswordForm.tsx`, `frontend/src/components/Sidebar.tsx`, `frontend/src/lib/session.ts`, `frontend/src/validations/schemas.ts`, `frontend/src/i18n/messages/ui.id.ts`, `frontend/src/i18n/messages/ui.en.ts`, `backend/internal/me/`, `backend/internal/httpserver/me_handlers.go`, `backend/internal/httpserver/server.go`, `backend/internal/audit/audit.go` (`ActionProfileUpdated`), `backend/cmd/server/main.go`, `docs/openapi.yaml`, `docs/IMPLEMENTATION-PLAN.md`.

### DEC-061 — Antarmuka pengajuan cuti frontend (`/cuti`)

- **Tanggal:** 2026-11-13
- **Pemilik:** Pemilik produk (item 1/4 dari daftar fitur tersisa: cuti)
- **Status:** Disetujui (arah pengguna)
- **Rincian:** Halaman baru `frontend/src/routes/_app/cuti.tsx` dengan daftar pengajuan cuti berpaginasi (search/sort/filter, ukuran halaman 10/25/50/100), form buat/ubah pengajuan cuti, dan tombol setujui/tolak untuk status `MENUNGGU` (hanya `cuti:approve[:homebase]`). Backend cuti (`GET/POST/PATCH /api/v1/cuti`, `POST /api/v1/cuti/{id}/approve,reject`) sudah tersedia dari sebelum; hanya antarmuka frontend yang belum ada sesuai `docs/PRD.md` ("Cuti belum memiliki antarmuka"). Sidebar menampilkan entri `nav.cuti` dengan ikon kalender, gated permission `cuti:read:any|homebase|self`. Skema Zod `cutiSchema`, `cutiPageSchema`, `cutiCreateSchema`, `cutiUpdateSchema`, `cutiActionSchema` ditambahkan ke `frontend/src/validations/schemas.ts`; data layer `frontend/src/lib/cuti.ts`; dialog `frontend/src/components/CutiDialogs.tsx`; tabel `frontend/src/components/CutiTable.tsx`; pesan i18n blok `cuti.*` ditambahkan ke `messages/ui.{id,en}.ts`; `routeTree.gen.ts` digenerate ulang (termasuk `/cuti`). Semua gate frontend lolos (tsc, eslint, prettier, build, vitest 59/59).
- **Bukti:** `frontend/src/routes/_app/cuti.tsx`, `frontend/src/lib/cuti.ts`, `frontend/src/components/CutiTable.tsx`, `frontend/src/components/CutiDialogs.tsx`, `frontend/src/components/Sidebar.tsx`, `frontend/src/validations/schemas.ts`, `frontend/src/i18n/messages/ui.id.ts`, `frontend/src/i18n/messages/ui.en.ts`, `frontend/src/routeTree.gen.ts`, `docs/PRD.md`, `docs/IMPLEMENTATION-PLAN.md`, `docs/openapi.yaml`.

### DEC-062 — Endpoint `PATCH /api/v1/me` untuk profil mandiri (nama, nomor HP)

- **Tanggal:** 2026-11-13
- **Pemilik:** Pemilik produk (bagian fitur Setting)
- **Status:** Disetujui (arah pengguna)
- **Rincian:** `GET /api/v1/me` sebelumnya hanya mengembalikan identitas sesi (`userId`, `email`, `roleName`, `homebaseId`, `permissions`). Ditambahkan `PATCH /api/v1/me` (hanya principal sendiri, identifier dari sesi bukan body) untuk mengubah `nama` dan `noHp` (nomor Indonesia, awalan 0/62/+62, 8–15 digit; teks kosong mengosongkan). Backend baru `backend/internal/me/` (domain aturan validasi, app use case, repository GORM dengan transaksi + jejak audit `profile.updated` tanpa mencatat nilai), handler `backend/internal/httpserver/me_handlers.go`, terdaftar di `server.go` dan di-wire di `cmd/server/main.go` (`MuxOptions.Me`). `ProfileResponse` dilengkapi `nama`/`noHp`. OpenAPI diperbarui: skema `CurrentAccount` + `UpdateProfileRequest`, operasi `PATCH /api/v1/me` (200/400/401/404/422/500). Tidak ada endpoint untuk mengubah email, NIP, PII terenkripsi, role, atau unit (hendak administrator). Pengujian: `backend/tests/me/domain/profile_test.go`, `backend/tests/me/app/service_test.go`, `backend/tests/httpserver/me_handlers_test.go`.
- **Bukti:** `backend/internal/me/domain/profile.go`, `backend/internal/me/app/service.go`, `backend/internal/me/repository/gorm.go`, `backend/internal/httpserver/me_handlers.go`, `backend/internal/httpserver/server.go`, `backend/internal/audit/audit.go` (`ActionProfileUpdated`), `backend/cmd/server/main.go`, `docs/openapi.yaml`, `docs/IMPLEMENTATION-PLAN.md`.

### DEC-063 — Penyesuaian frontend untuk CSP enforcing dan verifikasi ulang ZAP serta e2e

- **Tanggal:** 2026-11-13
- **Pemilik:** Pemilik produk (item 5 dan 6 daftar pekerjaan MVP: pemindaian ulang OWASP ZAP dan e2e ulang)
- **Status:** Disetujui (arah pengguna)
- **Rincian:** Verifikasi ulang memunculkan empat hal yang perlu keputusan teknis. (1) **Font brand:** `styles.css` mengimpor Fraunces dan Manrope dari `fonts.googleapis.com`, tetapi CSP enforcing `style-src 'self'; font-src 'self'` menolaknya, sehingga kedua font tidak pernah termuat dan aplikasi bergantung pada host pihak ketiga. Font kini disajikan sendiri dari `frontend/public/fonts/` (subset latin variabel) beserta lisensi OFL 1.1; aturan `@import` ke Google Fonts dihapus. (2) **Konfigurasi e2e:** `playwright.config.ts` memakai `VITE_API_BASE_URL` absolut ke port 8080, sehingga browser memblokir panggilan API sebelum `page.route` sempat menanganinya karena `connect-src 'self'`. Konfigurasi kini memakai path same-origin `/api/v1`, sesuai proksi Vite pada pengembangan harian dan tanpa melonggarkan CSP. (3) **Warna tautan:** aturan global `a { color: ... }` berada di luar layer Tailwind sehingga menimpa utilitas warna tombol primer; aturan itu dipindahkan ke `@layer base`, tombol utama kembali putih di hijau, dan rasio kontrasnya 6,40 (ambang 4,5). (4) **Risiko residual CSP:** `'unsafe-inline'` pada script dan style dipakai untuk inline SSR; ini dicatat sebagai F-09 dan diterima untuk sekarang karena sudah menjadi bagian kebijakan `DEC-056`. Menghilangkannya memerlukan nonce atau hash, yang belum dikerjakan. Alert `SQL Injection` dari pemindaian ulang ditriage menjadi positif palsu, lihat `docs/SECURITY-TEST-OWASP-2025.md` §5.2.
- **Bukti:** `frontend/src/styles.css`, `frontend/public/fonts/` (Fraunces dan Manrope subset latin, `OFL-Fraunces.txt`, `OFL-Manrope.txt`), `frontend/playwright.config.ts`, `docs/SECURITY-TEST-OWASP-2025.md` §5.2, `docs/IMPLEMENTATION-PLAN.md`.
