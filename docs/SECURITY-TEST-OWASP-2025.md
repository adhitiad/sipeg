# Uji Keamanan OWASP Top 10:2025 — SIMPEG

Tanggal: 2026-11-13 (pemindaian awal dan pemindaian ulang pada hari yang sama)
Status: Uji dinamis selesai pada infrastruktur disposable lokal. Tidak ada temuan keamanan berkategori kritis atau tinggi yang terbukti pada cakupan yang diuji. Delapan temuan awal (F-01 sampai F-08) sudah diperbaiki; F-03, F-06, dan F-08 terverifikasi lewat build dan tes, sedangkan F-01, F-02, F-04, dan F-07 kini terverifikasi ulang lewat pemindaian dinamis. Pemindaian ulang memunculkan satu alert baru `SQL Injection` (plugin 40018) yang ditriage menjadi positif palsu (§5.2) dan satu risiko residual yang diterima: CSP memakai `'unsafe-inline'` untuk script dan style (F-09, `DEC-056`).
Alat: OWASP ZAP 2.17.0 (Automation Framework), ditambah matriks otorisasi manual empat role.
Target: backend `http://localhost:8081`, frontend `http://localhost:3001` (dev server pada pemindaian awal, preview build produksi pada pemindaian ulang), PostgreSQL disposable `simpeg_owasp`.

> **Catatan cakupan (2026-11-16):** pemindaian ini mencakup backend & frontend **per 13 November 2026**. Modul yang menyusul — penggajian inti (skala, tunjangan, honor, BPJS, PPh 21 TER, THR, kontrak/pesangon, master pensiun), presensi mesin, cuti lanjutan, dan RBAC admin (`/api/v1/admin/rbac/*`, termasuk UI `/admin/rbac` dan `/admin/pensiun`) — **dipindai ulang pada 2026-11-16** dua tahap terhadap database disposable lokal `simpeg_zap` (dihapus setelah pengujian; migrasi v33 bersih, 4 akun uji, binary dari sumber ini, `APP_ENV=development`, RATE_LIMIT diturunkan):
>
> 1. **Matriks otorisasi manual 20 endpoint × 4 role** pada disposable `simpeg_owasp3` (mencakup cakupan unit/self/any) — **0 anomali** (atasan & operator hanya unitnya, pegawai hanya dirinya; penolakan tercatat `authz.denied`; `POST /api/v1/admin/rbac/roles` dengan body kosong menjawab `503` — batasan kecil, bukan kebocoran informasi).
> 2. **Authenticated ZAP active scan** dengan ZAP 2.17.0 (Automation Framework, `245 URL` di-import dari kontrak, HTTP Sender script menyuntikkan `Authorization: Bearer zapscan-admin-20261116` untuk GET/HEAD/OPTIONS dari akun admin). Cakupan: `payroll/bpjs*, honor-*, pph21-ter, scale, schedules, slips, slips/generate, thr/*, pemberhentian/*, kontrak, pesangon-aturan, pensiun/*, surat-keputusan, perpanjangan, holidays, admin/rbac/*, attendance/*, employees/import`. Mode: passive scan 5 menit + active scan 20 menit max, 2 thread. Hasil: **2 tipe alert** — (a) `CSP: Failure to Define Directive with No Fallback` (3 instance URL tertentu; extension header-only; residual `F-09` yang sudah diterima secara kebijakan — **CSP frontend dan HSTS/TLS pada lapisan deployment produksi belum terverifikasi**, jadi header keamanan produksi tetap terbuka sebagai item pra-rilis) dan (b) `Information Disclosure — Sensitive Information in URL` (2 instance: `arsip?search=ZAP`, `surat-keputusan?search=ZAP&userId=` — bukan data sensitif nyata, false positive hasil fuzzing). **Tidak ada SQLi, XSS, path traversal, command injection, atau SSRF terdeteksi.** Endpoint tulis (`POST/PATCH/DELETE`) tidak dipindai secara aktif dengan sesi (sengaja, untuk tidak memicu mutasi massal) — **tetap terbuka untuk bukti lebih lanjut sebelum rilis produksi**; OpenAPI diimpor apa adanya (tanpa user-provided data), sehingga scan hanya melihat parameter berparameter yang dinyatakan di kontrak.
>
> Sisa cakupan yang terbuka: ZAP tidak dijalankan pada atasan/operator/pegawai (hanya admin sesi — API-level otorisasi dinilai lewat matriks manual); Active scan terbatas 20 menit; belum ada Postman/OWASP ZAP scan untuk form/non-API UI front-end. Lihat item **Pemindaian ulang OWASP** di `docs/IMPLEMENTATION-PLAN.md` §6. Rute baru **wajib** didokumentasikan di kontrak OpenAPI seperti pada `DEC-088`/`DEC-089`.

## 1. Ruang lingkup

| Termasuk                                                         | Tidak termasuk                                                                                                              |
| ---------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------- |
| API `/api/v1` tanpa sesi dan dengan sesi `admin`                 | Database produksi Supabase dan database uji `simpeg_perf`                                                                   |
| Empat role sintetis: `admin`, `operator`, `atasan`, `pegawai`    | Google OAuth, MFA, dan alur pendaftaran mandiri (belum diverifikasi live; keduanya diimplementasikan 2026-11-13, `DEC-058`) |
| Halaman publik frontend dan `/login`, `/workspace`, `/admin/slo` | Pemindaian frontend dengan sesi login (crawl terautentikasi)                                                                |
| Endpoint autentikasi Better Auth `/api/auth/*`                   | TLS/HTTPS, HSTS, dan konfigurasi reverse proxy produksi                                                                     |
| Penanganan galat, pembatasan laju, logging keamanan, header      | Unggah file ke Cloudinary, SMTP/Redis/RabbitMQ nyata (QStash & S3 tidak dipakai)                        |
| Kontrak `docs/openapi.yaml` versus rute yang terdaftar           | Aplikasi mobile dan beban data berskala produksi                                                                            |

Uji hanya dijalankan terhadap infrastruktur lokal yang dapat dibuang. Tidak ada kredensial atau data pegawai nyata yang dipakai. Akun dan sesi dibuat khusus untuk pengujian dan dihapus bersama database `simpeg_owasp`.

Pemindaian ulang pada 2026-11-13 memakai target yang sama dan cakupan yang lebih luas: backend hasil build binary terbaru, preview frontend dari build produksi (bukan dev server), kontrak OpenAPI kanonis `docs/openapi.yaml` versi terkini, dan database `simpeg_owasp` yang sudah dimigrasikan ke versi Goose 24. Sesi admin dibuat langsung di tabel `session` supaya pemindaian API tidak bergantung pada runtime autentikasi frontend.

## 2. Lingkungan uji

| Komponen                   | Nilai                                                                                                                                                                                          |
| -------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| ZAP                        | 2.17.0, daemon `127.0.0.1:8090`, sesi terpisah per rencana                                                                                                                                     |
| Add-on                     | Automation Framework 0.62.0, OpenAPI Support 58.0.0, Active Scan Rules 84.0.0, Report Generation 0.46.0, Ajax Spider 23.33.0, GraalVM JavaScript 0.14.0, Script Console 45.20.0, Spider 0.20.0 |
| Backend                    | `cmd/server`, `APP_ENV=development`, port 8081                                                                                                                                                 |
| Frontend                   | Vite dev server TanStack Start, port 3001                                                                                                                                                      |
| Database                   | PostgreSQL 18 lokal, `simpeg_owasp`, Goose versi 20, 69 tabel, 68 tabel ber-RLS                                                                                                                |
| Akun uji                   | 4 akun sintetis dengan role berbeda, kata sandi khusus uji                                                                                                                                     |
| Sesi API                   | Sesi opaque GORM (`session`) dengan masa berlaku 7 hari                                                                                                                                        |
| Batas laju saat pemindaian | `RATE_LIMIT_REQUESTS=100000`, `RATE_LIMIT_AUTH_REQUESTS=10000`                                                                                                                                 |

Batas laju dinaikkan agar active scan tidak dihentikan `429`. Konsekuensinya pembatasan laju **tidak** diuji ulang secara dinamis dalam sesi ini; perilakunya tetap tercakup tes unit `backend/tests/httpserver/ratelimit_test.go`.

Perubahan lingkungan pada pemindaian ulang:

| Komponen          | Pemindaian awal                | Pemindaian ulang                                                                     |
| ----------------- | ------------------------------ | ------------------------------------------------------------------------------------ |
| Backend           | `go run ./cmd/server`          | binary hasil `go build ./cmd/server`                                                 |
| Frontend          | Vite dev server                | `vite preview` atas build produksi, aset hasil build diverifikasi 24 dari 24 Termuat |
| Database          | Goose versi 20                 | Goose versi 24 (migrasi 000021 sampai 000024), 70 tabel                              |
| Kontrak untuk ZAP | salinan yang diperbaiki manual | `docs/openapi.yaml` kanonis, disalin apa adanya                                      |
| Sesi admin        | injeksi token via HTTP Sender  | baris `session` sintetis, token `zapscan-admin-token-rescan`                         |

## 3. Metode

### 3.1 Rencana ZAP

Tiga rencana Automation Framework dijalankan berurutan, masing-masing pada sesi ZAP baru agar alert tidak tercampur. Setelah perbaikan F-01, F-02, dan F-07, dua rencana API dijalankan ulang (`plan-unauth-postfix.yaml`, `plan-auth-postfix.yaml`) pada backend hasil build terbaru.

| Rencana                     | Job                                                                                                                                  | Hasil                                                                                                                                                      |
| --------------------------- | ------------------------------------------------------------------------------------------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `plan-unauth.yaml`          | `openapi` (45 operasi), `requestor` 32 permintaan, `passiveScan-wait`, `activeScan` (12 menit, 6 thread, jeda 20 ms), `report`       | Selesai 00:36:54Z; 9 peringatan requestor; 2 tipe alert / 4 instans                                                                                        |
| `plan-auth.yaml`            | injeksi token `admin` lewat HTTP Sender, `requestor` 18 permintaan lintas role, `passiveScan-wait`, `activeScan`, `report`           | Selesai 00:50:08Z; 3 peringatan requestor; 3 tipe alert / 6 instans                                                                                        |
| `plan-frontend.yaml`        | `requestor` 10 permintaan (termasuk login gagal dan pendaftaran), `spider` (kedalaman 6), `passiveScan-wait`, `activeScan`, `report` | Selesai 00:58:19Z; 0 peringatan; 6 tipe alert / 22 instans                                                                                                 |
| `plan-unauth-postfix.yaml`  | sama dengan `plan-unauth.yaml`, ditambah `requestor` untuk `cuti` dan `tridharma`                                                    | Selesai 01:45:14Z; 9 peringatan (hanya rute F-03 `404`); 2 tipe alert / 4 instans                                                                          |
| `plan-auth-postfix.yaml`    | sama dengan `plan-auth.yaml`, ditambah `cuti` untuk `operator`/`atasan`, `tridharma/evaluasi` (termasuk jalur `sort`)                | Selesai 01:51:01Z; **0 peringatan**; 2 tipe alert / 4 instans, tanpa `Application Error Disclosure`                                                        |
| `plan-unauth-rescan.yaml`   | `openapi` atas kontrak kanonis terkini, `requestor` 32 permintaan, `passiveScan-wait`, `activeScan`, `report`                        | Selesai 15:39:10Z; 3 tipe alert / 5 instans: `SQL Injection` (tinggi), `Information Disclosure` dan `User Agent Fuzzer` (informasional)                    |
| `plan-auth-rescan.yaml`     | injeksi token admin lewat HTTP Sender, `requestor` lintas role, `passiveScan-wait`, `activeScan`, `report`                           | Selesai 15:51:34Z; 3 tipe alert / 5 instans, sama dengan rencana tanpa sesi                                                                                |
| `plan-frontend-rescan.yaml` | `requestor` halaman publik, `spider`, `passiveScan-wait`, `activeScan`, `report`                                                     | Selesai 15:40:32Z; 5 tipe alert / 17 instans: dua alert CSP `unsafe-inline` (sedang) dan tiga alert informasional; alert header keamanan lama tidak muncul |

Pemindaian ulang memakai sesi ZAP baru tanpa `auth-sender` aktif pada rencana tanpa sesi; percobaan pertama sempat memakai skrip yang masih termuat dari sesi sebelumnya dan dibuang karena seluruh permintaan tanpa sesi berubah menjadi `200`. Angka pada baris `plan-unauth.yaml` berasal dari sesi yang bersih: `GET /api/v1/me` tanpa sesi tetap `401`.

Kontrak OpenAPI yang dipakai ZAP adalah salinan yang diperbaiki di direktori artefak. Kontrak kanonis `docs/openapi.yaml` masih memiliki kekurangan deklarasi (lihat F-05).

Catatan reproduksi: pada Automation Framework, `requests` harus berada di level job, bukan di bawah `parameters`. Bentuk yang salah tidak menghasilkan galat; job `requestor` selesai dalam 0 detik tanpa mengirim permintaan. Rencana frontend sempat dijalankan dalam bentuk itu dan diulang setelah diperbaiki; laporan yang dirujuk di sini berasal dari jalur yang benar.

Injeksi token pada pemindaian terautentikasi dibatasi pada metode baca (`GET`/`HEAD`/`OPTIONS`) dan melewati `/api/v1/auth/*`. Pembatasan ini diperlukan karena `POST /api/v1/auth/logout` mencabut sesi opaque yang sedang dipakai; tanpa pembatasan, pemindai mencabut sesinya sendiri di tengah scan. Konsekuensi: jalur tulis terautentikasi diuji lewat `requestor` dengan header eksplisit, bukan lewat active scan.

Catatan teknis pemindaian ulang: endpoint `runPlan` pada API ZAP bersifat asinkron dan hanya mengembalikan `planId`, sehingga percobaan pertama sempat membuat tiga sesi beruntun tanpa menunggu job selesai. Penantian yang benar memakai file laporan yang ditulis job terakhir pada rencana. Selain itu, preview frontend harus dijalankan dari build yang sama persis dengan aset yang dipindai; ketika build diganti di tengah pemindaian, preview masih menyajikan manifest lama dan aset berryg (`404`) muncul sebagai artefak, bukan cacat aplikasi. Frontend karena itu dipindai ulang sekali lagi terhadap build produksi final yang sudah diverifikasi lengkap (24 dari 24 aset Termuat, kedua font brand Termuat, tanpa galat konsol).

### 3.2 Matriks otorisasi manual

ZAP tidak memahami semantik izin, sehingga A01 dinilai terutama dari matriks manual: 5 identitas (tanpa sesi dan 4 role) × 8 endpoint, ditambah pemeriksaan cakupan daftar pegawai.

| Endpoint                             | Tanpa sesi | pegawai       | atasan | operator | admin         |
| ------------------------------------ | ---------- | ------------- | ------ | -------- | ------------- |
| `GET /api/v1/me`                     | 401        | 200           | 200    | 200      | 200           |
| `GET /api/v1/employees`              | 401        | 200 (1 baris) | 200    | 200      | 200 (4 baris) |
| `GET /api/v1/audit-logs`             | 401        | 403           | 403    | 200      | 200           |
| `GET /api/v1/admin/slo/metrics`      | 401        | 403           | 403    | 403      | 200           |
| `GET /api/v1/categories`             | 401        | 403           | 200    | 200      | 200           |
| `GET /api/v1/holidays`               | 401        | 403           | 200    | 200      | 200           |
| `GET /api/v1/payroll/schedules`      | 401        | 403           | 403    | 500      | 200           |
| `POST /api/v1/holidays/2026/refresh` | 401        | 403           | 403    | 403      | 503           |

Cakupan data pegawai: `pegawai` menerima 1 baris dari 1 total (dirinya), sedangkan `admin`, `operator`, dan `atasan` menerima 4 baris dari 4. Filter cakupan unit pada daftar pegawai terbukti bekerja.

`GET /api/v1/cuti` diuji terpisah: `admin` 200 (0 baris), tiga role lain 503 karena cacat query (F-01), sehingga cakupan cuti untuk non-admin **belum dapat dinilai** dari data nyata pada pemindaian asli. Setelah F-01 diperbaiki, keempat role menerima `200` dengan baris uji yang sama (§5.1).

### 3.3 Probe cakupan rute

Probe status terhadap 29 rute × 4 role menemukan:

- 9 rute di kontrak mengembalikan `404` untuk semua role: `arsip`, `golongan`, `keluarga`, `pelatihan`, `penghargaan`, `potongan`, `sertifikasi`, `sister`, `tunjangan`.
- `tridharma/evaluasi` mengembalikan `503` untuk semua role (F-02); setelah diperbaiki menjadi `200` dengan data (§5.1).
- `attendance/my-records` dan `attendance/unit-summary` mengembalikan `422` tanpa parameter wajib (perilaku validasi, bukan cacat).
- Router mendaftarkan 138 penanganan rute, sedangkan kontrak memuat 33 path dan 45 operasi. Selisih ini adalah drift dokumentasi (F-05), bukan bukti rute tak terlindungi: seluruh rute ber-permission menolak permintaan tanpa sesi dengan `401`.
- **Pembaruan status (2026-11-14):** pernyataan "138 penanganan rute", "33 path", dan "45 operasi" adalah snapshot pemindaian pada 2026-11-13. Penutupan drift rute pada 2026-11-14 membuat kontrak sepadan dengan router: 147 operasi pada 78 path di `docs/openapi.yaml`, seluruh operasi terdaftar terdokumentasi, dan kontrak dikoreksi (`DEC-069`).

## 4. Hasil per kategori

### A01 Broken Access Control — teruji sebagian

Bukti:

- Seluruh endpoint ber-permission yang terdaftar menolak permintaan tanpa sesi (`401`) dan menolak role tanpa grant (`403`); pada pemindaian ulang tidak ada lagi endpoint yang gagal `5xx` sebelum otorisasi selesai pada rute yang dipindai. Sembilan path kontrak yang tidak terdaftar menjawab `404` untuk semua identitas, sehingga tidak membocorkan data. Cacat `5xx` yang dulu menutupi otorisasi (`cuti`, jadwal penggajian, `tridharma`) sudah diperbaiki dan diuji ulang; lihat §3.1 dan §5.1.
- `GET /api/v1/payroll/slips` hanya membuka data sendiri untuk `pegawai`, dan menolak `atasan` dengan `403`.
- `GET /api/v1/admin/slo/metrics` hanya `200` untuk `admin`.
- `PUT /api/v1/employees/{id}/role` menolak `pegawai`, `atasan`, dan `operator` dengan `403`; hanya `admin` yang lolos otorisasi (payload uji kemudian ditolak `400`, bukan masalah izin).
- `GET /api/v1/audit-logs/export` menolak `pegawai` dan `atasan` dengan `403`, serta membuka `operator` sesuai `audit:read:homebase` dan `admin` sesuai `audit:read`. Setelah F-07 diperbaiki, ekspor `admin` dan `operator` menghasilkan 1.423 baris data; cakupan baris mengikuti unit principal.
- Halaman `/admin/slo` dapat dimuat tanpa sesi, tetapi SSR hanya menampilkan status "Memuat sesi Anda..." dan tidak memuat metrik; data tetap memerlukan token admin pada API.

Belum teruji: IDOR dengan identifier milik pengguna lain. ZAP hanya mengirim UUID placeholder dari kontrak (`00000000-0000-4000-8000-0000000000b1`, `unitId`, `arsipId`), sehingga BOLA pada record nyata hanya tercakup tes otomatis `backend/tests/httpserver/scope_handlers_test.go`, bukan uji dinamis. Sembilan rute `404` (F-03) juga belum dapat dinilai.

### A02 Security Misconfiguration — teruji sebagian

- Backend mengirim `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, `Referrer-Policy: no-referrer`, `Cross-Origin-Opener-Policy: same-origin`, `Permissions-Policy`, dan `X-API-Version`. Tidak ada `Content-Security-Policy` (F-06).
- CORS hanya mengizinkan origin dari `TRUSTED_ORIGINS`; wildcard ditolak dan daftar kosong berarti tanpa akses lintas origin.
- `APP_ENV=production` mewajibkan `BETTER_AUTH_SECRET` ≥ 32 karakter tanpa pola bawaan dan `TRUSTED_ORIGINS` tidak kosong.
- RLS aktif default-deny pada 68 tabel aplikasi di database uji; migrasi tidak memakai `FORCE` sehingga backend sebagai pemilik tabel tetap dapat menulis.
- Frontend dev server pada saat pemindaian awal tidak mengirim CSP, `X-Frame-Options`, maupun `X-Content-Type-Options` (F-04). Pada sesi lanjutan 2026-11-13, dev/preview server dikonfigurasi mengirim `Content-Security-Policy` **enforcing** beserta `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, dan `Referrer-Policy: no-referrer` (`frontend/vite.config.ts`); cakupan header produksi pada lapisan deployment masih perlu diverifikasi.
- Pemindaian ulang frontend (F-04) mengonfirmasi perbaikan itu: tiga alert header yang sebelumnya muncul (`Content Security Policy (CSP) Header Not Set`, `Missing Anti-clickjacking Header`, `X-Content-Type-Options Header Missing`) tidak lagi ditemukan. Header juga diperiksa langsung pada respons `/` dan `/login` dari preview build produksi.
- Risiko residual yang kini terlihat: CSP memakai `'unsafe-inline'` untuk script dan style, sehingga ZAP melaporkan `CSP: script-src unsafe-inline` dan `CSP: style-src unsafe-inline` tingkat sedang. Ini konsekuensi inline SSR yang disetujui pada `DEC-056`, bukan kegagalan; penghilangan `unsafe-inline` memerlukan nonce atau hash dan belum dikerjakan. Dicatat sebagai F-09.
- Tidak ada kredensial bawaan; akun uji dibuat eksplisit pada database disposable.

### A03 Software Supply Chain Failures — teruji pada konfigurasi

- `go.sum` dikomit, versi toolchain dipin, `govulncheck` berjalan di CI dan terakhir melaporkan "No vulnerabilities found." setelah Go dinaikkan ke 1.27.4 (`DEC-045`).
- `frontend/bun.lock` adalah satu-satunya lockfile; CI memakai `bun install --frozen-lockfile`.
- Workflow CI memakai `permissions: contents: read`, dan ada pemindaian kebocoran connection string.
- Dependabot mingguan aktif untuk Go dan GitHub Actions.
- ZAP tidak memasang Retire.js, sehingga kerentanan pustaka JavaScript di browser **tidak** diperiksa secara dinamis dalam uji ini.

### A04 Cryptographic Failures — teruji sebagian

- Header kredensial, body, dan nilai query string tidak dicatat pada log akses maupun security event.
- `auth_failed` pada runtime Better Auth mencatat `path`, `reason`, dan waktu saja; tidak ada email, kata sandi, atau token. Terverifikasi dari log frontend saat uji login gagal.
- Sesi API memakai bearer token opaque; tidak ada cookie sesi API yang dapat dipalsukan lewat browser pada jalur ini.
- ZAP menandai `userId` pada query string sebagai "Sensitive Information in URL". Nilainya adalah placeholder, bukan rahasia, dan backend tidak mencatat nilai query string; ini positif palsu.
- TLS, HSTS, enkripsi kolom sensitif, dan klasifikasi data belum diuji dan belum ditetapkan.

### A05 Injection — teruji

- Active scan (Active Scan Rules 84.0.0) tidak menemukan SQL injection, XSS, command injection, maupun path traversal pada 45 operasi API dan 32 URL frontend yang tercakup rencana `plan-unauth-rescan.yaml` dan `plan-frontend-rescan.yaml` (sesi pemindaian awal, kontrak ±45 operasi; hasil final pasca-penutupan drift pada 2026-11-14 ada pada `DEC-069` dan penutupan tersebut **bukan** perbaikan keamanan).
- Sort dan filter memakai allowlist kolom; nilai di luar allowlist ditolak `400 invalid_query`. Identifier tabel/kolom tidak berasal dari input mentah.
- Dekode JSON menolak field tak dikenal, body melebihi batas, dan identifier tidak valid.
- Temuan cacat query pada F-01 dan F-02 adalah kesalahan sintaks SQL, bukan jalur injeksi: tidak ada input pengguna yang masuk ke teks SQL tanpa parameter.
- Pemindaian ulang memunculkan alert `SQL Injection` (plugin 40018) pada `PATCH /api/v1/arsip/{arsipId}` (`mimeType`, `fileUrl`) dan `POST /api/v1/organizations` (`description`). Alert itu ditriage menjadi positif palsu (§5.2): respons tidak berubah saat kondisi boolean dibalik, payload masuk ke PostgreSQL sebagai bind parameter, dan teks SQL hanya berisi placeholder.

### A06 Insecure Design — teruji pada dokumen

- `docs/THREAT-MODEL.md` mencakup data pegawai, persetujuan, pembayaran, cuti, impor/ekspor, dan kegagalan dependency.
- Cacat desain yang terlihat dari uji dinamis: kegagalan query diterjemahkan menjadi `500`/`503` pada rute yang seharusnya mengembalikan data atau `403`, dan rute yang didokumentasikan tidak terdaftar. Keduanya masuk F-01, F-02, dan F-05.
- Alur persetujuan cuti berjenjang belum dapat diuji karena modulnya belum lengkap.

### A07 Authentication Failures — teruji sebagian

- Tanpa token, token palsu, dan skema `Authorization` non-Bearer ditolak `401`.
- Login dengan kata sandi salah dan email tidak dikenal menghasilkan status, body, dan kode yang identik (`401`, `Invalid email or password`, `INVALID_EMAIL_OR_PASSWORD`), sehingga tidak ada enumerasi akun lewat respons.
- Pendaftaran mandiri ditolak `400 EMAIL_PASSWORD_SIGN_UP_DISABLED`, sesuai `disableSignUp`.
- Sesi dicabut saat logout (sesi opaque), akun nonaktif ditolak, dan role dibaca ulang dari tabel `Role` setiap permintaan sehingga perubahan role berlaku tanpa sesi baru.
- Pembatasan laju endpoint autentikasi lebih ketat daripada endpoint biasa, tetapi tidak diuji dinamis karena batas dinaikkan untuk pemindaian.
- Google OAuth dan MFA diimplementasikan di kode 2026-11-13 (`DEC-058`) tetapi tidak pernah diaktifkan atau diuji dalam pemindaian ZAP ini; alur state/callback dan faktor kedua tetap tidak dapat dinilai dari hasil pemindaian ini.
- Observasi: `reason` pada log internal membedakan `INVALID_EMAIL` dan `INVALID_PASSWORD`, meskipun respons publik tidak. Ini membantu investigasi dan hanya terlihat oleh operator log.

### A08 Software or Data Integrity Failures — teruji pada konfigurasi

- Signature QStash diverifikasi atas raw body, URL tujuan, masa berlaku, dan rotasi kunci (transport lama tidak dipakai; RabbitMQ adalah transport produksi); callback dengan signature tidak valid ditolak dan dicatat `webhook_signature_rejected` tanpa mencatat nilai signature.
- Idempotensi event presensi diuji: retry identik mengembalikan event yang sama, key sama dengan fakta berbeda ditolak `409`.
- Jejak audit append-only dari sisi aplikasi, dijaga tes pemindai source; satu-satunya penghapusan adalah retensi terjadwal.
- Tidak ada integritas artefak build (SBOM/signature) dan tidak ada unggah file dalam cakupan, sehingga bagian itu tidak dinilai.

### A09 Security Logging and Alerting Failures — teruji sebagian

- Security event terstruktur tersedia: `auth_failed`, `access_denied`, `rate_limited`, `authz.denied` (jejak audit), `webhook_signature_rejected`, dan kegagalan email outbox.
- Uji dinamis menghasilkan entri `auth_failed` berisi `{"event":"auth_failed","path":"/sign-in/email","reason":"INVALID_EMAIL","ts":...}` tanpa kredensial.
- Perubahan role menulis `employee.role.updated`; pembacaan jejak audit menulis `audit.accessed`; pencabutan sesi menulis `session.revoked`.
- Dasbor `/admin/slo` menampilkan error rate dan latensi per rute untuk admin.
- Belum ada aturan alert pada infrastruktur mana pun, dan dasbor belum mencakup saturasi, health dependency, antrean, atau kegagalan notifikasi.

### A10 Mishandling of Exceptional Conditions — teruji

- Respons `500` dan `503` memiliki body kosong; tidak ada stack trace, nama tabel, atau detail driver yang bocor. Ini juga alasan ZAP hanya memberi `Application Error Disclosure` tingkat low berbasis status, bukan kebocoran isi. Alert itu tidak lagi muncul setelah perbaikan F-01, F-02, dan F-07 (§3.1).
- Malformed JSON, field tak dikenal, body melebihi batas, dan identifier tidak valid ditolak `400`/`422` dengan format error seragam `{"error":{...}}`.
- Kegagalan dependency yang terkelola mengembalikan `503` tanpa detail internal; `holidays/2026/refresh` mengembalikan `503` karena penyedia eksternal tidak tersedia, bukan karena kebocoran.
- Timeout statement database diterapkan lewat `DB_STATEMENT_TIMEOUT`. Timeout Redis/RabbitMQ/SMTP terhadap provider nyata belum diuji.
- Cacat pada F-01 dan F-02 menunjukkan kegagalan internal memang tidak membocorkan detail, tetapi juga tidak tertangani sebagai kondisi yang dapat dipulihkan. F-07 menunjukkan pola yang lebih berbahaya: setelah status `200` dikirim, kegagalan stream ditelan tanpa sinyal ke pengguna sehingga berkas ekspor tampak berhasil. Ketiganya sudah diperbaiki dan diuji ulang (§5.1); F-07 sekarang mengirim `500` bila batch pertama gagal sebelum header CSV terkirim.
- Validasi identifier asing pada jalur tulis belum lengkap: `POST /api/v1/tridharma/evaluasi` dengan `periodeAkademikId` kosong meneruskan string kosong ke SQL, menghasilkan `SQLSTATE 22P02` dan `503 service_unavailable` (F-08). Ini bukan kebocoran detail, tetapi respons `5xx` untuk kesalahan input klien.

## 5. Temuan

| ID   | Keparahan                          | Kategori | Temuan                                                                                                                                                                                                                                                                                      | Bukti                                                                                                            | Status                                                                                                                                                                                                                     |
| ---- | ---------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| F-01 | Tinggi (fungsional)                | A10, A05 | Identifier camelCase tidak dikutip pada klausa `Where` GORM menyebabkan `SQLSTATE 42703`; `GET /api/v1/cuti` `503` untuk `operator`/`atasan`/`pegawai`, `GET /api/v1/payroll/schedules` `500` untuk `operator`, dan `PUT /api/v1/payroll/schedules/{unitId}` `500` untuk `admin`/`operator` | Log backend `column "unitId" does not exist`; `evidence\zap-scope-probe.txt`; ZAP `Application Error Disclosure` | **Diperbaiki 2026-11-13**, lihat §5.1                                                                                                                                                                                      |
| F-02 | Sedang                             | A10      | `GET /api/v1/tridharma/evaluasi` `503` untuk semua role karena kesalahan sintaks SQL di dekat `ASC`                                                                                                                                                                                         | Log backend; `evidence\zap-scope-probe.txt`                                                                      | **Diperbaiki 2026-11-13**, lihat §5.1                                                                                                                                                                                      |
| F-03 | Sedang                             | A02      | Sembilan rute yang didokumentasikan mengembalikan `404` karena pendaftaran rute bersyarat (`if api.arsip != nil` dan sejenisnya) tidak pernah aktif                                                                                                                                         | `backend/internal/httpserver/directory_api.go`, probe rute                                                       | **Diperbaiki 2026-11-13**, lihat §11                                                                                                                                                                                       |
| F-04 | Sedang                             | A02      | Frontend tidak mengirim CSP dan `X-Frame-Options`; `X-Content-Type-Options` juga tidak ada                                                                                                                                                                                                  | ZAP `frontend.json`: CSP 5 instans, anti-clickjacking 5, nosniff 5; header respons diperiksa langsung            | **Diperbaiki dan terverifikasi 2026-11-13**: pemindaian ulang frontend tidak lagi melaporkan ketiga alert header itu (dev/preview kini CSP **enforcing** + header lain; produksi/deployment belum diverifikasi), lihat §11 |
| F-05 | Sedang                             | A02, A09 | Kontrak OpenAPI dan router tidak sinkron: 9 path kontrak tidak terdaftar, dan 138 penanganan rute tidak seluruhnya terdokumentasi; kontrak juga masih merujuk `CutiDTO`, `ArsipDTO`, `UserID`, dan parameter `arsipId` yang tidak dideklarasikan                                            | Peringatan requestor ZAP; impor OpenAPI gagal sebelum salinan diperbaiki                                         | **Sebagian diperbaiki 2026-11-13**; **drift rute terbuka ditutup penuh 2026-11-14** (seluruh 147 operasi terdaftar terdokumentasi, `DEC-066`), lihat §11                                                                   |
| F-06 | Rendah                             | A02      | API tidak mengirim `Content-Security-Policy`                                                                                                                                                                                                                                                | Header respons `/health`                                                                                         | **Diperbaiki 2026-11-13**, lihat §11                                                                                                                                                                                       |
| F-07 | Sedang                             | A10, A01 | `GET /api/v1/audit-logs/export` mengembalikan `200` dengan CSV kosong (hanya header) untuk `admin` dan `operator`; pengguna tidak diberi tahu kegagalan dan entri `audit.exported` tidak ditulis                                                                                            | Log backend `SQLSTATE 22P02: invalid input syntax for type uuid: ""` pada `gorm_catalog.go:42`; respons 58 byte  | **Diperbaiki 2026-11-13**, lihat §5.1                                                                                                                                                                                      |
| F-08 | Rendah                             | A10      | Identifier asing kosong pada jalur tulis lolos validasi dan mencapai SQL; `POST /api/v1/tridharma/evaluasi` dengan `periodeAkademikId: ""` menjawab `503 service_unavailable` (`SQLSTATE 22P02`) alih-alih `422`                                                                            | Log backend `SELECT count(*) FROM "evaluasiBkd" ... "periodeAkademikId" = ''`; respons `503`                     | **Diperbaiki 2026-11-13**, lihat §11                                                                                                                                                                                       |
| F-09 | Sedang (risiko residual, diterima) | A02      | CSP frontend memakai `'unsafe-inline'` untuk script dan style karena inline SSR; ZAP melaporkan `CSP: script-src unsafe-inline` dan `CSP: style-src unsafe-inline`                                                                                                                          | ZAP `frontend-rescan.json` (plugin 10055, 5 instans per alert)                                                   | **Diterima 2026-11-13** sesuai `DEC-056`; penghilangan `unsafe-inline` memerlukan nonce atau hash, belum dikerjakan                                                                                                        |

Catatan keparahan: F-01, F-02, F-07, dan F-08 adalah cacat ketersediaan/integritas fungsi dan penanganan input, bukan bypass otorisasi. Tidak ada temuan yang memungkinkan pembacaan atau perubahan data lintas role. F-09 bukan cacat baru: frontend sebelumnya tidak mengirim CSP sama sekali, jadi penggunaan `'unsafe-inline'` ini adalah harga yang disetujui atas adanya proteksi, dan menghilangkannya memerlukan perubahan cara script dirender.

### 5.1 Verifikasi perbaikan F-01, F-02, F-07

Perbaikan diterapkan pada kode, seluruh `go test ./...` lulus, dan perilaku diverifikasi ulang pada backend yang dibangun dari kode terbaru terhadap database disposable `simpeg_owasp` (bukan Supabase produksi). Data uji dibuat lewat API agar query diuji pada baris nyata, bukan tabel kosong.

| Temuan | Perubahan                                                                                                                                                                                                                                                                                                                                                                            | Bukti setelah perbaikan                                                                                                                                                                                                                                                                                                                                                                                                                                                                 |
| ------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| F-01   | Identifier camelCase dan tabel `"user"` dikutip ganda di `directory/repository` (`gorm_arsip`, `gorm_golongan`, `gorm_keluarga`, `gorm_pelatihan`, `gorm_penghargaan`, `gorm_sertifikasi`, `gorm_sister`, `gorm_leave`) dan `payroll/repository` (`gorm_potongan`, `gorm_tunjangan`, `schedule`); `listing.OrderClause` mengeluarkan `"id"` bila kolom bawaan tidak ada di allowlist | `GET /api/v1/cuti` `200` dengan 1 baris untuk `admin`, `operator`, `atasan`, dan `pegawai`; `GET /api/v1/payroll/schedules` `200` dengan 1 baris untuk `admin`/`operator` dan `403` untuk `atasan`/`pegawai` (sesuai grant); `PUT /api/v1/payroll/schedules/{unitId}` `200` untuk `admin`. Tes: `TestListCutiQuotesCamelCaseColumns`, `TestUpsertScheduleQuotesUnitIdColumn`, `TestListSchedulesQuotesUnitIdColumn`                                                                     |
| F-02   | Peta sort duplikat di `tridharma/repository/gorm.go` dihapus; seluruh pemanggilan memakai allowlist `tridharma/domain` (yang memuat `tanggalEvaluasi`); `OrderClause` tidak lagi dapat menghasilkan klausa tanpa kolom                                                                                                                                                               | `GET /api/v1/tridharma/evaluasi` `200` dengan 1 baris; `?sort=tanggalEvaluasi&order=desc` `200` (jalur yang dulu gagal `503`); `?sort=nama` `400 invalid_query`; `GET /api/v1/tridharma/periode` `200`. Tes: `TestListEvaluasiBKDOrdersByDefaultColumn`, `TestListTridharmaPenelitianSortsByResearchColumns`, `TestOrderClauseNeverEmitsColumnlessClause`                                                                                                                               |
| F-07   | `StreamAuditLogs` melewati predikat keyset pada batch pertama dan membangun ulang query dasar per batch; `exportAuditLogs` menunda status `200` dan header CSV sampai baris pertama tiba, menjawab `500 internal_error` bila batch pertama gagal, dan meratakan buffer lewat `flushResponse`                                                                                         | Ekspor `admin` `200`, 335.889 byte, 1.424 baris (1 header + 1.423 data); ekspor `operator` `200`, 335.867 byte, jumlah baris sama dengan cakupan unit; ekspor `?aksi=authz.denied` `200` dengan 1.397 baris; `audit.exported` tercatat ("1423 baris diekspor", "authz.denied — 1397 baris diekspor"). Jalur gagal dicakup `TestAuditExportReportsStreamFailure` (`500`, tanpa header CSV) karena kegagalan batch pertama tidak dapat dipaksakan pada database sehat tanpa merusak skema |

Batas verifikasi ini:

- Sembilan rute `404` (F-03) tetap tidak dapat diuji dinamis, sehingga perbaikan F-01 pada `arsip`, `golongan`, `keluarga`, `pelatihan`, `penghargaan`, `sertifikasi`, `sister`, `potongan`, dan `tunjangan` hanya tercakup tes repository, bukan permintaan HTTP nyata.
- Perbaikan diuji pada database disposable dengan 4 akun sintetis dan data uji, bukan pada data produksi.
- Pemindaian ZAP ulang sudah dijalankan (§3.1): `plan-auth-postfix` selesai tanpa peringatan requestor, seluruh permintaan `cuti`, jadwal penggajian, dan `tridharma` sesuai kode yang diharapkan, dan `Application Error Disclosure` tidak lagi muncul. Alert tersisa pada kedua laporan hanya `Information Disclosure - Sensitive Information in URL` dan `User Agent Fuzzer`, keduanya informational. Frontend dipindai ulang pada sesi berikutnya; F-04 terverifikasi hilang (§5.2).

### 5.2 Triage alert SQL Injection dan verifikasi F-04 (pemindaian ulang 2026-11-13)

#### Alert SQL Injection (plugin 40018)

Pemindaian ulang melaporkan satu alert berisiko tinggi dengan keyakinan medium pada kedua rencana API. ZAP menuliskan "The page results were successfully manipulated using the boolean conditions", dengan instance:

| Rencana           | Metode dan path                 | Parameter      | Payload                     |
| ----------------- | ------------------------------- | -------------- | --------------------------- |
| Tanpa sesi        | `PATCH /api/v1/arsip/{arsipId}` | `mimeType`     | `John Doe AND 1=1 -- `      |
| Tanpa sesi        | `PATCH /api/v1/arsip/{arsipId}` | `tanggalMulai` | `John Doe' OR '1'='1' -- `  |
| Tanpa sesi        | `POST /api/v1/organizations`    | `description`  | `... ' AND '1'='1' -- `     |
| Dengan sesi admin | `PATCH /api/v1/arsip/{arsipId}` | `fileUrl`      | `John Doe' AND '1'='1' -- ` |

Triage dilakukan pada tiga sisi, dan hasilnya konsisten: tidak ada jalur injeksi.

**Kode.** Tidak ada input pengguna yang masuk ke teks SQL. Seluruh `backend/internal` hanya punya tiga pemakaian SQL mentah, ketiganya memakai placeholder dan parameter terpisah: `audit/cleaner.go` (`DELETE FROM "auditLog" WHERE "createdAt" < ?`), `payroll/repository/schedule.go` (`WHERE g."status" IN (?, ?)`), dan `directory/repository/gorm_catalog.go` (CTE dengan `?`). Semua jalur tulis memakai `Updates(map[string]any)` milik GORM, yang mengikat nilai, bukan menyusun kalimat SQL.

**Perilaku.** Kedua payload pada tiap endpoint diputar ulang secara manual dengan status dan body dibandingkan:

| Endpoint dan parameter                              | Kondisi `1=1`    | Kondisi `1=2`    |
| --------------------------------------------------- | ---------------- | ---------------- |
| `PATCH /arsip/{arsipId}` `mimeType`, tanpa sesi     | 401              | 401              |
| `PATCH /arsip/{arsipId}` `mimeType`, sesi admin     | 400 `invalid_id` | 400 `invalid_id` |
| `PATCH /arsip/{arsipId}` `fileUrl`, sesi admin      | 400 `invalid_id` | 400 `invalid_id` |
| `PATCH /arsip/{arsipId}` `tanggalMulai`, tanpa sesi | 401              | 401              |
| `POST /organizations` `description`, sesi admin     | 201              | 201              |

Body identik setelah `requestId` acak dinormalisasi. Perbedaan yang dibaca ZAP sebagai "manipulasi boolean" punya dua sebab yang tidak berkaitan dengan SQL. Pertama, `requestId` pada amplop error berbeda tiap permintaan. Kedua, pada `POST /organizations` dengan nama unit yang sama, permintaan pertama membuat baris dan permintaan berikutnya menabrak batasan unik sehingga menjawab 409; saat nama dibuat berbeda, kedua permintaan sama-sama menjawab 201. Jalur `PATCH /arsip/{arsipId}` bahkan berhenti di validasi UUID pada lapisan transport, sebelum menyentuh repository.

**Penyimpanan dan log database.** `log_statement` diaktifkan pada database disposable selama triage. PostgreSQL menerima:

```
INSERT INTO "Homebase" ("id","createdAt","updatedAt","name","description","jenis","parentId","aktif","latitude","longitude","radiusM") VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
DETAIL:  Parameters: $5 = 'Zaproxy alias impedit ... '' AND ''1''=''1'' -- ', $6 = 'UNIT', ...
```

```
UPDATE "Homebase" SET "description"=$1,"updatedAt"=$2 WHERE "id" = $3
DETAIL:  Parameters: $1 = 'Zaproxy alias impedit ... '' AND ''1''=''1'' -- ', $2 = '...', $3 = '76831dcd-...'
```

Payload masuk sebagai nilai bind dan disimpan apa adanya, lalu dibaca kembali tanpa perubahan. Kalimat SQL hanya berisi placeholder. Pengaturan `log_statement` dikembalikan ke `none` dan baris uji dihapus setelah triage.

Kesimpulan triage: **positif palsu**. Alert dicatat di §6 agar tidak terhitung sebagai temuan terbuka.

#### Verifikasi F-04 pada frontend

Pemindaian ulang frontend memakai preview atas build produksi final, bukan dev server. Perbandingan dengan laporan lama:

| Alert pada laporan lama                      | Status pada pemindaian ulang |
| -------------------------------------------- | ---------------------------- |
| Content Security Policy (CSP) Header Not Set | Tidak muncul                 |
| Missing Anti-clickjacking Header             | Tidak muncul                 |
| X-Content-Type-Options Header Missing        | Tidak muncul                 |
| CSP: script-src unsafe-inline                | Muncul (F-09, diterima)      |
| CSP: style-src unsafe-inline                 | Muncul (F-09, diterima)      |

Header juga diperiksa langsung pada respons `/` dan `/login`: CSP enforcing, `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, `Referrer-Policy: no-referrer`. Frontend kini menyajikan font brand (Fraunces dan Manrope) dari repo sendiri, sehingga CSP `font-src 'self'` tetap berlaku dan tidak ada permintaan ke `fonts.googleapis.com`. Aturan tautan global dipindahkan ke `@layer base` agar utilitas warna Tailwind pada tombol primer tetap menang; tombol utama terukur rasio kontras 6,40 terhadap latar hijau, di atas ambang 4,5.

## 6. Positif palsu dan observasi

| Alert ZAP                                                        | Penilaian                                                                                                                                                                      |
| ---------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Information Disclosure - Sensitive Information in URL (`userId`) | Positif palsu: nilai placeholder, bukan rahasia; backend tidak mencatat nilai query string                                                                                     |
| User Agent Fuzzer                                                | Informational; variasi User-Agent tidak mengubah perilaku teramati                                                                                                             |
| Application Error Disclosure                                     | Dipicu status `500`/`503`, bukan isi respons; body kosong. Merujuk F-01/F-02 dan hilang setelah keduanya diperbaiki                                                            |
| Modern Web Application                                           | Penanda TanStack Start pada HTML, bukan kelemahan                                                                                                                              |
| Authentication Request Identified                                | Endpoint login terdeteksi; sesuai desain                                                                                                                                       |
| SQL Injection (plugin 40018)                                     | Positif palsu: respons identik saat kondisi boolean dibalik, payload masuk ke PostgreSQL sebagai bind parameter, dan teks SQL hanya berisi placeholder. Triage lengkap di §5.2 |

## 7. Keterbatasan

1. Pemindaian berjalan pada konfigurasi pengembangan, bukan produksi: `APP_ENV=development`, tanpa TLS, dan batas laju dinaikkan. Frontend sudah memakai preview atas build produksi, tetapi lapisan deployment produksi (Nitro/nginx/ingress/CDN) belum diverifikasi, sehingga cakupan header di produksi tetap terbuka.
2. ZAP memakai UUID placeholder dari kontrak, sehingga IDOR/BOLA pada record nyata tidak diuji dinamis. Bukti A01 untuk hal ini berasal dari tes otomatis dan matriks role, bukan ZAP.
3. Pemindaian asli tidak dapat menilai `cuti` dan `tridharma/evaluasi` karena `5xx`; keduanya sudah diperbaiki dan diuji ulang secara langsung di luar ZAP (§5.1). Sembilan rute `404` (F-03) masih tidak dapat dinilai.
4. Active scan pada frontend tidak berjalan dengan sesi login, sehingga halaman terautentikasi (`/workspace`, `/admin/slo` dengan data) tidak dipindai.
5. Tidak ada pemindaian komposisi perpustakaan JavaScript (Retire.js) dan tidak ada pemindaian container/image.
6. Google OAuth, MFA, unggah file Cloudinary, dan alur ekspor/impor Excel tidak termasuk cakupan (Google OAuth dan MFA diimplementasikan di kode 2026-11-13 tetapi belum diuji live).
7. Rate limit, timeout dependency eksternal, RabbitMQ, SMTP, Redis, dan Cloudinary tidak diuji dinamis dalam sesi ini. QStash & S3 tidak dipakai dan tidak relevan untuk diuji.
8. Triage alert SQL Injection dibatasi pada parameter yang dilaporkan ZAP (`mimeType`, `fileUrl`, `tanggalMulai`, `description`) dan pada pemeriksaan kode seluruh jalur tulis repository. Belum ada fuzzing diperluas untuk setiap parameter pada body JSON.
9. CSP frontend masih memakai `'unsafe-inline'` untuk script dan style (F-09). Karena inline SSR masih dipakai, perlindungan terhadap injeksi script bergantung pada `default-src 'self'` dan `script-src 'self'`, bukan pada whitelist origin yang ketat.
10. Uji ini tidak menggantikan uji penetrasi dengan data dan beban produksi; hasil hanya berlaku untuk cakupan dan versi kode yang diuji.

## 8. Artefak dan reproduksi

Seluruh artefak berada di luar repo karena riwayat permintaan ZAP memuat token bearer sintetis.

| Artefak                                             | Lokasi                                                                                                                                      |
| --------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------- |
| Laporan API tanpa sesi                              | `C:\Users\humas\AppData\Local\Temp\zap-owasp\reports\api-unauth.json` dan `.html`                                                           |
| Laporan API dengan sesi admin                       | `C:\Users\humas\AppData\Local\Temp\zap-owasp\reports\api-auth-admin.json` dan `.html`                                                       |
| Laporan API tanpa sesi (setelah perbaikan)          | `...\reports\api-unauth-postfix.json` dan `.html`                                                                                           |
| Laporan API dengan sesi admin (setelah perbaikan)   | `...\reports\api-auth-admin-postfix.json` dan `.html`                                                                                       |
| Laporan API tanpa sesi (pemindaian ulang)           | `...\reports\api-unauth-rescan.json` dan `.html`                                                                                            |
| Laporan API dengan sesi admin (pemindaian ulang)    | `...\reports\api-auth-admin-rescan.json` dan `.html`                                                                                        |
| Ringkasan pemindaian ulang                          | `evidence\zap-postfix-scan.txt` dan `E:\code\sipeg\logs\zap-rescan-summary.txt`                                                             |
| Laporan frontend                                    | `C:\Users\humas\AppData\Local\Temp\zap-owasp\reports\frontend.json` dan `.html`                                                             |
| Laporan frontend (pemindaian ulang)                 | `...\reports\frontend-rescan.json` dan `.html`                                                                                              |
| Laporan frontend sebelum pemindaian ulang (salinan) | `...\reports\sebelum-rescan\frontend.json` dan `.html`                                                                                      |
| Rencana ZAP                                         | `plan-unauth.yaml`, `plan-auth.yaml`, `plan-frontend.yaml`                                                                                  |
| Rencana ZAP pemindaian ulang                        | `plan-unauth-rescan.yaml`, `plan-auth-rescan.yaml`, `plan-frontend-rescan.yaml`                                                             |
| Injeksi token                                       | `auth-sender.js`, `auth-sender-rescan.js`                                                                                                   |
| Kontrak untuk ZAP                                   | `openapi.yaml` (salinan yang diperbaiki), `openapi-rescan.yaml` (salinan `docs/openapi.yaml` kanonis)                                       |
| Skrip triage SQL Injection                          | `probe-sqli.cjs`, `probe-sqli.ps1`, `capture-sqli-log.ps1`, `capture-sqli-update.ps1`                                                       |
| Pemeriksa kontras dan font                          | `contrast-check.cjs`, `font-check.cjs`                                                                                                      |
| Matriks otorisasi                                   | `evidence\zap-authz-matrix.txt`, `evidence\zap-authz-matrix.ps1`                                                                            |
| Probe cakupan rute                                  | `evidence\zap-scope-probe.txt`, `evidence\zap-scope-probe.ps1`                                                                              |
| Ekspor audit setelah perbaikan F-07                 | `evidence\audit-export-admin-postfix.csv`, `evidence\audit-export-operator-postfix.csv`, `evidence\audit-export-filtered-admin-postfix.csv` |
| Log backend dan frontend                            | `logs\backend-8081.log`, `logs\backend-8081.err`, `logs\frontend-3001.err`, `logs\frontend-3001-rescan.log`                                 |

Menjalankan ulang:

```powershell
# ZAP daemon
& 'C:\Program Files\ZAP\Zed Attack Proxy\zap.bat' -daemon -host 127.0.0.1 -port 8090 `
  -config api.key=zap-owasp-2025 -dir E:\code\sipeg\logs\zap

# Rencana (sesi baru diperlukan agar alert tidak tercampur)
Invoke-RestMethod "http://127.0.0.1:8090/JSON/core/action/newSession/?name=owasp&overwrite=true&apikey=zap-owasp-2025"
Invoke-RestMethod "http://127.0.0.1:8090/JSON/automation/action/runPlan/?filePath=<path-plan>&apikey=zap-owasp-2025"
```

Sesi opaque uji dapat dicabut oleh `POST /api/v1/auth/logout`; jika token admin hilang di tengah pemindaian, sesi harus dibuat ulang sebelum rencana dijalankan lagi. `auth-sender.js` mencegah hal ini dengan tidak menyentuh `/api/v1/auth/*`.

Untuk pemindaian ulang, sesi admin dibuat langsung di tabel `session` pada database disposable (`insert into session (...) values (..., 'zapscan-admin-token-rescan', ...)`), sehingga pemindaian API tidak bergantung pada runtime autentikasi frontend. Token itu hanya berlaku untuk `simpeg_owasp` lokal dan tidak boleh dipakai di lingkungan lain.

## 9. Tindak lanjut

| Prioritas | Tindakan                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      | Rujukan |
| --------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------- |
| 1         | ~~Perbaiki klausa `Where` yang memakai identifier camelCase tanpa kutip ganda; tambahkan tes repository yang mengeksekusi SQL terhadap skema nyata~~ — selesai 2026-11-13, §5.1                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               | F-01    |
| 2         | ~~Perbaiki sintaks query `tridharma/evaluasi`~~ — selesai 2026-11-13, §5.1                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    | F-02    |
| 3         | ~~Putuskan per rute: daftarkan penanganan `arsip`, `golongan`, `keluarga`, `pelatihan`, `penghargaan`, `potongan`, `sertifikasi`, `sister`, `tunjangan`, atau hapus dari kontrak sampai siap~~ — selesai 2026-11-13: penyebabnya bukan rute rusak, melainkan struct `directoryAPI` tidak diisi field layanan di `RegisterDirectoryRoutes` sehingga blok `if api.<modul> != nil` tidak pernah aktif; seluruh field kini diisi dan rute terdaftar                                                                                                                                                                                                                                                               | F-03    |
| 4         | ~~Tetapkan header keamanan frontend pada server produksi (CSP, `X-Frame-Options`, `X-Content-Type-Options`); mulai CSP dalam mode report-only bila risiko inline script belum ditinjau~~ — selesai di repo 2026-11-13 (sesi lanjutan): dev dan preview server mengirim CSP **enforcing** (`script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; frame-ancestors 'none'`), `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, `Referrer-Policy: no-referrer`; header produksi pada lapisan deployment (Nitro/nginx/ingress/CDN) masih menunggu keputusan/verifikasi                                                                                                                   | F-04    |
| 5         | ~~Sinkronkan `docs/openapi.yaml` dengan router; tambahkan `CutiDTO`, `ArsipDTO`, `UserID`, dan deklarasi parameter `arsipId`~~ — selesai 2026-11-13: referensi menunjuk schema yang ada (`Cuti`, `Arsip`), parameter `UserID` (query) dan `ArsipId` (path) dideklarasikan, path `DELETE /api/v1/arsip/{arsipId}/archive` digabung ke `DELETE /api/v1/arsip/{arsipId}`, seluruh `$ref` resolve, dan tidak ada operasi terdokumentasi yang tidak terdaftar; penanganan rute tambahan pada modul SKP/tridharma serta sebagian rute presensi internal sempat tercatat sebagai drift dan ditutup 2026-11-14; seluruh 147 operasi terdaftar kini terdokumentasi di `docs/openapi.yaml` (0 hilang/ekstra, `DEC-066`) | F-05    |
| 6         | ~~Tambahkan CSP pada API sebagai pengerasan~~ — selesai 2026-11-13: `SecurityHeadersMiddleware` mengirim `Content-Security-Policy: default-src 'none'; frame-ancestors 'none'; base-uri 'none'` pada setiap respons                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           | F-06    |
| 7         | ~~Perbaiki keyset pagination ekspor audit agar tidak mengirim string kosong sebagai `uuid`, dan jangan kirim `200` sebelum batch pertama berhasil dibaca~~ — selesai 2026-11-13, §5.1                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                         | F-07    |
| 8         | ~~Validasi identifier asing pada jalur tulis: tolak string kosong/bukan UUID sebelum menyentuh SQL dan kembalikan `422`, bukan `503`; audit jalur `POST`/`PATCH` lain untuk pola yang sama~~ — selesai 2026-11-13: helper `requireUUID`/`requireOptionalUUID` memvalidasi di lapisan transport; jalur `POST`/`PATCH` yang diaudit dan ditambal mencakup modul `tridharma` (5), `skp` (3), `hr` (10), pegawai (create/patch, `homebaseId`/`kategoriPegawaiId`), organisasi (create/patch, `parentId`), dan cuti                                                                                                                                                                                                | F-08    |
| 9         | ~~Ulangi rencana ZAP setelah perbaikan F-01 sampai F-03 agar A01 dapat dinilai pada `cuti`, jadwal penggajian, dan `tridharma`~~ — selesai 2026-11-13; A01 kini teruji pada ketiganya, sembilan rute `404` tetap terbuka (§3.1)                                                                                                                                                                                                                                                                                                                                                                                                                                                                               | —       |
| 10        | ~~Pemindaian ulang ketiga rencana terhadap build terbaru, termasuk frontend~~ — selesai 2026-11-13: alert `SQL Injection` ditriage menjadi positif palsu (§5.2) dan F-04 terverifikasi hilang pada laporan frontend                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           | —       |
| 11        | Verifikasi header keamanan pada lapisan deployment produksi (Nitro/nginx/ingress/CDN)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                         | F-04    |
| 12        | Ganti `'unsafe-inline'` pada CSP frontend dengan nonce atau hash, lalu tinjau ulang kebijakan CSP                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             | F-09    |

## 10. Kesimpulan

Pada cakupan yang diuji, tidak ada kerentanan kritis atau tinggi yang membuka akses lintas role, eksekusi kode, atau kebocoran data. Otorisasi level rute konsisten menolak permintaan tanpa sesi dan menolak role tanpa grant, cakupan data pegawai bekerja, dan kegagalan internal tidak membocorkan detail.

Tiga cacat yang ditemukan sudah diperbaiki dan diverifikasi ulang pada backend hasil build terbaru dengan data uji nyata (§5.1): klausa camelCase tanpa kutip ganda (F-01), sintaks query `tridharma` (F-02), dan ekspor audit yang mengirim `200` dengan CSV kosong (F-07). Setelah perbaikan, `cuti` dan jadwal penggajian mengembalikan `200` dengan data, `tridharma/evaluasi` mengembalikan `200` termasuk jalur `sort`, dan ekspor audit menghasilkan 1.423 baris dengan cakupan per unit. Dengan demikian A01 tidak lagi terhalang `5xx` pada endpoint tersebut; yang tersisa adalah sembilan rute `404` (F-03) yang belum dapat diuji dinamis.

Pemindaian ulang ketiga rencana pada 2026-11-13 tidak mengubah kesimpulan itu. Alert berisiko tinggi `SQL Injection` yang muncul pada kedua rencana API ditriage menjadi positif palsu melalui pemeriksaan kode, reproduksi manual payload, dan jejak bind parameter di log PostgreSQL (§5.2). Pemindaian frontend mengonfirmasi F-04: tiga alert header keamanan yang sebelumnya muncul tidak lagi ditemukan, dan yang tersisa adalah risiko residual `unsafe-inline` pada CSP yang sudah diterima (`DEC-056`, F-09).

Klaim "semua kategori lulus" tetap belum dapat dibuat. Setelah penutupan drift rute 2026-11-14, penahan berikutnya bukan lagi ketidaksinkronan kontrak OpenAPI: yang menahan adalah header keamanan frontend yang belum diverifikasi pada lapisan produksi/deployment (F-04, dev/preview sudah menerapkan CSP enforcing), risiko residual CSP frontend (F-09), serta kebutuhan verifikasi produksi/integrasi nyata (F-04, OAuth, alerting) yang dicatat sebagai residual dalam `DEC-066`.

## 11. Perbaikan F-03 sampai F-08 (2026-11-13)

### F-03

Penyebab: `RegisterDirectoryRoutes` tidak mengisi field `keluarga`, `golongan`, `arsip`, `sertifikasi`, `pelatihan`, `sister`, `penghargaan`, `tunjangan`, dan `potongan` pada struct `directoryAPI`, sehingga blok `if api.<modul> != nil` tidak pernah aktif dan rute tidak terdaftar. Perbaikan: seluruh field diisi di `backend/internal/httpserver/directory_api.go`. `go build ./...` dan `go test ./...` lulus. Rute kini terdaftar; penilaian dinamis terhadap rute-rute ini (mis. akses file/Cloudinary, permintaan tulis) masih berada di luar cakupan pemindaian mana pun — tidak ada klaim keamanan baru yang dibuat hanya karena rute terdaftar.

### F-04

Dev dan preview server frontend (`frontend/vite.config.ts`, plugin `simpeg-security-headers`) kini mengirim: `Content-Security-Policy: default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'` (enforcing), `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, `Referrer-Policy: no-referrer`. CSP memakai `'unsafe-inline'` pada script/style karena kebutuhan inline SSR saat ini. Diverifikasi pada `bun run preview` 2026-11-13 (sesi lanjutan): header terkirim pada `/`, `/about`, `/privacy`, dan `/robots.txt`. Konfigurasi `nitro.routeRules` ditambahkan sebagai upaya best-effort untuk build/layanan produksi, tetapi cakupan header pada lapisan deployment produksi (Nitro/nginx/ingress/CDN/Cloudflare) tetap perlu diverifikasi; repo belum memiliki konfigurasi deployment akhir untuk itu.

Status setelah pemindaian ulang 2026-11-13: ketiga alert header pada laporan frontend lama tidak lagi muncul (§5.2). Font brand juga dipindahkan dari `fonts.googleapis.com` ke `frontend/public/fonts/` supaya CSP `font-src 'self'` tetap berlaku; kedua file font (Fraunces dan Manrope, lisensi OFL 1.1 ikut disimpan) termuat tanpa galat konsol. Aturan tautan global dipindahkan ke `@layer base` agar utilitas warna Tailwind pada tombol primer tetap menang; tombol utama terukur rasio kontras 6,40 terhadap latar hijau.

### F-05

- Referensi `#/components/schemas/CutiDTO` dan `ArsipDTO` diganti ke schema yang sudah ada (`Cuti`, `Arsip`).
- Parameter `UserID` (query, uuid) dan `ArsipId` (path, uuid) dideklarasikan; tiga referensi `parameters/ID` pada path arsip diganti menjadi `parameters/ArsipId` (`ID` tetap milik `cutiId`).
- `DELETE /api/v1/arsip/{arsipId}/archive` digabung menjadi `DELETE /api/v1/arsip/{arsipId}` (sesuai pola arsip-pegawai pada `/employees/{employeeId}`), sehingga tidak ada operasi terdokumentasi yang tidak terdaftar.
- Seluruh `$ref` pada kontrak resolve; pemeriksaan dilakukan dengan `js-yaml` (parse) dan pemindaian lokal seluruh `$ref`. Validasi akhir pascaperbaikan: YAML valid, 78 path, 1156 `$ref`, 0 hilang, 0 `operationId` duplikat (`DEC-069`).
- **Pembaruan status (2026-11-14):** drift rute yang tercatat di titik ini telah ditutup. Seluruh 147 operasi yang terdaftar di backend kini terdokumentasi di `docs/openapi.yaml` (78 path, 0 hilang/ekstra, 0 `operationId` duplikat), termasuk modul HR/SKP/Tridharma, `POST /api/v1/auth/session`, dan `POST /api/v1/internal/qstash/slo/evaluate`; kontrak dikoreksi pada `DEC-069`. Kondisi "94 penanganan rute belum terdokumentasi" adalah keadaan pada 2026-11-13 dan sudah tidak berlaku pada 2026-11-14. Ini adalah drift dokumentasi, bukan bukti ruta tak terlindungi (semua rute tetap deny-by-default).

### F-06

`SecurityHeadersMiddleware` mengirim `Content-Security-Policy: default-src 'none'; frame-ancestors 'none'; base-uri 'none'` pada setiap respons API (JSON-only; tidak ada dokumen yang sah dimuat). Tercakup tes `TestSecurityHeadersArePresent`.

### F-08

Lapisan transport kini memvalidasi identifier asing sebelum SQL: `requireUUID`/`requireOptionalUUID` (helper di `backend/internal/httpserver/directory_types.go`) mengembalikan `422` bila identifier bukan UUID. Jalur yang diaudit dan ditambal: `tridharma` (`evaluasi`, `pendidikan`, `penelitian`, `pengabdian`, `penunjang`; `userId`, `periodeAkademikId`, `dinilaiOleh`), `skp` (`log`, `penilaian`, `target`; `userId`, `skpPeriodeId`, `dinilaiOleh`), modul `hr` (`arsip`, `golongan`, `keluarga`, `pelatihan`, `penghargaan`, `potongan`, `sertifikasi`, `sister`, `tunjangan`, `cuti`; `userId`), pegawai (`homebaseId`, `kategoriPegawaiId` pada create/patch), dan organisasi (`parentId` pada create/patch). Tes: `backend/tests/httpserver/foreign_id_validation_test.go` (422 pada input tidak valid; jalur sukses tetap dilayani layanan).

Catatan batas verifikasi: perbaikan F-03 dan F-08 diverifikasi lewat build, `go vet`, dan `go test ./...` (lulus); tidak dijalankan terhadap database produksi. Pemindaian ulang 2026-11-13 memverifikasi F-04 pada frontend dan menandai alert `SQL Injection` sebagai positif palsu (§5.2). Delapan dari sembilan modul yang semula menjawab `404` (golongan, keluarga, pelatihan, penghargaan, potongan, sertifikasi, sister, tunjangan) tidak ikut dipindai karena tidak ada di kontrak `docs/openapi.yaml`; hanya `arsip` yang tercatat di kontrak dan karena itu ikut diimpor ZAP. F-03 dan F-08 belum dapat dinyatakan terverifikasi dinamis untuk modul-modul tersebut.
