# SIMPEG

SIMPEG (Sistem Informasi Kepegawaian) adalah proyek sistem informasi untuk pengelolaan data dan proses kepegawaian.

## Status

Repo ini masih berupa fondasi proyek, belum aplikasi yang siap dijalankan end-to-end.

- Backend Go/GORM dan pemetaan model tersedia di `backend/`.
- `prisma/schema.prisma` menjadi referensi skema PostgreSQL; backend menggunakan GORM, bukan Prisma Client.
- Database target adalah Supabase PostgreSQL, tetapi koneksi belum dapat digunakan sampai `DATABASE_URL` disediakan.
- Frontend TanStack Start belum dibuat.
- API HTTP menyediakan health check, identitas sesi sementara, CRUD direktori, endpoint impor pegawai, serta daftar audit log dan kategori. Seluruh endpoint data berada di bawah prefix versi `/api/v1`; path tanpa versi ditolak dengan 400. Grant role statis sudah dikonfigurasi sesuai matriks di `docs/PRD.md` §11; operator dan atasan sudah punya cakupan baca per unit beserta turunannya pada data pegawai, unit organisasi, dan jejak audit. Login/OAuth belum tersedia. Baca `AGENTS.md`, `docs/PRD.md` §11, serta status implementasi di `backend/README.md` sebelum mengubah atau membuka akses.
- Jejak audit bersifat append-only dan aksesnya dibatasi per unit. Jejak untuk perubahan role dan pembayaran belum ada karena endpoint/modulnya belum dibuat; masa retensi dan prosedur arsip belum ditetapkan.
- `/health` (liveness) dan `/ready` (readiness, memeriksa dependency dengan timeout) sudah dipisahkan. Log ditulis sebagai JSON terstruktur dengan correlation ID `X-Request-ID`; kredensial, body, dan nilai query string tidak pernah dicatat.
- Migration SQL direktori tersedia dan migration runner dipilih (Goose, konfigurasi `backend/.goose.yaml`), tetapi proses penerapan migrasi belum diuji pada database nyata.
- Environment deployment pertama ditetapkan sebagai Supabase PostgreSQL production. Prasyarat sebelum deploy (migration diterapkan, runner punya workflow, backup/restore teruji) belum terpenuhi.
- Integrasi email/cache/queue belum tersedia sebagai alur aplikasi lengkap.
- Tujuan produk, ruang lingkup MVP, dan keputusan yang masih terbuka ada di `docs/PRD.md`.

## Stack

- Backend: versi Go yang tercantum di `backend/go.mod` dan GORM.
- Database: PostgreSQL terkelola melalui Supabase.
- Frontend yang direncanakan: TanStack Start, shadcn/ui, TanStack Store, TanStack Table, dan TanStack Query.
- Layanan pendukung yang direncanakan/dikonfigurasi: Upstash Redis, QStash, SMTP email, Google OAuth, serta penyimpanan objek kompatibel S3.

Status konfigurasi layanan eksternal tidak berarti integrasinya sudah dipakai oleh aplikasi. Baca PRD dan aturan agent sebelum mengimplementasikannya.

## Backend

Persyaratan: versi Go yang tercantum di `backend/go.mod` (saat ini Go 1.26.0).

Jalankan pengujian:

```powershell
Set-Location backend
gofmt -w .
go test ./...
go vet ./...
go build ./...
go mod verify
```

GitHub Actions menjalankan pemeriksaan format, verifikasi dependency, vet, test, dan build backend pada push dan pull request. Frontend belum tersedia, sehingga workflow frontend belum dapat ditambahkan.

Paket database menyediakan `OpenPostgres(dsn)` untuk membuka koneksi GORM dengan pemetaan nama tabel dan kolom sesuai schema referensi. Isi `DATABASE_URL` dengan connection string Supabase PostgreSQL di lingkungan backend sebelum menghubungkan aplikasi. Jangan menaruh nilainya di README atau commit ke Git.

Belum ada server HTTP atau perintah `go run` untuk menjalankan aplikasi. Jangan gunakan GORM `AutoMigrate` untuk membuat skema produksi; tool migrasi masih perlu diputuskan.

## Variabel lingkungan

Simpan nilai lokal di `.env` atau gunakan secret manager pada deployment. File `.env` dan variasinya diabaikan oleh Git; jangan memasukkan kredensial ke source, dokumentasi, log, atau bundle frontend.

| Variabel | Kegunaan |
| --- | --- |
| `DATABASE_URL` | Koneksi backend ke Supabase PostgreSQL. |
| `UPSTASH_REDIS_REST_URL`, `UPSTASH_REDIS_REST_TOKEN` | Redis Upstash untuk cache/state sementara setelah integrasinya dibuat. |
| `QSTASH_URL`, `QSTASH_TOKEN`, `QSTASH_CURRENT_SIGNING_KEY`, `QSTASH_NEXT_SIGNING_KEY` | Mengirim pekerjaan ke QStash dan memverifikasi callback di backend. |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASS`, `SMTP_SECURE`, `SMTP_FROM` | Pengiriman email dari backend. Gunakan TLS/STARTTLS sesuai port dan kebijakan provider. |
| `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET` | OAuth Google bila alur autentikasi sudah diimplementasikan. |
| `AWS_ENDPOINT_URL_S3`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_REGION` | Penyimpanan objek kompatibel S3 bila integrasinya sudah diimplementasikan. |

Nama variabel di atas mendokumentasikan konfigurasi yang dibutuhkan atau tersedia; bukan jaminan semua integrasi sudah aktif. Notifikasi gaji hanya boleh dikirim setelah backend menyimpan konfirmasi pembayaran, bukan saat slip baru diterbitkan.

## Aturan agent

Baca `AGENTS.md` dan file terkait di `agents/rules/` sebelum mengubah proyek. Schema database harus diselaraskan dengan model GORM. Batas ukuran source file adalah 387 baris; pengecualian hanya sampai 397 baris dengan alasan yang jelas.
