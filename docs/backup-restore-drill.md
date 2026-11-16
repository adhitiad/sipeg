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

_Lampirkan log waktu/perintah ringkas (tanpa kredensial) bila perlu. Setelah selesai, ringkas hasil di catatan operasi dan perbarui `docs/RUNBOOK.md`/`docs/DECISIONS.md` jika ada perubahan prosedur atau target._
