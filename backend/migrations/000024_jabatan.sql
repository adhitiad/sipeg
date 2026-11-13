-- +goose Up
-- Tabel referensi jabatan untuk SIMPEG (data kepegawaian inti).
--
-- Jabatan adalah entitas master: nama jabatan dan jenisnya (struktural atau
-- fungsional). Penempatan pegawai pada jabatan tetap berupa teks pada kolom
-- user.jabatanStruktural / user.jabatanFungsional sampai ada keputusan
-- produk untuk memindahkan penempatan ke referensi ini; migrasi ini hanya
-- memperkenalkan master jabatan dan CRUD-nya.
--
-- Row Level Security default deny selaras dengan tabel aplikasi lain; tidak
-- ada policy khusus karena jabatan adalah data referensi.
-- +goose StatementBegin
DO $$
BEGIN
    CREATE TYPE "JabatanJenis" AS ENUM ('STRUKTURAL', 'FUNGSIONAL');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

CREATE TABLE IF NOT EXISTS "Jabatan" (
    "id"        uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    "nama"      text NOT NULL UNIQUE,
    "jenis"     "JabatanJenis" NOT NULL DEFAULT 'STRUKTURAL',
    "aktif"     boolean NOT NULL DEFAULT true,
    "createdAt" timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt" timestamp(3) NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS "jabatan_jenis_idx" ON "Jabatan" ("jenis");

ALTER TABLE "Jabatan" ENABLE ROW LEVEL SECURITY;

-- +goose Down
DROP TABLE IF EXISTS "Jabatan";
DROP TYPE IF EXISTS "JabatanJenis";
