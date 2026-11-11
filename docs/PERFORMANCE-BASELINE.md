# Baseline Performa dan Kapasitas SIMPEG

Tanggal: 2026-11-11
Status: Baseline awal dari pengukuran unit-level (fake store, tanpa database nyata). Angka ini adalah titik acuan untuk menetapkan SLO setelah ada database staging yang mewakili production.

## 1. Metode Pengukuran

Pengukuran dilakukan dengan `go test -bench=Benchmark -benchmem` pada paket `backend/internal/httpserver`. Tiga benchmark yang diukur:

- `BenchmarkListEmployeesConversion` — konversi DTO daftar pegawai (100 data, pageSize=100)
- `BenchmarkAuditLogDTOConversion` — konversi DTO jejak audit (100 data)
- `BenchmarkUnitScopeResolution` — resolusi cakupan unit untuk operator/atasan

Semua benchmark menggunakan fake store, bukan PostgreSQL nyata. Oleh karena itu, angka meluas biaya konversi DTO dan logika aplikasi, **bukan** latensi query database atau jaringan penuh.

## 2. Hasil Baseline

| Benchmark | Latency (ns/op) | Latency (ms/op) | Memory (B/op) | Allocs (op) |
|-----------|-----------------|-----------------|---------------|-------------|
| ListEmployeesConversion | 127,100 | ~0.127 | 55,891 | 48 |
| AuditLogDTOConversion | 65,923 | ~0.066 | 34,671 | 51 |
| UnitScopeResolution | 12,986 | ~0.013 | 8,515 | 39 |

## 3. Workload Realistis (Estimasi Awal)

Estimasi berikut adalah asumsi awal untuk perencanaan kapasitas, bukan komitmen. Harus diverifikasi setelah peluncuran pilot dengan data nyata.

| Parameter | Estimasi Awal | Keterangan |
|-----------|--------------|------------|
| Pengguna bersamaan (pilot) | 10–25 | Admin, operator dari satu fakultas dan satu biro |
| Data pegawai | ~1.000–5.000 baris | Universitas Subang, satu fakultas + satu biro |
| Pola pencarian | Baca-dominan (90% baca, 10% tulis) | Daftar/cari pegawai lebih sering dari pembuatan profil |
| Impor/ekspor | Bulanan, <1.000 baris per batch | Impor massal pegawai oleh operator |
| Login | Beberapa kali per hari per pengguna | Sesi berlaku 24 jam (default DEC-029) |
| Burst notifikasi | H+1 setelah tanggal gajian | Email slip tersedia dikirim ke semua pegawai dalam unit |

## 4. Rekomendasi SLO Awal (Menunggu Validasi)

Angka di bawah adalah usulan awal yang harus ditinjul ulang setelah baseline nyata dari database staging:

| Target | Nilai Usulan | Keterangan |
|--------|--------------|------------|
| p95 latency daftar pegawai | < 200 ms | Termasuk konversi DTO + query database |
| p99 latency daftar pegawai | < 500 ms | Bila database dan jaringan sehat |
| p95 latency detail pegawai | < 100 ms | Satu record, lebih sederhana dari daftar |
| Throughput daftar pegawai | > 50 req/detik | Berdasarkan 0.127 ms overhead DTO saja |
| Error rate | < 1% | Kesalahan klien (4xx) tidak dihitung |
| Ketersediaan (SLO) | 99.9% | Sesuai `SLO_TARGET` di env |

## 5. Keterbatasan Baseline

- Tidak termasuk latensi query PostgreSQL (belum ada database staging)
- Tidak termasuk latensi jaringan antara frontend dan backend
- Tidak termasuk beban Redis, QStash, atau SMTP
- Tidak mencakup impor/ekspor berkas Excel (menggunakan pustaka eksternal, lebih mahal)

## 6. Langkah Selanjutnya

1. Siapkan database staging yang mirip production
2. Jalankan benchmark dengan database nyata untuk mengukur latensi query
3. Ukur beban impor/ekspor Excel dengan data sintetis
4. Tetapkan SLO final berdasarkan hasil pengukuran nyata
5. Konfigurasikan alert untuk error rate dan latensi di atas ambang

## Referensi

- `backend/internal/httpserver/benchmark_test.go` — kode benchmark
- `docs/DECISIONS.md` DEC-020 — target ketersediaan, RPO, RTO, dan SLO (belum disetujui final)
- `.env.example` — `SLO_TARGET=99.9`, `RPO_HOURS=24`, `RTO_HOURS=4`
