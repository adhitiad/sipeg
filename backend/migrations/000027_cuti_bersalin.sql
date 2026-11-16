-- +goose Up
-- Sub-jenis cuti bersalin: MELAHIRKAN (3 bulan) atau KEGUGURAN (1,5 bulan).
-- Kolom kosong pada baris lama; aplikasi mewajibkan sub-jenis untuk BERSALIN baru.
ALTER TABLE "Cuti" ADD COLUMN IF NOT EXISTS "subJenisBersalin" text;

-- +goose Down
ALTER TABLE "Cuti" DROP COLUMN IF EXISTS "subJenisBersalin";