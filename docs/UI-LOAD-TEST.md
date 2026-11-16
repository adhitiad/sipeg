# Uji Beban UI — SIMPEG

- **Tanggal:** 2026-11-14
- **Lingkungan (bukan produksi):** PostgreSQL 18.4 lokal, database disposable `simpeg_uiload` (dihapus setelah selesai), migrasi Goose `000001`–`000024`, `perfbench seed --employees=5000 --months=6 --audit=10000` (5000 pegawai, 49 unit). Backend Go di `:8099` (`APP_ENV=development`). Frontend dev server di `:3103` (proxy same-origin `/api/v1` → backend; waktu pengukuran memakai perangkat lunak dev, bukan build produksi). Login nyata lewat runtime Better Auth tertanam dengan akun admin uji sintetis. Chromium lewat Playwright (`@playwright/test`).
- **Sifat pengukuran:** satu kali jalan (single pass), bukan uji beban konkuren; mengukur pengalaman satu pengguna pada dataset skala produksi. Uji beban bersamaan/soak backend ada di `docs/PERFORMANCE-BASELINE.md`.

## Hasil (timings)

| Tahap                                                       | Waktu                                |
| ----------------------------------------------------------- | ------------------------------------ |
| Muat halaman masuk (`/login`, `load`)                       | 131 ms                               |
| Masuk (isi kredensial → `/workspace`)                       | 589 ms                               |
| Klik "Pegawai" → tabel 20 baris tampil                      | 293 ms                               |
| Ganti ukuran halaman 20 → 100 (100 baris tampil)            | 217 ms                               |
| Pencarian tanpa hasil (`zzz-tidak-ada`) → empty state       | 581 ms (debounce 300 ms + roundtrip) |
| Klik "Atur ulang pencarian" → tabel tampil kembali          | 824 ms                               |
| Pencarian sempit (`Pegawai Sintetis 000123`) → baris tampil | 64 ms                                |
| Kesalahan halaman (`pageerror`)                             | 0                                    |

Observasi: API `GET /api/v1/employees` menjawab `200` pada tiap tarikan (lihat log backend); pagination/sort/pencarian dipersempit server, sehingga browser tidak pernah menerima himpunan penuh 5000 baris.

## Yang diperbaiki selama uji

1. Keadaan kosong "Manajemen Pegawai" tidak menampilkan kontrol apa pun, padahal pesannya meminta pengguna mengubah pencarian — pengguna terjebak. Ditambahkan tombol **"Atur ulang pencarian"** (membersihkan pencarian dan filter unit) pada empty state (`EmployeeManagementPage.tsx`).
2. Spesifikasi beban tersimpan sebagai `frontend/e2e/perf.spec.ts` (env-gated `PERF_STACK=1`, tidak berjalan di CI):

```powershell
# Infrastruktur uji (semuanya lokal, DB disposable):
psql -U postgres -d postgres -c "CREATE DATABASE simpeg_uiload"
cd E:\code\sipeg\backend
$env:DATABASE_URL="postgres://postgres@localhost:5432/simpeg_uiload?sslmode=disable"
go run ./cmd/migrate up
$env:PII_ENCRYPTION_KEY="<base64 32 byte>"
go run ./cmd/perfbench seed --employees=5000 --months=6 --audit=10000
# Jalankan backend (:8099) dan frontend dev (:3103, proxy /api/v1 → 8099), buat satu akun admin
# Better Auth pada DB tersebut (penggunaan kembali pola scripts/provision-accounts.ts), lalu:
cd E:\code\sipeg\frontend
$env:PERF_STACK="1"; $env:PERF_PASSWORD="<sandi akun uji>"
bun --bun run e2e -- perf.spec.ts
```

## Batas dan caveat

- Pengukuran memakai dev server (bukan `vite build` + `preview`/hosting produksi), jadi angka adalah batas bawah yang wajar; belum memperhitungkan CDN, HTTP/2, kompresi, atau cold start.
- Satu viewport desktop (default Playwright Chromium). Uji beban rendering pada layar ponsel/jaringan lambat belum dilakukan di sini.
- Uji memuat dataset pegawai; halaman berat lain (penggajian/audit/SLO dengan 10.000 baris) mengikuti pola yang sama tetapi belum diukur satu per satu.
- Lingkungan staging/produksi representatif belum tersedia; lihat `docs/STAGING-RELEASE-CHECKLIST.md`.
