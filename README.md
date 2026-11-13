# SIMPEG

SIMPEG (Sistem Informasi Kepegawaian) adalah proyek sistem informasi untuk pengelolaan data dan proses kepegawaian.

## Status

Repo ini masih berupa fondasi proyek, belum aplikasi yang siap dijalankan end-to-end.

- Backend Go/GORM dan pemetaan model tersedia di `backend/`.
- Skema database ditentukan oleh migrasi SQL Goose di `backend/migrations/`; backend menggunakan GORM.
- Database target adalah Supabase PostgreSQL, tetapi koneksi belum dapat digunakan sampai `DATABASE_URL` disediakan.
- Frontend TanStack Start tersedia di `frontend/` (Bun): halaman depan publik, About/Privacy/Terms/Terms-of-Reference, masuk/keluar lewat Better Auth tertanam di server frontend pada `/api/auth/*` (pendaftaran publik dimatikan; akun dibuat administrator), bilingual Indonesia/English, tema light/dark (bawaan light), sonner untuk umpan balik. Verifikasi end-to-end masuk/keluar terhadap database uji ada di `DEC-049`; desain UI masih draf menunggu persetujuan produk (`DESIGN.md`).
- API HTTP menyediakan health check, identitas sesi, CRUD direktori, endpoint impor pegawai, serta daftar audit log dan kategori. Seluruh endpoint data berada di bawah prefix versi `/api/v1`; path tanpa versi ditolak dengan 400. Grant role statis sudah dikonfigurasi sesuai matriks di `docs/PRD.md` §11; operator dan atasan punya cakupan baca per unit beserta turunannya pada data pegawai, unit organisasi, dan jejak audit. Login tersedia lewat runtime Better Auth tertanam di frontend (kata sandi); Google OAuth dan MFA belum dikonfigurasi. Baca `AGENTS.md`, `docs/PRD.md` §11, serta status implementasi di `backend/README.md` sebelum mengubah atau membuka akses.
- Jejak audit bersifat append-only dan aksesnya dibatasi per unit. Perubahan role (`employee.role.updated`) sudah berjejak audit; masa retensi bawaan 90 hari (`AUDIT_RETENTION_DAYS`).
- `/health` (liveness) dan `/ready` (readiness, memeriksa dependency dengan timeout) sudah dipisahkan. Log ditulis sebagai JSON terstruktur dengan correlation ID `X-Request-ID`; kredensial, body, dan nilai query string tidak pernah dicatat.
- Metrik SLO disimpan di PostgreSQL; halaman admin `/admin/slo` menampilkan agregat per rute (permission `slo:read`, admin saja, `DEC-050`). Login gagal pada lapisan runtime Better Auth tercatat sebagai `{ event: 'auth_failed', path, reason }` tanpa kredensial. RPO/RTO awal diratifikasi 24 jam / 4 jam; drill backup/restore dijalankan terhadap database disposable lokal (PITR dan restore produksi belum diuji) — `DEC-050`, `docs/RUNBOOK.md` §8E.
- Migration SQL sampai versi 000023 (`000023_keluarga_nik_encrypt.sql`) sudah diterapkan pada database Supabase `postgres` dan database uji `simpeg_perf` (Goose 23). Workflow deployment tersedia di `.github/workflows/deploy-migrations.yml`; eksekusi di GitHub menunggu environment/secret `DATABASE_URL` (lihat `docs/IMPLEMENTATION-PLAN.md` bagian 2 dan `DEC-046`). Kolom identitas PII (`user.nik`, `user.nip`, `user.npwp`, `RiwayatPendidikan.noIjazah`, `KeluargaPegawai.nik`) disimpan terenkripsi (AES-256-GCM) dengan `PII_ENCRYPTION_KEY`; lihat `DEC-053` dan `DEC-054`.
- Uji dinamis OWASP Top 10:2025 dengan ZAP 2.17.0 pada infrastruktur disposable lokal selesai; tidak ada temuan kritis atau tinggi, tetapi ada cacat query yang membuat sebagian endpoint gagal `5xx`, sembilan rute terdokumentasi yang belum terdaftar, dan header keamanan frontend yang belum ditetapkan — `docs/SECURITY-TEST-OWASP-2025.md`.
- Integrasi email/cache/queue sudah diimplementasikan di backend (worker outbox SMTP bila SMTP dikonfigurasi, cache Redis untuk kategori, RabbitMQ/QStash untuk pekerjaan terjadwal), tetapi belum dipakai oleh alur lengkap end-to-end dan pengiriman nyata ke kotak masuk belum terukur (SMTP Google Workspace menolak kredensial; butuh *app password*) — lihat `docs/IMPLEMENTATION-PLAN.md` bagian 5.
- Tujuan produk, ruang lingkup MVP, dan keputusan yang masih terbuka ada di `docs/PRD.md`.

## Stack

- Backend: versi Go yang tercantum di `backend/go.mod` dan GORM.
- Database: PostgreSQL terkelola melalui Supabase.
- Frontend: TanStack Start, shadcn/ui, TanStack Store, TanStack Table, dan TanStack Query (folder `frontend/`).
- Layanan pendukung yang direncanakan/dikonfigurasi: Upstash Redis, QStash, SMTP email, Google OAuth, serta penyimpanan objek kompatibel S3.

Status konfigurasi layanan eksternal tidak berarti integrasinya sudah dipakai oleh aplikasi. Baca PRD dan aturan agent sebelum mengimplementasikannya.

## Backend

Persyaratan: versi Go yang tercantum di `backend/go.mod` (saat ini Go 1.27.4).

Jalankan pengujian:

```powershell
Set-Location backend
gofmt -w .
go test ./...
go vet ./...
go build ./...
go mod verify
```

GitHub Actions menjalankan pemeriksaan format, verifikasi dependency, vet, test, dan build backend pada push dan pull request. Untuk perubahan di `frontend/`, `.github/workflows/frontend.yml` menjalankan format, lint, typecheck, unit test, build produksi, dan E2E.

Paket database menyediakan `OpenPostgres(dsn)` untuk membuka koneksi GORM dengan pemetaan nama tabel dan kolom sesuai schema referensi. Isi `DATABASE_URL` dengan connection string Supabase PostgreSQL di lingkungan backend sebelum menghubungkan aplikasi. Jangan menaruh nilainya di README atau commit ke Git.

Server HTTP tersedia di `backend/cmd/server` dan dijalankan dengan `DATABASE_URL` di lingkungan (lihat bagian deployment). Jangan gunakan GORM `AutoMigrate` untuk membuat skema produksi; migrasi diterapkan lewat Goose (`backend/migrations/`, `cmd/migrate`).

## Deployment (docker/podman)

Broker RabbitMQ berdiri sendiri pada `rabbitmq/docker-compose.yml` dan stack backend pada `docker-compose.yml` (`migrations`, `jobsched`, `backend`; semua binary dari satu image `backend/Dockerfile`). Broker dan backend tetap berjalan pada satu server melalui jaringan bersama `simpeg-broker`.

```bash
# Isi .env terlebih dahulu (DATABASE_URL, TRUSTED_ORIGINS, BETTER_AUTH_SECRET).
docker compose -f rabbitmq/docker-compose.yml up -d   # broker lebih dulu
docker compose up -d --build                          # atau: podman-compose up -d --build
curl http://localhost:8080/ready
```

Saat pengembangan, jalankan hanya broker (`cd rabbitmq && docker compose up -d`) dan backend native dengan `RABBITMQ_URL=amqp://simpeg:simpeg@localhost:5672/`. Detail broker, topologi antrean, dan verifikasi manual ada di `rabbitmq/README.md`. QStash tetap tersedia sebagai alternatif cloud bila `RABBITMQ_URL` tidak diisi (`DEC-042`).

## Variabel lingkungan

Simpan nilai lokal di `.env` atau gunakan secret manager pada deployment. File `.env` dan variasinya diabaikan oleh Git; jangan memasukkan kredensial ke source, dokumentasi, log, atau bundle frontend.

| Variabel | Kegunaan |
| --- | --- |
| `DATABASE_URL` | Koneksi backend ke Supabase PostgreSQL. |
| `UPSTASH_REDIS_REST_URL`, `UPSTASH_REDIS_REST_TOKEN` | Redis Upstash untuk cache/state sementara setelah integrasinya dibuat. |
| `QSTASH_URL`, `QSTASH_TOKEN`, `QSTASH_CURRENT_SIGNING_KEY`, `QSTASH_NEXT_SIGNING_KEY` | Mengirim pekerjaan ke QStash dan memverifikasi callback di backend. |
| `RABBITMQ_URL`, `RABBITMQ_USER`, `RABBITMQ_PASSWORD` | Broker RabbitMQ pengganti QStash pada deployment docker/podman (`DEC-042`). |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASS`, `SMTP_SECURE`, `SMTP_FROM` | Pengiriman email dari backend. Gunakan TLS/STARTTLS sesuai port dan kebijakan provider. |
| `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET` | OAuth Google bila alur autentikasi sudah diimplementasikan. |
| `ALLOWED_EMAIL_DOMAINS` | Domain email yang diizinkan untuk akses SIMPEG (pisahkan dengan koma). |
| `INVITE_ONLY` | Mode undangan: true = akses hanya diizinkan bagi pengguna yang diundang selain domain allowlist. |
| `SESSION_LIFETIME` | Masa berlaku sesi yang diterbitkan backend, default 23 jam. |
| `AWS_ENDPOINT_URL_S3`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_REGION` | Penyimpanan objek kompatibel S3 bila integrasinya sudah diimplementasikan. |

Nama variabel di atas mendokumentasikan konfigurasi yang dibutuhkan atau tersedia; bukan jaminan semua integrasi sudah aktif. Notifikasi gaji hanya boleh dikirim setelah backend menyimpan konfirmasi pembayaran, bukan saat slip baru diterbitkan.

## Aturan agent

Baca `AGENTS.md` dan file terkait di `agents/rules/` sebelum mengubah proyek. Schema database harus diselaraskan dengan model GORM. Batas ukuran source file adalah 387 baris; pengecualian hanya sampai 397 baris dengan alasan yang jelas.
