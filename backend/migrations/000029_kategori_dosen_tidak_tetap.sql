-- Menambahkan kategori status pegawai DOSEN_TIDAK_TETAP (DEC-083):
-- dosen S3/Profesor yang memasuki BUP dapat diperpanjang masa kerjanya per
-- 5 tahun melalui perjanjian kerja tahunan dan ditetapkan sebagai dosen
-- tidak tetap. `gajiDariNegara=false`, `butuhKontrak=true` (wajib perjanjian
-- tahunan), `wajibBpjs=true`; `bupSesuaiYayasan=true` (batas usia tetap
-- mengikuti yayasan/konfigurasi, bukan BUP umum). `JenisSk` sudah memuat
-- `PHK` — tidak perlu penambahan enum.

-- +goose Up
INSERT INTO "kategoriStatusPegawai" ("id", "kode", "nama", "deskripsi", "gajiDariNegara", "butuhKontrak", "wajibBpjs", "bupSesuaiYayasan") VALUES
  ('d0000000-0000-0000-0000-000000000006', 'DOSEN_TIDAK_TETAP', 'Dosen tidak tetap (pasca-BUP)',
   'Dosen S3/Profesor pasca-BUP yang diperpanjang sebagai dosen tidak tetap; honor per SKS; perjanjian kerja tahunan.',
   false, true, true, true)
ON CONFLICT ("kode") DO NOTHING;

-- +goose Down
DELETE FROM "kategoriStatusPegawai"
WHERE "kode" = 'DOSEN_TIDAK_TETAP'
  AND NOT EXISTS (SELECT 1 FROM "user" WHERE "kategoriStatusId" = 'd0000000-0000-0000-0000-000000000006');
