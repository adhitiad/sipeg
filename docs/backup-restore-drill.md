# Checklist & formulir uji backup/restore (Supabase Free, $0)

> **Hasil drill 2026-11-16 (staging, bukan produksi)** — lengkap di `docs/DECISIONS.md` `DEC-100`:
> dump `pg_dump` ≈ **59 detik**, 2.555.361 byte; restore ke container disposable `postgres:16-alpine` ≈ **5 detik** (4 galat khusus Supabase `supabase_vault`/`transaction_timeout`, diabaikan — tidak menyentuh skema aplikasi); **verifikasi identik dengan sumber**: 89 tabel, 89 RLS, Goose 33, `user`=102 (100 sintetis + 2 pilot), `gaji`=301, `session`=30, `account`=2, `auditLog`=50.161. Artefak dihapus. **Belum menggantikan uji di DB produksi.**

Pemilik: **adhitiad (admin)**. Terkait: `docs/DECISIONS.md` `DEC-050`, `DEC-065`, `docs/RUNBOOK.md` §7 & §8E.  
Tujuan: membuktikan prosedur pemulihan, mengukur durasi pulih dan titik data, serta membandingkan dengan target awal **RPO 24 jam / RTO 4 jam** (nilai awal, belum terukur produksi).

> **Biaya: harus $0.** Gunakan fitur/alat yang tersedia tanpa biaya tambahan: Supabase tier **Free** (backup/mekanisme restore yang tersedia pada project), `pg_dump`/`pg_restore`/`psql` (client PostgreSQL), dan database uji **disposable** lokal/container. **Jangan** mengaktifkan fitur berbayar (PITR umumnya tidak tersedia di Free) kecuali sudah diputuskan terpisah.
>
> **Keamanan:** jangan menaruh kredensial/connection string di repo atau catatan yang dibagikan. Jangan menimpa database produksi. Gunakan data sintetis bila memungkinkan; bila uji memakai salinan data nyata, ikuti prosedur yang disetujui dan hapus artefak/cadangan sementara setelah verifikasi.

## 1) Metadata uji (isi saat eksekusi)

| Field                                     | Nilai                                                                         |
| ----------------------------------------- | ----------------------------------------------------------------------------- |
| Tanggal/waktu mulai (WIB/UTC)             |                                                                               |
| Tanggal/waktu selesai                     |                                                                               |
| Pelaksana                                 | adhitiad (admin) / ...                                                        |
| Lingkungan sumber                         | produksi Supabase / staging / disposable (pilih & jelaskan)                   |
| Supabase project ref                      |                                                                               |
| Supabase tier & region                    | Free, region: ...                                                             |
| Metode backup/restore                     | dashboard/fitur Supabase Free / `pg_dump`+`pg_restore` / kombinasi (jelaskan) |
| Target restore (disposable, non-produksi) | nama DB/host: ...                                                             |
| Versi Goose sumber (sebelum)              |                                                                               |
| Komit/tag aplikasi (jika relevan)         |                                                                               |

## 2) Verifikasi kemampuan tier (wajib sebelum klaim PITR/RPO)

- [ ] Fitur backup/restore yang **tersedia** pada project Supabase target diverifikasi (dashboard/dokumentasi project): ______________________________
- [ ] **PITR tersedia?** ☐ Ya ☐ Tidak ☐ Tidak pasti. Jika tidak tersedia (umum pada Free), titik pemulihan mengikuti backup harian/ekspor yang tersedia — catat batasannya: ______________________________
- [ ] Kebijakan retensi backup platform (sesuai tier) dicatat: ______________________________

## 3) Prasyarat & keamanan

- [ ] Target restore adalah database **terpisah/disposable** (bukan produksi) dan disetujui.
- [ ] Akses kredensial hanya dari environment/secret manager lokal; tidak disalin ke repo/chat/log.
- [ ] Ruang penyimpanan sementara untuk dump/artefak aman dan akan dibersihkan.
- [ ] `psql`, `pg_dump`, `pg_restore` tersedia (atau gunakan container `postgres:16-alpine` secara lokal).

## 4) Langkah eksekusi & pengukuran

Catat waktu tiap langkah (untuk RTO).

| Langkah                                        | Mulai | Selesai | Durasi | Catatan                  |
| ---------------------------------------------- | ----- | ------- | ------ | ------------------------ |
| (A) Ambil cadangan / siapkan sumber restore    |       |         |        | ukuran dump/artefak: ... |
| (B) Pulihkan ke target disposable              |       |         |        | metode: ...              |
| (C) Verifikasi pascapulih                      |       |         |        | lihat bagian 5           |
| (D) Bersihkan (hapus DB uji/artefak sementara) |       |         |        |                          |

**Titik data pemulihan:** waktu/identitas backup yang dipulihkan = __________________ (bandingkan dengan waktu insiden hipotetis untuk estimasi RPO).

### Contoh perintah referensi ($0, jalankan lokal terhadap endpoint yang sah)

> Ganti placeholder; set `SOURCE_URL`/`TARGET_URL` di environment, jangan hardcode. Contoh memakai Docker lokal untuk target disposable.

```bash
# Target disposable lokal (contoh)
docker run --name simpeg-restore-drill -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=simpeg_restore -p 54329:5432 -d postgres:16-alpine

# Dump dari sumber (butuh akses jaringan/kredensial yang sah)
pg_dump "$SOURCE_URL" -Fc -f restore_drill.dump

# Restore ke target disposable
pg_restore --clean --if-exists --no-owner --no-privileges -d "$TARGET_URL" restore_drill.dump
# atau: psql "$TARGET_URL" -v ON_ERROR_STOP=1 -f dump.sql
```

(Sesuaikan dengan mekanisme restore yang tersedia pada Supabase Free; jika memakai fitur restore platform, catat langkah UI/API dan waktu yang terukur.)

## 5) Verifikasi pascapulih (wajib)

- [ ] Restore selesai **tanpa galat fatal** (catat peringatan jika ada): ______________________________
- [ ] **Migrasi/skema:** versi Goose dan keselarasan skema sesuai sumber.
  - `SELECT version_id, is_applied FROM goose_db_version ORDER BY id DESC LIMIT 5;` → __________
  - (opsional, dari repo backend dengan `DATABASE_URL` target uji) `go run ./cmd/perfbench schema` → __________
- [ ] **Objek & keamanan skema:** jumlah tabel/indeks/constraint/RLS sesuai harapan: __________
- [ ] **Hitungan baris kunci** (bandingkan sumber vs hasil, atau catat snapshot):

| Tabel/objek          | Sumber | Hasil restore | Cocok? |
| -------------------- | -----: | ------------: | ------ |
| `user`               |        |               |        |
| `session`            |        |               |        |
| `account`            |        |               |        |
| `auditLog`           |        |               |        |
| (domain kritis lain) |        |               |        |

- [ ] **Kesiapan aplikasi (jika diuji):** `/ready` = 200 terhadap DB hasil restore; sampel baca read-only berhasil: __________
- [ ] **Artefak dibersihkan:** dump sementara dihapus, DB/container disposable dihapus: __________

## 6) Hasil & evaluasi

| Metrik                                            | Target awal    | Hasil terukur | Status                                  |
| ------------------------------------------------- | -------------- | ------------- | --------------------------------------- |
| RPO (kehilangan data maks, sesuai kemampuan tier) | 24 jam (awal)  |               | ☐ Memenuhi ☐ Tidak ☐ Terbatas oleh tier |
| RTO (durasi hingga layanan/data siap)             | 4 jam (awal)   |               | ☐ Memenuhi ☐ Tidak                      |
| Keutuhan skema/data verifikasi                    | identik/sesuai |               | ☐ OK ☐ Ada temuan                       |

**Temuan & tindak lanjut:**
-

-

**Kesimpulan:** ☐ Lulus (bukti memadai untuk lingkup uji ini) ☐ Lulus dengan catatan ☐ Gagal/perlu uji ulang

**Penanggung jawab & tanggal tanda tangan:** ______________________________

---

## Lampiran — Drill 2026-11-17 (sumber dev Supabase → target disposable lokal)

> **Catatan:** drill ini dijalankan terhadap sumber **dev Supabase** (bukan produksi) dan target **PostgreSQL 18 lokal disposable** (`simpeg_restore_drill`, dihapus setelah verifikasi). Tidak menggantikan uji pada project produksi. Hasil dicatat untuk bukti prosedur dan pengukuran durasi.

### Metadata

| Field | Nilai |
|---|---|
| Tanggal/waktu mulai (WIB) | 2026-11-17 09:39:39 (UTC 02:39:39) |
| Tanggal/waktu selesai | 2026-11-17 09:40:32 (UTC 02:40:32) |
| Pelaksana | adhitiad (admin) |
| Lingkungan sumber | Supabase dev (staging), pooler sesi porta 5432 |
| Supabase project ref | emsraijcyvxuxdrqujbd |
| Supabase tier & region | dev, ap-northeast-1 (Tokyo) |
| Metode backup/restore | `pg_dump -Fc` + `pg_restore --clean --if-exists --no-owner --no-privileges` |
| Target restore (disposable) | PostgreSQL 18 lokal, DB `simpeg_restore_drill` (dihapus setelah verifikasi) |
| Versi Goose sumber | 33 |
| Komit/tag aplikasi | working tree (backend 2026-11-17) |

### Verifikasi kemampuan tier

- Fitur backup/restore yang tersedia: `pg_dump`/`pg_restore` konsisten via pooler sesi (porta 5432, `default_query_exec_mode=simple_protocol`). PITR tidak diuji pada target ini.
- PITR tersedia? Tidak pasti (perlu verifikasi pada project produksi target).
- Kebijakan retensi backup platform: mengikuti tier Supabase project (belum diverifikasi pada target produksi).

### Prasyarat & keamanan

- [x] Target restore adalah database terpisah/disposable (lokal, bukan produksi).
- [x] Akses kredensial hanya dari environment; tidak disalin ke repo/chat/log.
- [x] Ruang penyimpanan sementara (dump) aman dan dibersihkan setelah verifikasi.
- [x] `psql`, `pg_dump`, `pg_restore` tersedia (PostgreSQL 18).

### Langkah eksekusi & pengukuran

| Langkah | Mulai | Selesai | Durasi | Catatan |
|---|---|---|---|---|
| (A) Ambil cadangan (`pg_dump -Fc`) | 09:39:39 | 09:40:29 | 49,94 detik | ukuran dump ~2,5 MB; SHA256 `D2461477EEA75947362A61832EA70B0BD6AA6571EEA222878C1515B490F845B5`; param `default_query_exec_mode` di-strip otomatis oleh helper |
| (B) Pulihkan ke target disposable | 09:40:29 | 09:40:32 | 2,59 detik | 3 galat non-fatal: extension `supabase_vault` (bukan skema aplikasi) |
| (C) Verifikasi pascapulih | 09:40:32 | 09:40:33 | ~1 detik | lihat di bawah |
| (D) Bersihkan (hapus DB uji + artefak) | — | — | — | DB `simpeg_restore_drill` dihapus; dump dihapus |

**Titik data pemulihan:** backup diambil 2026-11-17 09:39:39 WIB (snapshot sumber dev).

### Verifikasi pascapulih

- [x] Restore selesai tanpa galat fatal (3 peringatan `supabase_vault` diabaikan — tidak menyentuh skema aplikasi).
- [x] Migrasi/skema: Goose versi 33 (cocok dengan sumber).
- [x] Objek & keamanan skema: 89 tabel, 89 tabel dengan RLS aktif.
- [x] Hitungan baris kunci: `user`=1, `session`=30, `account`=2, `auditLog`=50161, `gaji`=301 — cocok dengan bukti drill sebelumnya.
- [x] Artefak dibersihkan: dump sementara dihapus, DB disposable dihapus.

### Hasil & evaluasi

| Metrik | Target awal | Hasil terukur | Status |
|---|---|---|---|
| RPO (kehilangan data maks) | 24 jam (awal) | ~50 detik (durasi dump) | Terbatas oleh tier (dev, bukan produksi) |
| RTO (durasi hingga data siap) | 4 jam (awal) | ~53 detik total (dump+restore+verifikasi) | Memenuhi (pada target dev) |
| Keutuhan skema/data | identik | Goose 33, 89 tabel, 89 RLS, hitungan baris cocok | OK |

### Temuan & tindak lanjut

- `pg_dump` tidak menerima parameter `default_query_exec_mode` pada DSN — helper kini men-strip query string otomatis sebelum memanggil `pg_dump`.
- 3 galat `supabase_vault` non-fatal pada restore (extension tidak tersedia di PostgreSQL lokal) — tidak menyentuh skema aplikasi; abaikan.
- Drill ini dilakukan pada sumber dev, bukan produksi; RPO/RTO produksi tetap harus diuji pada project Supabase target.

### Kesimpulan

Lulus dengan catatan (bukti memadai untuk lingkup dev; uji produksi tetap wajib).

**Penanggung jawab & tanggal tanda tangan:** adhitiad (admin) — 2026-11-17

---

_Lampirkan log waktu/perintah ringkas (tanpa kredensial) bila perlu. Setelah selesai, ringkas hasil di catatan operasi dan perbarui `docs/RUNBOOK.md`/`docs/DECISIONS.md` jika ada perubahan prosedur atau target._
