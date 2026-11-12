-- +goose Up
-- Menambahkan jenis dokumen baru dan kolom metadata berkas pada tabel
-- "arsipDokumenPegawai".
--
-- Jenis dokumen baru sesuai kebutuhan dokumen pegawai/dosen:
--   - FOTO               foto pegawai (png, jpg, jpeg, webp, maks 3.1 MB)
--   - IJAHAN_TERAKHIR    ijazah terakhir (pdf, maks 5.1 MB)
--   - SK_KEPEGAWAIAN_KONTRAK  SK kepegawaian / kontrak (pdf, maks 7.1 MB)
--   - BPJS_KETENAGAKERJAAN  kartu BPJS Ketenagakerjaan (png, jpg, jpeg, webp, pdf, maks 1.1 MB)
--   - BPJS_KESEHATAN     kartu BPJS Kesehatan (png, jpg, jpeg, webp, pdf, maks 1.1 MB)
--   - SERTIFIKAT_SERDOS  sertifikat serdos (pdf, maks 5.1 MB)
--
-- Kolom metadata berkas ditambahkan agar backend dapat memvalidasi ukuran
-- dan tipe berkas tanpa menangani isi berkas itu sendiri (berkas disimpan
-- di Cloudinary, backend hanya menyimpan URL + metadata).
ALTER TYPE "JenisDokumenArsip" ADD VALUE 'FOTO';
ALTER TYPE "JenisDokumenArsip" ADD VALUE 'IJAHAN_TERAKHIR';
ALTER TYPE "JenisDokumenArsip" ADD VALUE 'SK_KEPEGAWAIAN_KONTRAK';
ALTER TYPE "JenisDokumenArsip" ADD VALUE 'BPJS_KETENAGAKERJAAN';
ALTER TYPE "JenisDokumenArsip" ADD VALUE 'BPJS_KESEHATAN';
ALTER TYPE "JenisDokumenArsip" ADD VALUE 'SERTIFIKAT_SERDOS';

ALTER TABLE "arsipDokumenPegawai" ADD COLUMN IF NOT EXISTS "fileSize" bigint;
ALTER TABLE "arsipDokumenPegawai" ADD COLUMN IF NOT EXISTS "mimeType" text;
ALTER TABLE "arsipDokumenPegawai" ADD COLUMN IF NOT EXISTS "cloudinaryPublicId" text;

-- +goose Down
ALTER TABLE "arsipDokumenPegawai" DROP COLUMN IF EXISTS "cloudinaryPublicId";
ALTER TABLE "arsipDokumenPegawai" DROP COLUMN IF EXISTS "mimeType";
ALTER TABLE "arsipDokumenPegawai" DROP COLUMN IF EXISTS "fileSize";
-- Catatan: enum value yang sudah ditambahkan tidak dapat dihapus dari
-- tipe enumerasi PostgreSQL. Down migration hanya menghapus kolom metadata.
