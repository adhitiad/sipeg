# Runbook insiden SIMPEG

Dokumen ini berisi tindakan yang harus dilakukan operator ketika ada gangguan.
Setiap bagian memuat: gejala, diagnosis, tindakan, dan hal yang **tidak** boleh
dilakukan. Jalankan bagian yang relevan, catat kronologi, dan hubungi pemilik
proses bila langkah perlu keputusan di luar wewenang teknis.

Aturan umum:

- **Jangan pernah menghapus atau mengubah baris `auditLog`.** Jejak audit bersifat
  append-only. Penghapusan hanya mungkin lewat prosedur arsip yang disetujui
  pemilik kebijakan, dan prosedur itu belum ditetapkan.
- **Jangan mengirim data pegawai ke luar sistem** saat diagnosis. Gunakan ID, kode
  error, dan korelasi request ID (`X-Request-ID`), bukan isi data.
- **Jangan mengubah skema database di production** tanpa migration yang sudah
  direview dan diuji di lingkungan terpisah.

## 1. Server tidak mau start

**Gejala:** proses berhenti saat start dengan pesan `config: ...` atau
`database connection setup failed`.

**Diagnosis:**

```bash
# Periksa variabel wajib
echo "${APP_ENV}" "${DATABASE_URL:+db-url-diset}" "${BETTER_AUTH_SECRET:+secret-diset}" "${TRUSTED_ORIGINS:+origins-diset}"
```

**Tindakan:**

1. Pastikan `DATABASE_URL` disetel dan menunjuk ke instance yang benar.
2. Pada `APP_ENV=production`, `BETTER_AUTH_SECRET` wajib ada, minimal 32 karakter,
   dan tidak boleh memuat pola bawaan seperti `change-me`.
3. Pada `APP_ENV=production`, `TRUSTED_ORIGINS` wajib diisi dan tidak boleh `*`.
4. Pastikan nilai pool database masuk akal: `DB_MAX_IDLE_CONNS` tidak boleh lebih
   besar dari `DB_MAX_OPEN_CONNS`.

**Jangan:** inserting nilai default untuk production agar start berhasil.

## 2. Database tidak dapat dijangkau

**Gejala:** `GET /ready` mengembalikan `503` dengan `checks.postgres.status` =
`unavailable`, atau log akses mencatat banyak `503`.

**Diagnosis:**

```bash
curl -s -o /dev/null -w '%{http_code}\n' http://localhost:8080/ready
```

Readiness menjalankan ping database dengan batas waktu (default 15 detik).
`GET /health` tetap `200` selama proses hidup — itu liveness, bukan kesiapan.

**Tindakan:**

1. Periksa apakah instance database aktif dan tidak sedang undergoing maintenance.
2. Periksa batas koneksi: `DB_MAX_OPEN_CONNS`mungkin perlu diturunkan bila
   instance dibagi dengan layanan lain dan mengalami connection exhaustion.
3. Jika hanya sebagian endpoint yang lambat, periksa `DB_STATEMENT_TIMEOUT`.
   Query yang melewati batas dibatalkan oleh driver PostgreSQL.

**Jangan:** menaikkan `DB_STATEMENT_TIMEOUT` tanpa investigating query yang
lambat — batas waktu ada untuk mencegah sumber daya habis.

## 3. Antrean atau timeout pada permintaan

**Gejala:** permintaan sering `503`, atau durasi pada log akses jauh melewati
batas normal.

**Tindakan:**

1. Lihat `duration_ms` pada log akses untuk mengidentifikasi endpoint lambat.
2. Periksa `totalItems` pada halaman daftar — endpoint daftar memakai pagination
   server-side dengan `pageSize` maksimum 100.
3. Pastikan tidak ada laporan yang meminta `pageSize` besar; nilai di luar batas
   ditolak dengan `400 invalid_query`.

## 4. Kebocoran kredensial

**Gejala:** ada dugaan password atau token bocor ke log, respons API, atau repo.

**Tindakan (urutan):**

1. **Putuskan akses lebih dulu.** Jika kredensial aktif, rotasi segera di
   secret manager. Rotasi lebih cepat daripada investigasi.
2. Periksa log akses. Header `Authorization`, body request, dan nilai query
   string tidak pernah dicatat oleh `internal/observability` — bila kredensial
   terlihat di sana, berarti ada jalur pencatatan lain yang perlu diperiksa.
3. Periksa respons API: bentuk error seragam hanya memuat `code`, `message`,
   `requestId`, `fields`, dan `rows`, tidak memuat nilai internal.
4. Periksa repository: `git log -p -S '<fragmen>'` untuk mencari apakah kredensial
   pernah masuk komitman. Riwayat commit tidak dapat "dihapus" dengan aman —
   seluruh kredensial yang pernah masuk komitman harus dianggap bocor.
5. Catat insiden beserta request ID terkait dan tindakan yang dilakukan.

**Jangan:** menampilkan isi `.env`, nilai `DATABASE_URL`, atau isi token di
issue, pesan, atau dokumen.

## 5. Akun atau sesi harus dicabut segera

**Gejala:**Pegawai keluar dari sistem, ada kecurigaan penyalahgunaan, atau
persona yang dinonaktifkan masih aktif.

**Tindakan:**

1. Backend menolak sesi untuk akun nonaktif secara otomatis: akun dengan
   `aktif = false` ditolak sebagai sesi tidak valid (`401`).
2. Cabut sesi dengan `POST /api/v1/auth/logout`. Endpoint ini menghapus baris
   `session` sehingga token tidak lagi berlaku, dan mencatat jejak `session.revoked`.
3. Untuk pencabutan massal, hapus baris `session` milik pengguna terkait
   **melalui prosedur operasional yang disetujui**, bukan dengan operasional
   manual terhadap database di luar aplikasi.

**Jangan:** mengedit baris `user.roleId` atau `user.aktif` langsung di database.
Perubahan role harus melalui endpoint yang berjejak audit.

## 6. Gangguan integrasi eksternal (Redis, QStash, SMTP, S3)

Integrasi tersebut belum diimplementasikan. Bila integrasi sudah aktif:

1.Setiap integrasi wajib punya fallback ke sumber kebenaran di PostgreSQL.
   Redis dan queue **tidak boleh** menjadi sumber kebenaran untuk data pegawai,
   payroll, atau pembayaran.
2. Callback QStash diverifikasi signature-nya terhadap raw request body sebelum
   diproses, dan handler harus idempoten karena retry dapat mengulang delivery.
3. Notifikasi gaji hanya dikirim setelah konfirmasi pembayaran tersimpan sebagai
   terkonfirmasi, bukan saat slip diterbitkan.

## 7. Pemulihan setelah penerapan migrasi

**Sebelum deploy:**

1. Tinjau seluruh migrasi yang belum diterapkan, termasuk urutan dan kompatibilitas
   mundur. Migrasi di `backend/migrations/` memakai nomor berurutan.
2. Jalankan di lingkungan terpisah dengan data sintetis lebih dulu.
3. Pastikan backup dan restore sudah diuji, bukan hanya backup.

**Setelah deploy:**

1. Verifikasi `GET /ready` mengembalikan `200`.
2. Verifikasi `GET /health` tetap `200`.
3. Cek log akses untuk error dan request ID bermasalah.

**Jika migrasi gagal di tengah jalan:**

1. Jangan menjalankan migrasi berikutnya secara manual.
2. Tentukan apakah migrasi bersifat reversible. Migrasi yang menambah kolom
   (`000003_audit_log_unit.sql`) dapat dibalik dengan `DROP COLUMN`; migrasi yang
   menambah kolom dengan `NOT NULL` tanpa default memerlukan langkah pemulihan
   data lebih dulu.
3. Dokumentasikan tindakan pada catatan deployment.

4. Migrasi `000019_cuti_eviden.sql` bersifat aditif dan dapat dibalik: hanya
   menambah kolom nullable, mengubah nama kolom `fileSuratUrl` menjadi
   `fileEvidenUrl` (tidak ada kolom yang dihapus secara permanen pada Up, dan
   Down mengembalikan nama lama), serta menambah enum `JenisEviden` yang baru
   digunakan model. Rollback lewat `goose down` aman pada database yang belum
   menulis ke kolom baru.

Workflow migrasi GitHub (`.github/workflows/deploy-migrations.yml`) belum
pernah dijalankan karena environment `staging`/`production` dan secret
`DATABASE_URL` belum dibuat di GitHub. Sebelum menjalankan workflow di
produksi, cobalah urutan perintahnya (`migrate status` → `migrate up` →
`migrate status`) pada database uji terlebih dahulu.

## 8. Skema dan mapping GORM tidak sinkron

Gejala: query gagal dengan `column "..." does not exist`, atau tes paritas skema
gagal.

**Diagnosis:** periksa status dan urutan Goose migrations, lalu jalankan
`go test ./tests/database/...` untuk memeriksa keberadaan tabel model,
indeks jalur query kritis, dan cakupan migrasi RLS. Tes ini tidak membuktikan
bahwa migrasi sudah diterapkan atau seluruh kolom GORM cocok dengan database.

Untuk pemeriksaan langsung terhadap database target, pakai subcommand `schema`
dari perkakas pengukuran. Perintah itu membandingkan kolom yang dinyatakan
model GORM dengan kolom yang benar-benar ada:

```bash
cd backend
export DATABASE_URL="postgres://<user>:<sandi>@<host>:5432/<database>?sslmode=require"
go run ./cmd/perfbench schema
```

Keluarannya memuat tiga kelompok selisih: model punya kolom tapi database
tidak, database punya kolom tapi model tidak, dan model tanpa tabel. Tabel
yang tidak ada pada database menandakan migrasi belum diterapkan.

**Tindakan:** verifikasi skema database target terhadap migrasi Goose dan
perbaiki migrasi/model melalui perubahan berversi yang ditinjau. Jangan memakai
`AutoMigrate` untuk memperbaiki skema production.

### 8.1 Koneksi database ditolak pooler

Gejala: permintaan menjawab `503` dengan `service_unavailable`, dan log
menampilkan `FATAL: (EMAXCONNSESSION) max clients reached in session mode`.

**Diagnosis:** pooler sesi Supabase membatasi jumlah koneksi concurrently per
proyek. Bila `DB_MAX_OPEN_CONNS` melampaui batas itu, PostgreSQL menolak
koneksi baru. Periksa batas pooler pada proyek Supabase dan nilai
`DB_MAX_OPEN_CONNS` yang aktif.

**Tindakan:** turunkan `DB_MAX_OPEN_CONNS` di bawah batas pooler, atau
pindahkan ke pooler transaksi yang melakukan multiplexing pada porta 6543.
Bawaan repository adalah 10 koneksi, yang memberi ruang di bawah batas
pooler tanpa menurunkan throughput pada beban wajar.

### 8.2 Publikasi dan jadwal pekerjaan ditolak (RabbitMQ self-hosted / QStash cloud)

Gejala: pekerjaan terjadwal tidak dieksekusi, atau antrean `simpeg.jobs.dead`
berisi pesan yang gagal diproses.

**Diagnosis:** bentuk transport tergantung pada deployment:

- **Self-hosted (docker/podman):** `RABBITMQ_URL` terisi, pekerjaan terjadwal
  dipindahkan ke RabbitMQ (`DEC-042`). Penjadwal `cmd/jobsched` menerbitkan
  pesan ke exchange `simpeg.jobs`, server mengonsumsi antrean `simpeg.jobs`.
  Periksa antrean dead-letter:

  ```bash
  docker compose -f rabbitmq/docker-compose.yml exec rabbitmq \
    rabbitmqctl list_queues name messages
  docker compose logs backend | grep mq_job_failed
  ```

- **Cloud (QStash):** `RABBITMQ_URL` kosong, pekerjaan terjadwal melalui
  QStash. Bentuk endpoint QStash harus sesuai (`/v2/publish` dan
  `/v2/schedules` ditambahkan otomatis). Periksa daftar jadwal:

  ```bash
  curl -H "Authorization: Bearer $QSTASH_TOKEN" \
    https://qstash-us-east-1.upstash.io/v2/schedules
  ```

**Tindakan:**
- RabbitMQ: bila `simpeg.jobs.dead` berisi pesan, periksa log `mq_job_failed`
  untuk penyebabnya, perbaiki, lalu jalankan ulang pekerjaannya dengan
  `jobsched publish <kind>`. Pesan dead-letter tidak kembali otomatis karena
  tick cron berikutnya memicu eksekusi baru. Pekerjaan idempoten sehingga
  pengulangan aman.
- QStash: jalankan ulang backend setelah konfigurasi diperbaiki. Backend
  mendaftarkan jadwal secara idempoten, sehingga penyalaan ulang tidak membuat
  jadwal ganda. Bila jadwal lama menunjuk tujuan yang sudah tidak dipakai,
  hapus lewat `DELETE /v2/schedules/{scheduleId}`.

### 8.3 Email tidak terkirim

Gejala: antrean keluar menumpuk dengan status `PENDING`, atau log worker
menyebut `smtp` pada pesan galat.

**Diagnosis:** mode keamanan SMTP ditentukan oleh `SMTP_SECURE` dan porta:

| Mode | Perilaku | Porta lazim |
|------|-----------|-------------|
| `SMTP_SECURE=true` | TLS langsung sejak koneksi pertama | 465 |
| `SMTP_SECURE=false` | Koneksi polos, dinaikkan lewat STARTTLS bila tersedia | 587 |
| `SMTP_SECURE` kosong | Mode ditebak dari porta: 465 memakai TLS langsung, porta lain memakai STARTTLS | — |

Galat `tls: first record does not look like a TLS handshake` berarti mode TLS
langsung dipakai pada porta submission. Galat `535` dari Google Workspace
berarti kredensial ditolak; Google Workspace menuntut *app password*, bukan
sandi akun biasa.

Bentuk `SMTP_FROM` boleh berupa alamat polos maupun bentuk baku
`"Nama Tampilan" <alamat@domain>`. Bentuk kedua dipakai apa adanya pada
header `From`, sedangkan perintah SMTP `MAIL FROM` memakai alamat polosnya.

**Tindakan:** samakan `SMTP_SECURE` dengan porta server, lalu gunakan *app
password* yang sah bila penyedia adalah Google Workspace.

## 8A. Pemeriksaan SLO terjadwal

**Gejala:** alert SLO tidak pernah datang, atau alert datang pada waktu yang
tidak wajar.

**Diagnosis:** tiga hal harus benar:

1. `SLO_ENABLED=true` mengaktifkan pengoleki metrik dan mendaftarkan
   pemeriksaan terjadwal.
2. Jadwal `slo.evaluate` terdaftar pada QStash (periksa daftar jadwal seperti
   pada bagian 8.2), atau pada mode self-hosted: `SLO_EVALUATE_ENABLED=true`
   pada penjadwal RabbitMQ dan pesan `slo.evaluate` mengalir setiap 5 menit
   (bagian 8B).
3. `SLO_ALERT_RECIPIENTS` berisi alamat penerima yang sah.

```bash
cd backend
export DATABASE_URL="postgres://<user>:<sandi>@<host>:5432/<database>?sslmode=require"
go run ./cmd/perfbench schema

# Metrik yang terkumpul pada jendela terakhir.
psql "$DATABASE_URL" -c \
  'SELECT "windowStart", "route", "requestCount", "errorCount", "latencyMaxMs"
     FROM "sloMetric" ORDER BY "windowStart" DESC LIMIT 10;'
```

**Tindakan:** metrik disimpan di PostgreSQL, bukan Redis. Bila tabel
`sloMetric` kosong, pengoleki tidak berjalan: periksa `SLO_ENABLED` dan log
slo_metric_flush_failed. Bila metrik terkumpul tetapi alert tidak dikirim,
periksa antrean keluar (outboxMessage) dan status pengiriman email. Untuk
melihat ringkasan agregat tanpa SQL, gunakan halaman admin `/admin/slo`
(`GET /api/v1/admin/slo/metrics`, permission `slo:read`, admin saja);
halaman menampilkan agregat per rute pada jendela 24 jam atau 7 hari, bukan
data mentah per jendela.

## 8B. Pekerjaan terjadwal RabbitMQ (self-hosted)

**Gejala:** pekerjaan harian (pembersihan jejak audit, pemberitahuan slip
gaji H+1) tidak berjalan, padahal deployment memakai docker/podman dengan
`RABBITMQ_URL` terisi.

**Diagnosis:** tiga komponen harus hidup:

1. Broker: `docker compose ps rabbitmq` harus berstatus healthy.
2. Penjadwal: log `jobsched` memuat `jobsched_schedule_ready` untuk setiap
   kind; `jobsched_published` muncul pada setiap tick.
3. Konsumen: log `backend` memuat `mq_consumer_started` dan `mq_job_done`
   setiap pekerjaan berjalan.

```bash
docker compose exec rabbitmq rabbitmqctl list_queues name messages
docker compose run --rm --entrypoint /bin/jobsched jobsched publish audit.cleanup
docker compose logs backend | grep mq_job
```

**Tindakan:** bila `simpeg.jobs.dead` berisi pesan, periksa log
`mq_job_failed` untuk mengetahui penyebabnya, perbaiki, lalu jalankan ulang
pekerjaannya dengan `jobsched publish <kind>`; pesan dead-letter tidak
kembali otomatis karena tick cron berikutnya sudah memicu eksekusi baru.
Pekerjaan idempoten sehingga pengulangan aman. Bila konsumen gagal
tersambung, log `mq_consumer_error` berulang setiap 5 detik; periksa
kredensial `RABBITMQ_URL` dan jaringan antar container.

## 9. Percobaan akses tidak sah

**Gejala:** log keamanan menunjukkan `auth_failed`, `access_denied`, atau
`rate_limited` berulang dari satu `client_ip`, atau jejak audit memuat aksi
`authz.denied`.

**Dua sumber bukti:**

| Sumber | Isi | Dipakai untuk |
|--------|-----|---------------|
| Log keamanan (`msg`: `auth_failed`, `access_denied`, `rate_limited`) | `client_ip`, `path`, `status`, `request_id`, `user_id`, `role` | Deteksi volume dan korelasi cepat |
| Log `auth_failed` runtime Better Auth (frontend) | `path`, `reason`, `ts` (tanpa `client_ip`/kredensial, `DEC-050`) | Menelusuri kegagalan login email/OAuth, termasuk Google OAuth |
| Jejak audit aksi `authz.denied` | aktor, unit cakupan, izin diminta, pola route | Menelusuri pihak yang mencoba akses di luar haknya |
| Log keamanan `webhook_signature_rejected` | `job`, `path`, `request_id` | Mendeteksi callback QStash yang tidak sah |

Kolom `client_ip` hanya tersedia pada log keamanan backend; aturan alert per
IP (lihat tabel ambang di bawah) berlaku untuk log backend. Log `auth_failed`
runtime frontend dapat dihitung per `path`/`reason`, bukan per IP.

`user_id` dan `role` hanya ada pada peristiwa yang terjadi setelah autentikasi
berhasil. Kegagalan autentikasi (`auth_failed`) tidak memuat aktor karena belum
ada identitas terverifikasi.

**Diagnosis:**

```bash
# Jejak penolakan di gerbang izin, tanpa membuka isi data pegawai.
psql "$DATABASE_URL" -c "select \"createdAt\", \"aktorId\", \"unitId\", \"keterangan\" from \"auditLog\" where \"aksi\" = 'authz.denied' order by \"createdAt\" desc limit 50;"
```

**Tindakan:**

1. Ambil `request_id` dari satu entry log keamanan, lalu telusuri log akses pada
   `request_id` yang sama untuk melihat urutan permintaan.
2. Cocokkan `aktorId` pada `authz.denied` dengan daftar akun pilot. Akses dari
   akun yang tidak memegang izin memunculkan dugaan salah grant atau
   penyalahgunaan wewenang.
3. Naikkan `RATE_LIMIT_AUTH_REQUESTS` hanya setelah memastikan traffic memang
   sah. Batas yang ada memperlambat credential stuffing.
4. Cabut sesi yang dicurigai melalui `POST /api/v1/auth/logout`, nonaktifkan akun
   bila perlu, lalu catat kronologi insiden.
5. Jika `webhook_signature_rejected` muncul berulang, periksa konfigurasi
   signing key QStash dan penjadwal, bukan isi log signature.

**Ambang alert awal.** Nilai berikut adalah titik awal untuk disetel operator,
belum berdasarkan baseline volume:

| Event | Syarat alert awal |
|-------|-------------------|
| `auth_failed` | lebih dari 20 kejadian dari satu `client_ip` dalam 5 menit |
| `access_denied` | lebih dari 50 kejadian dari satu `client_ip` dalam 5 menit |
| `rate_limited` | muncul berulang pada path autentikasi |
| `webhook_signature_rejected` | lebih dari 3 kejadian dalam 10 menit |

**Jangan:** menganggap `access_denied` sebagai bug. `403` pada endpoint
terlindungi adalah hasil yang benar untuk role yang tidak memegang izin, dan
perlu diperlakukan sebagai bukti percobaan akses, bukan tanda gangguan layanan.

## 8C. Pantauan Redis Upstash (cache data referensi)

**Gejala:** pembacaan kategori pegawai pada `GET /api/v1/categories` lebih
lambat dari biasanya, atau log `redis_cache_miss` berulang.

**Diagnosis:** tiga komponen:

1. Koneksi: `UPSTASH_REDIS_REST_URL` dan `UPSTASH_REDIS_REST_TOKEN` terisi,
   dan `RedisEnabled()` mengembalikan `true` pada production.
2. Respons: `GET {UPSTASH_REDIS_REST_URL}/ping` mengembalikan `PONG`.
3. Hit/miss: metrik cache (jika diinstrumenkan) menunjukkan rasio hit yang
   menurun secara tajam.

```bash
curl -H "Authorization: Bearer $UPSTASH_REDIS_REST_TOKEN" \
  "$UPSTASH_REDIS_REST_URL/ping"
```

**Tindakan:** bila Redis gagal, backend tetap membaca dari PostgreSQL lewat
`ReadThrough` (cache-saja, bukan sumber kebenaran). Tidak ada gangguan
layanan, hanya penurunan performa pembacaan referensi. Periksa status instance
Upstash pada dashboard penyedia.

## 8D. Timeout, retry, dan circuit breaker pada dependency eksternal

**Gejala:** permintaan ke QStash, SMTP, atau penyedia hari libur menyebabkan
timeout berulang, dan antrean keluar (outbox) menumpuk.

**Diagnosis:**
- QStash: periksa `qstash_schedule_failed` dan `qstash_job_payload_failed`
  (bagian 8.2).
- SMTP: periksa log `smtp` pada pesan galat worker outbox (bagian 8.3).
- Penyedia hari libur: kalender dibaca dari tabel `HariLibur` lokal, tidak
  bergantung pada penyedia runtime; penyegaran hanya dipicu admin.

**Tindakan:** backend menggunakan outbox persisten di PostgreSQL sehingga
pesan tidak hilang saat dependency eksternal down. Worker mencoba ulang
dengan backoff eksponensial yang dibatasi; setelah jumlah percobaan maksimum,
pesan pindah ke status `FAILED` sebagai dead-letter. Circuit breaker belum
diimplementasikan pada rilis ini karena integrasi eksternal (QStash, SMTP,
API hari libur) belum semua diimplementasikan production. Tambahkan circuit
breaker saat integrasi tersebut dipakai pada production.

## 8E. Uji backup dan restore database

**Gejala:** tidak ada bukti bahwa backup dapat dipulihkan dalam waktu yang
ditentukan.

**Diagnosis:**
- `RPO_HOURS` (bawaan 24 jam) dan `RTO_HOURS` (bawaan 4 jam) dikonfigurasi
  melalui `.env` dan divalidasi `config.Load()`. Nilai awal diratifikasi
  24 jam / 4 jam pada 13 November 2026 (`DEC-050`); keduanya belum diukur
  terhadap produksi.
- Backup dan PITR (point-in-time recovery) dijalankan melalui tier Supabase
  yang dipilih, bukan melalui skrip backend.

**Bukti drill (13 November 2026, `DEC-050`):** drill dijalankan terhadap
database disposable lokal `simpeg-verify-pg` (postgres:16-alpine), **bukan**
Supabase production:

- `pg_dump` dari `postgres`: 134.982 byte, 0,35 detik;
- restore ke database baru `simpeg_restore_drill`: 1,36 detik, tanpa galat
  (`ON_ERROR_STOP=1`);
- perbandingan sumber vs hasil identik: Goose 20 versi, 69 tabel, 181
  indeks, 167 constraint, 68 tabel dengan RLS aktif, `user`=4, `session`=8,
  `account`=4, `auditLog`=8;
- database `simpeg_restore_drill` dihapus setelah verifikasi;
- **belum diuji:** PITR, restore pada tier Supabase berbayar, dan restore
  pada database target produksi. Uji di lingkungan terpisah tetap wajib
  dilakukan sebelum klaim RTO 4 jam dianggap terpenuhi untuk produksi.

**Tindakan:** uji backup/restore pada database terpisah dengan data sintetis,
catat waktu yang dibutuhkan untuk pulihkan ke titik waktu tertentu, dan
bandingkan dengan `RPO_HOURS` dan `RTO_HOURS`. Dokumentasikan hasil uji pada
catatan operasi. Jangan gunakan data pegawai produksi untuk uji restore.

## 9. Dashboard dan alert

**Gejala:** operator tidak mengetahui layanan mana yang gagal, dampak ke
pengguna, atau langkah pemulihannya.

**Diagnosis:** backend menyediakan log terstruktur JSON satu baris per entri
dengan correlation ID (`X-Request-ID`), dan metrik SLO disimpan di PostgreSQL
(`sloMetric`) dan ditampilkan pada halaman admin `/admin/slo` (backend:
`GET /api/v1/admin/slo/metrics`, permission `slo:read`, admin saja,
baca-saja, lihat `DEC-050`). Halaman ini mencakup error rate, latency
(rata-rata/p95/p99/maks) per rute, dan jendela 24 jam atau 7 hari —
**tidak** mencakup saturasi, health dependency, keterlambatan antrean, atau
kegagalan notifikasi, dan **aturan alert belum terpasang pada
infrastruktur mana pun** pada rilis ini. Tanpa `SLO_ENABLED=true` pada
backend, `sloMetric` kosong dan halaman menampilkan total nol.

**Tindakan:** dasbor internal adalah titik awal yang sudah ada, bukan
platform monitoring. Pasang aturan alert pada platform monitoring yang
dipakai organisasi (atau scraper sederhana yang membaca
`GET /api/v1/admin/slo/metrics`) dengan metrik berikut sebagai titik awal:

| Metrik | Sumber | Ambang alert awal |
|--------|--------|-------------------|
| Error rate API | `sloMetric.errorCount / requestCount` | > 1% dalam 5 menit |
| Latency p95 | histogram bucket pada `sloMetric` (dihitung oleh evaluasi `slo.evaluate`, bukan kolom tunggal) | > 800 ms dalam 5 menit |
| Koneksi pool PostgreSQL | `DB_MAX_OPEN_CONNS` vs aktif | > 80% batas pooler |
| Antrean keluar (outbox) | `outboxMessage` status `FAILED` | > 0 pesan |
| Pesan gagal QStash | `qstash_job_payload_failed` | > 3 dalam 10 menit |
| Email gagal | `outbox` worker error SMTP | > 0 pesan permanen |

Jenis alert:
- **Kritis:** API error rate > 5%, koneksi pool > 90%, outbox dead-letter > 10
  pesan.
- **Peringatan:** API error rate > 1%, latency p95 > 800 ms, email gagal
  permanen.

Log keamanan (`auth_failed`, `access_denied`, `rate_limited`) dan jejak audit
(`authz.denied`) juga boleh diekspor ke SIEM untuk deteksi insiden keamanan
(bagian 9).

## 10. Autentikasi dan Sesi

### 10.1 Model sesi yang didukung

Backend mendukung dua jenis sesi:

| Jenis | Penyimpanan | Revokasi | Keterangan |
|-------|-------------|----------|------------|
| JWT Better Auth | Better Auth runtime (frontend) | Milik runtime Better Auth | Verifikasi via `BETTER_AUTH_SECRET` dengan `golang-jwt/jwt/v5` |
| Sesi opaque GORM | Tabel `session` di PostgreSQL | `POST /api/auth/logout` menghapus baris `session` | Resolusi via `auth.CombinedSessionResolver` |

**Penting:** runtime Better Auth **terpasang tertanam di server frontend** pada `/api/auth/*` (lihat `DEC-049`): sesi 7 hari dengan rolling refresh 1 hari, pendaftaran publik dimatikan, akun dibuat administrator. Masuk/keluar terverifikasi terhadap database uji disposable untuk seluruh role; logout di frontend memanggil `POST /api/v1/auth/logout` (mencabut sesi opaque) dan `POST /api/auth/sign-out` (mencabut sesi runtime Better Auth). Google OAuth dan MFA TOTP diimplementasikan 2026-11-13 (`DEC-058`); verifikasi live menunggu kredensial Google Workspace dan redirect URI yang terdaftar.

### 10.2 Logout dan pencabutan sesi

**Gejala:** sesi tetap dapat digunakan setelah `POST /api/auth/logout`, atau token JWT tetap valid setelah dicabut.

**Diagnosis:**
- `POST /api/auth/logout` (daftarkan via `httpserver.RegisterLogoutRoute`) **hanya mencabut sesi opaque GORM** (menghapus baris `session` di PostgreSQL). Penrevokasi token JWT tetap menjadi tanggung jawab runtime Better Auth, bukan backend Go.
- Role dibaca dari tabel `Role` pada setiap permintaan, sehingga perubahan role berlaku tanpa sesi baru.
- Akun nonaktif (`aktif = false`) dan baris `session` yang dihapus ditolak sebagai `401`.

**Tindakan:**
1. Untuk sesi opaque: panggil `POST /api/auth/logout`, lalu verifikasi permintaan berikutnya dengan token yang sama menerima `401`.
2. Untuk token JWT: cabut melalui runtime Better Auth (`POST /api/auth/sign-out`; runtime terpasang tertanam di server frontend sejak `DEC-049`).
3. Jika akun dicurigai berbahaya: nonaktifkan akun (`aktif = false`) pada tabel `user`, lalu sesi berikutnya akan ditolak.

### 10.3 Kegagalan resolver sesi

**Gejala:** permintaan menerima `503` dengan kode `service_unavailable` saat memeriksa sesi.

**Diagnosis:** resolver sesi tidak dapat terhubung ke database PostgreSQL. Periksa `DATABASE_URL` dan koneksi pooler.

**Tindakan:** verifikasi koneksi database:
```bash
cd backend
export DATABASE_URL="postgres://<user>:<sandi>@<host>:5432/<database>?sslmode=require"
go run ./cmd/migrate status
```

### 10.4 Pemilihan permission dan cakupan

**Gejala:** endpoint menerima `403` meski payload dan token valid.

**Diagnosis:** grant statis dikonfigurasi di `backend/cmd/server/main.go` (matriks disetujui di `docs/PRD.md` §11). Role atau izin yang tidak tercantum tetap ditolak (deny-by-default). Permission berakhiran `:any` = lintas seluruh organisasi (khusus `admin`); `:homebase` = dibatasi pada unit penugasan principal beserta seluruh turunannya (`operator`, `atasan`); `:self` = hanya record milik principal sendiri (`pegawai`).

**Tindakan:** verifikasi grant pada role principal melalui `GET /api/v1/me` dan cek matriks izin di `docs/PRD.md` §11. Jangan memperluas grant tanpa memverifikasi route dan query.


