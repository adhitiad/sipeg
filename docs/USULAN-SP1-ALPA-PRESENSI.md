# Usulan Alur SP1 dari Presensi (peringatan otomatis) — menunggu keputusan pemilik

- **Status:** usulan alur (belum diputuskan, **belum diimplementasikan**).
- **Rujukan:** `DEC-077` (pipeline presensi; SP1 sengaja tidak terbit otomatis), `docs/PRD.md` §5.9 dan §3B, `backend/internal/models/models_lifecycle.go` (model `Peringatan`).

## 1. Kondisi sekarang

| Aspek                 | Status                                                                                                                           |
| --------------------- | -------------------------------------------------------------------------------------------------------------------------------- |
| Data presensi harian  | Ada (`presensi`, diturunkan dari `attendanceEvent`; status `H`, `A`, `LD`, `C`, `S`, `I`)                                        |
| Laporan alpa berulang | Ada: `GET /api/v1/attendance/alpa-suspects` (kandidat alpa berulang, untuk ditindaklanjuti manual)                               |
| Komponen alpa di slip | Sudah ada: komponen `ALPA` (Rp 25.000 per hari alpa tanpa keterangan, maks Rp 750.000/bulan) untuk kategori tertentu (`DEC-076`) |
| Model `Peringatan`    | Ada (`userId`, `tingkat`, `alasan`, `nomorSurat`, `tanggal`, `penerbit`, `sanksi`, `sanksiCatatan`) — **tanpa endpoint**         |
| SP1                   | **Tidak pernah terbit otomatis**; SP1 contemplates diterbitkan manual                                                            |
| Konsekuensi di slip   | Pemotongan alpa di slip sudah otomatis; SP1 (peringatan/sanksi) **belum** memengaruhi slip                                       |

## 2. Hal yang harus diputuskan lebih dulu

1. **Pemicu.** Kapan SP1 terbit? Usulan awal: alpa tanpa keterangan (status `A`) berulang dalam rentang tertentu (mis. minimal 3 hari dalam 1 bulan, atau minimal 5 hari dalam 3 bulan) **dan** tidak ada sakit/cuti/izin yang sah. Rentang mana yang benar?
2. **Persetujuan.** Apakah SP1 terbit otomatis begitu ambang terpenuhi, atau perlu persetujuan admin/kabiro lebih dulu? Usulan: perlu persetujuan — SP1 memengaruhi hak pegawai dan tidak sebaiknya terbit tanpa peninjauan.
3. **Nomor surat.** Format nomor SP1 (mis. `SP1-<tahun>-<urutan>` per pegawai atau per unit)? Siapa yang memastikan nomornya unik?
4. **Tingkat peringatan.** Isi `tingkat` (mis. `SP1`, `SP2`, `SP3`) mengikuti repetisi (SP2 bila alpa berulang lagi)? Berapa tingkat maksimum yang relevan untuk kebijakan kepegawaian yayasan?
5. **Hubungan dengan slip.** Apakah SP1 yang sudah terbit memengaruhi slip (mis. potongan tambahan atau pengurangan hari kerja)? Jika ya, dasar dan besaran potongannya apa, dan apakah konsisten dengan komponen `ALPA` yang sudah ada?
6. **Hubungan dengan modul disiplin.** Apakah SP1 memicu modul mutasi/disiplin lain (mis. pengajuan sanksi)? Modul tersebut belum ada.
7. **Pemberitahuan.** Siapa yang diberi tahu (pegawai, atasan, HR) dan lewat kanal apa (email? aplikasi)? Apakah pegawai menerima salinan sebelum berlaku?
8. **Koreksi.** Bila keadaan berubah (mis. sakit terlanjur disetujui setelah SP1 terbit), apakah SP1 dapat dibatalkan/dicoret, dan bagaimana jejaknya?

## 3. Usulan alur (draft, belum disetujui)

```
1. Pipeline presensi menutup periode harian → rekap alpa per pegawai.
2. Kandidat SP1 = pegawai dengan alpa tanpa keterangan ≥ ambang, tanpa sakit/cuti/izin sah,
   dan belum punya SP1 terbuka untuk periode yang sama.
3. Kandidat masuk daftar "alpa suspects" (sudah ada) → admin meninjau dan membuat SP1.
4. Penyelesaian SP1 (backend, satu transaksi):
   - buat baris Peringatan (userId, tingkat, alasan, nomorSurat, tanggal, penerbit),
   - catat jejak audit `peringatan.sp1.created` (aktor, unit, target),
   - opsional: tandai slip periode berjalan sebagai terpotong (jika diputuskan pada butir 5).
5. SP1 yang sudah terbit tidak diedit; koreksi lewat pembatalan dengan alasan (butir 8).
```

## 4. Setelah keputusan

1. Catat keputusan di `docs/DECISIONS.md`.
2. Migrasi Goose bila perlu (mis. indeks unik nomor surat per penerbit/tahun, field alasan terstruktur).
3. Backend: endpoint `GET/POST /api/v1/peringatan` (daftar + terbitkan SP1), permission baru `peringatan:sp1:manage` (admin/kabiro), audit `peringatan.sp1.created`/`withdrawn`, integrasi ke mesin slip bila diputuskan.
4. Frontend: halaman `/admin/peringatan` (kandidat dari laporan alpa, form SP1, riwayat SP1 pegawai).
5. Verifikasi dinamis pada database disposable (alpa → kandidat → SP1 → audit), lalu pencatatan bukti.

**Selama keputusan belum ada:** tidak ada SP1 otomatis, tidak ada endpoint peringatan, dan komponen `ALPA` di slip tetap menjadi satu-satunya konsekuensi otomatis atas alpa.
