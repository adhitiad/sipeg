# Frontend SIMPEG

Aplikasi frontend SIMPEG memakai TanStack Start (React + TanStack Router),
shadcn/ui, TanStack Query, TanStack Table, dan TanStack Store. Frontend
berjalan sebagai aplikasi terpisah dan hanya membaca data bisnis lewat
kontrak HTTP backend Go pada prefix versi `/api/v1`.

## Kebutuhan lingkungan

- Bun 1.4 atau lebih baru sebagai package manager dan peluncur skrip.
- Node.js 20 atau lebih baru. Vitest 5 menjalankan worker jsdom dengan runtime
  Node; runtime Bun 1.4.2 belum dapat menjalankan worker tersebut (galat
  `EventTarget` lintas realm). Skrip `bun run test` sudah memakai Node lewat
  shebang, sehingga tidak perlu tindakan tambahan.

## Menjalankan

```bash
bun install
cp .env.example .env
bun --bun run dev
```

Server pengembangan berjalan di `http://localhost:3000`. Pengujian E2E memakai
port khusus `3101` (`bun --bun run dev:e2e`) supaya tidak bentrok dengan
server pengembangan yang sedang berjalan.

| Skrip          | Perintah                   | Keterangan                             |
| -------------- | -------------------------- | -------------------------------------- |
| `dev`          | `vite dev --port 3000`     | Server pengembangan                    |
| `build`        | `vite build`               | Build produksi (klien dan SSR)         |
| `preview`      | `vite preview --port 3000` | Pratinjau hasil build                  |
| `typecheck`    | `tsc --noEmit`             | Gate tipe, harus nol galat             |
| `lint`         | `eslint .`                 | Gate lint, harus nol galat dan warning |
| `format`       | `prettier --write .`       | Formatter of record                    |
| `format:check` | `prettier --check .`       | Dipakai CI                             |
| `test`         | `vitest run`               | Unit dan komponen test                 |
| `e2e`          | `playwright test`          | Alur end-to-end di Chromium            |

Perintah di atas dijalankan lewat Bun, misalnya `bun --bun run build` dan
`bun run test`.

## Konfigurasi environment

| Variabel             | Wajib | Keterangan                                                                                                                                                                        |
| -------------------- | ----- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `VITE_API_BASE_URL`  | ya    | Basis URL backend Go, contoh `http://localhost:8080/api/v1`                                                                                                                       |
| `VITE_AUTH_BASE_URL` | tidak | Basis URL runtime autentikasi Better Auth. Kosong berarti runtime tertanam di origin yang sama (`/api/auth`), inilah bawaan; isi hanya bila runtime dipasang pada origin terpisah |
| `VITE_APP_URL`       | tidak | Basis URL situs web untuk tag meta berbagi (Open Graph, Twitter), mis. `https://simpeg.unisubang.ac.id`. Kosong berarti bawaan produksi                                           |

### Variabel server-only (runtime `/api/auth`, dibaca dari `frontend/.env`)

| Variabel               | Wajib | Keterangan                                                  |
| ---------------------- | ----- | ----------------------------------------------------------- |
| `DATABASE_URL`         | ya    | Koneksi Postgres untuk runtime Better Auth                  |
| `BETTER_AUTH_SECRET`   | ya    | Secret sesi, minimal 32 karakter                            |
| `TRUSTED_ORIGINS`      | tidak | Origin yang boleh memanggil runtime, dipisah koma           |
| `BETTER_AUTH_URL`      | tidak | Base URL eksplisit, contoh `http://localhost:3000`          |
| `GOOGLE_CLIENT_ID`     | tidak | Client ID Google OAuth; kosong = tombol Google nonaktif     |
| `GOOGLE_CLIENT_SECRET` | tidak | Client secret Google OAuth; kosong = tombol Google nonaktif |

Bun memuat `.env` dari direktori kerja `frontend/`, bukan akar repo. Salin
nilai server-only dari `.env` akar ke `frontend/.env` (sudah di-`.gitignore`)
bila ingin menjalankan runtime auth. Tanpa `BETTER_AUTH_SECRET` yang terisi,
runtime mengembalikan `500 auth_not_configured` dan halaman masuk menampilkan
keadaan belum dikonfigurasi.

## Batas tanggung jawab

- Frontend tidak mengakses PostgreSQL, Supabase Data API, Redis, RabbitMQ, atau
  kredensial email. Semua data bisnis berasal dari backend Go.
- Sesi dibawa lewat header `Authorization: Bearer <token>` sesuai skema
  `bearerSession` pada `docs/openapi.yaml`. Cookie tidak disertakan karena
  middleware CORS backend hanya mengirim `Access-Control-Allow-Origin` tanpa
  `Access-Control-Allow-Credentials`, sehingga request lintas origin dengan
  cookie ditolak browser.
- Token sesi hanya hidup di memori tab (`src/lib/auth-token.ts`) dan tidak
  pernah ditulis ke `localStorage` atau `sessionStorage`.
- Route guard pada `src/lib/require-auth.tsx` hanya membantu navigasi. Backend
  tetap menjadi satu-satunya penentu hak akses dan cakupan unit.
- Nama peran pada `GET /api/v1/me` hanya dipakai untuk teks antarmuka. Matriks
  permission tidak disalin ke frontend.

## Struktur

| Lokasi            | Isi                                                                               |
| ----------------- | --------------------------------------------------------------------------------- |
| `src/routes/`     | Rute berbasis berkas; `routeTree.gen.ts` dibuat generator dan tidak diedit manual |
| `src/features/`   | Komponen halaman yang dipakai lebih dari satu rute                                |
| `src/components/` | Komponen bersama dan komponen shadcn/ui di `src/components/ui`                    |
| `src/lib/`        | Klien API, skema Zod, sesi, penjaga rute, dan pemetaan peran                      |
| `e2e/`            | Alur Playwright dengan data sintetis                                              |

## Status tahap awal

Sudah tersedia: landing page publik beserta halaman statis (tentang, privasi,
syarat penggunaan, kerangka acuan), runtime autentikasi Better Auth tertanam
pada `/api/auth/*` (masuk/keluar; seluruh role terverifikasi terhadap
database uji, belum terhadap produksi), tata letak responsif dengan bilingual
Indonesia/English dan tema light/dark (light bawaan), halaman `workspace`
(sementara, sesudah masuk), halaman admin `/admin/slo` (agregat metrik SLO,
permission `slo:read`, `DEC-050`), serta gerbang mutu (format, lint, tipe,
unit test, E2E). Panel masuk menampilkan pesan bila runtime autentikasi
memang tidak dikonfigurasi, bukan mengirim permintaan yang pasti gagal.

Sudah tersedia di antarmuka: ruang kerja (`/workspace`), modul pegawai
(daftar/detail/manajemen), organisasi, jabatan, kategori, audit, presensi
(mesin absensi + presensi), penggajian (slip: daftar, detail, unduh PDF
dibatasi 3× per pegawai/periode, generate otomatis, konfirmasi pembayaran,
verifikasi publik `/verify/slip/{token}`), cuti (pengajuan/persetujuan
berjenjang dengan eviden dan sub-jenis bersalin), SLO admin, pengaturan akun,
dan halaman admin `/admin/mesin-pin`. Backend berjalan pada prefix `/api/v1`;
kontrak rute ada di `docs/openapi.yaml`.
