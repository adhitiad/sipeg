# Usulan Alur Surat Keputusan (SK) — menunggu keputusan pemilik

- **Status:** usulan (belum diputuskan). Modul **baca** sudah terimplementasi; penerbitan manual belum.
- **Rujukan:** `DEC-083` (SK PHK otomatis saat persetujuan pemberhentian), `DEC-091`, `docs/PRD.md` §5.9 (SK Yayasan vs SK Rektor), `backend/migrations/000025_payroll_core.sql` (tabel `suratKeputusan`).

## 1. Kondisi sekarang

| Aspek                                               | Status                                                                                                                       |
| --------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------- |
| Model `suratKeputusan`                              | Ada (id, `nomorSk` unik, `jenis`, `penerbit`, `userId`, `homebaseId`, `tanggalSk`, masa berlaku, `fileUrl`, `metadata` JSON) |
| Penerbitan otomatis                                 | SK-PHK terbit saat admin menyetujui pengajuan pemberhentian (nomor `SK-PHK-<tahun>-NNNN`)                                    |
| SK pengangkatan/mutasi/jabatan struktural/penugasan | **Belum ada** alur penerbitan                                                                                                |
| Modul baca                                          | **Sudah ada** (lihat §2)                                                                                                     |

## 2. Yang sudah terimplementasikan (baca saja)

- `backend/internal/sk/{domain,app,repository}` + `backend/internal/httpserver/sk_handlers.go`.
- Endpoint `GET /api/v1/surat-keputusan` (daftar, pencarian, filter `jenis`/`penerbit`/`userId`, pagination) dan `GET /api/v1/surat-keputusan/{id}` (detail).
- Permission: `sk:read:any` (admin, seluruh organisasi) dan `sk:read:homebase` (operator & atasan — **hanya SK pegawai pada cakupan unit beserta turunannya**; SK di luar cakupan dijawab `404`, bukan `403`, agar keberadaan SK tidak terkonfirmasi).
- UI: `/admin/sk` (daftar + filter + dialog detail), sidebar gated `sk:read:any`/`sk:read:homebase`.
- Tidak ada endpoint tulis: SK terbit dari alur kerja, bukan CRUD manual.
- Bukti: `docs/DECISIONS.md` `DEC-091`.

## 3. Usulan alur penerbitan (perlu diputuskan)

Bagian berikut **belum** diimplementasikan; menunggu keputusan pemilik (kabiro kepegawaian & yayasan).

1. **Kapan SK terbit.** Saat pengajuan disetujui (seperti SK-PHK sekarang) atau setelah persetujuan atasan lain (mis. HUMAN RESOURCES) — mana yang benar? Untuk pengangkatan/mutasi, pengajuan apa yang menjadi pemicu?
2. **Siapa yang menerbitkan.** Usulan: admin (kantor pusat) menerbitkan atas nama `YAYASAN` atau `REKTOR` sesuai jenis SK; operator hanya bisa mengajukan.
3. **Penomoran.** Usulan: `SK-<JENIS>-<tahun>-<urutan>` per penerbit (mis. `SK-MUTASI-2026-0001`); urutan dihitung dalam satu transaksi bersama pembuatan baris.
4. **Dokumen fisik/digital.** Apakah SK perlu berkas PDF (unduhan) dan arsip dokumen (tautan ke `arsipDokumenPegawai`)? Saat ini `fileUrl` hanya kolom kosong.
5. **Retensi & pembatalan.** Apakah SK dapat dibatalkan/dicoret, dan bagaimana history-nya (immutable vs pembatalan dengan SK pembatalan)?
6. **Integrasi ke slip/kontrak.** Apakah SK pengangkatan/mutasi otomatis memperbarui `kontrakKerja` atau komponen gaji? (Sekarang SK PHK tidak mengubah kontrak; status kontrak diubah lewat alur kontrak.)
7. **Dokumen pengesah.** Apakah metadata SK perlu memuat siapa penanggung jawab dan nomoroia?

## 4. Setelah keputusan

- Keputusan dicatat di `docs/DECISIONS.md` sebagai entri baru (nomor berikutnya), lalu:
  1. backend: lapis `domain/app/repository/httpserver` untuk penerbitan (+ audit `sk.created`/`sk.canceled`, nomor urut transaksional);
  2. migrasi baru bila diperlukan (mis. tabel `suratKeputusan`_seq atau kolom baru) — lewat Goose, staging lebih dulu;
  3. frontend: dialog Terbitkan SK + unduhan PDF bila disepakati;
  4. verifikasi dinamis pada database disposable, lalu pencatatan bukti di `docs/DECISIONS.md`.
