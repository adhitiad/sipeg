# Usulan Potongan Cooperasi & Pinjaman (cicilan/sisa) — menunggu keputusan pemilik

- **Status:** usulan desain/kontrak (belum diputuskan, **belum diimplementasikan**).
- **Rujukan:** `docs/PRD.md` §5.9 ("Potongan koperasi/pinjaman: jenis potongan tersedia di `komponenPotongan` (`KOPERASI`, `KAS_BON`); aturan cicilan/sisa pinjaman dan penagihan bulanan menunggu keputusan"), `DEC-067`, `DEC-071`.

## 1. Kondisi sekarang

| Aspek                    | Status                                                                                                                                                                       |
| ------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Master komponen potongan | Ada: `komponenPotongan` per pegawai (`jenis` `KOPERASI`/`KAS_BON`, `persentase`, `nominal`, `porsiInstitusi`, `porsiPegawai`, `aktif`) — dikelola admin di `/admin/potongan` |
| Pemotongan di slip       | Sudah dipakai mesin hitung slip (`internal/payroll`) sebagai komponen `POTONGAN`                                                                                             |
| Cicilan & sisa pinjaman  | **Belum ada** — tidak ada data pokok/cicilan/sisa, tidak ada tagihan bulanan, tidak ada pelunasan                                                                            |
| Aturan bisnis            | PRD §5.9: cicilan/sisa dan penagihan bulanan **menunggu keputusan**                                                                                                          |

## 2. Hal yang harus diputuskan lebih dulu

1. **Sumber nominal cicilan.** Apakah nominal bulanan berasal dari (a) komponen `KAS_BON`/`KOPERASI` yang sudah ada (nominal tetap), atau (b) data pinjaman terpisah dengan cicilan dan sisa yang berkurang tiap bulan?
2. **Pokok dan bunga.** Apakah ada bunga/jasa? Berapa persen, dan apakah bunga dihitung dari sisa (annuity) atau flat?
3. **Tanggal potong.** Tanggal berapa setiap bulan tagihan dipotong dari slip (mengikuti `jadwalGaji`)? Apa yang terjadi bila slip belum terbit atau terlambat?
4. **Prioritas potong.** Bila total potongan melebihi take-home pay, urutan potong mana yang mendahului (pinjaman, BPJS, PPh 21, dan lain-lain)? Apakah aturan prioritas ini perlu ditulis di modul penggajian?
5. **Pelunasan.** Apakah ada pelunasan dipercepat, dan bagaimana dicatat agar historis tetap benar?
6. **Potong gagal.** Apa yang terjadi bila pegawai tidak bisa dipotong (izin/cuti, take-home pay kecil)? Apakah tagihan ditunda ke periode berikutnya, dan apakah ada denda?
7. **Pinjaman dari luar sistem** (misalnya KUR atau koperasi yang/air terjadi di luar SIMPEG): dicatat sebagai komponen nominal tetap (tanpa cicilan/sisa), atau sebagai pinjaman dengan angsuran?

## 3. Usulan skema (draft, belum disetujui)

Bila keputusan diambil pada arah (b) — pinjaman dengan cicilan dan sisa:

```
potonganPinjaman
  id uuid pk
  userId uuid fk -> "user"
  jenisPotongan enum        -- KOPERASI | KAS_BON (pakai enum yang sudah ada)
  pokok numeric(14,2)      -- pokok pinjaman
  bungaPct numeric(7,4)    -- 0 bila flat
  cicilanBulan int          -- jumlah periode
  sisa numeric(14,2)       -- sisa pokok; berkurang saat tagihan dilunasi
  tanggalMulai date
  tanggalSelesai date
  status enum               -- AKTIF | LUNAS | DIBATALKAN
  createdAt/updatedAt timestamptz

potonganTagihan
  id uuid pk
  pinjamanId uuid fk -> potonganPinjaman
  periode text              -- "YYYY-MM" (ikut periode slip)
  nominal numeric(14,2)     -- nominal tagihan bulan ini
  status enum               -- TERJADIL | TERTAGIH | DIBATALKAN
  slipId uuid null fk       -- slip yang memotong tagihan ini (null bila belum)
  createdAt/updatedAt timestamptz
```

Ketentuan minimum yang perlu diputuskan sebelum implementasi:

- apakah tagihan wajib menempel pada slip, atau boleh dipotong di periode lain;
- apakah `komponenPotongan` yang ada tetap dipakai berdampingan, atau digantikan oleh `potonganPinjaman`;
- urutan potong terhadap PPh 21 dan BPJS, serta dasar pengurangannya;
- apakah ada penagihan di luar slip (misalnya pembayaran mandiri).

## 4. Setelah keputusan

1. Catat keputusan di `docs/DECISIONS.md`.
2. Migrasi Goose baru (tabel di atas + RLS default-deny seperti tabel lain).
3. Backend lapis `domain/app/repository/httpserver`: master pinjaman, generate tagihan bulanan, integrasi ke mesin slip (komponen `POTONGAN_PINJAMAN`), audit `potongan.pinjaman.created/updated` dan `potongan.tagihan.generated`.
4. Frontend: halaman `/admin/pinjaman` (daftar pinjaman, status cicilan, riwayat tagihan).
5. Verifikasi dinamis pada database disposable; laporan sisa pinjaman per pegawai.

**Selama keputusan belum ada:** tidak ada kode, migrasi, atau perubahan perhitungan slip untuk cicilan/sisa/tagihan. Yang berjalan sekarang hanyalah komponen `KOPERASI`/`KAS_BON` dengan nominal tetap.
