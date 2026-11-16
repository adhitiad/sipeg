# Verifikasi Dinamis Seluruh Rute dan IDOR (2026-11-14)

- **Tanggal:** 2026-11-14
- **Ruang lingkup:** seluruh rute terdaftar pada backend, matriks peran (`admin`, `operator`, `atasan`, `pegawai`), skenario IDOR/cakupan lintas unit, versioning, logout/pencabutan sesi, dan penolakan akun nonaktif.
- **Lingkungan (bukan produksi):** PostgreSQL 18.4 lokal (`localhost:5432`), database disposable `simpeg_dynverify` (dibuat dan **dihapus setelah selesai**), migrasi Goose `000001`–`000024`, data sintetis (6 akun `@example.test`, 4 unit pohon `Homebase`, 2 slip gaji `TERBIT`), backend dev di port `8099` (`APP_ENV=development`).
- **Yang tidak diverifikasi di sini:** autentikasi perangkat presensi dengan kredensial nyata, pengiriman SMTP nyata, alur Better Auth sungguhan (runtime frontend), callback legacy `internal/qstash/*` secara dinamis (tidak dipanggil pada deployment mana pun sejak `DEC-102`; rute tetap ada dengan autentikasi signature — diverifikasi unit test), dan database target produksi/Supabase.

> **Catatan pembaruan (2026-11-16):** verifikasi ini mencakup migrasi `000001`–`000024` dan rute yang ada per 14 November 2026. Rute & modul yang menyusul — RBAC admin (`/api/v1/admin/rbac/*`), master pensiun (`/api/v1/pensiun/*`), THR, presensi, cuti lanjutan, kontrak/pesangon, dan HR — **belum termasuk sapuan ini**; bukti terpisah ada di `docs/DECISIONS.md` (`DEC-083`–`DEC-090`) dan pemindaian ulang menyeluruh (ZAP + IDOR) pada cakupan final tetap menjadi item terbuka di `docs/IMPLEMENTATION-PLAN.md` §6–§7.

## 1. Perbaikan yang dilakukan selama verifikasi

1. **`backend/internal/outbox` tidak dapat dikompilasi:** `GormStore.Stats()` (pekerjaan berjalan di working tree) mengembalikan tipe `Stats` yang belum didefinisikan. Tipe `Stats` minimal ditambahkan di `outbox.go` (`pending/processing/done/failed`, `OldestPendingAt`, `FailedByKind`) sehingga `go build ./...` hijau. Bila pekerjaan observabilitas outbox dilanjutkan, sambungkan tipe ini ke pembaca yang dipakai (`httpserver.OutboxStatsReader`).
2. **Bug SQL pada `GET /api/v1/attendance/unit-records`** (`internal/attendance/gorm_read.go`): `LEFT JOIN "homebase"` → `SQLSTATE 42P01` (relasi tidak ada; tabel sebenarnya `"Homebase"`). Diperbaiki menjadi `LEFT JOIN "Homebase"`, query diekstrak ke helper `unitDayScope`, dan ditambahkan tes regresi dry-run `TestListUnitDayJoinMemakaiTabelHomebaseBertandaPetik`. Sesudah perbaikan: admin `200` (6 baris), operator/atasan `200` (4 baris sesuai cakupan); `go test ./internal/attendance/...` hijau.

## 2. Hasil sapuan tanpa autentikasi (270+ permintaan)

| Kelompok                                                                   | Jumlah | Status                                                                                                            |
| -------------------------------------------------------------------------- | ------ | ----------------------------------------------------------------------------------------------------------------- |
| `GET /health`, `GET /ready` (publik)                                       | 2      | `200`                                                                                                             |
| Rute terproteksi tanpa token                                               | 142    | `401`                                                                                                             |
| `POST /api/v1/internal/qstash/{payroll/notify,audit/cleanup,slo/evaluate}` | 3      | `404` (tidak terdaftar: instance tanpa QStash) — callback legacy, tidak dipanggil sejak `DEC-102`; signature diverifikasi unit test                                            |

Tidak ada rute terproteksi yang dapat diakses tanpa sesi.

## 3. Matriks peran (dengan sesi valid)

Legenda: `200` sesuai kewenangan; `403` ditolak deny-by-default/cakupan; `503` layanan tidak dikonfigurasi; `422` validasi input.

| Endpoint                                                                                                                                           | admin                             | operator                   | atasan                  | pegawai                                                                  |
| -------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------- | -------------------------- | ----------------------- | ------------------------------------------------------------------------ |
| `GET /api/v1/me`                                                                                                                                   | 200                               | 200                        | 200                     | 200                                                                      |
| `PATCH /api/v1/me` (`nama`/`noHp`)                                                                                                                 | 200                               | —                          | —                       | —                                                                        |
| `GET /api/v1/employees` (keseluruhan/cakupan/self)                                                                                                 | 200 (6)                           | 200 (4, cakupan)           | 200 (4, cakupan baca)   | 200 (1, diri)                                                            |
| `GET /api/v1/employees/summary`                                                                                                                    | 200                               | 200 (4)                    | 200 (4)                 | 200 (1)                                                                  |
| `GET /api/v1/organizations`                                                                                                                        | 200 (4)                           | 200 (2, cakupan)           | 200 (2, cakupan)        | 403                                                                      |
| `GET /api/v1/audit-logs`, `.../export`                                                                                                             | 200                               | 200 (cakupan)              | 403                     | 403                                                                      |
| `GET /api/v1/categories`                                                                                                                           | 200                               | 200                        | 200                     | **403** (pegawai tidak diberi `category:read` — drift dokumen, lihat §5) |
| `GET /api/v1/jabatan`                                                                                                                              | 200                               | 200                        | 200                     | 200                                                                      |
| `GET /api/v1/payroll/slips`                                                                                                                        | 200 (2)                           | 200 (1, cakupan)           | 403                     | 200 (1, diri)                                                            |
| `GET /api/v1/payroll/slips/{A,B}`                                                                                                                  | 200/200                           | 200/**403** (lintas unit)  | 403/403                 | 200 (A milik sendiri)/**403** (B)                                        |
| `POST .../slips/{B}/payment` (konfirmasi)                                                                                                          | 200 lalu **409** saat diulang     | 403                        | 403                     | 403                                                                      |
| `GET /api/v1/payroll/schedules`                                                                                                                    | 200                               | 200 (cakupan)              | 403                     | 403                                                                      |
| `GET /api/v1/holidays`                                                                                                                             | 200                               | 200                        | 200                     | 403                                                                      |
| `GET /api/v1/attendance/my-records?from&to`                                                                                                        | 403 (tak ada grant `:self`)       | 403                        | 403                     | 200 (diri)                                                               |
| `GET /api/v1/attendance/unit-summary?from&to`                                                                                                      | 200                               | 200                        | 200 (cakupan)           | 403                                                                      |
| `GET /api/v1/attendance/unit-records?date`                                                                                                         | 200 (6)                           | 200 (4)                    | 200 (4)                 | 403                                                                      |
| `GET /api/v1/admin/ops/status` (`ops:read`)                                                                                                        | 200                               | 403                        | 403                     | 403                                                                      |
| `GET /api/v1/admin/slo/metrics` (`slo:read`)                                                                                                       | 200                               | 403                        | 403                     | 403                                                                      |
| `GET /api/v1/cloudinary/upload-signature`                                                                                                          | 503 (layanan tidak dikonfigurasi) | 503                        | 503                     | 503                                                                      |
| Endpoint (GET list)                                                                                                                                | admin                             | operator                   | atasan                  | pegawai                                                                  |
| ---                                                                                                                                                | ---                               | ---                        | ---                     | ---                                                                      |
| HR (`cuti`, `keluarga`, `golongan`, `arsip`, `sertifikasi`, `pelatihan`, `sister`, `penghargaan`, `tunjangan`, `potongan`, `skp/*`, `tridharma/*`) | 200                               | 200 (cakupan sesuai modul) | 200 (baca sesuai modul) | 200 (read-self sesuai modul)                                             |
| `POST /api/v1/auth/logout` → `GET /me`                                                                                                             | 204 → 401 setelahnya              | —                          | —                       | —                                                                        |

## 4. IDOR dan penolakan lintas cakupan (semuanya lulus)

- Pegawai A membaca profil karyawan B (unit lain) → `403`; membaca milik sendiri → `200`.
- Operator (cakupan Fakultas Teknik) mengedit pegawai B (Biro Kepegawaian) → `403` (gate cakupan di layanan, bukan hanya handler).
- Operator mengedit pegawai C (unitnya) → `200`; dikembalikan ke nilai semula.
- Atasan membaca karyawan lintas unit → `403`; admin → `200`.
- Slip gaji B diakses pegawai A / operator (luar cakupan) → `403`.
- Pegawai tidak dapat mengubah role siapa pun (termasuk role sendiri) → `403`; admin mengubah role pegawai C → `200`.
- Pegawai tidak dapat mengonfirmasi pembayaran (termasuk slip miliknya) → `403`; admin → `200`, pengulangan → `409` (idempoten).
- Akun nonaktif (`pegawaiB`, `aktif=false`) → sesi ditolak `401`; setelah diaktifkan kembali → `200`.
- `GET /api/me` (tanpa versi) → `400 unversioned_path`; `GET /api/v2/me` → `404`; `GET /api/v1/me` → header `X-API-Version: v1`.

## 5. Temuan dan drift dokumen/kontrak

1. **Dokumen menyatakan `pegawai` memiliki `category:read`, tetapi `policy.go` tidak memberikannya** → pegawai `403` pada `GET /api/v1/categories`. Perbaiki dokumen (PRD) atau tetapkan grant dengan persetujuan pemilik; jangan ubah policy tanpa keputusan.
2. **`docs/openapi.yaml` belum mendokumentasikan** `PATCH /api/v1/me`, `GET /api/v1/admin/ops/status`, `GET /api/v1/admin/slo/metrics`, `POST /api/v1/auth/session` (bootstrap) — drift yang konsisten dengan DEC-056 (rute terdaftar belum seluruhnya terdokumentasi).
   **Status pembaruan (2026-11-14):** semua item tersebut sudah terdokumentasi setelah penutupan drift rute pada `DEC-066`; item ini ditindaklanjuti sebagai catatan historis dan ketidaksempurnaan tersebut sudah ditutup. `POST /api/v1/auth/session` pada waktu itu juga masih menggambarkan perilaku cookie yang tidak ada; kontrak dikoreksi pada `DEC-069` dan perilaku aktual sesuai handler backend.
3. Rute internal (legacy) `internal/qstash/*` diverifikasi hanya sampai "tidak terdaftar tanpa kredensial" (`404`); autentikasi signature (HS256 raw body) ditutup oleh unit test, bukan uji dinamis di sini. Sejak `DEC-102` rute ini tidak dipanggil pada deployment mana pun; transport pekerjaan terjadwal adalah RabbitMQ.
4. `my-records`/`unit-summary` mewajibkan `from`/`to`, `unit-records` mewajibkan `date`; tanpa parameter → `422`. Sesuai kontrak dan kini **dideklarasikan `required: true`** di OpenAPI; ketidakdokumentasian tersebut ditutup pada penutupan drift rute 2026-11-14 (`DEC-066`).
5. `Jabatan`/`Category`: pegawai tidak diberi `category:read` (lihat #1); `jabatan:read` diberikan ke semua role (terverifikasi `200`).

## 6. Cara mereproduksi

```powershell
# 1. Buat database disposable dan terapkan migrasi
psql -U postgres -d postgres -c "CREATE DATABASE simpeg_dynverify"
cd E:\code\sipeg\backend
$env:DATABASE_URL="postgres://postgres@localhost:5432/simpeg_dynverify?sslmode=disable"
go run ./cmd/migrate up

# 2. Seed sintetis (roles, Homebase, user, session, slip gaji)
psql -U postgres -d simpeg_dynverify -f <path-seed>.sql

# 3. Jalankan server (APP_ENV=development, PORT=8099, PII_ENCRYPTION_KEY=base64 32 byte)
# 4. Jalankan skrip uji dinamis (urllib Python), baca hasil dari stdout/laporan
# 5. Hapus database setelah selesai
psql -U postgres -d postgres -c "DROP DATABASE simpeg_dynverify"
```

Seed, skrip uji, dan hasil mentah tersimpan di direktori temp sesi (`simpeg_dynverify_seed.sql`, `simpeg_dyntest.py`, `simpeg_dyntest2.py`, `simpeg_dyntest_results.json`). Database disposable sudah dihapus; jangan gunakan kredensial produksi untuk uji ini.

## 8. Verifikasi dinamis modul master skala gaji (2026-11-14, `DEC-073`)

- **Lingkungan:** DB disposable lokal `simpeg_tahap2` (migrasi 000001–000025; 5 kategori status dari seed migrasi; admin/pegawai sintetis; `simpeg_tahap2` tetap ada untuk pengembangan lanjutan; bukan staging/produksi).
- **18 skenario lulus:** 401 tanpa sesi; 201 create; 409 duplikat `kode` dan 409 duplikat `(skala, kategori, tingkat)`; 200 list/get/patch; 422 tanggal non-ISO & nominal negatif; 403 pegawai (list/get/post); 400 UUID tidak valid; 204 delete detail; detail `kategoriKode` = `DOSEN_TETAP_YAYASAN`.
- **Konfirmasi runtime:** endpoint baru terdaftar bersama modul lain; audit transaksional (`skala.*`) tertulis di `auditLog` (verifikasi manual via psql).

## 9. Verifikasi dinamis modul honor SKS, BPJS, PPh 21 TER (2026-11-14, `DEC-074`)

- **Lingkungan:** DB disposable lokal `simpeg_tahap2` (migrasi 000001–000025; seed `@example.test` admin/pegawai; 1 `periodeAkademik` sintetis; `simpeg_tahap2` dihapus setelah selesai).
- **21 skenario lulus:** 401 tanpa sesi; 403 pegawai pada tulis (dan pada baca di luar scope pada honor-periode/bpjs-peserta — pegawai hanya melihat miliknya); 201 create; 409 duplikat `(userId, periodeAkademikId)` honor-periode dan `(jenis, berlakuMulai)` bpjsKonfigurasi; 422 nominal negatif; `totalHonor` dihitung server menjadi `1200000` → setelah `PATCH sks=9` menjadi `900000`; `GET /api/v1/payroll/honor-rate/current?asOf=2026-02-01` mengembalikan `perSks=100000`.

## 10. Verifikasi dinamis mesin hitung slip dan PDF (2026-11-14, `DEC-075`)

- **Lingkungan:** DB disposable lokal `simpeg_tahap2` (migrasi Goose `000001`–`000025`, versi 25; seed sintetis `@example.test` admin/pegawai, 5 kategori status dari migrasi, 1 `periodeAkademik` AKTIF 2026-1, skala `SK-2026` efektif 2026-01-01 (kategori `DOSEN_TETAP_YAYASAN` tingkat 3 = Rp 9.500.000), `honorSksRate` Rp 100.000/SKS, tunjangan FUNGSIONAL Rp 1.500.000, potongan KOPERASI Rp 100.000, BPJS KESEHATAN 1%/4% dan KETENAGAKERJAAN 2%/3,7% (dasar `GAJI_POKOK`), tabel TER sintetis (0% hingga Rp 10.000.000, 5% di atasnya; **bukan tabel resmi PMK 168/2023**); backend dev di port 8099 (`APP_ENV=development`, `PAYROLL_SLIP_ORG_NAME=Universitas Subang Sintetis`); database **dihapus setelah selesai**.
- **Hasil `POST /api/v1/payroll/slips/generate`:** 401 tanpa sesi; 403 pegawai (deny-by-default, `payroll:manage` hanya admin); 400 `userId` bukan UUID; 422 `periode` di luar `YYYY-MM` (`2026-13`) dan 422 `kategoriStatusId` tanpa `tingkat`; 200 happy path (DRAF; gaji pokok Rp 9.500.000 dari skala; bruto Rp 11.000.000; potongan KOPERASI 100.000, BPJS_KESEHATAN 95.000, BPJS_KETENAGAKERJAAN 190.000, PPH21 550.000 (= 5% × bruto, kategori TER B untuk `K/2`)); regenerate slip DRAF → 200 dengan **id slip dan nomor slip tetap** (idempotent); slip berstatus TERBIT → **409** `conflict`; audit `payroll.slip.generated` tertulis dengan `unitId` = unit pegawai (2 entri baru; entri sebelum perbaikan `unitId=NULL`).
- **Hasil `GET /api/v1/payroll/slips/{id}/pdf`:** 200 `application/pdf` (35.140 byte, magic `%PDF-`, `Content-Disposition: inline`, `Cache-Control: private, no-store`); pegawai membaca PDF slip miliknya → 200; admin → 200; `payrollId` bukan UUID → 400; UUID tidak ada → 404; `Cache-Control` melarang penyimpanan bersama. PDF tidak dapat dipratinjau visual pada sesi ini (viewer tidak tersedia); struktur diverifikasi sampai level byte/header.
- **Isi slip diverifikasi lewat `psql`:** `gajiTunjangan` = FUNGSIONAL 1.500.000; `gajiPotongan` = KOPERASI 100.000, BPJS_KESEHATAN 95.000, BPJS_KETENAGAKERJAAN 190.000, PPH21 550.000; `gaji.PotonganPph21` = 550.000. Gaji bersih = 11.000.000 − 935.000 = **Rp 10.065.000**.
- **Pembelajaran selama verifikasi:** (1) `UpsertSlip` tidak memuat asosiasi; respons awal menampilkan `pendapatan`/`potongan`/`nama` kosong — diperbaiki dengan muat ulang slip lewat jalur otorisasi `payroll.Get` di handler (cakupan tetap ditegakkan); (2) `payroll.slip.generated` awalnya tidak mengisi `unitId`; diperbaiki (`GetEmployeeHomebaseID`) agar selaras invariant jejak audit per unit; (3) entri tunjangan/potongan audit yang ada sebelumnya menulis `userId` ke kolom `unitId` — **tidak diubah di sini** (di luar lingkup perbaikan; dicatat sebagai temuan).
- **Belum diverifikasi:** halaman verifikasi publik slip (token `KodeVerifikasi` tersimpan; rute belum ada); tabel TER resmi PMK 168/2023 (seed uji bersifat sintetis; wajib diisi admin sebelum slip nyata); branding/QR pada PDF (`DEC-071`/`DEC-072`); reliabilitas pemetaan PTKP → kategori TER terhadap teks resmi PMK.

## 11. Verifikasi dinamis potongan alpa pada mesin slip (2026-11-14, `DEC-076`)

- **Lingkungan:** DB disposable lokal `simpeg_tahap2` (migrasi Goose `000001`–`000025`, versi 25; seed sintetis: admin/pegawai/gawai `@example.test`, 5 kategori status dari migrasi, `skalaGaji SK-2026` dengan detail `TENDIK_TETAP` tingkat 1 = Rp 5.000.000, `periodeAkademik` AKTIF 2026-1, BPJS & TER sintetis; tambahan presensi: 3 alpa + 1 izin + 1 sakit di Juni 2026, 31 alpa di Juli 2026, seluruh `Presensi` bersumber `seed`); backend dev di port 8099; database **dihapus setelah selesai**.
- **Hasil 14 pemeriksaan lulus:** `POST /api/v1/payroll/slips/generate` Juni 2026 → 200, potongan memuat `ALPA` **Rp 75.000** (3 × 25.000; izin/sakit tidak dipotong), gaji pokok Rp 5.000.000 dari skala `TENDIK_TETAP`, bruto Rp 6.500.000; Juli 2026 (31 alpa) → ALPA **Rp 750.000** (batas maksimal); `pegawai` → 403; DOSEN_LB (honor-only, tanpa `kategoriStatusId`) → 200 tanpa komponen ALPA; DOSEN_LB dengan `kategoriStatusId` → 404 (tidak ada skala, sesuai DEC-070: LB tidak menerima gaji pokok yayasan); regenerate DRAF → 200, potongan ALPA tetap (idempotent); `GET .../slips/{id}/pdf` → 200 `application/pdf`; komponen ALPA tersimpan di `gajiPotongan`; jejak audit `payroll.slip.generated` terisi `unitId`.
- **Unit test baru:** `tests/payroll/domain/alpa_test.go` (AlpaDeduction: negatif/0/1/10/29/30/31/100; EligibleForAlpaDeduction: DOSEN_TETAP_YAYASAN/TENDIK_TETAP/TENDIK_KONTRAK true; DOSEN_DPK/DOSEN_LB/"" false) dan `tests/payroll/app/generate_test.go` (5 skenario: alpa 3 hari → 75.000; 40 hari → cap 750.000; kategori tidak tercakup → tanpa ALPA; 0 alpa → tanpa komponen; pegawai → `payrollDomain.ErrForbidden`). `go build`, `go vet`, `gofmt`, `go test ./...` hijau.
- **Belum diverifikasi / di luar cakupan:** penurunan `Presensi` dari `AttendanceEvent` (aturan H/I/S/A/C/LD, PIN→pegawai); pengisian `Presensi` oleh admin/operator; penegakan SP1 otomatis; dampak alpa pada slip TERBIT/TERBAYAR (generate menolak dengan 409 sebelum menghitung); perhitungan terhadap tabel TER resmi PMK 168/2023.

## 12. Verifikasi dinamis pipeline presensi (2026-11-14, `DEC-077`)

- **Lingkungan:** DB disposable lokal `simpeg_tahap2` (migrasi Goose `000001`–`000026`, versi 26 — `000026_mesin_absensi_pin.sql` diterapkan hanya di sini); seed sintetis: admin/pegawai/honorer `@example.test`, 1 mesin `MesinAbsensi` ('Mesin Finger 06'), pegawai terpetakan ke PIN `1001` (pegawai utama) dan `1002` (honorer), tap Juni–Juli 2026 (pegawai utama 4 hari di Juni; honorer Juni 4 hari + Juli Senin–Sabtu kecuali 07-08 dan 07-22), `HariLibur` sintetis 2026-06-05, 1 baris `Presensi` manual (status S, 2026-06-03); seed dihapus bersama database setelah selesai.
- **Hasil 20 pemeriksaan lulus:** list pin kosong → 200; create pin 1001 → 201 (+DTO berisi userId); duplikat PIN → 409; **import CSV** → 200 `imported=1` dengan 2 issue (baris 2 duplikat PIN, baris 4 duplikat user) sesuai desain; list pin → 2 item; **pengisian manual** 2026-06-03 → 201 `created=true` (baris manual diverifikasi tetap `S` setelah derivasi — `skippedManual=1`); pegawai → **403** pada list/derive/suspects; **derive** Juni–Juli 2026 → 200 (`created=121`, `skippedManual=1`, `byStatus: A=70, H=33, LD=18` — dikonfirmasi langsung ke DB: tap → H, 06-05/Sunday → LD, 06-03 manual → S tetap); **alpa-suspects** (`until=2026-07`, `months=2`) → memuat pegawai utama (20/27, total 47) dan honorer (21/2, total 23) dengan jumlah per bulan sesuai DB; **slip honorer Juni** → 200, potongan ALPA **Rp 525.000** (= 21 hari alpa × 25.000, cocok dengan hitungan DB; izin/sakit tidak memotong); **hapus pin** → 204; hapus kedua kali → 404; tanpa sesi → 401.
- **Jejak audit:** seluruh entri `attendance.pin.created`, `attendance.pin.deleted`, `attendance.presensi.upserted`, `attendance.presensi.derived` terisi `unitId` (terverifikasi via psql). 121 baris `sumber=derivasi_mesin`, 1 baris `sumber=manual` pada `Presensi`.
- **Temuan nyata selama verifikasi:** (1) **Bug zona waktu pada penurunan status:** server PostgreSQL beroperasi di `Asia/Jakarta`; seed pertama menanam event 01:00 WIB tanpa offset, yang tersimpan sebagai 01:00 WIB = hari sebelumnya 18:00 UTC — derivasi (yang memotong hari di UTC) mengaitkan tap ke tanggal yang salah (event Juli muncul di akhir Juni). **Diperbaiki** dengan 2 lapis: (a) seed uji menanam event dengan `AT TIME ZONE 'UTC'` (eksplisit), dan (b) `EventsPerPinPerDay` memotong `occurredAt` pada **kalender Asia/Jakarta** (`AT TIME ZONE 'Asia/Jakarta'`), bukan UTC — tap 00:00–07:00 WIB jatuh pada tanggal Jakarta-nya. (2) **Bug scan GORM:** `First` pada query tanpa model menambah `ORDER BY pin_id` yang tidak ada → 503; diganti `Take`. (3) **Bug mapping kolom DTO:** field `UserID` tidak ter-map ke `userId`; ditambahkan tag kolom eksplisit.
- **Belum diverifikasi / di luar cakupan:** tap nyata dari mesin produksi (event yang ada saat ini hanya berasal dari seed); pengisian `HariLibur` produksi (**kerja:** `POST /api/v1/holidays/2026/refresh` terhadap provider api.co.id sungguhan → 25 libur nasional 2026 tersimpan di disposable; derivasi Februari–Maret 2026 menghasilkan `LD` pada Imlek (16–17 Feb), rangkaian Nyepi/Idul Fitri (18–24 Mar), dan Minggu; terverifikasi langsung ke DB; DB disposable dihapus setelah selesai); dampak `Peringatan` (SP1 manual); perhitungan slip NYATA terhadap tabel TER resmi PMK 168/2023.

## 13. Verifikasi dinamis slip digital A5 + QR + verifikasi publik slip + pemberitahuan H-1 (2026-11-14, `DEC-078`)

- **Lingkungan:** DB disposable lokal `simpeg_tahap2` (migrasi Goose `000001`–`000026`, versi 26); seed sintetis (`@example.test` admin/pegawai; 1 unit, `kategoriStatusId` TENDIK_TETAP, skala `SK-2026`, PTKP `K/2`, TER sintetis); backend dev di port 8099 (`APP_ENV=development`, `PAYROLL_SLIP_ORG_NAME=Universitas Subang Sintetis`, `APP_BASE_URL=http://localhost:3000`); migrasi `000025`/`000026` diterapkan hanya di sini; **database dihapus setelah selesai**.
- **Hasil pemeriksaan lulus:**
  - `POST /api/v1/payroll/slips/generate` (periode `2026-06`, TENDIK_TETAP tingkat 1) → 200; slip `SL-2026-06-0001`; kolom `kodeVerifikasi` terisi **22 karakter base64url** (dibaca langsung via psql: `R8L3sUK8Gid1gGWzDOj_Ew`).
  - `GET /api/v1/verify/slip/{token}` **tanpa sesi** → 200; isi persis `{"nama":"Pegawai Sintetis","periode":"2026-06","periodeLabel":"Juni 2026","status":"DRAF","statusLabel":"Draf"}` — **nominal tidak pernah tampil**; `Cache-Control: no-store`.
  - Token bentuk salah (`token-bukan-22-karakter`) → **400**; token 22 karakter base64url yang sah tetapi tidak dikenal (`a`×22) → **404**; token persis 22 karakter (`aaaa…aaaa`, pola alak) → 404 (bukan 500).
  - `GET /api/v1/payroll/slips/{id}/pdf` (dengan sesi admin) → 200 `application/pdf`, **36.400 byte**, magic `%PDF-`; MediaBox `0 0 420.00 595.00` (A5 portrait); font tertanam **Tinos** dan **Tinos-Bold**; **1 image XObject** (QR verifikasi); nama instansi dari `PAYROLL_SLIP_ORG_NAME` pada kop. Dua kali unduhan PDF menghasilkan **2 entri jejak audit `payroll.slip.pdf.downloaded`** (aktor = admin, target = slip, `unitId` = unit pemilik slip) — jalur server hanya mencatat, tidak membatasi jumlah unduhan (batas 3× tetap di klien).
  - Audit `payroll.slip.generated` untuk slip tersebut **terisi `unitId`** (`e0000000-…-0001`).
  - Frontend `/verify/slip/{token}` terdaftar sebagai rute publik; build produksi dan `bun --bun run e2e` hijau (43/43, termasuk 2 uji baru `slip-verify.spec.ts`: halaman publik tampil + token tidak sah; batas unduh PDF 3× per pegawai/periode berhenti setelah 3 kali).
- **Efek samping yang disengaja (tercatat di `DEC-078`):** `PaydayDate` periode P = tanggal gajian bulan **P+1** (periode Juni dibayar 25 Juli); `NotifyDate` = **H−1**; `FindDueReminders` menghapus filter status (seluruh pemilik slip menerima pemberitahuan, bukan hanya TERBIT/TERBAYAR); slip tanpa `kodeVerifikasi` (data lama) dilewati tanpa pemberitahuan.
- **Belum diverifikasi / di luar cakupan:** pengiriman email H-1 ke kotak masuk nyata (transport ter-stub di unit test — multipart/related + QR CID terverifikasi di `tests/notify/slip_test.go`); slip A5 bermerek di data produksi; QR decode visual (diperiksa sampai level MediaBox/image object, bukan pemindai fisik); perhitungan TER resmi PMK 168/2023 (wajib diisi admin sebelum slip nyata).

## 14. Verifikasi dinamis hak cuti Universitas (2026-11-14, `DEC-079`)

- **Lingkungan:** DB disposable lokal `simpeg_tahap2` (migrasi Goose `000001`–`000027`, versi 27 — `000027_cuti_bersalin.sql` diterapkan hanya di sini); seed sintetis (`@example.test` admin/pegawai; pegawai a2 `tanggalMasuk` 2020-01-15 untuk syarat masa kerja; kategori DOSEN_TETAP_YAYASAN tingkat 3 = Rp 9.500.000); backend dev di port 8099 (`APP_ENV=development`); database **dihapus setelah selesai**.
- **Hasil pemeriksaan lulus:**
  - `POST /api/v1/cuti` jenis BERSALIN tanpa `subJenisBersalin` → **422**; dengan `MELAHIRKAN` + eviden surat dokter lengkap → **201** (id tercatat).
  - `POST /api/v1/cuti` IZIN_PENTING MENIKAH tanggal 2026-11-01 → 2026-11-08 (8 hari kalender > batas 7) → **422**.
  - `POST /api/v1/cuti` TAHUNAN 2026-11-02 → 2026-11-06 (5 hari kerja) → **201**; `POST /api/v1/cuti/{id}/approve` (admin) → **200**; `SaldoCuti` (pegawai, 2026) terkonfirmasi via psql → **5/12/0** (terpakai/kuota/carry).
  - Pengajuan yang **tumpang tindih** dengan pengajuan MENUNGGU/DISETUJUI milik pegawai yang sama → **409 Conflict** (diuji pada TAHUNAN tumpang tindih dengan TAHUNAN DISETUJUI, dan SAKIT tumpang tindih dengan TAHUNAN DISETUJUI; berlaku juga saat approve: penolakan atomik dalam transaksi).
  - Dua pengajuan SAKIT_BERKEPANJANGAN disetujui berturut-turut → `PeriodeSakit` terkonfirmasi via psql → `1/100` (Juni) lalu `2/100` (Juli): bulan ke berjalan, persen sesuai tier.
  - `POST /api/v1/cuti` PANJANG, masa kerja ≥ 6 tahun → **201**; `tanggalMasuk` diset 2024-01-15 → PANJANG **422**; ulang dengan `untukHaji: true` → **201** (tanggal uji 2027-04-01..2027-05-31, tidak bertumpuk dengan pengajuan lain).
  - `POST /api/v1/payroll/slips/generate` periode `2026-06` (pegawai a2, `PeriodeSakit` di-set `bulanKe=6`, `persenBerjalan=75` via psql) → **200**; `gajiPokok` pada slip = **Rp 7.125.000** (75% × Rp 9.500.000); audit `payroll.slip.generated` memuat catatan `sakit berkepanjangan bulan ke-6; gaji pokok dibayar 75%`.
- **Unit test tambahan:** `tests/directory/domain/leave_rules_test.go` (BERSALIN dengan/tanpa sub-jenis & batas durasi MELAHIRKAN 3 bulan / KEGUGURAN 45 hari; izin penting per sub-jenis; PANJANG 2 bulan; `SisaCuti` hangus; `PersenSakitBerkepanjangan`; `KalenderHariAntara`), `tests/payroll/app/generate_test.go` (slip dengan PeriodeSakit 75% → gaji pokok terpotong; tanpa PeriodeSakit → penuh), `tests/httpserver/hr_leave_handlers_test.go` (422 tanpa sub-jenis; 201 dengan sub-jenis; 422 izin penting > 7 hari). `go build`, `go vet`, `gofmt`, `go test ./... -count=1` hijau.
- **Belum diverifikasi / di luar cakupan:** carry-over nyata tahun berganti (butuh dua tahun kalender / simulasi); hangus 30 Juni (pola lazy terlihat di log server: `SaldoCuti` dibaca pada akses); cuti panjang real di tahun berjalan; integrasi PeriodeSakit untuk bulan >16 (mekanisme sama); koordinasi approver berjenjang TingkatTerusan (tidak berubah); email/notifikasi cuti.

## 15. Verifikasi dinamis pembatalan cuti (2026-11-14, `DEC-080`, `DEC-081`)

- **Lingkungan:** DB disposable lokal `simpeg_tahap2` (migrasi Goose `000001`–`000028`, versi 28 — `000027_cuti_bersalin.sql` dan **`000028_cuti_batal_status.sql`** diterapkan hanya di sini); seed sintetis (admin/pegawai `@example.test`, `tanggalMasuk` 2020-01-15); backend dev di port 8099; database **dihapus setelah selesai**.
- **Cacat nyata ditemukan di percobaan pertama:** `ALTER TYPE "StatusPersetujuan" ADD VALUE` — nilai `BATAL` belum terdaftar di enum PostgreSQL bawaan (`000001_directory_crud.sql:48`), sehingga `UPDATE "Cuti" SET status='BATAL'` gagal `SQLSTATE 22P02` dan endpoint menjawab **503** (dikonfirmasi di log server; migrasi 000028 ditambahkan). Setelah migrasi `000028` diterapkan, **200** pada kedua jalur pembatalan.
- **Hasil pemeriksaan lulus:**
  - TAHUNAN 5 hari kerja → approve 200 → `SaldoCuti` `5/12/0` (psql) → **cancel** → **200**, `status: BATAL`, saldo kembali **`0/12/0`** (psql).
  - **Cancel lagi** pada pengajuan yang sama → **409** (sudah `BATAL`).
  - TAHUNAN tanpa approve (`MENUNGGU`) → cancel → **200**, `status: BATAL`, saldo tidak berubah.
  - TAHUNAN → reject → **cancel** → **409** (sudah `DITOLAK`).
  - **Cancel oleh pegawai** (`cuti:approve:*` tidak di-grant) → **403**.
  - Jejak audit `cuti.canceled` terkonfirmasi via psql: **2 entri**, `aktorId` = admin pada kedua entri; `sasaran` = id pengajuan.
- **Pembatalan mandiri oleh pegawai (`DEC-081`):** pegawai cancel pengajuan miliknya (MENUNGGU) → **200** `BATAL`; approve lalu pegawai cancel → **200**, saldo kembali `0/12`; pegawai cancel pengajuan orang lain → **403**; cancel ulang (sudah BATAL) → **409**; approver-class tidak berubah (admin cancel 200/409); audit mencatat kedua jalur.
- **Belum diverifikasi / di luar cakupan:** pembatalan `SAKIT_BERKEPANJANGAN` yang sudah disetujui (PeriodeSakit sengaja tidak dihapus — gaji bulan berjalan perlu tinjauan admin gaji); pembatalan concurrent dari dua atasan (mutex baris pada `FOR UPDATE`).

## 16. Verifikasi dinamis perbaikan rute `POST /api/v1/cuti` untuk admin (2026-11-14, `DEC-082`)

- **Lingkungan:** DB disposable lokal `simpeg_tahap2` (migrasi Goose `000001`–`000028`, versi 28); seed sintetis; backend dev di port 8099; database **dihapus setelah selesai**.
- **Latar:** grant `cuti:create:any` di policy untuk `admin` tidak terhubung ke middleware rute — admin selalu mendapat **403** (dikonfirmasi di log server sebelum perbaikan).
- **Hasil pemeriksaan lulus:** `POST /api/v1/cuti` oleh admin untuk pegawai lain → **201** (id tercatat); `POST /api/v1/cuti` oleh admin jenis SAKIT dengan eviden surat dokter lengkap → **201**. Uji handler `TestCutiCreateAdminDapatMembuatUntukPegawaiLain` (201) ditambahkan ke `tests/httpserver/hr_leave_handlers_test.go`.
- **Catatan:** jalur pegawai (`cuti:create` self-only) dan operator (`cuti:create:homebase`) tidak berubah; `cuti:create:any` hanya diberikan ke `admin`.

## 17. Verifikasi dinamis modul pemberhentian + perpanjangan pasca-BUP (2026-11-15, `DEC-083`)

- **Lingkungan:** DB disposable lokal `simpeg_tahap2` (migrasi Goose `000001`–`000030`, versi 30 — `000029_kategori_dosen_tidak_tetap.sql` dan **`000030_pemberhentian_status.sql`** diterapkan hanya di sini); seed sintetis (`@example.test` admin/pegawai, `tanggalMasuk` 2020-06-16, kategori `DOSEN_TETAP_YAYASAN`); backend dev di port 8099 (`APP_ENV=development`, `PAYROLL_SLIP_ORG_NAME=Universitas Subang Sintetis`); database **dihapus setelah selesai**.
- **Hasil pemeriksaan lulus:**
  - `POST /api/v1/pesangon-aturan` (admin, kategori `DOSEN_TETAP_YAYASAN`): baris `masaKerjaTahun=0 → 2 bulan`; `masaKerjaTahun=5 → 4 bulan + 1 penghargaan` → tersimpan (201).
  - `POST /api/v1/pemberhentian` (admin) BUP tanpa aturan → approve → **200**, `kompensasiBulan: 0` (tabel kosong → tetap disetujui).
  - `POST /api/v1/pemberhentian` BUP kedua dengan aturan terisi → approve → **200**, `kompensasiBulan: 5` (base 4 dari aturan 5 tahun, tambahan 1 = floor(6/5)).
  - Pengajuan dengan `alasan` tidak dikenal → **422**.
  - `POST /api/v1/perpanjangan` oleh operator → **403**; oleh admin → **201** (`PKWT-2026-0001`; `kategoriStatusId` pegawai berubah ke `DOSEN_TIDAK_TETAP`; kontrak tercatat `PKWT` aktif; audit `perpanjangan.created` tertulis).
  - `GET /api/v1/kontrak?jenis=PKWT` → **200**, satu kontrak untuk pegawai tersebut.
  - Audit `pemberhentian.proposed`/`pemberhentian.approved` → **3/2 entri**; `pemberhentian.rejected` → 1 entri; `suratKeputusan` PHK `SK-PHK-2026-0001`/`SK-PHK-2026-0002` (`penerbit=YAYASAN`) tersimpan; `Pemberhentian` row berisi `status`/`alasan`/`kompensasiBulan`.
- **Temuan nyata selama verifikasi:** (1) tabel pesangonAturan mengimplementasikan upsert berdasar unique key `(kategoriStatusId, masaKerjaTahun)` — admin dapat mengganti nilai tanpa duplikat; (2) `approve` pada alasan yang bukan HORMAT menghasilkan `kompensasiBulan: 0` tanpa menahan persetujuan (kompensasi adalah rinciannya diformulasikan; approval tetap sah tanpa nilai kompensasi bila aturan belum diisi — keputusan desain disengaja, tercatat di `DEC-083`).
- **Belum diverifikasi / di luar cakupan:** perpanjangan multi-siklus (dua kali perpanjangan berturut-turut); validasi tanggal kontrak (tumpang tindih); formula `penghargaanBulan`/`gantiKerugianBulan` ke audit/rincian slip; integrasi `Pemberhentian` ke mesin slip (nominal tidak tercetus di slip payroll saat ini); **belum** pada staging/produksi (migrasi `000025`–`000030` belum diterapkan).

## 18. Verifikasi dinamis lanjutan modul pemberhentian + kontrak pasca-BUP (2026-11-15, `DEC-083`)

- **Lingkungan:** DB disposable lokal `simpeg_tahap2` (migrasi Goose `000001`–`000030`); pegawai `pegawai@example.test` (`a0000000-…-0002`) dengan `tanggalMasuk` 2020-06-16 dan kategori `DOSEN_TETAP_YAYASAN`; skala `SK-2026` (11+ tingkat; tingkat 3 = Rp 9.500.000); `pesangonAturan` 0thn→2 / 5thn→4; backend dev di port 8099. Database dihapus setelah selesai.
- **Hasil pemeriksaan lulus:**
  - `POST /api/v1/pemberhentian` (admin, BUP, tanggal 2026-11-15) → **201**; `POST .../approve` → **200**, `kompensasiBulan: 5`; `user.aktif` → **false** setelah setuju; audit `pemberhentian.approved` tercatat "pegawai dinonaktifkan".
  - `POST /api/v1/payroll/slips/generate` periode `2026-10` (periode sebelum pemberhentian) → **200** (`gajiPokok=Rp 9.500.000`, total 11.000.000).
  - `POST /api/v1/payroll/slips/generate` periode `2026-11` (bulan pemberhentian) → **200**; komponen pendapatan: `FUNGSIONAL` Rp 1.500.000 + **`PESANGON` Rp 47.500.000** (5 bulan × Rp 9.500.000); total Rp 58.500.000.
  - `POST /api/v1/payroll/slips/generate` periode `2026-12` (setelah pemberhentian) → **409 conflict**.
  - Perpanjangan pertama pasca-BUP (admin) → **201** `PKWT-2026-0001`; perpanjangan kedua (tumpang tindih dengan kontrak AKTIF) → **409 conflict**; override `allowOverlap: true` → **201** `PKWT-2026-0002` (audit `perpanjangan.created` memuat klausa override).
- **Belum diverifikasi / di luar cakupan:** (1) slip pemberhentian bagi pegawai dengan `periodeSakit` berjalan pada bulan pemberhentian (interaksi sakit-berkepanjangan × PESANGON belum diuji end-to-end); (2) `Pemberhentian` dengan alasan tidak hormat pada bulan pemberhentian (PESANGON harus tidak muncul — kompensasi 0 — logika ada, belum diverifikasi dinamis); (3) UI unduh slip untuk slip yang memuat komponen PESANGON (tidak ada spec e2e khusus); (4) PDF slip A5 dengan komponen PESANGON (lebar komponen/jenis belum diuji visual); (5) approve pemberhentian tidak mengubah kepesertaan DPLK (`pensiunKepesertaan.aktif` tidak dimatikan — keputusan desain: data DPLK sengaja dipertahankan untuk administrasi iuran terpisah); (6) **belum** pada staging/produksi (migrasi `000025`–`000030` belum diterapkan).

## 19. Verifikasi dinamis THR + pembatalan slip saat approve pemberhentian (2026-11-15, `DEC-084`)

- **Lingkungan:** DB disposable lokal `simpeg_tahap2` (migrasi Goose `000001`–`000031`, versi 31 — `000031_slip_batal_status.sql` diterapkan hanya di sini); pegawai `pegawai@example.test`; backend dev port 8099. Database dihapus setelah selesai. (Supersedes catatan e2e 43/43 pada §17 — e2e sekarang 49/49.)
- **Hasil pemeriksaan lulus:**
  - `POST /api/v1/payroll/slips/generate` `2026-10` → **200** (DRAF); `2026-12` → **200** (DRAF, dipakai untuk uji pembatalan).
  - `POST /api/v1/payroll/thr/config` (2027, `GAJI_KOTOR`, syarat 3 bln, berlaku 2027-03-09) → **201**.
  - `POST /api/v1/payroll/thr/generate` sebelum ada slip `TERBIT` → **0 dibuat**, 3 pegawai dilewati (`no_terbit_slip`).
  - Slip `2026-10` ditandai `TERBIT` → `GET /api/v1/payroll/thr/generate` → **generated=1**, slip **`THR-2027-0001`**, `totalGaji=Rp 11.000.000` (bruto slip TERBIT), tunjangan komponen `THR` Rp 11.000.000; run kedua → idempotent (slip digantikan, tidak berlipat); audit `payroll.thr.generated` tertulis (2 entri).
  - `GET /api/v1/payroll/slips?periode=2027-03` setelah perubahan listing (periode cocok sebulan penuh) → **`THR-2027-0001` tampil**; PDF slip THR → **200** (`%PDF-`, 36.308 byte).
  - `POST /api/v1/pemberhentian` (BUP) + `POST .../approve` → **200**, pegawai nonaktif; slip DRAF `2026-12` → **`BATAL`** (listing); PDF slip `2026-12` → **409**; audit `pemberhentian.approved` memuat "N slip DRAF masa depan dibatalkan".
- **Belum diverifikasi / di luar cakupan:** alur penerbitan slip THR (`DRAF` → `TERBIT`) untuk THR (admin melakukan langkah yang sama seperti slip reguler — belum dites UI); interaksi PESANGON × PeriodeSakit pada bulan pemberhentian; pembedaan THR pada halaman `/payroll/slips` frontend di luar komponen; iuran BPJS untuk slip THR (tanpa potongan sesuai `DEC-084`); **belum** pada staging/produksi (migrasi `000025`–`000031` belum diterapkan); TER untuk THR-only note belum diverifikasi (PPh 21 THR di luar perhitungan).

## 20. Verifikasi dinamis THR — alur lengkap + pemberhentian + kompensasi (2026-11-15, `DEC-084`/`DEC-083`)

- **Lingkungan:** DB disposable lokal `simpeg_tahap2` (migrasi Goose `000001`–`000031`, versi 31); backend dev port 8099; pegawai dengan `PeriodeSakit` bulan ke-2 (75%), `tanggalMasuk` 2020-06-16, kategori `DOSEN_TETAP_YAYASAN`, skala `SK-2026` tingkat 3 (Rp 9.500.000). Database dihapus setelah selesai.
- **Hasil pemeriksaan lulus:**
  - **Pratinjau kompensasi sebelum approve:** `GET /api/v1/pemberhentian/{id}/kompensasi-preview` → **200**, `{honoured:true, tersedia:true, masaKerjaTahun:6, pesangonBulan:4, tambahanBulan:1, totalBulan:5}` — identik dengan nilai saat approve (`kompensasiBulan=5`).
  - **PESANGON × PeriodeSakit interaksi end-to-end:** approve BUP → generate slip `2026-11` → `total gaji pokok = Rp 7.125.000` (75% × 9.500.000); komponen: `FUNGSIONAL` 1.500.000 + **PESANGON Rp 35.625.000** (5 × 7.125.000); total slip **Rp 44.250.000**. Audit `payroll.slip.generated` memuat dua catatan (sakit, pesangon).
  - **Alur slip lengkap** (`generate` → `terbit` → `payment`): generate → **200** `DRAF`; `POST /api/v1/payroll/slips/{id}/terbit` → **200** `TERBIT`; terbit ulang → **409 conflict**; `POST .../payment` → **200** `TERBAYAR`; audit `payroll.slip.published` dan `payroll.slip.paid` tertulis.
  - **THR dengan slip TERBAYAR sebagai sumber:** `POST /api/v1/payroll/thr/generate` → **generated=1** (`THR-2027-0001`, `totalGaji=Rp 8.625.000` — bruto slip TERBAYAR bulan pemberhentian saat gaji pokok terpotong sakit 75%); regenerate → idempotent; `POST .../terbit` → **200** `TERBIT`; `POST .../payment` → **200** `TERBAYAR`.
  - **PDF slip** (penerimaan kolom): terunduh **200**, `%PDF-`, 36/37 KB (3 komponen pendapatan tergambar; tanpa alat PDF-to-text di lingkungan agen — verifikasi visual komponen PESANGON wajib dilakukan manusia sebelum pilot).
  - e2e: **49/49** (spec baru: `admin menerbitkan slip DRAF menjadi TERBIT`).
- **Belum diverifikasi / batasan dicatat:** PDF-to-text visual check PESANGON (perlu manusia); refresh UI setelah approve pemberhentian menampilkan status terbarui tanpa reload manual (belum otomatis); notifikasi H-1 THR tercakup alur reguler unit berjadwal (belum diuji terpisah); **belum** pada staging/produksi (migrasi `000025`–`000031` belum diterapkan).

## 21. Verifikasi hibrida UI + backend + database: alur pemberhentian × sakit berkepanjangan (2026-11-15, `DEC-083`/`DEC-084`)

- **Lingkungan:** backend dev nyata di `localhost:8099` (DB disposable `simpeg_tahap2`, migrasi `000001`–`000031`) + server dev frontend `localhost:3101` dengan proksi `/api/v1` → backend; pengujian Playwright **hanya menyimulasikan autentikasi** (`POST /api/auth/sign-in/email` dan `GET /api/v1/me`), seluruh panggilan modul pemberhentian/penggajian diteruskan ke backend nyata. Spec: `frontend/e2e/hybrid-pemberhentian.spec.ts` — dilewati (skipped) bila backend uji tidak hidup, sehingga `bun run e2e` tanpa backend tetap hijau.
- **Alur yang dijalankan lewat UI sungguhan:** propose BUP (dialog) → **pratinjau kompensasi** (backend nyata: `{masaKerjaTahun:6, pesangonBulan:4, tambahanBulan:1, totalBulan:5}`) → approve → generate slip `2026-11` (tingkat 3, kategori `DOSEN_TETAP_YAYASAN`) → detail slip.
- **Hasil yang terverifikasi (backend + UI + DB):**
  - Slip `SL-2026-11-0001`: gaji pokok **Rp 7.125.000** (PeriodeSakit bulan ke-2 = 75%), pendapatan `FUNGSIONAL` Rp 1.500.000 + **`PESANGON` Rp 35.625.000**, total **Rp 44.250.000**; DB: `Pemberhentian` DISETUJUI `kompensasiBulan=5`, `user.aktif=false`.
  - **PDF A5 diperiksa visual** (dirender lewat Chromium): kolom PENERIMAAN memuat baris **PESANGON — Rp 35.625.000** di bawah `Gaji pokok Rp 7.125.000`; POTONGAN KOPERASI/BPJS/PPH21; total penerimaan Rp 44.250.000; gaji bersih Rp 41.723.750; QR + tautan verifikasi; tata letak A5 rapi (tanpa tumpang tindih). Artefak: tangkapan layar + berkas PDF.
  - Test hibrida **lulus** (1 uji, ~4 detik); e2e total repo **49/49** tanpa backend + 1 skipped hibrida.
- **Temuan nyata selama verifikasi hibrida (dan perbaikannya):** (1) **pengajuan dengan UUID pegawai yang tidak dikenal** menghasilkan **503** (keterbatasan FK database) → kini **`404`** (pemeriksaan keberadaan pegawai di lapisan aplikasi, `domain.ErrNotFound`); (2) **UUID sintetis pada seed disposable** (`a0000000-0000-0000-…`) **bukan UUID RFC-valid** sehingga ditolak `z.uuid()` di frontend — artefak data uji, bukan cacat produksi (UUID produksi = v4); seed disposable diperbaiki ke bentuk RFC-v4 (`-4000-8000-9000-`) sehingga alur UI nyata dapat dijalankan; (3) token sesi pada halaman disimpan **di memori** — navigasi lewat `page.goto` (muat ulang) mengeluarkan pengguna ke `/login`; pengujian hibrida memakai klik sidebar.
- **Belum diverifikasi:** pengujian hibrida pada browser non-Chromium; environments tanpa `VITE_API_PROXY_TARGET` (proxy hanya aktif bila env tersebut disetel); migrasi `000025`–`000031` **belum** diterapkan pada staging/produksi.

## 22. Verifikasi dinamis kebijakan THR final (`DEC-085`: rencana, prorata admin, tanpa iuran)

- **Lingkungan:** DB disposable `simpeg_tahap2` (migrasi `000001`–`000031`); pegawai `PeriodeSakit` bulan 2 (75%), `tanggalMasuk` 2020-06-16, kategori `DOSEN_TETAP_YAYASAN`, skala tingkat 3 (Rp 9.500.000); backend dev port 8099. Database dihapus setelah selesai.
- **Hasil pemeriksaan lulus:**
  - Slip gaji `2026-11` → `TERBIT` → `TERBAYAR` (sumber nominal THR).
  - `GET /api/v1/payroll/thr/plan?tahun=2027` → **200**: 1 kandidat (`nama`, `masaKerjaTahun: 6`, `nominalOtomatis: 8.625.000` = bruto slip TERBAYAR, `sumber: GAJI_KOTOR`) + 2 pegawai dilewati (`no_terbit_slip`).
  - `POST /api/v1/payroll/thr/generate` dengan `overrides:[{userId, nominal: 5.500.000}]` → **200**, `generated: 1`, `created[0].nominal = 5.500.000`, `created[0].sumber = "input admin (prorata)"`.
  - Generate ulang tanpa override → `created[0].nominal = 8.625.000`, `sumber = "otomatis"` (slip `DRAF` digantikan; nomor tetap `THR-2027-0001`).
  - DB: slip THR `totalGaji = Rp 8.625.000`, komponen `THR` satu baris, **0 baris potongan** (tanpa iuran apa pun); audit `payroll.thr.generated` mencatat sumber nominal pada kedua run.
  - e2e halaman THR: **2/2** (rencana dimuat, nominal diubah admin, body generate berisi `overrides` sesuai).
- **Belum diverifikasi:** THR pada periode yang sama dengan slip gaji reguler (periode slip berbeda: tanggal berlaku vs tanggal 1), slips slip THR berbulan; konfirmasi pajak THR oleh akuntan (di luar sistem); **belum** pada staging/produksi.

## 7. Kesimpulan

Seluruh rute terdaftar menolak akses tanpa sesi (`401`), matriks peran **berjalan sesuai grant runtime**, dan tidak ditemukan kebocoran IDOR pada skenario lintas cakupan yang diuji. Dua cacat nyata ditemukan dan diperbaiki/dicatat (§1, §5). Verifikasi dinamis ini melengkapi — bukan menggantikan — pengujian otomatis (`go test ./...`) dan tinjauan kontrak OpenAPI.
