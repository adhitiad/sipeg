# SIMPEG

**Sistem Informasi Kepegawaian (SIMPEG)** adalah platform pengelolaan data dan proses kepegawaian yang dikembangkan untuk kebutuhan institusi (target: Universitas Subang). Proyek ini memisahkan **backend API** (Go) dan **frontend web** (TanStack Start) dengan kontrak HTTP/JSON berversi.

> **Status proyek:** dalam pengembangan aktif. Repositori ini berisi fondasi, modul inti, dan integrasi pendukung; sebagian alur belum end-to-end siap produksi. Detail kebijakan, batas implementasi, dan keputusan terbuka terdokumentasi di `docs/`.

---

## Ringkasan

| Aspek       | Keterangan                                                                                                    |
| ----------- | ------------------------------------------------------------------------------------------------------------- |
| Backend     | Go + GORM, API HTTP/JSON di bawah `/api/v1`, migrasi SQL Goose                                                |
| Database    | PostgreSQL (Supabase), skema dikelola lewat migrasi berversi (bukan `AutoMigrate`)                            |
| Frontend    | TanStack Start (React), Bun, shadcn/ui, TanStack Query/Store/Table                                            |
| Autentikasi | Better Auth tertanam di server frontend (`/api/auth/*`), sesi bearer ke API (`Authorization: Bearer <token>`) |
| Kontrak API | OpenAPI 3.1 (`docs/openapi.yaml`), 135 path / 231 operasi, sinkron dengan router backend per 16 November 2026 |

### Cakupan yang sudah tersedia (ringkas)

- **Direktori & organisasi:** data pegawai, unit/homebase, kategori, master jabatan, impor pegawai, arsip dokumen (metadata), serta struktur organisasi.
- **Akses & audit:** otorisasi deny-by-default berbasis permission/role, cakupan data per unit/self/any, jejak audit append-only dengan retensi terjadwal, ekspor audit (CSV).
- **Presensi:** ingestion event mesin, pembacaan record/ringkasan sesuai cakupan (status harian diturunkan dari record yang ada; fondasi/pemetaan perangkat masih terbatas).
- **Penggajian:** slip gaji (PDF A5 bermerek + QR verifikasi publik), konfirmasi pembayaran manual (transaksional, notifikasi tanpa nominal), jadwal gajian per unit, kalender hari libur, dan mekanisme notifikasi berbasis outbox. Modul penggajian inti yang **sudah terimplementasi**: master skala gaji yayasan, komponen tunjangan & potongan, honor per SKS, BPJS, PPh 21 TER, THR (termasuk `PERSEN_TAKE_HOME`), kontrak PKWT/PKWTT + perpanjangan pasca-BUP, aturan pesangon, alur pemberhentian (SK PHK otomatis), dan master pensiun (konfigurasi per kategori status + kepesertaan DPLK). Aturan bisnisnya disetujui pada `DEC-067`/`DEC-070`; asumsi yang masih menunggu konfirmasi pemilik tercatat di `DEC-071` (penyedia & iuran DPLK, pelaporan LLDikti, potongan koperasi/pinjaman).
- **Akses berbasis peran (RBAC scoped, `DEC-087`–`DEC-089`):** matriks grant statis (deny-by-default) tetap sumber kebijakan, dan lapisan RBAC berbasis _scoped assignment_ mulai menegakkan keputusan secara otoritatif bagi pengguna yang sudah punya assignment. Modul admin RBAC (`/api/v1/admin/rbac/*`) mengelola role, permission, matriks role-permission, assignment per pengguna beserta cakupan unit, dan delegasi; UI `/admin/rbac`. Perkakas `backend/cmd/rbacseed` memetakan grant statis ke data RBAC untuk pengguna yang sudah ada (dry-run bawaan, `--apply` idempoten).
- **Administrasi:** arsip dokumen, master Surat Keputusan (baca-saja — SK terbit dari alur kerja seperti persetujuan pemberhentian yang menerbitkan SK-PHK; penerbitan manual belum diputuskan), serta modul RBAC admin.
- **Operasional:** health/readiness terpisah, logging terstruktur dengan correlation ID, rate limit & security headers, metrik SLO admin (agregat), status operasional antrean keluar (agregat outbox), dukungan pekerjaan terjadwal via RabbitMQ (self-hosted) atau QStash (alternatif).
- **Observabilitas & pemulihan (templat $0, self-hosted):** stack Prometheus/Alertmanager/Grafana/Loki + exporter host/container/blackbox/RabbitMQ di `observability/`, formulir & helper uji backup/restore di `docs/backup-restore-drill.md` dan `scripts/backup-restore-drill.ps1`. Pemasangan di target produksi belum terbukti — lihat `docs/IMPLEMENTATION-PLAN.md` §7.

> Batas penting: tidak semua keputusan kebijakan di `docs/PRD.md` berarti sudah terimplementasi penuh. Selalu verifikasi perilaku aktual pada konfigurasi, middleware, handler, dan database target.

---

## Struktur repositori

```text
.
├─ backend/                 # API Go/GORM, migrasi, worker/penjadwal, perkakas operator
│  ├─ cmd/                  # server, migrate, jobsched, perfbench
│  ├─ internal/             # domain/app/repository/httpserver dan integrasi pendukung
│  └─ migrations/           # sumber skema SQL Goose
├─ frontend/                # Aplikasi web TanStack Start (Bun)
├─ docs/                    # PRD, OpenAPI, keputusan, runbook, threat model, baseline, dsb.
├─ observability/           # Templat stack monitoring/alerting/log self-hosted ($0, Docker)
├─ rabbitmq/                # Stack broker RabbitMQ mandiri + jaringan simpeg-broker
├─ scripts/                 # Perkakas operator (mis. helper uji backup/restore)
├─ docker-compose.yml       # Stack backend: migrations, jobsched, backend
├─ agents/rules/            # Aturan proyek, backend, frontend, arsitektur
└─ AGENTS.md                # Instruksi agen & konvensi repo
```

---

## Prasyarat

- **Go** sesuai `backend/go.mod` (saat ini Go 1.27.4)
- **Bun** (untuk frontend)
- **PostgreSQL** (Supabase atau kompatibel) untuk menjalankan backend secara nyata
- **Docker/Podman** (opsional, untuk deployment/stack tercontainer)

---

## Menjalankan secara lokal

### Backend

Backend membaca konfigurasi dari environment (tidak auto-load `.env`). Variabel utama: `DATABASE_URL`; untuk verifikasi JWT Better Auth set `BETTER_AUTH_SECRET`. Lihat `backend/README.md` dan `.env.example`.

```powershell
Set-Location backend
gofmt -w .
go mod download
go mod verify
go vet ./...
go test ./...
go build ./...
```

Menjalankan server (pastikan `DATABASE_URL` dan variabel terkait sudah terisi di environment):

```powershell
# contoh (PowerShell)
$env:DATABASE_URL = "postgres://..."
$env:BETTER_AUTH_SECRET = "..."
go run ./cmd/server
```

Endpoint operasional: `GET /health` (liveness), `GET /ready` (readiness). Endpoint data berada di bawah `/api/v1/...`.

### Frontend

Frontend bersifat mandiri secara build, namun bergantung pada API backend untuk data bisnis persisten. Detail aturan: `agents/rules/frontend.md`.

```bash
cd frontend
bun install
bun --bun run dev
```

Perintah kualitas yang tersedia (`frontend/package.json`): `format`, `lint`, `typecheck`, `test`, `build`, `e2e`. Gunakan `bun --bun run <script>` sesuai konvensi repo.

### Pengembangan dengan RabbitMQ (opsional)

Untuk menjalankan broker saja dan backend native:

```bash
cd rabbitmq
docker compose up -d
# lalu jalankan backend/jobsched native dengan RABBITMQ_URL=amqp://simpeg:simpeg@localhost:5672/
```

Lihat `rabbitmq/README.md` untuk topologi, verifikasi manual, dan catatan keamanan.

---

## Deployment (Docker/Podman)

Stack broker RabbitMQ berdiri sendiri di `rabbitmq/docker-compose.yml`, sedangkan stack backend (`migrations`, `jobsched`, `backend`) ada di `docker-compose.yml` root. Keduanya terhubung lewat jaringan bersama `simpeg-broker`.

```bash
# Siapkan .env terlebih dahulu (DATABASE_URL, TRUSTED_ORIGINS, BETTER_AUTH_SECRET, dll.)
docker compose -f rabbitmq/docker-compose.yml up -d   # broker lebih dulu
docker compose up -d --build                          # atau: podman-compose up -d --build
curl http://localhost:8080/ready
```

**Catatan penting**

- `APP_ENV=production` menolak start bila konfigurasi rahasia/origin tidak memenuhi syarat (lihat `backend/README.md`).
- Migrasi dijalankan oleh layanan `migrations` (`/bin/migrate`) sebelum backend bergantung padanya.
- Migrasi `000025_payroll_core.sql` (skema penggajian inti) dan `000026`–`000032` (presensi, cuti, kategori, slip BATAL, RBAC scoped) sudah disetujui/terdaftar di repo tetapi **belum diterapkan pada staging/produksi**. Karena sudah terdaftar, `migrate up` akan menerapkannya otomatis, jadi jalankan hanya lewat workflow pada staging lebih dulu, lalu verifikasi `perfbench schema`/`goose_db_version`, baru produksi dengan persetujuan pemilik (`DEC-068`, `DEC-087`).
- Seed RBAC (`go run ./cmd/rbacseed --apply`) dijalankan **setelah** migrasi diterapkan di target, mengikuti urutan pada `docs/RUNBOOK.md` §RBAC dan `docs/STAGING-RELEASE-CHECKLIST.md`.
- Mode QStash tetap tersedia sebagai alternatif cloud bila `RABBITMQ_URL` tidak diisi (`DEC-042`).

### Monitoring, alerting, dan log (stack opsional, $0)

Stack observability self-hosted terpisah ada di `observability/` (Prometheus, Alertmanager, Grafana, Loki + Promtail, node-exporter, cAdvisor, blackbox-exporter, metrik RabbitMQ). Templat ini bergabung ke jaringan eksternal `simpeg-broker`, mem-probe `backend:8080/health` dan `/ready`, serta melakukan scrape metrik RabbitMQ pada `rabbitmq:15692`.

```bash
cd observability
cp .env.example .env        # ganti kata sandi Grafana; jangan commit .env
# isi penerima alert nyata di alertmanager.yml (wajib), lalu:
docker compose up -d
```

> Status: **templat siap-disesuaikan, belum terpasang di target produksi**. Wiring, penerima alert, dan uji tembak wajib dibuktikan di infrastruktur; lihat `observability/README.md`, `docs/RUNBOOK.md` §9/§9A, dan `docs/IMPLEMENTATION-PLAN.md` §7.

### Uji backup/restore ($0)

Formulir dan checklist drill ada di `docs/backup-restore-drill.md`; helper PowerShell `scripts/backup-restore-drill.ps1` menjalankan dump/restore/verifikasi ke target **disposable** (kredensial hanya dari env `SOURCE_DATABASE_URL`/`TARGET_DATABASE_URL`). Kemampuan PITR/backup mengikuti tier Supabase target (tier Free umumnya tidak setara PITR penuh) dan harus diverifikasi per project sebelum diklaim.

### Masuk ke container

Perintah di bawah memakai Docker Compose v2 (`docker compose`); pada Podman gunakan `podman compose`/`podman-compose` dan `podman exec`.

1. **Lihat container yang berjalan**

   ```bash
   docker ps
   docker compose ps
   docker compose -f rabbitmq/docker-compose.yml ps
   ```

2. **Masuk ke shell container (jika image menyediakan shell)**

   - RabbitMQ (dari folder `rabbitmq/`):

     ```bash
     cd rabbitmq
     docker compose exec rabbitmq sh
     # atau: docker exec -it <nama-container-rabbitmq> sh
     ```

     Contoh: `rabbitmqctl list_queues name messages`. UI manajemen: `http://localhost:15672` (kredensial bawaan `simpeg`/`simpeg` bila belum diubah; jangan terekspos di production).

   - Layanan backend (`backend`, `jobsched`, `migrations`): image berbasis `distroless/static` **tanpa shell**, sehingga `exec ... sh` tidak tersedia.

3. **Memeriksa/menjalankan layanan backend tanpa shell**

   ```bash
   docker compose logs -f backend
   docker compose logs -f jobsched
   docker compose logs migrations

   docker compose ps
   docker inspect <nama-container>

   # perintah sekali jalan (contoh)
   docker compose run --rm jobsched publish audit.cleanup
   ```

   Layanan `migrations` berjalan sekali lalu berhenti (`restart: "no"`). Jalankan ulang bila perlu: `docker compose run --rm migrations`.

4. **Perintah broker dari root repo**

   ```bash
   docker compose -f rabbitmq/docker-compose.yml exec rabbitmq rabbitmqctl list_queues name messages
   ```

---

## Variabel lingkungan

Simpan nilai lokal di `.env` (diabaikan Git) atau secret manager. **Jangan** memasukkan kredensial, token, atau data pribadi ke source, dokumentasi, log, maupun bundle frontend. Rujukan lengkap dan perilaku default ada di `backend/README.md` dan `.env.example`.

| Variabel                                             | Kegunaan (ringkas)                                                             |
| ---------------------------------------------------- | ------------------------------------------------------------------------------ |
| `DATABASE_URL`                                       | Koneksi PostgreSQL (wajib untuk backend)                                       |
| `BETTER_AUTH_SECRET`                                 | Verifikasi/kontrak sesi Better Auth (wajib & kuat pada production)             |
| `TRUSTED_ORIGINS`                                    | Origin CORS yang diizinkan (wajib pada production, tanpa wildcard)             |
| `APP_ENV`, `PORT`                                    | Lingkungan dan porta server                                                    |
| `PII_ENCRYPTION_KEY`                                 | Enkripsi kolom PII di level aplikasi (wajib pada production)                   |
| `RABBITMQ_URL`, `RABBITMQ_USER`, `RABBITMQ_PASSWORD` | Broker RabbitMQ untuk pekerjaan terjadwal self-hosted                          |
| `QSTASH_*`                                           | Penjadwalan/callback QStash (alternatif cloud)                                 |
| `SMTP_*`                                             | Pengiriman email dari backend                                                  |
| `UPSTASH_REDIS_REST_*`                               | Cache Redis (data referensi non-sensitif, cache-only)                          |
| `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`           | OAuth Google (bila dikonfigurasi/digunakan)                                    |
| `ALLOWED_EMAIL_DOMAINS`, `INVITE_ONLY`               | Kebijakan akses akun (sesuai keputusan yang berlaku)                           |
| `SESSION_LIFETIME`                                   | Masa berlaku sesi (bila relevan dengan runtime yang dipakai)                   |
| `SLO_*`, `HOLIDAY_API_*`, `CLOUDINARY_*`, `AWS_*`    | Fitur/observabilitas/integrasi spesifik (aktif sesuai konfigurasi & keputusan) |

> Tercantum di sini tidak berarti integrasi sudah aktif end-to-end. Lihat `docs/IMPLEMENTATION-PLAN.md`, `docs/DECISIONS.md`, dan `backend/README.md`.

---

## Kualitas & verifikasi

- **Backend:** `gofmt`, `go vet`, `staticcheck`, `go test -race`, `go build`, `go mod verify`, `govulncheck`, serta pemeriksaan CI (`.github/workflows/backend.yml`).
- **Frontend:** format, lint, typecheck, unit test, build, dan E2E sesuai workflow (`.github/workflows/frontend.yml`).
- **Database/migrasi:** perubahan skema harus selaras antara model GORM dan migrasi Goose; verifikasi keselarasan terhadap database target dilakukan secara eksplisit (bukan disimpulkan dari kode).
- **Verifikasi dinamis:** fitur yang menyentuh database diverifikasi pada **database disposable** (migrasi baru → seed sintetis → permintaan nyata lewat HTTP, lalu database dihapus). Prinsipnya: jangan pernah menyentuh staging/produksi tanpa persetujuan pemilik; menyatakan "terimplementasi" tanpa bukti runtime tidak sah. Contoh skenario dan batasan tiap modul ada di `docs/DYNAMIC-ROUTE-VERIFICATION.md` dan `docs/DECISIONS.md`.
- **E2E hibrida:** sebagian spec Playwright memakai backend uji sungguhan (auth disimulasikan) dan **dilewati** bila backend atau dataset disposable belum siap — bukan dihitung lulus. Daftar spec hibrida: `hybrid-pemberhentian`, `hybrid-pensiun`, `hybrid-rbac`.

---

## Dokumentasi

- `docs/PRD.md` — kebutuhan produk, ruang lingkup, dan catatan keputusan kebijakan
- `docs/openapi.yaml` — kontrak HTTP API (OpenAPI 3.1)
- `docs/DECISIONS.md` — log keputusan (pemilik, tanggal, status)
- `docs/IMPLEMENTATION-PLAN.md` — checklist/status pekerjaan
- `docs/RUNBOOK.md` — panduan operasional & penanganan insiden
- `docs/THREAT-MODEL.md` — model ancaman
- `docs/PERFORMANCE-BASELINE.md` — hasil & batasan pengukuran performa
- `docs/SECURITY-TEST-OWASP-2025.md` — hasil uji keamanan dinamis (ZAP) dan keterbatasannya
- `docs/DYNAMIC-ROUTE-VERIFICATION.md` — verifikasi dinamis seluruh rute, matriks peran, dan skenario IDOR
- `docs/STAGING-RELEASE-CHECKLIST.md` — checklist kesiapan staging & gerbang rilis
- `docs/UI-LOAD-TEST.md` — hasil uji beban UI lokal dan keterbatasannya
- `docs/backup-restore-drill.md` — formulir/checklist uji backup/restore (Supabase Free)
- `docs/USULAN-ALUR-SURAT-KEPUTUSAN.md` — usulan alur penerbitan SK (menunggu keputusan pemilik; modul baca sudah ada)
- `docs/USULAN-POTONGAN-KOPERASI-PINJAMAN.md` — usulan cicilan/sisa/tagihan bulanan potongan (menunggu keputusan)
- `docs/USULAN-SP1-ALPA-PRESENSI.md` — usulan alur SP1 dari presensi (menunggu keputusan)
- `backend/README.md` — detail runtime, arsitektur lapisan, integrasi, dan endpoint
- `observability/README.md` — templat stack monitoring/alerting/log self-hosted
- `rabbitmq/README.md` — operasi broker dan topologi

---

## Aturan pengembangan

- Baca `AGENTS.md` serta `agents/rules/project.md`, `agents/rules/backend.md`, `agents/rules/frontend.md`, dan `agents/rules/architecture.md` sebelum mengubah kode.
- Sumber skema produksi adalah migrasi Goose; jangan gunakan GORM `AutoMigrate` untuk production.
- Teks yang menghadap pengguna dan dokumentasi ditulis dalam Bahasa Indonesia; identifier kode dalam bahasa Inggris.
- Jangan commit kredensial, token, atau data pribadi nyata. Gunakan data sintetis untuk pengembangan dan pengujian.
- Batas ukuran source file yang ditulis tangan mengikuti aturan proyek (lihat `AGENTS.md`).
