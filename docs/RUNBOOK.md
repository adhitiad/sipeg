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

## 6. Gangguan integrasi eksternal (Redis, SMTP)

Integrasi dukungan di repo ini membatasi diri pada interface terkontrol: Redis **cache saja**, RabbitMQ (self-hosted) untuk pekerjaan terjadwal, SMTP untuk notifikasi. QStash tidak dipakai pada deployment manapun (`DEC-042`, `DEC-102`). Aturan integrasi eksternal:

1. Setiap integrasi wajib punya fallback ke sumber kebenaran di PostgreSQL.
Redis dan queue **tidak boleh** menjadi sumber kebenaran untuk data pegawai,
payroll, atau pembayaran. 2. Pekerjaan yang dijadwalkan lewat RabbitMQ mengakhiri siklusnya di `simpeg.jobs` / `simpeg.jobs.dead`; bila pekerjaan terjadwal gagal berulang, periksa antrean dan `jobsched`. 3. Notifikasi gaji hanya dikirim setelah konfirmasi pembayaran tersimpan sebagai
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

## 7A. Seed RBAC scoped (pasca-migrasi `000032`)

Kapan perlu: setelah migrasi `000032_rbac_scoped.sql` terpasang di target dan
sebelum RBAC dipakai untuk pencabutan izin. Tanpa assignment, evaluator RBAC
tidak berkuasa dan grant statis tetap berlaku (`DEC-087`–`DEC-089`).

**Prasyarat:** migrasi `000001`–`000032` terpasang (`go run ./cmd/migrate status`
→ versi 32), backup target sudah diuji, dan Environment GitHub/mesin operator
memiliki `DATABASE_URL` untuk database tersebut.

**Langkah (urutan wajib):**

1. **Dry-run lebih dulu** — tidak menulis apa pun:

   ```bash
   cd backend
   $env:DATABASE_URL = "postgres://..."
   go run ./cmd/rbacseed
   ```

   Periksa ringkasan: jumlah role/permission/matriks yang akan dibuat dan
   jumlah pengguna yang akan mendapat assignment. Pengguna dengan role tak
   dikenal atau akun nonaktif **dilewati** (reported, bukan error).

2. **Verifikasi versi skema** pada target:

   ```bash
   go run ./cmd/perfbench schema
   ```

   Selisih kolom pada tabel `rbac_*` harus nol sebelum seed dijalankan.

3. **Terapkan (hanya di staging lebih dulu):**

   ```bash
   go run ./cmd/rbacseed --apply
   ```

   Penulisan berjalan dalam satu transaksi (gagal = tidak ada yang berubah).
   Perintah bersifat **idempoten**: menjalankan ulang hanya menambahkan yang
   hilang (role/permission/matriks), pengguna yang sudah punya assignment
   dilewati.

4. **Verifikasi setelah seed:**

   ```sql
   SELECT code, tier_level FROM rbac_roles ORDER BY code;
   SELECT r.code, count(*) FROM rbac_role_permissions rp
     JOIN rbac_roles r ON r.id = rp.role_id GROUP BY r.code;
   SELECT s.user_id, r.code, s.unit_id FROM rbac_user_role_scopes s
     JOIN rbac_roles r ON r.id = s.role_id ORDER BY r.code;
   ```

   Assignment admin transcends organisasi (`unit_id` kosong); operator/atasan/
   pegawai memakai unit penugasannya sebagai informasi cakupan.

5. **Uji penegakan (opsional, staging):** hapus satu grant role-permission,
   lalu panggil endpoint terkait memakai sesi pengguna yang terdampak —
   hasilnya harus `403` meski grant statis mengizinkan (RBAC otoritatif).
   Kembalikan grant dengan `go run ./cmd/rbacseed --apply`.

6. **Produksi:** hanya setelah staging bersih, gate rilis terpenuhi, dan
   persetujuan pemilik tertulis (`docs/STAGING-RELEASE-CHECKLIST.md`).

**Bila perlu membatalkan:** hapus assignment yang dibuat seed (baris
`rbac_user_role_scopes` untuk pengguna terkait). Tanpa assignment, evaluator
kembali ke grant statis. Hapus matriks/role hanya bila tidak ada pengguna
yang memakainya.

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

### 8.2 Publikasi dan jadwal pekerjaan ditolak (RabbitMQ self-hosted)

Gejala: pekerjaan terjadwal tidak dieksekusi, atau antrean `simpeg.jobs.dead` berisi pesan yang gagal diproses.

**Transport aktif:** self-hosted docker/podman — `RABBITMQ_URL` terisi, `cmd/jobsched` menerbitkan pesan ke exchange `simpeg.jobs`, dan server mengonsumsi antrean yang sama (`DEC-042`). **QStash tidak dipakai** (`DEC-102`). Periksa antrean dead-letter:

```bash
docker compose -f rabbitmq/docker-compose.yml exec rabbitmq \
  rabbitmqctl list_queues name messages
docker compose logs backend | grep mq_job_failed
```

**Tindakan:**

- RabbitMQ: bila `simpeg.jobs.dead` berisi pesan, periksa log `mq_job_failed` untuk penyebabnya, perbaiki, lalu jalankan ulang pekerjaannya dengan `jobsched publish <kind>`. Pesan dead-letter tidak kembali otomatis karena tick cron berikutnya memicu eksekusi baru. Pekerjaan idempoten sehingga pengulangan aman.

### 8.3 Email tidak terkirim

Gejala: antrean keluar menumpuk dengan status `PENDING`, atau log worker
menyebut `smtp` pada pesan galat.

**Diagnosis:** mode keamanan SMTP ditentukan oleh `SMTP_SECURE` dan porta:

| Mode                 | Perilaku                                                                       | Porta lazim |
| -------------------- | ------------------------------------------------------------------------------ | ----------- |
| `SMTP_SECURE=true`   | TLS langsung sejak koneksi pertama                                             | 465         |
| `SMTP_SECURE=false`  | Koneksi polos, dinaikkan lewat STARTTLS bila tersedia                          | 587         |
| `SMTP_SECURE` kosong | Mode ditebak dari porta: 465 memakai TLS langsung, porta lain memakai STARTTLS | —           |

Galat `tls: first record does not look like a TLS handshake` berarti mode TLS
langsung dipakai pada porta submission. Galat `535` (atau `534`) dari Google
Workspace berarti kredensial ditolak; Workspace menuntut _app password_
(dengan 2FA), bukan sandi akun biasa — pada percobaan 2026-11-16 Gmail
menjawab `534 5.7.9 Application-specific password required`.

Bentuk `SMTP_FROM` boleh berupa alamat polos maupun bentuk baku
`"Nama Tampilan" <alamat@domain>`. Bentuk kedua dipakai apa adanya pada
header `From`, sedangkan perintah SMTP `MAIL FROM` memakai alamat polosnya.

**Tindakan:** samakan `SMTP_SECURE` dengan porta server, lalu gunakan _app
password_ yang sah bila penyedia adalah Google Workspace.

## 8A. Pemeriksaan SLO terjadwal

**Gejala:** alert SLO tidak pernah datang, atau alert datang pada waktu yang
tidak wajar.

**Diagnosis:** tiga hal harus benar:

1. `SLO_ENABLED=true` mengaktifkan pengoleki metrik di dalam proses backend.
2. Pemeriksaan `slo.evaluate` dijadwalkan lewat RabbitMQ (`cmd/jobsched` → antrean `simpeg.jobs`): `SLO_EVALUATE_ENABLED=true` pada penjadwal dan pesan `slo.evaluate` mengalir setiap 5 menit (bagian 8B); tidak ada pendaftaran cron eksternal (QStash tidak dipakai, `DEC-102`).
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
gaji H-1, sejak `DEC-078`; sebelumnya H+1) tidak berjalan, padahal deployment memakai docker/podman dengan
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

| Sumber                                                               | Isi                                                              | Dipakai untuk                                                 |
| -------------------------------------------------------------------- | ---------------------------------------------------------------- | ------------------------------------------------------------- |
| Log keamanan (`msg`: `auth_failed`, `access_denied`, `rate_limited`) | `client_ip`, `path`, `status`, `request_id`, `user_id`, `role`   | Deteksi volume dan korelasi cepat                             |
| Log `auth_failed` runtime Better Auth (frontend)                     | `path`, `reason`, `ts` (tanpa `client_ip`/kredensial, `DEC-050`) | Menelusuri kegagalan login email/OAuth, termasuk Google OAuth |
| Jejak audit aksi `authz.denied`                                      | aktor, unit cakupan, izin diminta, pola route                    | Menelusuri pihak yang mencoba akses di luar haknya            |
| Log keamanan `webhook_signature_rejected`                            | `job`, `path`, `request_id`                                      | Mendeteksi callback tidak sah (transport lama QStash — tidak dipakai)  |

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
5. Jika `webhook_signature_rejected` muncul berulang, periksa konfigurasi penjadwalan RabbitMQ/dead-letter; tidak ada callback QStash yang harus diuji karena tidak dipakai (`DEC-102`).

**Ambang alert awal.** Nilai berikut adalah titik awal untuk disetel operator,
belum berdasarkan baseline volume:

| Event                        | Syarat alert awal                                          |
| ---------------------------- | ---------------------------------------------------------- |
| `auth_failed`                | lebih dari 20 kejadian dari satu `client_ip` dalam 5 menit |
| `access_denied`              | lebih dari 50 kejadian dari satu `client_ip` dalam 5 menit |
| `rate_limited`               | muncul berulang pada path autentikasi                      |
| `webhook_signature_rejected` | lebih dari 3 kejadian dalam 10 menit                       |

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

**Gejala:** permintaan ke penyedia hari libur, Redis, atau SMTP menyebabkan
timeout berulang, dan antrean keluar (outbox) menumpuk.

**Diagnosis:**

- API hari libur: kalender dibaca dari tabel `HariLibur` lokal, tidak
  bergantung pada penyedia runtime; penyegaran hanya dipicu admin.
- SMTP: periksa log `smtp` pada pesan galat worker outbox (bagian 8.3).
- RabbitMQ / antrean pekerjaan terjadwal: lihat bagian 8.2/8B.
- Status antrean keluar (agregat, tanpa payload/identitas): `GET /api/v1/admin/ops/status`
  (permission `ops:read`, admin) mengembalikan jumlah pesan per status
  (`PENDING`/`PROCESSING`/`DONE`/`FAILED`), usia pesan tertua yang belum
  selesai, dan jumlah `FAILED` per jenis pekerjaan. Gunakan ini untuk
  memantau keterlambatan antrean dan kegagalan notifikasi/pengiriman.

**Tindakan:** backend menggunakan outbox persisten di PostgreSQL sehingga
pesan tidak hilang saat dependency eksternal down. Worker mencoba ulang
dengan backoff eksponensial yang dibatasi; setelah jumlah percobaan maksimum,
pesan pindah ke status `FAILED` sebagai dead-letter. Circuit breaker belum
diimplementasikan pada rilis ini karena integrasi eksternal (RabbitMQ, SMTP,
API hari libur) belum semuanya teruji pada produksi. Tambahkan circuit
breaker saat integrasi tersebut dipakai pada produksi.

## 8E. Uji backup dan restore database

**Status backup Supabase Free harus dipahami dengan benar:** project Free tidak mendapat backup database harian otomatis dan tidak mendapat PITR. Dokumentasi Supabase menyarankan project Free mengekspor data secara berkala melalui Supabase CLI `db dump` dan menyimpan salinan off-site. Backup harian otomatis tersedia pada plan Pro/Team/Enterprise; PITR merupakan add-on plan berbayar dengan persyaratan compute tersendiri. Jangan menulis "backup platform harian aktif" atau "PITR tersedia" pada SIMPEG Free.

**Target awal:** `RPO_HOURS=24`, `RTO_HOURS=4` (nilai kebijakan, belum terbukti pada project produksi). Pada tier Free, RPO 24 jam hanya mungkin didekati bila ekspor database berhasil dijalankan sekurang-kurangnya setiap 24 jam, artefak disimpan terenkripsi di lokasi terpisah, dan kegagalan job terdeteksi. Tidak ada jaminan PITR per detik.

**Bukti drill terbaru (2026-11-16, `DEC-100`):** staging/disposable, bukan project Supabase produksi:
- Dump `pg_dump` sekitar 59 detik, 2.555.361 byte.
- Restore ke `postgres:16-alpine` sekitar 5 detik; empat galat terkait `supabase_vault`/`transaction_timeout` dicatat sebagai komponen khusus Supabase, bukan dianggap otomatis aman untuk seluruh lingkungan.
- Verifikasi dibanding sumber: 89 tabel, 89 RLS, Goose 33, `user`=102 (100 sintetis + 2 pilot), `gaji`=301, `session`=30, `account`=2, `auditLog`=50.161.
- Artefak sementara dibersihkan.
- **Belum diuji:** ekspor terjadwal berulang dari project Free aktual, restore ke project disposable Supabase dengan konfigurasi target, verifikasi akses aplikasi pascapulih, serta RPO/RTO produksi.

### Ekspor yang benar untuk Supabase Free

Supabase CLI `db dump` default menghasilkan schema dump saja; tidak menyertakan row data atau custom roles. Untuk backup yang berguna, siapkan minimal tiga artefak:

1. `schema.sql` — schema aplikasi.
2. `data.sql` — seluruh row data yang perlu dipulihkan.
3. `roles.sql` — custom roles tingkat cluster bila project memang memilikinya.

CLI saat ini memakai `--remote=<project-ref>`. Flag `--linked` masih didukung pada CLI v3, tetapi sudah deprecated dan dihapus pada CLI v4. Perintah berikut membutuhkan Supabase CLI dan Docker yang tersedia di mesin backup, serta hak akses yang sah:

```powershell
# Project ref bukan password. Login lewat `supabase login` atau credential store CLI.
$ProjectRef = "<SUPABASE_PROJECT_REF>"
$BackupDir = "D:\simpeg-backups\$(Get-Date -Format 'yyyyMMdd-HHmmss')"
New-Item -ItemType Directory -Force -Path $BackupDir | Out-Null

supabase db dump "--remote=$ProjectRef" --file (Join-Path $BackupDir "schema.sql")
if ($LASTEXITCODE -ne 0) { throw "Schema dump gagal" }

supabase db dump "--remote=$ProjectRef" --data-only --file (Join-Path $BackupDir "data.sql")
if ($LASTEXITCODE -ne 0) { throw "Data dump gagal" }

supabase db dump "--remote=$ProjectRef" --role-only --file (Join-Path $BackupDir "roles.sql")
if ($LASTEXITCODE -ne 0) { throw "Role dump gagal" }

Get-ChildItem $BackupDir -File | Get-FileHash -Algorithm SHA256 |
  Format-Table Path, Hash
```

Perhatikan batas cakupan:
- CLI mengecualikan schema Supabase yang dikelola platform, termasuk `auth`, `storage`, dan schema yang dibuat extension. Periksa dengan teliti bila ada fitur aplikasi yang bergantung pada schema tersebut.
- File yang disimpan melalui Supabase Storage API tidak ikut dipulihkan oleh database dump karena database hanya menyimpan metadata objek. SIMPEG menggunakan Cloudinary untuk berkas dokumen; inventaris, akses, dan pemulihan objek Cloudinary harus memiliki prosedur/provider drill tersendiri.
- Data dump berisi data pegawai/keuangan. Lokasi backup harus terenkripsi, aksesnya terbatas, checksum diverifikasi, dan salinannya disimpan off-site. Jangan simpan artefak di repository atau lampirkan ke issue/chat.
- Sebelum restore ke project baru, audit default privileges target. Dokumentasi Supabase memperingatkan tabel dapat mewarisi privilege default target; privilege `anon`/`authenticated` harus diselaraskan dengan model keamanan aplikasi sebelum data dibuka.

### Prosedur restore drill

1. **Verifikasi project sumber:** catat project ref, tier, region, waktu snapshot/ekspor, versi CLI, dan konfigurasi compute. Jangan menyimpan URL koneksi atau password di formulir.
2. **Siapkan target terpisah:** gunakan project/database disposable yang disetujui. Jangan restore dengan `--clean` ke database produksi.
3. **Ambil artefak:** unduh set schema/data/roles dari lokasi off-site, verifikasi checksum dan hasil decrypt pada host yang berwenang.
4. **Tinjau SQL sebelum eksekusi:** identifikasi owner, grant/default privileges, extension, dan objek yang tidak termasuk dump. Pastikan perbedaan tier/extension tidak mengubah kebijakan akses.
5. **Restore roles, schema, lalu data** (sesuaikan dengan isi hasil dump dan hak akses target):
   ```powershell
   psql "$env:TARGET_DATABASE_URL" -v ON_ERROR_STOP=1 -f ".\roles.sql"
   psql "$env:TARGET_DATABASE_URL" -v ON_ERROR_STOP=1 -f ".\schema.sql"
   psql "$env:TARGET_DATABASE_URL" -v ON_ERROR_STOP=1 -f ".\data.sql"
   ```
   Jika project tidak menggunakan custom roles, `roles.sql` boleh kosong/tidak diaplikasikan. Periksa dokumentasi SQL hasil dump; jangan mengasumsikan urutan tersebut menggantikan review dependensi role/owner.
6. **Verifikasi teknis:** cek koneksi, status seluruh migrasi Goose, jumlah tabel/index/constraint/RLS, jumlah row tabel penting, integrity constraints, dan sampel query read-only. Jalankan binary backend dengan konfigurasi target uji dan pastikan `GET /ready` = 200. Uji login/authorization tanpa membuka data pilot ke pengguna.
7. **Ukur RPO/RTO:** catat timestamp data terakhir pada sumber, timestamp artefak, waktu mulai restore, waktu DB siap, dan waktu API siap. RPO = jarak antara insiden/snapshot titik data dan data terakhir yang berhasil dipulihkan; RTO = durasi sampai layanan siap digunakan, bukan sekadar sampai `psql` selesai.
8. **Bersihkan:** hapus project disposable dan artefak plaintext setelah verifikasi; simpan hanya laporan/checksum tanpa PII atau rahasia.

**Formulir dan helper:** `docs/backup-restore-drill.md`, `scripts/backup-restore-drill.ps1` (pg_dump/pg_restore generik), dan `scripts/supabase-free-backup.ps1` (ekspor schema/data/roles khusus project Free). Helper tidak melakukan backup otomatis/off-site, tidak mengaktifkan PITR, dan tidak menggantikan restore drill.

**Kriteria penutupan produksi:** setidaknya beberapa siklus ekspor terjadwal sukses; kegagalan backup menghasilkan alert; salinan terenkripsi berada di lokasi off-site yang terpisah dari project Supabase; restore drill ke project disposable berhasil dan hasil RPO/RTO aktual dicatat. Sampai bukti itu ada, item backup produksi tetap belum selesai.

Referensi resmi:
- Supabase Database Backups: https://supabase.com/docs/guides/platform/backups
- Supabase CLI `db dump`: https://supabase.com/docs/reference/cli/supabase-issue-feature

## 9. Dashboard, monitoring, dan alert

**Gejala:** operator tidak mengetahui layanan mana yang gagal, dampak ke
pengguna, atau langkah pemulihannya.

**Sumber sinyal yang tersedia di aplikasi (sudah ada):**

- **Health:** `/health` (liveness) dan `/ready` (readiness dependency wajib,
  saat ini PostgreSQL).
- **Log terstruktur:** JSON satu baris ke stdout dengan `X-Request-ID`; redaksi
  kredensial/body/query (`internal/observability`). Peristiwa keamanan
  (`auth_failed`, `access_denied`, `rate_limited`, `webhook_signature_rejected`)
  dan pekerjaan (`mq_job_*`, `outbox_*`, `slo_*`) tercatat sebagai event.
- **Metrik SLO (agregat):** `sloMetric` di PostgreSQL, halaman admin `/admin/slo`
  dan `GET /api/v1/admin/slo/metrics` (`slo:read`, admin). Mencakup error rate
  dan latensi (rata-rata/p95/p99/maks) per rute pada jendela 24 jam/7 hari.
  Bergantung `SLO_ENABLED=true`.
- **Status operasional antrean keluar:** `GET /api/v1/admin/ops/status`
  (`ops:read`, admin) memuat agregat outbox per status, usia pesan tertua yang
  belum selesai, dan kegagalan per jenis — sinyal keterlambatan antrean dan
  kegagalan notifikasi/pengiriman asinkron (`DEC-065`).

Dasbor internal adalah titik awal, bukan platform monitoring penuh; cakupan
saturasi, health dependency eksternal, metrik Redis/SMTP/storage (bukan QStash/S3 — keduanya tidak dipakai), dan backlog broker RabbitMQ memerlukan
pengumpulan di infrastruktur.

**Stack monitoring/alerting yang dipilih (`DEC-065`, target: Docker + RabbitMQ
self-hosted + stdout JSON + Supabase):** observability self-hosted berbasis
Docker (templat awal di `observability/`): Prometheus + Alertmanager + Grafana,
Loki + Promtail/Grafana Alloy untuk log stdout JSON, node-exporter & cAdvisor
untuk saturasi host/container, blackbox-exporter untuk probe `/health`/`/ready`
dan endpoint kritis, serta metrik RabbitMQ via plugin/endpoint Prometheus
management. Aturan alert dipasang di Alertmanager/Prometheus; penerima dan
eskalasi ditetapkan pemilik (adhitiad/admin). **Pemasangan, wiring ke target,
dan uji tembak alert masih harus dibuktikan di infrastruktur.**

Panduan pemasangan, konfigurasi, dan uji tembak alert ada di
`observability/README.md`. Sumber sinyal aplikasi yang dipakai stack itu:
`GET /api/v1/admin/slo/metrics` (`slo:read`) untuk agregat error/latensi,
`GET /api/v1/admin/ops/status` (`ops:read`) untuk agregat antrean keluar, dan
log JSON terstruktur ke stdout. Status operasional antrean keluar yang dibaca
`/admin/ops/status` bersifat agregat per status, usia pesan tertua, dan
kegagalan per jenis; payload, dedupe key, dan penerima tidak pernah diekspos.

**Ambang alert awal (titik awal, wajib disesuaikan dengan baseline produksi):**

| Metrik                      | Sumber                                                             | Ambang alert awal                                             |
| --------------------------- | ------------------------------------------------------------------ | ------------------------------------------------------------- |
| Error rate API              | `sloMetric.errorCount / requestCount` atau metrik HTTP teragregasi | > 1% dalam 5 menit (kritikal > 5%)                            |
| Latency p95                 | histogram bucket `sloMetric` / evaluasi `slo.evaluate`             | > 800 ms dalam 5 menit                                        |
| Saturasi host/container     | node-exporter/cAdvisor (CPU, memori, disk)                         | CPU/memori tinggi berkelanjutan (mis. > 85% selama 5 menit)   |
| Pool/koneksi PostgreSQL     | metrik pool aplikasi/DB Supabase                                   | > 80% batas (kritikal > 90%)                                  |
| Antrean keluar (outbox)     | `GET /api/v1/admin/ops/status` atau query `outboxMessage`          | `FAILED` > 0 (kritikal > 10); `PENDING`/usia tertua meningkat |
| Antrean broker              | metrik RabbitMQ (`simpeg.jobs`, `simpeg.jobs.dead`)                | pesan di dead-letter > 0; backlog bertumbuh                   |
| Kegagalan notifikasi/SMTP   | `FAILED` outbox per kind `email.*`, log `smtp`                    | > 0 pesan permanen                             |
| Health dependency           | probe `/ready`, blackbox/health Redis/RabbitMQ/Supabase            | tidak sehat/timeout berulang                                  |

Log keamanan (`auth_failed`, `access_denied`, `rate_limited`, `webhook_signature_rejected`)
dan jejak audit (`authz.denied`) dapat diekspor ke SIEM/log alerting untuk
deteksi insiden (lihat bagian insiden keamanan).

## 9A. Retensi dan operasi log produksi

**Kebijakan (`DEC-065`, pemilik: adhitiad/admin):**

- **Sumber:** log aplikasi JSON ke stdout (container Docker), dikumpulkan oleh
  Promtail/Grafana Alloy ke Loki (atau stack log setara pada target).
- **Retensi log agregasi:** awal **30 hari** (dapat diubah sesuai kapasitas;
  nilai efektif wajib dicatat pada konfigurasi deployment). Retensi ini
  terpisah dari retensi data di database.
- **Retensi data terjadwal di database:** jejak audit `AUDIT_RETENTION_DAYS`
  (bawaan 90 hari, pembersihan terjadwal) dan metrik SLO `SLO_RETENTION_DAYS`
  (bawaan 30 hari).
- **Akses:** dibatasi admin/operator yang berwenang sesuai kebijakan
  organisasi; jangan menyalin log ke saluran publik.
- **Isi & redaksi:** log tidak boleh memuat kredensial, body permintaan, atau
  nilai query string; penelusuran insiden memakai `request_id`, `msg`, `path`,
  `status`, dan metadata non-sensitif yang memang dicatat.
- **Operasi:** pastikan label/field untuk korelasi (`request_id`, service/job)
  konsisten, rotasi/retensi di-backend log aktif, dan prosedur pencarian
  insiden diuji. **Konfigurasi agregasi/retensi/akses pada deployment target
  masih harus dipasang dan diverifikasi.**

## 10. Autentikasi dan Sesi

### 10.1 Model sesi yang didukung

Backend mendukung dua jenis sesi:

| Jenis            | Penyimpanan                    | Revokasi                                          | Keterangan                                                     |
| ---------------- | ------------------------------ | ------------------------------------------------- | -------------------------------------------------------------- |
| JWT Better Auth  | Better Auth runtime (frontend) | Milik runtime Better Auth                         | Verifikasi via `BETTER_AUTH_SECRET` dengan `golang-jwt/jwt/v5` |
| Sesi opaque GORM | Tabel `session` di PostgreSQL  | `POST /api/auth/logout` menghapus baris `session` | Resolusi via `auth.CombinedSessionResolver`                    |

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
