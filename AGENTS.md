# Instruksi agen — SIMPEG

SIMPEG adalah sistem informasi kepegawaian. Repo ini masih fondasi, bukan aplikasi siap jalankan end-to-end.

## Struktur repo

- `backend/` — HTTP API Go/GORM (module `simpeg/backend`, Go 1.27.4). Entry point: `backend/cmd/server/main.go`.
- `backend/cmd/perfbench/` — perkakas operator untuk pengukuran terhadap dependensi nyata (seed, schema, explain, latency, e2e, deps, slo). Perkakas ini hanya membaca kredensial dari environment dan tidak boleh memuat nilai rahasia di dalam repo.
- `backend/migrations/` — sumber skema aplikasi dalam migrasi SQL Goose; backend memetakan tabel dengan GORM di `backend/internal/models`. Keberadaan migrasi tidak membuktikan bahwa migrasi sudah diterapkan pada database target.
- `backend/internal/slo/` — area Service Level Objective: `domain` (aturan dan perhitungan persentil), `app` (use case pemeriksaan dan alert), `repository` (port persistensi metrik), dan `collector.go` (pengoleki metrik dalam proses). Metrik disimpan di PostgreSQL, bukan Redis.
- `docs/` — `docs/PRD.md` (baseline produk; item proposed/open bukan persyaratan yang dikonfirmasi), `docs/openapi.yaml` (kontrak HTTP API OpenAPI 3.1), `docs/DECISIONS.md` (log keputusan: pemilik, tanggal, status), `docs/RUNBOOK.md` (tindakan operator saat insiden), `docs/IMPLEMENTATION-PLAN.md` (checklist status pekerjaan), `docs/THREAT-MODEL.md` (ancaman per area data dan dependency), `docs/PERFORMANCE-BASELINE.md` (hasil pengukuran performa, uji beban, dan uji soak beserta keterbatasannya).
- `backend/Dockerfile`, `docker-compose.yml` (stack backend: migrasi, penjadwal, API), dan `rabbitmq/` (stack broker RabbitMQ yang berdiri sendiri dengan jaringan `simpeg-broker`) — deployment docker/podman, lihat DEC-042.
- `agents/rules/` — `project.md`, `backend.md`, `frontend.md`, `architecture.md`. Baca sebelum mengubah kode; konvensi ini mengikat.
- Frontend (TanStack Start) direncanakan tapi belum dibuat. Tidak ada file `opencode.json`.
- Saat frontend mulai dibuat, ikuti `agents/rules/frontend.md` untuk urutan setup yang benar: scaffold dengan `bunx @tanstack/cli@latest create`, lalu `bunx shadcn@latest init -t start` **sebelum** menulis komponen, dan `bunx shadcn@latest add <component>` untuk komponen UI. Package manager adalah Bun — lockfile `bun.lock`, jangan `package-lock.json`/`yarn.lock`/`pnpm-lock.yaml`. Tooling mutu wajib dipasang sejak scaffold: Vitest, ESLint, `tsc --noEmit`, Playwright, Prettier, dan Zod (lihat seksi "Quality tooling" di `agents/rules/frontend.md`).

## Perintah pengembang (jalankan di `backend/`)

Format (menulis ulang file):

```bash
gofmt -w .
```

CI memeriksa format dengan `gofmt -l .` dan gagal jika ada file yang belum diformat — jalankan `gofmt -w .` sebelum push.

Verifikasi + tes + build (urutan CI):

```bash
go mod download
go mod verify
go vet ./...
go test ./...
go build ./...

```

Tes satu paket:

```bash
go test ./internal/httpserver/...
go vet ./internal/database/...
```

## Hal penting yang mudah terlewat

- `config.Load()` baca env melalui `os.Getenv` — tidak auto-load `.env`. Setel `DATABASE_URL` (dan `BETTER_AUTH_SECRET` untuk verifikasi JWT Better Auth) di lingkungan sebelum menjalankan server. `APP_ENV` default `development`; `PORT` default `8080`. Pada `APP_ENV=production` backend menolak start bila `BETTER_AUTH_SECRET` kosong/kurang dari 32 karakter/memuat pola bawaan, atau `TRUSTED_ORIGINS` kosong.
- CORS hanya mengizinkan origin dari `TRUSTED_ORIGINS`; wildcard ditolak. Kosong berarti tidak ada akses lintas origin.
- Batas laju per klien diterapkan `httpserver.RateLimiter` (in-memory per instance); endpoint autentikasi punya batas lebih ketat. Batas waktu statement database diterapkan lewat `DB_STATEMENT_TIMEOUT` pada context permintaan. Keduanya dikonfigurasi lewat env — lihat `backend/README.md`.
- Row Level Security diaktifkan default-deny pada seluruh tabel aplikasi lewat `backend/migrations/000004_row_level_security.sql`. Jangan menambahkan policy RLS tanpa persetujuan; policy harus mencerminkan cakupan organisasi pengguna. Migrasi itu sengaja tidak memakai FORCE agar backend (role pemilik tabel) tetap bisa diakses.
- Integrasi layanan pendukung memakai paket terpisah: `backend/internal/outbox` (antrean persisten di PostgreSQL), `backend/internal/qstash` (publisher + verifikasi signature raw body), `backend/internal/mq` (RabbitMQ untuk pekerjaan terjadwal self-hosted, lihat DEC-042), `backend/internal/notify` (SMTP), `backend/internal/cache` (Redis, cache-saja). Worker outbox berjalan di background bila SMTP dikonfigurasi. Jangan menambah kode yang tidak dipakai.
- Aturan yang ditegakkan kode dan tidak boleh dilanggar: notifikasi gaji tidak boleh memuat nominal (konstruktor `notify.SalaryPaidNotice` dan payload `notify.SalaryPaidPayload` tidak menerima parameter nominal) dan hanya boleh dikirim setelah pembayaran terkonfirmasi. Sumber konfirmasi sudah ada: modul penggajian (`internal/payroll`) menyimpan baris `pembayaran` yang menempel pada slip gaji, dan `notify.EmailDeliverer.Confirmations` memakai `payroll/app.Service.ConfirmedPayment`. Redis hanya untuk cache data referensi — kunci sesi, token, kredensial, penggajian, dan status pembayaran otomatis ditolak.
- Struktur backend memakai empat lapis per area bisnis: `domain` (aturan dan nilai tanpa I/O), `app` (use case yang memeriksa permission dan cakupan), `repository` (port persistensi + adapter GORM), dan `httpserver` (transport, DTO, middleware). Area direktori: `internal/directory/{domain,app,repository}`; area penggajian: `internal/payroll/{domain,app,repository}`. Kontrak daftar bersama ada di `internal/listing` (`listing.Options`). Jangan menaruh logika bisnis di handler; handler hanya menerjemahkan HTTP → use case dan error → status HTTP.
- Tes tidak butuh database nyata (handler pakai fake; tes `database` parsing skema GORM). `DATABASE_URL` hanya diperlukan untuk menjalankan server.
- Jangan pakai GORM `AutoMigrate` untuk skema. Migration runner dipilih: Goose (SQL berversi di `backend/migrations/`), dan workflow penerapannya ada di `.github/workflows/deploy-migrations.yml` (lihat DEC-033). Migrasi **sudah** diterapkan pada database Supabase `postgres` (18 versi Goose sejak 2026-11-11) dan pada database uji terpisah `simpeg_perf`; `000002_attendance_events.sql` sudah ikut terpasang bersama migrasi lain. Jangan menyimpulkan keadaan database dari isi repo — periksa `goose_db_version` pada database target.
- Endpoint yang butuh permission mengembalikan `403` bila role tidak punya grant (deny-by-default). `GET /api/v1/me` butuh header `Authorization: Bearer <session-token>`.
- Jika ada perubahan database, implementasikan struktur yang disetujui pada mapping GORM dan migrasi SQL Goose. Package `internal/models` menyediakan `models.All()` sebagai registry model; daftarkan model baru agar ikut pemeriksaan mapping. Tes database memeriksa tabel, indeks query kritis, dan cakupan migrasi RLS, tetapi tidak membuktikan migrasi sudah diterapkan atau seluruh kolom GORM sama dengan database target. Jalankan `go test ./internal/database/...` setelah mengubah model atau migrasi, dan `go run ./cmd/perfbench schema` untuk memeriksa keselarasan model dengan database target secara langsung.
- Keselarasan model dan skema harus diverifikasi terhadap database target, bukan disimpulkan dari kode. Model GORM menulis seluruh kolomnya pada `INSERT`, sehingga kolom yang dinyatakan model tetapi tidak ada pada tabel membuat setiap path yang menulis baris itu gagal. `docs/PERFORMANCE-BASELINE.md` §8 mencatat lima cacat yang ditemukan lewat pengukuran nyata, termasuk `pembayaran.updatedAt` yang membuat konfirmasi pembayaran gaji selalu gagal.
- Ukuran pool koneksi harus di bawah batas pooler penyedia. Pooler sesi Supabase membatasi jumlah koneksi concurrently per proyek; ketika `DB_MAX_OPEN_CONNS` melampaui batas itu, PostgreSQL menolak koneksi baru dengan `EMAXCONNSESSION` dan backend menjawab `503`. Bawaan repository adalah 10 koneksi.
- Backend memakai GORM dan migrasi Goose. Jangan gunakan GORM `AutoMigrate` untuk skema production. Status database harus diverifikasi melalui migrasi dan pemeriksaan database target, bukan disimpulkan dari kode model.
- Teks yang menghadap pengguna dan dokumentasi dalam Bahasa Indonesia; identifier kode dalam bahasa Inggris.
- Sumber kebenaran: keputusan/kebijakan produk yang disetujui ada di `docs/PRD.md` (khusus matriks akses: §11); checklist/status pekerjaan ada di `docs/IMPLEMENTATION-PLAN.md`; perilaku aktual harus diverifikasi pada konfigurasi entry point, middleware, dan handler backend. Jangan menyamakan persetujuan kebijakan dengan implementasi lengkap. Jika dokumen bertentangan dengan kode, laporkan dan perbaiki dokumentasi yang stale; jangan menambah grant atau mengklaim cakupan akses tanpa verifikasi route dan query.
- Status otorisasi saat ini: grant statis sudah dikonfigurasi di `backend/cmd/server/main.go` (matriks disetujui di `docs/PRD.md` §11). Permission berakhiran `:any` = lintas seluruh organisasi (khusus `admin`); permission berakhiran `:homebase` = dibatasi pada unit penugasan principal **beserta seluruh turunannya** pada pohon `Homebase` (`operator`, `atasan`); permission berakhiran `:self` = hanya record milik principal sendiri (`pegawai`). Cakupan ditegakkan di lapisan aplikasi pada data pegawai, unit organisasi, slip gaji, dan jejak audit: daftar difilter lewat `listing.Options.ScopeHomebaseIDs`, setiap record diperiksa per-pegawai, dan import operator menempatkan pegawai pada unit penugasannya. Operator hanya dapat membuat unit di bawah induk yang sudah berada dalam cakupan. Permission `audit:read` (admin) membuka `GET /api/v1/audit-logs` untuk seluruh jejak; `audit:read:homebase` (operator) hanya untuk unitnya; `category:read` membuka `GET /api/v1/categories`; `payroll:read:any`/`payroll:manage` (admin) membuka modul penggajian, `payroll:read:homebase` (operator) untuk unitnya, `payroll:read:self` (pegawai) untuk slipnya sendiri; `payroll:schedule` (admin, seluruh unit) dan `payroll:schedule:homebase` (operator, unitnya beserta turunannya) membuka `GET/PUT /api/v1/payroll/schedules`; `holiday:read` (admin, operator, atasan) membuka `GET /api/v1/holidays`; `holiday:manage` (admin) membuka `POST /api/v1/holidays/{year}/refresh`. **Batas yang belum terpenuhi:** ingestion presensi menerima sesi pengguna dengan permission `attendance:event:ingest` selain autentikasi perangkat (`X-Device-ID` + bearer token), sehingga endpoint itu belum hanya bisa diakses mesin; perubahan role pegawai melalui `PUT /api/v1/employees/{id}/role` sudah berjejak audit (`employee.role.updated`). Tes cakupan: `backend/internal/httpserver/scope_handlers_test.go`, `organization_scope_test.go`, `backend/internal/directory/app/employee_test.go`, `backend/internal/payroll/app/service_test.go`. Lihat `docs/PRD.md` §11 dan backend README sebelum mengubah izin atau menyatakan endpoint aman/tersedia.
- Jejak audit ditulis lewat paket `backend/internal/audit` dan bersifat **append-only** dari sisi kode aplikasi — tidak ada jalur hapus/ubah pada entri yang sudah tertulis; dijaga tes pemindai source. Satu-satunya penghapusan adalah retensi yang disetujui: `audit.Cleaner` menghapus entri yang lebih lama dari `AUDIT_RETENTION_DAYS` hari (default 90, `0` berarti tanpa pemotongan) dan pekerjaan itu dijalankan terjadwal lewat callback QStash. Setiap entri menyimpan `unitId` (unit cakupan saat aksi terjadi) sehingga akses audit dapat dibatasi per unit. Perubahan kolom ini perlu disinkronkan pada model GORM dan migrasi `backend/migrations/000003_audit_log_unit.sql`.
- Format error seragam di satu paket `backend/internal/apierror` (dipakai bersama oleh `auth` dan `httpserver`): `{"error":{"code","message","requestId","fields","rows"}}`. Gunakan `apierror.Code*`; jangan menulis bentuk error sendiri. Middleware `httpserver.JSONErrorMiddleware` mengubah 404/405 router menjadi JSON yang sama.
- Versi API memakai prefix path `/api/v1`. Konstanta `httpserver.APIPrefix` dan `httpserver.Version` — jangan hardcode `/api/...` di route atau tes; pakai `APIPrefix+"/..."`. Path tanpa versi ditolak `400 unversioned_path` oleh `VersionGuard`; versi tak dikenal `404`. Setiap respons memuat header `X-API-Version`. Versi naik hanya untuk perubahan yang merusak kompatibilitas.
- `/health` = liveness (selalu 200 saat proses hidup). `/ready` = readiness: memeriksa dependency wajib dengan timeout, mengembalikan 503 bila gagal, dan **tidak** membocorkan detail kegagalan. Jangan gabungkan keduanya.
- Log memakai `backend/internal/observability`: JSON satu baris per entri dengan correlation ID `X-Request-ID`. Header kredensial, body, dan nilai query string **tidak boleh** dicatat. Request ID dari klien hanya diterima bila karakternya aman, selain itu digenerate ulang.
- Sort dan filter daftar memakai `listing.Options` (`internal/listing`). Kolom sort berasal dari allowlist di sisi repository (`directory/domain/list.go`, `payroll/domain/sort.go`) — kolom SQL tidak boleh berasal dari input mentah. Nilai di luar allowlist ditolak `400 invalid_query`. Nama kolom camelCase harus dikutip ganda (`"createdAt"`) di SQL.
- Mata uang sistem adalah Rupiah Indonesia (IDR) dan zona waktu operasional adalah Asia/Jakarta (UTC+7). Penyimpanan timestamp memakai UTC (`internal/timeutil`, `database.OpenPostgres` memaksa `timezone=UTC`), sedangkan tampilan dan aturan kalender memakai Asia/Jakarta. Nominal diformat dengan `internal/money` (`money.Format` → `Rp 1.500.000`); jangan menulis format angka sendiri dan jangan memakai simbol mata uang lain.
- Callback QStash tidak memakai sesi bearer. `POST /api/v1/internal/qstash/audit/cleanup` diverifikasi dengan signature HS256 atas raw body dan subjek yang harus cocok dengan URL tujuan (`qstash.Verifier`). Jadwal harian didaftarkan lewat `internal/jobs.EnsureSchedule` dan idempoten: jangan membuat jadwal kedua untuk tujuan yang sama. Pada deployment self-hosted (`RABBITMQ_URL` terisi), pendaftaran jadwal QStash dilewati dan pekerjaan berpindah ke RabbitMQ: penjadwal `backend/cmd/jobsched` menerbitkan pesan dan server mengonsumsinya lewat `internal/mq` (DEC-042). Bentuk endpoint QStash bersifat khusus: publish dan jadwal memakai destination pada path, sedangkan cron, percobaan ulang, dan metode pada header `Upstash-*`. `QSTASH_URL` boleh berisi root API regional; jalur `/v2/publish` dan `/v2/schedules` ditambahkan otomatis. Endpoint jadwal harus mengikuti region token.
- Transport SMTP mendukung TLS langsung (porta 465) dan STARTTLS (porta 587). Peningkatan ke TLS terjadi otomatis bila server menawarkannya dan selalu sebelum autentikasi. Bentuk `SMTP_FROM` berupa `"Nama Tampilan" <alamat@domain>` sah dan dipakai apa adanya pada header, sedangkan perintah `MAIL FROM` memakai alamat polosnya (`notify.SplitAddress`).
- DTO publik ada di `backend/internal/httpserver/directory_types.go`, `audit_handlers.go`, dan `payroll_handlers.go`; model GORM tidak boleh diekspos langsung sebagai skema API.
- Backend mendukung dua jenis sesi: JWT Better Auth (verifikasi via `BETTER_AUTH_SECRET` dengan `golang-jwt/jwt/v5`) dan sesi opaque GORM (database token), melalui `auth.CombinedSessionResolver` di `backend/internal/auth/jwt.go`. `POST /api/auth/logout` (daftarkan via `httpserver.RegisterLogoutRoute`) hanya mencabut sesi opaque GORM; penrevokasian token JWT tetap milik runtime Better Auth.
- Resolusi sesi membaca role dari tabel `Role` pada setiap permintaan, jadi perubahan role berlaku tanpa sesi baru; akun nonaktif dan baris `session` yang dihapus ditolak sebagai `401`. Status implementasi tabel `session`/`account`/`verification`/`twofactor` ada di `docs/PRD.md` §11.
- File sumber yang ditulis tangan maksimal 387 baris; batas mutlak 397 baris hanya dengan alasan yang terdokumentasi; lebih dari 397 harus dibagi. File generated dikecualikan.
- Jangan komit kredensial, token, atau data pribadi nyata. `.env` berada di `.gitignore`.

## CI

- `.github/workflows/backend.yml` berjalan pada push/PR: `go mod download`, `go mod verify`, `gofmt -l .`, `go vet`, `staticcheck`, `go test -race`, `go build`, `govulncheck`, dan pemindaian kebocoran connection string. Belum ada workflow frontend.
- Dependabot dijadwalkan mingguan untuk Go dan GitHub Actions (`.github/dependabot.yml`).
