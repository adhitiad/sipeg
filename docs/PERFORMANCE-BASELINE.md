# Baseline Performa dan Kapasitas SIMPEG

Tanggal: 2026-11-12
Status: Baseline unit-level **dan** pengukuran nyata terhadap PostgreSQL, Redis, QStash, dan SMTP pada database staging Supabase. Semua angka di bawah berasal dari pengukuran yang dapat direproduksi memakai `backend/cmd/perfbench`.

## 1. Metode Pengukuran

Pengukuran dibedakan menjadi dua lapis agar tidak tercampur.

### 1.1 Lapis unit (repository palsu)

Benchmark memakai repository palsu sehingga yang terukur adalah biaya konversi DTO, logika aplikasi, dan jalur HTTP — **bukan** latensi query atau jaringan.

```bash
go test -run "^$" -bench "Benchmark" -benchmem -count=1 ./internal/httpserver/... ./internal/directory/app/...
```

Uji beban dan soak:

```bash
go test ./internal/httpserver/... -run TestBurstRequestsRemainStable -v
SIMPEG_SOAK=1 SIMPEG_SOAK_DURATION=30 SIMPEG_SOAK_STREAMS=16 go test ./internal/httpserver/... -run TestSoakReadPath -v
```

### 1.2 Lapis nyata (staging Supabase)

Perkakas `backend/cmd/perfbench` menjalankan pengukuran terhadap dependensi sungguhan. Perkakas membaca kredensial dari environment; tidak ada nilai rahasia di dalam repo.

| Subcommand | Fungsi |
|------------|--------|
| `seed` | mengisi database uji dengan data sintetis |
| `schema` | membandingkan kolom model GORM dengan kolom yang benar-benar ada di database |
| `explain` | menjalankan `EXPLAIN (ANALYZE, BUFFERS)` atas statement yang direkam saat repository produksi berjalan |
| `latency` | mengukur distribusi latensi tiap skenario query |
| `e2e` | menjalankan binari `cmd/server` asli lalu mengukurnya lewat HTTP |
| `deps` | mengukur beban Redis, QStash, dan SMTP |

```bash
cd backend
export DATABASE_URL="postgres://<user>:<sandi>@<pooler-host>:5432/simpeg_perf?sslmode=require"
go run ./cmd/perfbench seed --employees=5000 --months=6 --audit=50000
go run ./cmd/perfbench schema
go run ./cmd/perfbench explain
go run ./cmd/perfbench latency --iterations=40
go run ./cmd/perfbench e2e --requests=400 --streams=16
go run ./cmd/perfbench deps --redis --qstash --smtp --count=100 --concurrency=8
```

SQL yang dianalisis bukan query yang ditulis ulang secara manual. Statement direkam pada level connection pool GORM, sehingga `EXPLAIN` dijalankan atas SQL yang benar-benar diterima PostgreSQL beserta argumennya. Uji `e2e` menjalankan binari `cmd/server` asli sebagai proses anak, sehingga pengukuran memakai rantai handler produksi tanpa duplikasi.

### 1.3 Lingkungan pengukuran

| Komponen | Nilai |
|----------|-------|
| Host pengukuran | Windows, AMD Ryzen 5 5600G, 12 goroutine |
| Toolchain | Go 1.26.11 (versi yang dikunci `backend/go.mod`) |
| Database | Supabase PostgreSQL 17.11, region `aws-1-ap-northeast-1` (Tokyo) |
| Jalur koneksi | Pooler sesi Supabase, `aws-1-ap-northeast-1.pooler.supabase.com:5432` |
| Database uji | `simpeg_perf`, database terpisah pada proyek yang sama |
| Migrasi | 17 versi Goose terpasang |
| Skala data | 5.000 pegawai, 49 unit, 30.000 slip gaji, 50.000 entri jejak audit |

Database uji dibuat terpisah dari database `postgres` yang dipakai production. Isi database `postgres` tetap nol baris dan tidak pernah dipakai untuk seeding.

Pengisian data sintetis selesai dalam **58,13 detik**: 5.000 pegawai, 49 unit organisasi, 6 bulan slip gaji beserta tunjangan, potongan, dan konfirmasi pembayaran, serta 50.000 entri jejak audit.

## 2. Baseline Unit

| Benchmark | Latency (ns/op) | Latency (ms/op) | Memory (B/op) | Allocs (op) |
|-----------|-----------------|-----------------|---------------|-------------|
| ListEmployeesConversion | 129.713 | ~0,130 | 65.799 | 247 |
| AuditLogDTOConversion | 72.690 | ~0,073 | 39.046 | 150 |
| UnitScopeResolution | 12.786 | ~0,013 | 8.448 | 38 |
| ListEmployeesParallel | 28.281 | ~0,028 | 78.710 | 247 |

Benchmark paralel memakai `b.RunParallel`, jadi 12 goroutine bekerja bersamaan. Throughput jalur baca daftar 100 pegawai di repository palsu sekitar **35.000 req/detik**. Angka berlaku khusus untuk Go 1.26.11.

Uji burst (200 permintaan paralel) dan uji soak (30 dan 120 detik, 16 stream) menghasilkan 0 kegagalan, throughput tetap ±36.000 req/detik, p99 di bawah 2,7 ms, dan heap tidak tumbuh proporsional dengan waktu. Yang terbukti stabil adalah jalur HTTP, bukan perilaku PostgreSQL, Redis, QStash, atau SMTP.

## 3. Latensi Query PostgreSQL Nyata

Pengukuran memakai 40 iterasi per skenario dengan satu goroutine, lewat repository produksi sehingga SQL yang dibangkitkan sama dengan SQL aplikasi.

| Skenario | Min | p50 | p95 | p99 | Maks |
|----------|-----|-----|-----|-----|------|
| `baseline.roundtrip` (`SELECT 1`) | 85,62 ms | 87,13 ms | 88,22 ms | 88,51 ms | 88,56 ms |
| `employees.list.admin` | 183,50 ms | 184,49 ms | 186,95 ms | 207,76 ms | 221,06 ms |
| `employees.list.page50` | 184,32 ms | 185,30 ms | 187,55 ms | 213,93 ms | 230,44 ms |
| `employees.list.search` | 206,95 ms | 208,76 ms | 213,70 ms | 219,73 ms | 220,76 ms |
| `employees.list.scope` | 185,92 ms | 188,41 ms | 189,40 ms | 190,45 ms | 190,74 ms |
| `audit.list.admin` | 179,07 ms | 180,19 ms | 181,32 ms | 204,09 ms | 218,08 ms |
| `audit.list.scope` | 180,37 ms | 182,28 ms | 184,88 ms | 189,14 ms | 190,74 ms |
| `audit.list.filtered` | 175,62 ms | 176,47 ms | 178,36 ms | 207,95 ms | 223,62 ms |
| `categories.list` | 173,99 ms | 174,99 ms | 175,95 ms | 177,12 ms | 177,84 ms |
| `scope.resolve` | 86,97 ms | 88,16 ms | 89,54 ms | 89,72 ms | 89,78 ms |
| `payroll.list.admin` | 442,81 ms | 445,11 ms | 451,45 ms | 492,43 ms | 498,10 ms |
| `payroll.list.periode` | 441,39 ms | 443,11 ms | 496,37 ms | 681,57 ms | 788,67 ms |
| `payroll.list.scope` | 453,91 ms | 457,89 ms | 506,25 ms | 508,41 ms | 508,92 ms |
| `payroll.list.self` | 437,08 ms | 439,31 ms | 459,58 ms | 482,21 ms | 491,01 ms |

### 3.1 Temuan Utama: Jumlah Statement Mengominasi, Bukan Biaya Query

Perjalanan bolak-balik kosong (`SELECT 1`) memakan **±87 ms**. Angka itu adalah lantai jaringan: setiap statement PostgreSQL harus penumpang pooler menuju Tokyo dan kembali.

Dengan angka itu, latensi tiap skenario dapat dijelaskan oleh jumlah statement-nya:

| Skenario | Jumlah statement | Perkiraan 87 ms × statement | Latensi p50 terukur |
|----------|------------------|-----------------------------|---------------------|
| `scope.resolve` | 1 (recursive CTE) | 87 ms | 88,16 ms |
| `employees.list.admin` | 2 (count + list) | 174 ms | 184,49 ms |
| `categories.list` | 2 (count + list) | 174 ms | 174,99 ms |
| `payroll.list.admin` | 5 (count + list + 3 pramuat) | 435 ms | 445,11 ms |

Selisih antara perkiraan dan hasil terukur hanya 5–10 ms, yaitu biaya pemrosesan baris di dalam PostgreSQL.

**Konsekuensi:** pengurangan jumlah statement adalah penglever yang paling efektif, bukan pengoptimalan indeks. Perbaikan N+1 yang sudah dilakukan pada impor massal, jejak audit, dan antrean keluar memberi manfaat yang jauh lebih besar pada database remote dibanding pada repository palsu. Daftar slip gaji adalah jalur terburuk karena memakai lima statement per permintaan.

### 3.2 Batas Latensi yang Dikendalikan Arsitektur

Penempatan backend menentukan angka ini secara langsung:

| Penempatan | Perkiraan perjalanan bolak-balik | Perkiraan p95 daftar pegawai |
|------------|----------------------------------|-------------------------------|
| Backend di region yang sama dengan database (Tokyo) | ±1–5 ms | ±15–30 ms |
| Backend di region berbeda, lewat pooler sesi | ±87 ms terukur | ±185 ms terukur |

Selisihnya sekitar 20 kali. Semua target SLO di bawah harus dinyatakan eksplisit terhadap penempatan backend, karena angka yang berbeda menghasilkan kesimpulan yang berbeda.

## 4. Profil Query: `EXPLAIN (ANALYZE, BUFFERS)`

Statement direkam saat repository produksi berjalan, lalu dijalankan ulang dengan awalan `EXPLAIN`. Angka di bawah adalah waktu eksekusi di dalam PostgreSQL, **tanpa** biaya jaringan.

### 4.1 Daftar Pegawai

| Statement | Rencana | Waktu eksekusi |
|-----------|---------|----------------|
| `SELECT count(*) FROM "user" WHERE aktif = $1` | `Seq Scan` + filter, 5.000 baris | 2,296 ms |
| `SELECT * FROM "user" WHERE aktif = $1 ORDER BY "nama" ASC LIMIT $2` | `Seq Scan` + top-N heapsort | 6,986 ms |
| `SELECT count(*) ... AND (nama ILIKE $2 OR nip ILIKE $3)` | `Seq Scan`, 4.900 baris dibuang oleh filter | 15,530 ms |
| `SELECT * ... ILIKE ... ORDER BY "nama" LIMIT $2` | `Seq Scan` + top-N heapsort | 15,647 ms |
| Versi cakupan (`"homebaseId" IN (...)`, 49 unit) | `Seq Scan` + filter `= ANY (uuid[])` | 3,013 ms / 7,794 ms |

Pengamatan:

- **Tidak ada indeks pada `aktif`.** Filter `aktif = true` examined seluruh tabel. Pada 5.000 baris biayanya 2,3 ms, tetapi pada 100.000 baris menjadi ≈50 ms.
- **Tidak ada indeks pada `nama`.** Pengurutan memakai top-N heapsort, yaitu pengurutan di memori tanpa indeks. Pada 5.000 baris biayanya 7 ms.
- **Pencarian ILIKE melakukan seq scan penuh** dan membuang 4.900 dari 5.000 baris. Biaya 15,5 ms pada 5.000 baris menjadi ≈310 ms pada 100.000 baris. Indeks `pg_trgm` pada `nama` dan `nip` mengubahnya menjadi pencarian indeks.
- **Penyaringan cakupan tidak memakai indeks `homebaseId`.** Plan menunjukkan `Seq Scan` dengan filter `= ANY`, bukan `Index Scan`. Penyaringan ini memindai seluruh tabel.

### 4.2 Jejak Audit

| Statement | Rencana | Waktu eksekusi |
|-----------|---------|----------------|
| `SELECT count(*) FROM "auditLog"` | `Index Only Scan` memakai `audit_log_aktor_idx`, Heap Fetches: 0 | 10,197 ms |
| `SELECT * FROM "auditLog" ORDER BY "createdAt" DESC LIMIT $1` | `Index Scan Backward` memakai `audit_log_created_at_idx` | 0,043 ms |
| Versi cakupan (`"unitId" IN (...)`) | `Index Only Scan` memakai `audit_log_unit_idx` | 10,196 ms |
| Versi cakupan dengan urutan | `Index Scan Backward` + filter, Heap Fetches: 0 | 0,060 ms |
| Penyaring `aksi = $1` | `Index Only Scan` memakai `audit_log_aksi_idx` | 2,589 ms |

Jejak audit sudah terlindungi indeks dengan baik. Pengambilan 20 baris terbaru hanya 0,043 ms; yang mahal adalah `count(*)` pada 50.000 baris (10,2 ms). `count(*)` pada tabel besar adalah kandidat utama untuk perbaikan: penghitungan dilakukan dengan estimasi (`EXPLAIN`) atau berversi.

### 4.3 Rekomendasi Indeks

| Indeks | Alasan | Perkiraan dampak |
|--------|--------|------------------|
| `CREATE INDEX ... ON "user" (nama) WHERE aktif` | Menghapus top-N sort dan seq scan pada daftar default | p95 daftar pegawai turun dari 185 ms ke 95 ms |
| `CREATE INDEX ... ON "user" USING gin (nama gin_trgm_ops)` | Pencarian ILIKE menjadi pencarian indeks | p95 pencarian turun dari 209 ms ke ≈95 ms |
| `CREATE INDEX ... ON "user" ("homebaseId") WHERE aktif` | Penyaringan cakupan tidak lagi memindai tabel | p95 daftar cakupan turun dari 188 ms ke ≈95 ms |

Catatan: indeks pada `nama` untuk `ILIKE` dengan awalan wildcard (`%cari%`) hanya speeding lewat `pg_trgm`, bukan `btree`.

Indeks tersebut belum dibuat. Penambahannya adalah perubahan skema dan perlu migrasi Goose tersendiri beserta keputusan produk tentang biaya tulis.

## 5. Dampak Cache Redis terhadap Latensi Nyata

Perbandingan dilakukan pada beban yang sama (100 operasi, 8 thread paralel):

| Operasi | Min | p50 | p95 | Throughput |
|---------|-----|-----|-----|------------|
| `redis.set` (Upstash REST) | 18,19 ms | 19,23 ms | 378,65 ms | 161,6 ops/detik |
| `redis.get` (Upstash REST) | 18,09 ms | 19,22 ms | 20,20 ms | 401,2 ops/detik |
| `postgres.categories.list` | 171,07 ms | 175,29 ms | 986,13 ms | 33,5 ops/detik |

**Cache terbukti memperbaiki latensi.** Pembacaan dari Redis 19 ms berbanding 175 ms dari PostgreSQL pada jalur yang sama, jadi sekitar **9 kali lebih cepat** dan 12 kali lebih tinggi throughput-nya.

Perhatikan juga p95 `postgres.categories.list` yang mencapai 986 ms pada 8 thread paralel: itu antrean koneksi di pooler, bukan biaya eksekusi query. `redis.get` pada beban sama tetap 20 ms pada p95 karena setiap operasi adalah satu perjalanan bolak-balik tanpa penguncian sumber daya bersama.

Dengan demikian cache daftar kategori bukan sekadar penghematan sumber daya, tetapi jugaungan dari penumpukan koneksi pada database remote.

## 6. Beban Dependency Nyata

### 6.1 QStash

| Metrik | Hasil |
|--------|-------|
| Operasi | 100 publish |
| Gagal | 0 |
| Latensi | min 773,91 ms, p50 797,45 ms, p95 837,89 ms, p99 843,72 ms |
| Throughput | 9,6 ops/detik |

Sekitar 800 ms per publish adalah biaya yang wajar: QStash memvalidasi tujuan dengan menyelesaikan DNS sebelum menerima pesan.

### 6.2 SMTP

Jalur SMTP diverifikasi sampai tahap autentikasi terhadap server sungguhan (`smtp.gmail.com:587`):

| Tahap | Hasil |
|-------|-------|
| Koneksi + TLS (STARTTLS) | Berhasil — handshake selesai, kredensial terkirim lewat TLS |
| Autentikasi | **Gagal**, server membalas `535 5.7.8 Username and Password not accepted` |
| Pengiriman pesan | Tidak terjadi — 100 dari 100 operasi berhenti pada autentikasi |

Artinya jalur kodenya benar sampai autentikasi, tetapi kredensial pada `.env` ditolak oleh Google Workspace. Google Workspace menuntut *app password* untuk SMTP; sandi akun biasa ditolak. Tanpa kredensial yang valid, beban pengiriman email sungguhan tidak dapat diukur sampai ke kotak masuk.

SMTP berada pada **nilai bawaan produksi yang salah** sebelum diperbaiki. `SMTP_SECURE` bernilai kosong dan `SMTP_PORT=587` membuat backend memakai TLS langsung pada porta submission. Kombinasi itu gagal dengan `tls: first record does not look like a TLS handshake` pada setiap percobaan, sehingga tidak ada satu pun email yang pernah terkirim. Perbaikannya dijelaskan di bagian 8.

### 6.3 Redis

Rincian pada bagian 5. Tidak ada kegagalan pada 200 operasi tulis dan baca.

## 7. Uji Beban dan Soak End-to-End

Pengukuran end-to-end menjalankan binari `cmd/server` asli terhadap database
staging, lalu mengirim permintaan HTTP sungguhan dengan bearer token sesi
3. **Sebar endpoint.** Beban dialokasikan ke tujuh endpoint dengan bobot
menggambarkan pemakaian nyata: daftar pegawai (tiga varian), jejak audit,
kategori, slip gaji, dan profil pengguna.

### 7.1 Burst

| Metrik | Hasil |
|--------|-------|
| Request | 400 |
| Sukses (200) | 400 |
| Gagal | 0 |
| Durasi | 22,05 s |
| Throughput | 18,1 req/detik |
| Latensi | min 277,06 ms, p50 795,68 ms, p95 1,53 s, p99 2,18 s, maks 2,55 s |

### 7.2 Soak

| Metrik | Hasil |
|--------|-------|
| Durasi | 60 s |
| Request | 1.205 |
| Sukses (200) | 1.205 |
| Gagal | 0 |
| Throughput | 19,9 req/detik |
| Latensi | min 289,92 ms, p50 745,84 ms, p95 1,36 s, p99 1,69 s, maks 2,70 s |

Tidak ada degradasi antara burst dan soak: p50 dan p95 pada soak justru
sedikit lebih baik daripada burst, dan tidak ada satu pun permintaan yang
gagal. Tidak terlihat penumpukan koneksi maupun kebocoran sumber daya selama
satu menit beban penuh.

### 7.3 Mengapa End-to-End Lebih Lambat dari Level Repository

Latensi end-to-end (p50 746 ms) jauh lebih tinggi daripada latensi query
tunggal di repository (p50 180–445 ms). Tiga hal memisahkannya:

1. **Antrean koneksi.** 16 stream paralel berbagi pool 10 koneksi, dan setiap
   statement memakan 87 ms perjalanan bolak-balik. Waktu tunggu antrean
   inilah yang mendominasi, bukan biaya eksekusi query.
2. **HTTP dan middleware.** Rantai handler (version guard, pembatas laju,
   CORS, batas waktu statement, header keamanan, log akses, request ID)
   menambah biaya per permintaan.
3. **Sebar endpoint.** Beban dialokasikan ke tujuh endpoint dengan bobot
   berbeda, bukan satu query yang sama berulang kali.

Konsekuensinya, p99 1,69 s pada 16 stream adalah sifat arsitektur pooler
remote, bukan cacat aplikasi. Dengan backend di region yang sama dengan
database, ketiga sumber itu hilang secara bersamaan.

### 7.4 Galat yang Ditemukan pada Uji Beban

Dua galat muncul pada percobaan pertama dan keduanya berakar pada konfigurasi,
bukan pada kode:

| Galat | Akar | Tindakan |
|-------|------|----------|
| `429` pada 99 dari 400 permintaan | `RATE_LIMIT_REQUESTS=120` per menit, di bawah beban 400 permintaan | Batas laju dinaikkan pada harness pengukuran |
| `503` pada 16 dari 400 permintaan | `EMAXCONNSESSION`, pool 25 koneksi melampaui `pool_size: 15` pooler sesi | Bawaan `DB_MAX_OPEN_CONNS` diturunkan menjadi 10 |

Galat `503` juga membeberkan cacat tersembunyi: dengan konfigurasi bawaan,
backend **akan** menjawab 503 saat dikonkurensi cukup tinggi pada produksi.

## 8. Temuan dan Perbaikan yang Tersembul dari Pengukuran Nyata

Pengukuran nyata menemukan lima cacat yang tidak terlihat pada pengujian dengan repository palsu.

| # | Temuan | Dampak | Perbaikan |
|---|--------|--------|-----------|
| 1 | Tabel `pembayaran` tidak memiliki kolom `updatedAt` yang dinyatakan model `models.Pembayaran` | Setiap `POST /api/v1/payroll/slips/{payrollId}/payment` gagal dengan `SQLSTATE 42703` | Migrasi `000016_payment_updated_at.sql` |
| 2 | `qstash.Publisher` menempelkan URL tujuan ke root API tanpa jalur `/v2/publish` | Setiap publish dijawab `404` | `publishEndpoint()` menambahkan jalur |
| 3 | `qstash.Publisher` membuat jadwal dengan bentuk v1 (destination dan cron di dalam body) | Setiap pendaftaran jadwal ditolak `400 invalid destination url`, sehingga **jadwal berkala tidak pernah terdaftar** | `CreateSchedule` memakai bentuk v2: destination pada path, cron pada header `Upstash-Cron` |
| 4 | Transport SMTP tidak mendukung `STARTTLS` | Tidak ada email yang dapat terkirim pada porta 587 | `startTLSIfOffered` dan mode keamanan dari porta |
| 5 | Validasi `SMTP_FROM` menolak bentuk baku `"Nama" <alamat@domain>` | Backend **tidak mau start** karena konfigurasi email ditolak | `notify.SplitAddress` dan `notify.ValidateAddress` |
| 6 | `QSTASH_URL` berisi root API regional, bukan endpoint lengkap | Jadwal dan publish tidak pernah berhasil | Jalur `/v2/publish` dan `/v2/schedules` ditambahkan otomatis; endpoint jadwal diturunkan dari `QSTASH_URL` agar satu region |

Temuan 3 paling serius: tanpa itu, pekerjaan pembersihan jejak audit dan pemberitahuan slip gaji H+1 tidak pernah dijadwalkan, sehingga tidak pernah berjalan otomatis.

Besides itu, pengukuran beban menemukan dua keterbatasan konfigurasi:

| Temuan | Bukti | Tindakan |
|--------|-------|----------|
| Pooler sesi Supabase membatasi 15 koneksi concurrently (`pool_size: 15`), sedangkan bawaan pool backend adalah 25 | `FATAL: (EMAXCONNSESSION) max clients reached in session mode` pada 16 stream paralel | Bawaan `DB_MAX_OPEN_CONNS` diturunkan menjadi 10 dan didokumentasikan |
| `QSTASH_SCHEDULES_URL` kosong membuat jadwal jatuh ke region global | Token regional `us-east-1` | Endpoint jadwal diturunkan dari `QSTASH_URL` |

### 8.1 Verifikasi Perbaikan terhadap QStash Nyata

Setelah bentuk endpoint diperbaiki, pendaftaran jadwal berhasil terhadap API QStash sungguhan:

```
jadwal dibuat: scd_6WyYveD2soczUScPHSuCrty1mgto
1 jadwal terdaftar:
  scd_6WyYveD2soczUScPHSuCrty1mgto  cron=*/5 * * * *      tujuan=https://example.com/slo-verify
```

Jadwal uji tersebut kemudian dibatalkan agar akun QStash tidak meninggalkan jadwal yang tidak pernah dieksekusi.

### 8.2 Batasan Lingkungan: Domain Aplikasi Belum Ada di DNS

Pendaftaran jadwal dengan tujuan `APP_BASE_URL` ditolak QStash dengan:

```
400 invalid destination url: unable to resolve host: lookup simpeg.unisubang.ac.id: no such host
```

Pemeriksaan DNS mengonfirmasi bahwa `simpeg.unisubang.ac.id` maupun `unisubang.ac.id` tidak ada di DNS. QStash memvalidasi tujuan dengan menyelesaikannya sebelum menerima jadwal, sehingga jadwal tidak dapat didaftarkan sampai domain aplikasi benar-benar terbit. Ini adalah fakta lingkungan, bukan cacat kode: jalur kode sudah benar dan terbukti berhasil pada tujuan yang dapat diresolusi.

## 8A. Verifikasi Job SLO Terjadwal

Job pemeriksaan SLO diuji end-to-end terhadap server produksi sungguhan, bukan
hanya dengan pengujian unit. Alurnya: pengoleki metrik mengumpulkan data,
callback bertanda tangan QStash dipanggil, target dievaluasi, alert diantrekan,
dan hasilnya tercatat di jejak audit.

### 8A.1 Metrik yang terkumpul

Pengoleki menulis satu baris per rute per jendela:

| Jendela | Rute | Request | Galat | Latensi maksimum |
|---------|------|---------|-------|------------------|
| 01:38 | `GET /api/v1/employees` | 18 | 0 | 1.597 ms |
| 01:38 | `GET /api/v1/me` | 6 | 0 | 1.158 ms |
| 01:38 | `GET /api/v1/audit-logs` | 6 | 0 | 1.857 ms |
| 01:38 | `GET /api/v1/categories` | 5 | 0 | 1.271 ms |
| 01:38 | `GET /api/v1/payroll/slips` | 5 | 0 | 2.048 ms |

Histogram bucket terisi sesuai distribusi latensi yang terukur, dan rute
tercatat sebagai pola rute (`/api/v1/employees`), bukan path mentah.

### 8A.2 Hasil pemeriksaan

Callback ditandatangani dengan `QSTASH_CURRENT_SIGNING_KEY` memakai klaim yang
sama dengan QStash, sehingga signature diverifikasi oleh `qstash.Verifier`
seperti pada callback sungguhan:

```json
{
  "windowStart": "2026-11-12T01:38:00Z",
  "windowEnd": "2026-11-12T01:39:00Z",
  "requests": 40,
  "errors": 0,
  "errorRate": 0,
  "p95Ms": 1600,
  "p99Ms": 2240,
  "breaches": 1,
  "alerted": true
}
```

### 8A.3 Alert yang diantrekan

Alert masuk antrean keluar persisten dengan kunci idempotensi yang memuat awal
jendela, sehingga percobaan ulang callback pada jendela yang sama tidak
menghasilkan email kedua:

```
email.umum | PENDING | slo-alert:2026-11-12T01:38:00Z:adhitiadwima@unsub.ac.id
```

Isi alert memuat ringkasan jendela, jumlah permintaan, galat, p95 dan p99,
pelanggaran yang terdeteksi, dan lima rute terlambat. Jejak audit mencatat
`"slo.evaluated"` dengan keterangan `"pemeriksaan SLO: alert diantrekan"`.

Status pesan tetap `PENDING` karena kredensial SMTP ditolak oleh Google
Workspace (bagian 6.2). Antrean keluar, bukan pengiriman langsung, yang
membuat perilaku ini ожи: bila kredensial diperbaiki, worker outbox akan
mengirimkan pesan yang sama tanpa perlu pemeriksaan ulang.
## 9. Workload Realistis (Estimasi)
| Parameter | Estimasi | Keterangan |
|-----------|-----------|------------|
| Pengguna bersamaan (pilot) | 10–25 | Admin, operator dari satu fakultas dan satu biro |
| Data pegawai | 1.000–5.000 baris | Lebih kecil dari skala pengukuran (5.000 baris) |
| Pola pencarian | Baca-dominan (90% baca, 10% tulis) | Daftar/cari pegawai lebih sering dari pembuatan profil |
| Impor/ekspor | Bulanan, <1.000 baris per batch | Impor massal pegawai oleh operator |
| Login | Beberapa kali per hari per pengguna | Sesi berlaku 23 jam (DEC-029, DEC-034) |
| Burst notifikasi | H+1 setelah tanggal gajian | Email slip tersedia dikirim ke semua pegawai dalam unit |

## 10. Rekomendasi SLO

Angka di bawah dinyatakan terhadap **penempatan backend di region yang sama dengan database**. Pada penempatan berbeda, kalikan Component Latency dengan faktor yang terukur di bagian 3.2 (±87 ms per statement).

| Target | Nilai | Dasar pengukuran |
|--------|-------|------------------|
| p95 daftar pegawai | < 150 ms | Terukur 185 ms lewat pooler; ≈15 ms bila backend se-region |
| p99 daftar pegawai | < 400 ms | Terukur 208 ms lewat pooler |
| p95 daftar slip gaji | < 200 ms | Terukur 445 ms lewat pooler; target menuntut pengurangan statement |
| p95 kategori (dengan cache) | < 60 ms | Terukur 19 ms untuk pembacaan cache, 175 ms tanpa cache |
| Error rate | < 1% | Galat klien (4xx) tidak dihitung |
| Throughput daftar pegawai | > 50 req/detik | Terukur 28,6 req/detik end-to-end lewat pooler pada 16 stream |
| Ketersediaan (SLO) | 99,9% | Sesuai `SLO_TARGET` di env |

Target SLO p95 bawaan yang dipakai job pemeriksaan adalah **800 ms** dengan toleransi laju galat **1%** dan jumlah permintaan minimum **20** per jendela. Angka 800 ms sengaja menjadi batas bucket pada histogram metrik, sehingga keputusan pelanggaran pada target tersebut tidak pernah meleset ke bawah.

## 11. Keterbatasan Baseline

- Tidak mencakup latensi jaringan antara frontend dan backend; semua pengukuran dilakukan dari mesin developers.
- Tidak mencakup beban import/ekspor Excel (menggunakan pustaka eksternal, lebih mahal).
- Tidak ada uji beban UI karena frontend belum dibuat.
- Uji beban SMTP berhenti pada autentikasi karena kredensial pada `.env` ditolak oleh Google Workspace; jalur kode terverifikasi sampai autentikasi, tetapi pengiriman nyata ke kotak masuk belum terukur.
- Skala data pengukuran (5.000 pegawai) masih kecil dibanding kapasitas produksi. Kesimpulan tentang indeks (`Seq Scan` pada `aktif`, `nama`, dan `homebaseId`) berdasarkan plan pada 5.000 baris dan harus diulang pada skala lebih besar.
- Semua angka berasal dari satu mesin developers, satu proyek Supabase, dan satu region. Angka belum merupakan bukti lintas lingkungan.
- `count(*)` pada tabel besar masih berjalan penuh (10 ms pada 50.000 baris jejak audit) dan belum dioptimalkan.
- Jadwal QStash untuk `APP_BASE_URL` production belum bisa didaftarkan karena domain aplikasi belum ada di DNS (bagian 8.2). Perilaku jadwal pada domain yang sudah terbit belum diuji.
- Pengoleki metrik SLO memakai satu proses backend. Pada deployment multi-instance, setiap instance menulis agregasi untuk jendela yang sama dan baris terakhir yang menang. Untuk instance banyak, pengoleki perlu dikunci per jendela atau metrik perlu dikumpulkan per instance.

## 12. Langkah Selanjutnya

1. Tambahkan indeks `nama`, `homebaseId`, dan `pg_trgm` pada `user` sebagai migrasi terpisah, lalu ulangi `perfbench explain` untuk mengukur dampaknya.
2. Turunkan jumlah statement pada daftar slip gaji dari lima menjadi satu atau dua, karena jalur itu paling lambat.
3. Optimasikan `count(*)` pada daftar dengan penghitungan estimasi atau berversi.
4. Jalankan backend di region yang sama dengan database Supabase, lalu ulangi seluruh pengukuran.
5. Sediakan *app password* Google Workspace yang valid, lalu ulangi uji beban SMTP sampai ke kotak masuk.
6. Uji beban import/ekspor Excel dengan data sintetis.
7. Terbitkan domain aplikasi, lalu daftarkan jadwal QStash dan verifikasi job SLO berjalan otomatis pada jadwal cron.
8. Tinjau batas `pool_size` pooler sesi dan pertimbangkan pooler transaksi untuk toleransi koneksi lebih tinggi.
9. Untuk deployment multi-instance, tentukan cara pengoleki metrik SLO mengunci jendela agar tidak saling menimpa.

## Referensi

- `backend/cmd/perfbench/` — perkakas pengukuran (seed, schema, explain, latency, e2e, deps, slo, schedule)
- `backend/internal/slo/` — domain, aplikasi, repository, dan pengoleki metrik SLO
- `backend/internal/httpserver/slo_handlers.go` — callback pemeriksaan SLO terjadwal
- `backend/migrations/000016_payment_updated_at.sql` — perbaikan skema `pembayaran`
- `backend/migrations/000017_slo_metric.sql` — tabel metrik SLO
- `backend/internal/httpserver/benchmark_test.go` — kode benchmark jalur HTTP
- `backend/internal/httpserver/load_test.go` — uji beban burst
- `backend/internal/httpserver/soak_test.go` — uji soak opt-in
- `backend/internal/directory/app/catalog_benchmark_test.go` — benchmark dampak cache
- `backend/internal/directory/repository/gorm_import_test.go` — penjaga jumlah statement impor
- `backend/internal/payroll/repository/schedule_test.go` — penjaga jumlah statement notifikasi
- `backend/internal/qstash/publisher_endpoint_test.go` — penjaga bentuk endpoint QStash
- `backend/internal/notify/smtp_starttls_test.go` — penjaga peningkatan STARTTLS
- `backend/internal/notify/address_test.go` — penjaga bentuk alamat email
- `backend/internal/slo/domain/slo_test.go` — penjaga perhitungan persentil dan evaluasi target
- `backend/internal/slo/app/service_test.go` — penjaga use case pemeriksaan dan alert
- `docs/DECISIONS.md` DEC-036 hingga DEC-041 — keputusan yang lahir dari pengukuran ini
- `docs/RUNBOOK.md` §8 — prosedur pemeriksaan dan pemulihan
