-- +goose Up
-- Menambahkan metadata eviden (bukti) pada pengajuan cuti.
--
-- Cuti melahirkan, cuti sakit, dan cuti harian (izin penting) wajib
-- menyertakan bukti. Bentuk bukti dibedakan agar approver dapat menilai
-- kecukupannya:
--   - SURAT_DOKTER  surat dokter, wajib untuk cuti sakit dan melahirkan.
--                    Nomor dan tanggal surat memakai kolom yang sudah ada.
--   - FOTO          foto atau tangkapan layar bertimestamp, untuk cuti harian.
--   - LAINNYA       bentuk bukti lain, wajib disertai waktu dan keterangan.
--
-- Kolom "fileSuratUrl" diubah nama menjadi "fileEvidenUrl" karena kini
-- menunjuk berkas bukti untuk semua jenis eviden, bukan hanya surat. Nama
-- lama belum pernah dipakai endpoint mana pun sehingga tidak ada data yang
-- kehilangan penanda.
CREATE TYPE "JenisEviden" AS ENUM ('SURAT_DOKTER', 'FOTO', 'LAINNYA');

ALTER TABLE "Cuti" RENAME COLUMN "fileSuratUrl" TO "fileEvidenUrl";
ALTER TABLE "Cuti" ADD COLUMN IF NOT EXISTS "jenisEviden" "JenisEviden";
ALTER TABLE "Cuti" ADD COLUMN IF NOT EXISTS "waktuEviden" timestamp(3);
ALTER TABLE "Cuti" ADD COLUMN IF NOT EXISTS "catatanEviden" text;

-- +goose Down
ALTER TABLE "Cuti" DROP COLUMN IF EXISTS "catatanEviden";
ALTER TABLE "Cuti" DROP COLUMN IF EXISTS "waktuEviden";
ALTER TABLE "Cuti" DROP COLUMN IF EXISTS "jenisEviden";
ALTER TABLE "Cuti" RENAME COLUMN "fileEvidenUrl" TO "fileSuratUrl";
DROP TYPE IF EXISTS "JenisEviden";
