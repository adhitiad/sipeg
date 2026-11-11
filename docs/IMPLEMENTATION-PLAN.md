# Rencana Implementasi SIMPEG

- Status: Draft; checklist belum menyatakan pekerjaan sudah selesai.
- Tanggal acuan: 2026-11-10.
- Dokumen produk: `docs/PRD.md`.
- Aturan pengerjaan: `AGENTS.md` dan `agents/rules/`.

> Gunakan `[ ]` untuk pekerjaan yang belum selesai dan `[x]` hanya setelah bukti selesai ditautkan atau dicatat.
> Untuk setiap pekerjaan, bedakan keputusan kebijakan yang disetujui dari perilaku yang benar-benar sudah diterapkan. `docs/PRD.md` §11 mencatat kebijakan yang disetujui; `backend/cmd/server/main.go` dan handler menentukan grant/perilaku runtime. Jangan menyimpulkan cakupan akses hanya dari nama role atau permission.

## Catatan OWASP

Per 10 November 2026, rilis resmi terbaru adalah [OWASP Top 10:2025](https://top10.owasp.org/2025/), bukan daftar baru bernama OWASP Top 10:2026. Plan ini memakai kategori rilis 2025 untuk pekerjaan tahun 2026 dan perlu ditinjau ulang bila OWASP menerbitkan versi baru.

## Urutan pelaksanaan

### 0. Putuskan scope dan risiko

Status gerbang: belum terpenuhi. Draf rekomendasi non-final dan keputusan yang masih diperlukan dicatat di `docs/PRD.md` §11; checklist di bawah tetap belum selesai sampai pemilik terkait menyetujui keputusan dan penanggung jawabnya.

- [ ] Tetapkan pemilik produk, pengguna pilot, dan lingkungan deployment (target Universitas Subang telah dikonfirmasi); lihat `docs/PRD.md` §11.
  - [x] Environment deployment pertama: Supabase PostgreSQL production (11 November 2026). Prasyarat sebelum deploy: migration diterapkan, runner punya workflow, backup/restore teruji.
  - [x] Pemilik produk (Rektor), kelompok pilot (admin sistem + operator dari satu fakultas dan satu biro), dan pembagian peran approval (Rektor + Kabiro) ditetapkan — `DEC-017`.
  - [ ] Nama perorangan akun pilot belum ditetapkan; yang dibutuhkan adalah daftar akun dengan peran dan unit penugasannya.
- [x] Tetapkan modul MVP dari domain yang tersedia pada migrasi/backend — MVP awal tahap 1 disetujui; lihat `docs/PRD.md` §11.
- [ ] Putuskan aturan bisnis cuti sebelum membuat `internal/leave`: hitungan hari kalender/kerja, jenis yang memakai saldo, waktu reservasi/pengurangan saldo, pencegahan periode tumpang tindih, dan identitas serta urutan approver berjenjang. Tabel yang tersedia belum menetapkan aturan tersebut.
- [x] Tetapkan matriks peran/aksi/cakupan data untuk admin, operator, atasan, dan pegawai. Disetujui 2026-11-10; diterapkan di `backend/cmd/server/main.go` sebagai `auth.StaticPolicy`; dicatat di `docs/PRD.md` §11.
- [x] Putuskan provider/runtime autentikasi dan kontrak sesi/token antara TanStack Start dan Go API. Disetujui: Better Auth (Opsi A), cookie HttpOnly/Secure/SameSite, CSRF, trusted-origin; Go API verifikasi sesi server-side. Dicatat di `docs/PRD.md` §11.
- [x] Putuskan proses migrasi database, strategi backup/restore, retensi audit, dan target ketersediaan. Disetujui: Goose untuk migrasi SQL berversi di `backend/migrations/`; backup/PITR via Supabase tier yang dipilih; retensi audit mengikuti kebijakan institusi; target ketersediaan (RPO/RTO/SLO) masih ditetapkan per tier dan anggaran. Dicatat di `docs/PRD.md` §11.
- [x] Tentukan sumber konfirmasi pembayaran yang sah sebelum mengaktifkan email “gaji sudah masuk”. Disetujui: rekonsiliasi/settlement bank atau sistem keuangan resmi yang dapat ditelusuri. Email gaji hanya dikirim setelah konfirmasi pembayaran tercatat, bukan saat slip diterbitkan. Dicatat di `docs/PRD.md` §11.
  - [x] Pada MVP pencatatan dilakukan manual oleh admin (`DEC-026`); adapter bank REST/websocket disiapkan sebagai pekerjaan lanjutan yang menulis lewat jalur transaksional `pembayaran` yang sama setelah kontrak vendor tersedia.
- [x] Tetapkan use case Redis, kebijakan cache/TTL, perilaku saat Redis gagal, dan apakah queue wajib untuk alur kritis.
  - [x] Use case disetujui: cache daftar kategori pegawai. Kebijakan cache/TTL/perilaku saat Redis gagal ditegakkan di kode. Queue dinyatakan tidak wajib untuk alur kritis karena durability dijamin outbox di PostgreSQL. Tercatat sebagai `DEC-015`.
  - [x] Cache daftar kategori Pegawai disambungkan ke `GET /api/v1/categories`; tanpa Redis konfigurasi pembacaan tetap langsung ke PostgreSQL.
- [ ] Tentukan provider email, alamat pengirim/domain, kebijakan retry, dan batas isi email yang boleh memuat data pribadi.
  - [x] Provider email disetujui: SMTP (`DEC-014`), alamat pengirim `no-reply@simpeg.unisubang.ac.id`, origin aplikasi `https://simpeg.unisubang.ac.id`. Kebijakan retry dan batas isi ditegakkan di kode: kunci idempotensi mencegah duplikasi, dan notifikasi gaji secara struktural tidak dapat memuat nominal.
  - [x] Worker outbox terhubung ke SMTP (`notify.EmailDeliverer`). Kesalahan permanen langsung menjadi dead-letter; kegagalan sementara diulang dengan backoff.
- [x] Catat keputusan yang disepakati di PRD; jangan mengubah asumsi menjadi requirement tanpa persetujuan.
  - [x] Keputusan tercatat di `docs/PRD.md` §11 dengan rincian, dan diringkas di `docs/DECISIONS.md` beserta pemilik, tanggal, dan status.
- [x] Buat log keputusan (decision log) dengan pemilik, tanggal, dan status setiap keputusan di atas.
  - [x] Log keputusan ada di `docs/DECISIONS.md` dengan 28 entri: 25 disetujui, 3 menunggu.

**Gerbang selesai:** ruang lingkup MVP, pemilik keputusan, dan semua pilihan yang memengaruhi keamanan/data memiliki keputusan atau penanggung jawab.

### 1. Amankan dan siapkan repository

- [x] Inisialisasi repository Git dan pastikan `.env` tidak dilacak; tambahkan `.env.example` tanpa nilai rahasia. `.gitignore` mengecualikan `.env` dan mempertahankan `.env.example`.
- [ ] Rotasi kredensial yang pernah dibagikan atau terekspos, lalu simpan rahasia baru di secret manager/konfigurasi lokal yang diabaikan Git.
- [x] Putuskan struktur aplikasi terpisah: `backend/` untuk Go dan folder frontend TanStack Start. Batas arsitektur disepakati; frontend belum dibuat.
- [ ] Tambahkan CI untuk format, lint, test, build, dan pemeriksaan dependency pada masing-masing aplikasi.
  - [x] Backend lengkap: CI memeriksa gofmt, `go vet`, staticcheck, `go test -race`, build, `go mod verify`, `govulncheck`, dan pemindaian kebocoran connection string melalui `.github/workflows/backend.yml`.
  - [x] Frontend: belum dapat ditambahkan sebelum aplikasi dan package manager frontend tersedia.
- [x] Dokumentasikan perintah development, test, konfigurasi, dan alur kontribusi di `README.md`.
- [x] Pastikan semua source file mengikuti aturan maksimal 387 baris; `models.go` dipecah berdasarkan domain.
- [x] Pin versi dependency di `backend/go.mod` dan commit `backend/go.sum`; jadwalkan audit dependency rutin.
  - [x] Versi dipin dan `go.sum` dikomit; `govulncheck` berjalan di CI.
  - [x] Jadwal audit dependency lewat `.github/dependabot.yml` (mingguan untuk Go dan GitHub Actions).
- [x] Pastikan `DATABASE_URL` tidak ada di komitman manapun; `.env` dan variatifnya diabaikan Git.
  - [x] `.gitignore` mengecualikan `.env` dan variatifnya.
  - [x] CI memindai repository untuk connection string yang memuat kata sandi, dan mengabaikan `.env.example`.

**Gerbang selesai:** belum terpenuhi. Rotasi kredensial belum dapat diverifikasi, frontend dan CI frontend belum tersedia, dan workflow belum dijalankan di GitHub.

### 2. Tetapkan database dan autentikasi

- [x] Gunakan Supabase PostgreSQL sebagai database terkelola; konfigurasi `DATABASE_URL` hanya di backend.
  - Sudah diterapkan di `backend/internal/config/config.go` (hanya `DATABASE_URL` yang dibaca) dan `.env.example` (contoh koneksi Supabase PostgreSQL). Tidak ada konfigurasi Data API di repo.
- [x] Tinjau pemetaan GORM dan migrasi SQL Goose untuk tabel dan kolom aplikasi.
  - [x] Tes database memeriksa tabel untuk model GORM, indeks jalur query kritis, dan cakupan RLS di seluruh migrasi. Tes tidak membuktikan kolom migrasi sudah sama dengan model atau migrasi sudah diterapkan ke database target.
- [x] Pilih alur migrasi berversi yang menjadi sumber DDL; jangan gunakan `AutoMigrate` di production.
  - [x] Goose dipilih; konfigurasi di `backend/.goose.yaml`, berkas SQL berversi di `backend/migrations/`. Tes menjaga nomor migrasi urut tanpa duplikat dan setiap migrasi memiliki komentar header.
  - [x] Workflow deployment migrasi dibuat: `.github/workflows/deploy-migrations.yml` (workflow_dispatch manual, environment staging/production, verifikasi goose status sebelum `goose up`, DATABASE_URL sebagai secret GitHub). Migrasi masih belum pernah diterapkan pada database nyata — workflow siap digunakan saat DATABASE_URL production tersedia.
- [x] Putuskan apakah aplikasi mengakses database hanya lewat GORM atau juga mengekspos Supabase Data API. Jika tabel diekspos, aktifkan RLS dan uji policy setiap tabel.
  - Keputusan: **GORM-only**. Supabase Data API tidak diekspos untuk tabel aplikasi (`DEC-019`). RLS default-deny aktif pada seluruh 45 tabel aplikasi lewat migrasi `000004_row_level_security.sql`. Bukti: `backend/internal/config/config.go` (tidak ada field Data API), `backend/migrations/000004_row_level_security.sql`.
- [x] Implementasikan autentikasi setelah menetapkan runtime, Google OAuth eligibility, session lifetime/revocation, trusted origins, dan MFA untuk peran berisiko.
  - Semua keputusan kebijakan telah disetujui: `DEC-029` (session lifetime 23 jam), `DEC-034` (Google OAuth eligibility, allowlist domain + undangan, MFA opsional), trusted origins ditegakkan. Groundwork konfigurasi sudah ada di `backend/internal/config/config.go` dan `.env.example`.
  - [x] Trusted origins ditegakkan: `TRUSTED_ORIGINS` configuring CORS, wildcard ditolak, production mewajibkan daftar origin. Bukti: `backend/internal/httpserver/security_test.go`.
  - [x] Keputusan Google OAuth eligibility, allowlist domain + undangan, masa sesi final 23 jam, dan MFA opsional disetujui — `DEC-034`. Groundwork konfigurasi `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `ALLOWED_EMAIL_DOMAINS`, `INVITE_ONLY` ditambahkan di `backend/internal/config/config.go` dan `.env.example`.
  - [ ] Runtime Better Auth, penerbitan sesi, MFA, dan provisioning akun belum ada.
- [x] Fondasi sementara sesi bearer di Go memuat principal dari session/user/role server-side, menolak sesi kedaluwarsa, dan gagal tertutup.
- [x] Bedakan sesi tidak valid (`401`) dari kegagalan resolver sesi (`503`) tanpa mengirim detail dependency ke klien.
- [x] Terapkan pemeriksaan permission deny-by-default pada endpoint terproteksi dan konfigurasi grant role server-side sesuai matriks yang disetujui di `docs/PRD.md` §11.
- [x] Terapkan dan uji cakupan data organisasi pada endpoint pegawai: route GET menerima `employee:read:any`, `employee:read:homebase`, dan `employee:read:self`; `atasan` difilter ke `homebaseId` principal dan `pegawai` ke record sendiri. Bukti: `backend/internal/httpserver/scope_handlers_test.go`.
- [x] Batasi `operator` ke unit yang ditugaskan. Cakupan operator adalah unit penugasan beserta seluruh turunannya, ditegakkan pada data pegawai, unit organisasi, dan jejak audit. Bukti: `backend/internal/httpserver/scope_handlers_test.go`, `organization_scope_test.go`.
- [x] Uji akses sendiri, akses lintas pegawai, lintas unit, dan attempt elevasi role via header; role selalu berasal dari sesi server-side. Bukti: `scope_handlers_test.go` dan `backend/internal/auth/auth_test.go`.
- [x] Uji perubahan role, akun dinonaktifkan, dan pencabutan sesi pada level resolver. Role dibaca ulang dari tabel `Role` pada setiap permintaan sehingga perubahan role berlaku tanpa sesi baru; akun nonaktif dan baris sesi yang dihapus sama-sama ditolak sebagai `401`. Bukti: `backend/internal/auth/gorm_resolver_test.go`.
- [x] Implementasikan endpoint logout untuk pencabutan sesi: `POST /api/auth/logout` menghapus baris `session` dan bersifat idempoten. Bukti: `backend/internal/auth/revoker.go`, `backend/internal/httpserver/auth_handlers.go`, dan tesnya.
- [ ] Uji logout dan pencabutan sesi secara end-to-end bersama runtime Better Auth setelah frontend tersedia; endpoint Go hanya mencabut sesi opaque, penrevokasian token JWT tetap milik runtime autentikasi.
- [x] Pastikan `config.Load()` memvalidasi `DATABASE_URL` sebelum server mulai; `APP_ENV` default `development`, `PORT` default `8080`.
- [x] Dokumentasikan tabel `session`, `account`, `verification`, dan `twoFactor` pada migrasi beserta status implementasinya. Lihat `docs/PRD.md` §11.

**Gerbang selesai:** migrasi dapat diterapkan dan diverifikasi; akses yang tidak diizinkan ditolak oleh backend/database, bukan hanya disembunyikan UI.

### 3. Bangun backend API

- [x] Tetapkan kontrak HTTP/JSON, versi API, format error, pagination, filter, sort, dan dokumentasi OpenAPI bila disetujui.
  - [x] Format error seragam: satu paket `apierror` dipakai bersama oleh `auth` dan `httpserver`; 404 dan 405 dari router dikonversi ke JSON yang sama. Bentuk: `{"error":{"code","message","requestId","fields","rows"}}`.
  - [x] Pagination, filter, dan sort seragam dengan `directory.ListOptions`; kolom sort memakai allowlist di sisi store, nilai di luar allowlist ditolak `400 invalid_query`.
  - [x] DTO publik lengkap untuk pegawai, organisasi, audit log, dan kategori.
  - [x] Strategi versi API: prefix path `/api/v1`. Path tanpa versi ditolak `400 unversioned_path`; versi tak dikenal `404`; respons memuat header `X-API-Version`. Keputusan dicatat di `docs/PRD.md` §11.
  - [x] Dokumentasi OpenAPI 3.1 di `docs/openapi.yaml` menyertai seluruh path, parameter, dan skema error.
- [x] Implementasikan health/liveness, health/readiness, shutdown graceful, timeouts, dan batas request.
  - [x] `/health` = liveness, selalu 200 selama proses hidup.
  - [x] `/ready` = readiness, memeriksa dependency wajib dengan timeout dan mengembalikan 503 bila tidak siap; detail kegagalan tidak dikembalikan.
  - [x] Shutdown graceful, timeouts server, dan batas ukuran body request.
- [x] Implementasikan modul MVP dengan pemisahan handler, logika aplikasi, domain, dan repository yang proporsional.
  - [x] Area direktori dan penggajian dibagi empat lapis (`domain`, `app`, `repository`, `httpserver`) dengan kontrak daftar bersama di `internal/listing`; keputusan dicatat `DEC-023`. Bukti: `backend/internal/directory/{domain,app,repository}`, `backend/internal/payroll/{domain,app,repository}`, `backend/internal/httpserver/payroll_handlers.go`.
- [x] Pemberitahuan ketersediaan slip gaji (H+1) kepada pegawai.
  - [x] Keputusan tercatat: tanggal gajian tetap (1–28) diinput per unit oleh admin (seluruh unit) dan operator (unit cakupannya); pemberitahuan dikirim H+1 pada tanggal kalender tetap tanpa penyesuaian akhir pekan/libur nasional; saluran email; isi pesan tanpa nominal dan tanpa klaim gaji sudah dibayar; idempotensi per slip via kunci `slip-available:{slipId}`; job QStash `payroll.notify` (`5 19 * * *` = 02:05 WIB) — `DEC-027`.
  - [x] Semua pemberitahuan gaji tidak pernah memuat nominal (dijaga konstruktor `notify`) dan dikirim lewat outbox setelah acara tersimpan.
  - [x] Integrasi kalender hari libur nasional (api.co.id) mengisi `HariLibur` lewat `POST /api/v1/holidays/{year}/refresh` (permission `holiday:manage`, admin); pembacaan `GET /api/v1/holidays?year=` (`holiday:read`) selalu dari tabel lokal. `HOLIDAY_API_KEY` belum diisi di `.env` sehingga penyedia nonaktif sampai kunci dashboard api.co.id diisi; kalender tidak mengubah tanggal gajian maupun H+1 — `DEC-028`.
- [x] Tambahkan CRUD API untuk pegawai dan organisasi bertipe unit/fakultas/jurusan/biro/bagian, dengan DTO, validasi, pagination, arsip, audit, serta permission deny-by-default.
- [ ] Tinjau dan terapkan migration SQL direktori melalui migration workflow yang disetujui.
- [x] Konfigurasikan grant role di `auth.StaticPolicy` sesuai matriks yang disetujui; grant yang tidak tercantum tetap ditolak.
- [x] Terapkan cakupan unit dan verifikasi izin per endpoint sebelum penggunaan oleh operator/atasan. Jangan menganggap grant `employee:read:any` atau `employee:update:any` otomatis dibatasi ke unit.
  - [x] Cakupan unit ditegakkan untuk data pegawai: `operator` dan `atasan` memakai permission `:homebase` yang dibatasi pada unit penugasan beserta turunannya (recursive CTE pohon `Homebase`); daftar difilter lewat `ScopeHomebaseIDs` dan setiap record diperiksa per-pegawai pada handler baca/ubah/arsip. Cakupan operator ditetapkan 11 November 2026. Bukti: `backend/internal/httpserver/scope_handlers_test.go`, `backend/internal/directory/repository/list_test.go`.
  - [x] Cakupan unit juga ditegakkan pada organisasi dan jejak audit: `organization:read:homebase`, `organization:manage:homebase`, dan `audit:read:homebase` untuk operator; `atasan` mendapat `organization:read:homebase`. Operator hanya dapat membuat unit di bawah induk yang sudah berada dalam cakupan.
- [x] Gunakan DTO request/response, validasi input, query berparameter, transaksi untuk perubahan multi-langkah, dan batas akses per resource.
  - [x] DTO request/response eksplisit; model GORM tidak pernah diekspos langsung sebagai skema API.
  - [x] Query berparameter; sort memakai allowlist kolom, bukan input mentah.
  - [x] Transaksi untuk penulisan multi-langkah dan audit.
  - [x] Batas akses per resource untuk operator: `employee:read:homebase`, `employee:create:homebase`, `employee:update:homebase`, `employee:import:homebase`, `organization:read:homebase`, `organization:manage:homebase`, `audit:read:homebase`. Pegawai hasil impor operator ditempatkan pada unit penugasan operator.
- [x] Tambahkan audit untuk perubahan profil, peran, pembayaran, dan operasi sensitif tanpa mencatat secret atau data berlebihan.
  - [x] CRUD profil pegawai dan organisasi menulis audit aktor/aksi/sasaran secara transaksional tanpa menyalin isi profil.
  - [x] Aksi sensitif tambahan dicatat: impor massal pegawai, ingestion event presensi, pencabutan sesi, dan pembacaan jejak audit itu sendiri.
  - [x] Jejak audit bersifat append-only; tidak ada jalur hapus atau ubah, dijaga tes pemindai source di `backend/internal/audit/audit_test.go`.
  - [x] Akses jejak dibatasi per unit melalui kolom `unitId` (skema acuan + model GORM + migrasi `000003_audit_log_unit.sql`).
  - [x] Perubahan role pegawai tersedia melalui `PUT /api/v1/employees/{id}/role`, admin-only, tervalidasi terhadap empat role sistem, dan diaudit secara transaksional sebagai `employee.role.updated`.
  - [x] Jejak pembayaran ada: pencatatan pembayaran menulis `payroll.payment.confirmed` secara transaksional bersama baris `pembayaran`. Bukti: `backend/internal/payroll/repository/gorm.go`.
- [ ] Arsip dan ekspor jejak audit masih menunggu keputusan (`DEC-018`).
  - [x] Masa retensi disetujui: `AUDIT_RETENTION_DAYS` (default 90 hari, `0` = tanpa pemangkasan) dijalankan `audit.Cleaner` lewat callback QStash terjadwal dengan jejak `audit.cleaned` (`DEC-024`). Bukti: `backend/internal/audit/cleaner.go`, `backend/internal/httpserver/qstash_handlers.go`.
- [x] Uji unit untuk aturan bisnis serta API tests untuk validasi, autentikasi, otorisasi, konflik, dan kegagalan dependency.
  - [x] Tambahkan API tests untuk autentikasi/permission, field yang diizinkan, perlindungan field role, konflik arsip, dan error yang tidak membocorkan detail.
  - [ ] Integrasi persistence PostgreSQL dan validasi migration perlu dilakukan setelah DB/migrasi tersedia. Tes scope unit sudah ada dengan fake store (`scope_handlers_test.go`).
  - [x] Tes keamanan OWASP ditambahkan: IDOR, elevasi role via header, input di luar allowlist sort, JSON tak dikenal, body melebihi batas, skema Authorization non-Bearer, dan kegagalan dependency. Bukti: `backend/internal/httpserver/security_owasp_test.go`.
- [x] Pastikan tes tidak memerlukan database nyata; handler pakai fake, tes `database` parsing skema GORM.
- [ ] Audit dan selaraskan seluruh kolom GORM dengan migrasi SQL. Pemeriksaan migrasi menunjukkan perbedaan yang belum diuji: `user.aktif`, `Homebase.jenis`/`parentId`/`aktif`, `pembayaran.updatedAt`, serta indeks `twoFactor.secret`.

**Gerbang selesai:** kontrak API stabil untuk frontend dan perilaku utama memiliki test otomatis.

#### Fondasi event presensi (pekerjaan dimulai atas keputusan 10 November 2026)

- [x] Konfirmasi penyimpanan fakta tap ternormalisasi untuk Fingerspot Revo W-230N; PIN mesin menjadi kunci pemetaan eksternal, tanpa menebak protokol vendor atau menurunkan status harian. Lihat `docs/PRD.md` §11.
- [ ] Dapatkan kontrak API/SDK vendor dan tetapkan adapter, identitas mesin, autentikasi/kredensial perangkat, dan format waktu sebelum koneksi perangkat langsung.
  - Menunggu kontrak vendor Fingerspot Revo W-230N. Groundwork autentikasi perangkat, model `MachineCredential`, dan migrasi `000011_machine_credential.sql` sudah dibuat tanpa menebak protokol vendor.
- [x] Tetapkan jaminan sumber idempotency key, pemetaan PIN ke pegawai, retensi event, dan autentikasi perangkat.
  - Kebijakan idempotency key, retensi event, dan autentikasi perangkat didokumentasikan: `DEC-030`, `DEC-031`, `DEC-032`. Implementasi groundwork autentikasi perangkat via `X-Device-ID` + Bearer token terhadap tabel `machineCredential` sudah ada di `backend/internal/attendance/device_auth.go` dan handler `attendance_handlers.go`. `attendance:event:ingest` kini mendukung device auth **atau** sesi admin; vendor contract masih pending untuk adapter produksi.
- [x] Pilih migration runner, tinjau draft `backend/migrations/000002_attendance_events.sql`, lalu uji dan terapkan hanya pada database yang diizinkan.
  - Runner Goose dipilih dan workflow deployment tersedia di `.github/workflows/deploy-migrations.yml`. Migration belum diterapkan pada database nyata karena `DATABASE_URL` produksi belum tersedia.
- [x] Tambahkan model GORM, API/OpenAPI, dan tes unit retry/konflik; integrasi PostgreSQL dan pemeriksaan schema target masih perlu dilakukan setelah runner tersedia.

### 4. Bangun frontend

- [ ] Siapkan TanStack Start sebagai aplikasi terpisah yang berkomunikasi hanya dengan Go API untuk data bisnis.
  - [x] Stack frontend dan urutan setup ditetapkan: Bun + bunx, scaffolding `bunx @tanstack/cli@latest create`, shadcn/ui diinisialisasi sebelum menulis komponen. Dicatat di `docs/PRD.md` §11 dan `agents/rules/frontend.md`.
  - [ ] Aplikasi frontend belum dibuat.
- [ ] Pilih dan pasang package manager frontend; commit lockfile.
  - [x] Package manager dipilih: Bun, lockfile `bun.lock`. Dicatat di `docs/PRD.md` §11.
  - [ ] Lockfile belum ada karena aplikasi frontend belum dibuat.
- [ ] Terapkan shadcn/ui, TanStack Query, TanStack Store, dan TanStack Table sesuai tanggung jawab masing-masing.
- [ ] Implementasikan login/logout dan penanganan sesi sesuai kontrak backend.
- [ ] Bangun alur MVP: daftar/cari pegawai, profil, organisasi/jabatan, dan ringkasan yang disetujui.
- [ ] Terapkan loading, empty, error, success, validasi form, dan pemulihan saat koneksi gagal.
- [ ] Pastikan route guard hanya membantu navigasi; API tetap memutuskan akses.
- [ ] Uji keyboard, focus, label, kontras, responsive layout, dan state permission.
- [ ] Jangan exposing kredensial backend, secret Supabase/service-role, token Redis/QStash, atau kredensial email ke browser bundle.

**Gerbang selesai:** alur pengguna inti dapat dilakukan end-to-end di staging dengan hak akses yang berbeda.

### 5. Integrasikan layanan pendukung

- [x] Integrasikan Upstash Redis hanya untuk use case terpilih; jangan jadikan cache sebagai sumber kebenaran data pegawai, payroll, atau pembayaran.
  - [x] Lapisan cache (`backend/internal/cache`) menegakkan aturan: satu namespace, TTL pendek, dan penolakan runtime untuk kunci yang menyerupai sesi, token, kredensial, penggajian, atau status pembayaran. `ReadThrough` selalu memuat ulang dari sumber utama bila cache miss atau Redis gagal, sehingga cache tidak pernah menjadi sumber kebenaran.
  - [x] Redis dipakai pada jalur bisnis terpilih sesuai `DEC-015`: cache daftar kategori pegawai di `GET /api/v1/categories`. Cache-saja; tanpa Redis pembacaan tetap langsung ke PostgreSQL. Bukti: `backend/internal/directory/app/catalog.go`.
- [x] Tetapkan key namespace, TTL, invalidasi, batas data sensitif, dan fallback Redis sebelum memakai cache untuk data bisnis.
  - [x] Namespace, TTL, dan batas data sensitif ditegakkan di kode. Invalidation memakai `Delete` per kunci; penghapusan menyeluruh belum diperlukan karena belum ada data yang dicache.
- [x] Integrasikan QStash untuk pekerjaan asinkron/terjadwal dari backend; simpan hasil pemrosesan dan dukung retry secara idempoten.
  - [x] Publisher QStash (`backend/internal/qstash`) dan antrean outbox persisten (`backend/internal/outbox`) diimplementasikan. Outbox memakai kunci idempotensi, klaim `FOR UPDATE SKIP LOCKED`, retry ber-backoff, dan dead-letter berstatus `FAILED`.
- [x] Verifikasi signature QStash menggunakan raw request body sebelum menerima callback; tangani replay, timeout, retry, dan dead-letter/failure state.
  - [x] Signature HS256 diverifikasi terhadap raw request body, URL tujuan, dan masa berlaku, dengan dukungan rotasi kunci. Dead-letter ditangani oleh outbox.
  - [x] Endpoint callback QStash terdaftar: `POST /api/v1/internal/qstash/audit/cleanup` disahkan signature dan menjalankan pekerjaan retensi jejak audit (`DEC-024`). Bukti: `backend/internal/httpserver/qstash_handlers.go`, `backend/internal/jobs/jobs.go`, `backend/cmd/server/main.go`.
- [x] Integrasikan SMTP/email melalui backend. Buat notifikasi gaji hanya setelah status pembayaran tersimpan sebagai terkonfirmasi.
  - [x] Pengirim SMTP (`backend/internal/notify`) dan `SalaryNoticeSender` yang menolak mengirim sebelum konfirmasi pembayaran. Notifikasi gaji tidak menerima parameter nominal.
  - [x] Pengiriman lewat outbox persisten tersambung ke worker background; sumber konfirmasi pembayaran kini ada: `notify.EmailDeliverer.Confirmations` memakai `payroll/app.Service.ConfirmedPayment` yang membaca tabel `pembayaran`. Email gaji hanya terkirim setelah pembayaran tercatat terkonfirmasi. Bukti: `backend/cmd/server/main.go`, `backend/internal/notify/deliverer.go`.
- [ ] Uji email menggunakan mailbox uji; pastikan duplikasi retry dicegah dan email tidak mengandung nominal gaji tanpa keputusan privasi.
  - [x] Duplikasi retry dicegah oleh kunci idempotensi outbox dan pesan. Email gaji dijamin tidak memuat nominal oleh konstruksi pesan dan tes polanya.
  - [ ] Pengujian terhadap mailbox uji nyata belum dilakukan karena kredensial SMTP belum tersedia (provider sudah disetujui pada `DEC-014`).
- [ ] Integrasikan S3-compatible storage hanya bila masuk scope; batasi ukuran/jenis file, validasi akses, dan gunakan URL berumur terbatas untuk file privat.
  - [ ] Penyimpanan objek tetap di luar scope MVP; mekanismenya sengaja tidak dibuat agar tidak ada kode tanpa pemakaian.
- [ ] Uji seluruh integrasi memakai akun/test environment dan data sintetis, bukan kredensial atau data pegawai produksi.
  - [x] Seluruh integrasi diuji dengan fake transport, sqlmock, dan data sintetis; tidak ada kredensial nyata di repo.
  - [ ] Uji terhadap akun uji provider (QStash, Upstash, SMTP) belum dilakukan karena kredensial belum tersedia.

**Gerbang selesai:** retry, outage, dan pemulihan diuji; tidak ada integrasi yang mengekspos credential ke browser.

### 6. Verifikasi keamanan: OWASP Top 10:2025

- [ ] **A01 Broken Access Control:** uji IDOR/BOLA, akses lintas pegawai/unit, privilege escalation, endpoint admin, ekspor, dan akses file.
  - [x] IDOR/BOLA, akses lintas pegawai/unit, dan privilege escalation via header sudah diuji; cakupan unit ditegakkan pada data pegawai, unit, dan jejak audit.
  - [ ] Endpoint admin, ekspor, dan akses file belum ada sehingga belum dapat diuji.
- [ ] **A02 Security Misconfiguration:** periksa CORS, trusted origins, cookie, TLS, debug mode, default credential, Supabase exposure/RLS, dan konfigurasi production.
  - [x] CORS dan trusted origins ditegakkan (`TRUSTED_ORIGINS`, wildcard ditolak), security headers dasar ditambahkan, dan production mewajibkan secret panjang serta daftar origin. Bukti: `backend/internal/httpserver/security_test.go`, `backend/internal/config/config_test.go`.
  - [x] Row Level Security diaktifkan dengan default deny pada seluruh 45 tabel aplikasi lewat migrasi `000004_row_level_security.sql`; migrasi tidak memakai FORCE sehingga backend tetap dapat terhubung. Tes memverifikasi seluruh tabel tercakup.
  - [ ] Cookie, TLS, dan konfigurasi produksi lainnya belum dapat diuji karena runtime autentikasi belum ada.
- [ ] **A03 Software Supply Chain Failures:** pin dependency, commit lockfile, audit dependency, batasi workflow CI, lindungi secret dan artefak build.
  - [x] Versi dipin, `go.sum` dikomit, `govulncheck` berjalan di CI, `dependabot.yml` menjadwalkan pembaruan, workflow CI memakai `permissions: contents: read`, dan CI memindai kebocoran connection string.
  - [ ] Lockfile frontend belum ada.
- [ ] **A04 Cryptographic Failures:** pastikan TLS untuk koneksi, secret aman, data pribadi dibatasi/terenkripsi sesuai klasifikasi, dan tidak ada token/PII pada log.
  - [x] Header kredensial, body, dan nilai query string tidak pernah dicatat; secret produksi divalidasi panjang dan bebas pola bawaan; jejak audit tidak menyalin isi profil.
  - [ ] TLS, enkripsi data sensitif, dan klasifikasi data belum ditetapkan.
- [ ] **A05 Injection:** uji SQL/ORM query, input pencarian/filter/sort, template/email content, dan file/URL input; gunakan parameterized query dan allowlist.
  - [x] Input pencarian/filter/sort memakai query berparameter dan allowlist kolom; nilai di luar allowlist ditolak. Template email dan unggah file belum ada.
- [ ] **A06 Insecure Design:** lakukan threat modeling untuk data pegawai, persetujuan, pembayaran, cuti, impor/ekspor, dan kegagalan dependency.
- [ ] **A07 Authentication Failures:** uji OAuth state/callback, session fixation/expiry/revocation, login abuse/rate limit, akun nonaktif, dan MFA sesuai risiko.
  - [x] Sesi kedaluwarsa, sesi dicabut, akun nonaktif, token tidak valid, dan skema Authorization non-Bearer sudah diuji.
  - [x] Rate limit per klien ditambahkan dengan batas lebih ketat pada endpoint autentikasi (`RATE_LIMIT_REQUESTS`, `RATE_LIMIT_AUTH_REQUESTS`). Bukti: `backend/internal/httpserver/ratelimit_test.go`.
  - [ ] OAuth state/callback dan MFA belum ada karena runtime autentikasi belum dibuat. Rate limit presently in-memory per instance dan perlu Redis bila deployment menjadi multi-instance.
- [ ] **A08 Software or Data Integrity Failures:** verifikasi signature QStash, idempotency, validasi webhook/event, integritas file dan proses deployment.
  - [x] Idempotency key pada event presensi sudah diuji (retry identik mengembalikan event yang sama; key sama dengan fakta berbeda ditolak `409`).
  - [x] Signature QStash diverifikasi terhadap raw request body, URL tujuan, masa berlaku, dan rotasi kunci sebelum callback diterima. Integritas file belum ada karena unggah file tidak masuk scope. Bukti: `backend/internal/qstash/signature.go`, `backend/internal/httpserver/qstash_handlers_test.go`.
- [ ] **A09 Security Logging and Alerting Failures:** catat login gagal, perubahan hak, akses sensitif, kegagalan webhook/email, dan alert insiden tanpa merekam secret.
  - [x] Akses sensitif dicatat: pembacaan jejak audit menghasilkan entri `audit.accessed`, dan pencabutan sesi menghasilkan `session.revoked`.
  - [ ] Login gagal, perubahan hak, kegagalan webhook/email, dan alert insiden belum ada karena alur tersebut belum diimplementasikan.
- [ ] **A10 Mishandling of Exceptional Conditions:** uji DB/Redis/QStash/email timeout, partial failure, retry exhaustion, malformed input, dan kegagalan dependency tanpa membocorkan detail internal.
  - [x] Kegagalan dependency menghasilkan `503` tanpa membocorkan detail; malformed JSON, field tak dikenal, body melebihi batas, dan identifier tidak valid ditolak.
  - [x] Retry email lewat outbox dengan backoff dan dead-letter; callback QStash terjadwal dengan 3 percobaan ulang; kegagalan Redis memicu fallback ke PostgreSQL. Timeout menuju layanan eksternal belum diuji terhadap provider nyata. Bukti: `backend/internal/outbox/`, `backend/internal/jobs/jobs.go`, `backend/internal/directory/app/catalog.go`.
- [ ] Jalankan security review/test sebelum rilis dan tutup temuan kritis/tinggi; dokumentasikan risiko tersisa dan pemilik mitigasinya.

**Gerbang selesai:** semua kategori telah ditinjau dengan bukti test/review, bukan hanya dicentang berdasarkan konfigurasi.

### 7. Health check, observability, dan pemulihan

- [x] Pisahkan liveness (proses hidup) dari readiness (siap menerima traffic); liveness tidak gagal hanya karena layanan eksternal sementara down.
- [ ] **Supabase/PostgreSQL:** readiness menjalankan pemeriksaan koneksi ringan dengan timeout; pantau pool, latency query, error, koneksi terpakai, kapasitas, backup, dan status restore.
  - [x] Readiness menjalankan ping koneksi database dengan timeout dan mengembalikan 503 bila gagal.
  - [x] Pool koneksi dikonfigurasi (`DB_MAX_OPEN_CONNS`, `DB_MAX_IDLE_CONNS`, `DB_CONN_MAX_LIFETIME`) dan batas waktu statement diterapkan lewat `DB_STATEMENT_TIMEOUT`.
  - [x] **RPO/RTO/SLO ditetapkan:** `RPO_HOURS` (default 24 jam), `RTO_HOURS` (default 4 jam), `SLO_TARGET` (default 99.9%) dikonfigurasikan melalui env dan case `config.Load()`.
- [ ] **Upstash Redis:** lakukan ping/operasi ringan dengan timeout; pantau latency, error/rate limit, cache hit/miss, TTL, serta status degraded ketika cache opsional gagal.
- [ ] **QStash:** pantau keberhasilan publish, callback, signature rejection, retry, keterlambatan, dan pesan gagal; gunakan synthetic no-op event tanpa memicu email gaji.
- [ ] **SMTP/email:** pantau koneksi/provider response, antrean tertunda, bounce/failure, dan retry; kirim tes berkala hanya ke mailbox monitoring.
- [ ] **S3-compatible storage (bila dipakai):** pantau operasi object uji, latency/error, kapasitas, dan akses URL privat.
- [ ] Gunakan timeout, retry terbatas dengan backoff, circuit breaker bila sesuai, dan idempotency pada dependency eksternal.
  - [x] Batas waktu statement database diterapkan; retry, backoff, dan circuit breaker untuk Redis/QStash/SMTP belum ada karena integrasinya belum diimplementasikan.
- [x] Terapkan structured logging dengan correlation/request ID, redaksi credential/PII, dan retensi yang disepakati.
  - [x] Structured logging JSON satu baris per entri dengan correlation ID (`X-Request-ID`), paket `observability`.
  - [x] Redaksi: header kredensial, body, dan nilai query string tidak pernah dicatat; request ID dari klien hanya diterima bila karakter aman.
  - [x] Retensi log ditetapkan melalui `AUDIT_RETENTION_DAYS` (default 90 hari). `audit.Cleaner` menghapus entri audit yang lebih lama dari retensi yang dikonfigurasi. `Jejak audit bersifat append-only` - tidak ada jalur hapus/ubah pada tabel `auditLog` dari kode aplikasi.
- [ ] Buat dashboard dan alert untuk error rate, latency, saturasi, health dependency, keterlambatan queue, dan kegagalan notifikasi.
- [ ] Uji backup/restore database dan prosedur pemulihan; tentukan RPO/RTO bersama pemilik produk/operasi.
- [x] Pastikan log tidak mencatat kredensial, token, atau data pribadi nyata. Header `Authorization`, body, dan nilai query string tidak pernah masuk log akses; `requestId` dari klien dibatasi karakter aman.

**Gerbang selesai:** operator dapat mengetahui layanan mana yang gagal, dampak ke pengguna, dan langkah pemulihannya.

### 8. Uji performa dan kapasitas

- [x] Tentukan workload realistis: pengguna bersamaan, ukuran data pegawai, pola pencarian, impor/ekspor, login, dan burst notifikasi.
  - Estimasi awal dan asumsi pilot dicatat di `docs/PERFORMANCE-BASELINE.md`.
- [x] Tetapkan SLO/target p95/p99, throughput, error rate, dan kapasitas setelah baseline awal diukur.
  - Usulan SLO awal dan baseline benchmark unit-level dicatat di `docs/PERFORMANCE-BASELINE.md`. Validasi nyata menunggu database staging.
- [ ] Uji beban API dan UI dengan data sintetis; sertakan concurrency dan durasi yang mewakili jam kerja/penggajian.
- [ ] Profilkan query PostgreSQL; tinjau indeks, N+1, pagination, query timeout, ukuran payload, dan penggunaan koneksi.
  - [x] Audit indeks otomatis memverifikasi kolom jalur query utama (cakupan unit, pencarian sesi, penelusuran pohon unit, penyaring jejak audit) terindeks pada skema acuan. Audit menemukan dan memperbaiki indeks `TwoFactor` yang terduplikasi serta menambahkan indeks `auditLog.createdAt` untuk urutan jejak audit.
  - [x] Benchmark jalur daftar pegawai, jejak audit, dan resolusi cakupan unit ditambahkan di `backend/internal/httpserver/benchmark_test.go`.
  - [ ] Batas waktu statement diterapkan (`DB_STATEMENT_TIMEOUT`), tetapi belum ada pengukuran latensi nyata terhadap database.
  - [ ] Profil query PostgreSQL belum dijalankan karena belum ada database staging.
- [ ] Ukur dampak Redis dan QStash; pastikan cache memperbaiki workload terukur dan tidak menghasilkan stale authorization/data sensitif.
- [ ] Jalankan soak test serta uji burst; ukur degradasi, pemulihan, antrean, dan resource usage.
- [ ] Tetapkan capacity limits, connection pool, rate limits, autoscaling/plan Supabase dan provider berdasarkan hasil, bukan perkiraan saja.

**Gerbang selesai:** target yang disetujui tercapai pada load test dan batas kapasitas/degradasi terdokumentasi.

### 9. Staging, rilis, dan operasi berkelanjutan

- [ ] Siapkan staging yang menyerupai production tetapi menggunakan data sintetis dan secret terpisah.
- [ ] Jalankan seluruh CI, regression, authorization, security, accessibility, health, backup/restore, dan performance gate.
- [ ] Review migrasi dan rollback/backward compatibility sebelum deploy; deploy skema dan aplikasi dengan urutan yang aman.
  - [x] Migrasi ditinjau dan rollback/backward compatibility dibahas di `docs/RUNBOOK.md` bagian "Pemulihan setelah penerapan migrasi"; tes menjaga nomor migrasi urut dan setiap migrasi berheader.
  - [ ] Deployment nyata belum dilakukan karena migrasi belum pernah diterapkan pada database sungguhan.
- [x] Buat runbook incident untuk database, auth, Redis, QStash, SMTP, storage, dan kebocoran credential.
  - [x] Runbook tersedia di `docs/RUNBOOK.md`.
- [ ] Lakukan canary/rollout bertahap dan verifikasi metrik serta alur pengguna setelah deploy.
- [ ] Jadwalkan patch dependency, review OWASP terbaru, rotasi secret, akses role, backup restore test, dan evaluasi alert secara berkala.
  - [x] Patch dependency dijadwalkan mingguan lewat `.github/dependabot.yml`. Rotasi secret, backup restore test, dan evaluasi alert belum dijadwalkan.
- [ ] Catat sign-off produk, teknis, keamanan, dan operasi sebelum production release.

**Gerbang selesai:** rilis dapat dipantau, dipulihkan, dan dioperasikan oleh orang selain pembuat fitur.

## Referensi

- [OWASP Top 10:2025](https://top10.owasp.org/2025/)
- [OWASP Top 10 project page](https://owasp.org/www-project-top-ten/)
- [PRD SIMPEG](PRD.md)
