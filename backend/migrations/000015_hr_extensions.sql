-- +goose Up
-- Extensi data kepegawaian untuk SIMPEG Universitas (HR extensions).
-- Tabel-tabel berikut mencatat keluarga/tanggungan, riwayat golongan,
-- komponen tunjangan & potongan, arsip dokumen, sertifikasi, pelatihan,
-- pemetaan ID eksternal SISTER/PDDIKTI, serta riwayat penghargaan.

-- ═════════════════════════════════ Enum types ═════════════════════════════════

-- +goose StatementBegin
DO $$
BEGIN
    CREATE TYPE "JenisHubunganKeluarga" AS ENUM ('SUAMI_ISTRI', 'ANAK');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
BEGIN
    CREATE TYPE "StatusTanggungan" AS ENUM ('TANGGUNGAN', 'BUKAN_TANGGUNGAN');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
BEGIN
    CREATE TYPE "JenisTunjangan" AS ENUM ('STRUKTURAL', 'FUNGSIONAL', 'BERAS', 'MAKANAN', 'TRANSPORTASI');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
BEGIN
    CREATE TYPE "JenisPotongan" AS ENUM ('PPH21', 'BPJS_KESEHATAN', 'BPJS_KETENAGAKERJAAN', 'KAS BON', 'KOPERASI');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
BEGIN
    CREATE Type "JenisDokumenArsip" AS ENUM ('KTP', 'NPWP', 'KARTU_KELUARGA', 'REKENING_BANK', 'SK_PENGANGKATAN_YAYASAN', 'SK_MENDIKBUD', 'PAKTA_INTEGRITAS', 'LAINNYA');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
BEGIN
    CREATE TYPE "JenisSertifikasi" AS ENUM ('SERDOS', 'PROFESI', 'KOMPETENSI', 'LAINNYA');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
BEGIN
    CREATE TYPE "StatusSertifikasi" AS ENUM ('AKTIF', 'KEDALUWARSA');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
BEGIN
    CREATE TYPE "JenisPelatihan" AS ENUM ('DIKLAT', 'LOKAKARYA', 'SOFT_SKILLS', 'LAINNYA');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
BEGIN
    CREATE TYPE "SistemEksternal" AS ENUM ('SISTER', 'PDDIKTI', 'LAINNYA');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
BEGIN
    CREATE TYPE "StatusSinkronisasi" AS ENUM ('AKTIF', 'NONAKTIF');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
BEGIN
    CREATE TYPE "JenisPenghargaan" AS ENUM ('DOSEN_TELOHAN', 'SATYALANCANA', 'LAINNYA');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- ═════════════════════════════════ Tabel baru ═══════════════════════════════════

-- Keluarga & tanggungan pegawai (Suami/Istri dan Anak) untuk BPJS dan
-- Tunjangan Keluarga/Anak.
CREATE TABLE IF NOT EXISTS "keluargaPegawai" (
    "id"             uuid PRIMARY KEY,
    "userId"         uuid NOT NULL REFERENCES "user" ("id") ON DELETE CASCADE,
    "hubungan"       "JenisHubunganKeluarga" NOT NULL,
    "nama"           text NOT NULL,
    "nik"            text NOT NULL,
    "tanggalLahir"   date NOT NULL,
    "pekerjaan"      text,
    "statusTanggungan" "StatusTanggungan" NOT NULL DEFAULT 'TANGGUNGAN',
    "createdAt"      timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"      timestamp(3) NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS keluarga_pegawai_user_idx ON "keluargaPegawai" ("userId");
CREATE INDEX IF NOT EXISTS keluarga_pegawai_nik_idx ON "keluargaPegawai" ("nik");

-- Riwayat golongan/pangkat pegawai (III/b, IV/a, dst.) dengan TMT dan
-- Angka Kredit sebagai penentu standar gaji pokok.
CREATE TABLE IF NOT EXISTS "riwayatGolongan" (
    "id"          uuid PRIMARY KEY,
    "userId"      uuid NOT NULL REFERENCES "user" ("id") ON DELETE CASCADE,
    "golongan"    text NOT NULL,
    "ruang"       text NOT NULL,
    "tmt"         date NOT NULL,
    "angkaKredit" integer NOT NULL DEFAULT 0,
    "nomorSk"     text,
    "createdAt"   timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"   timestamp(3) NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS riwayat_golongan_user_idx ON "riwayatGolongan" ("userId");
CREATE INDEX IF NOT EXISTS riwayat_golongan_tmt_idx ON "riwayatGolongan" ("tmt");

-- Komponen tunjangan per pegawai (struktural, fungsional, beras, makan,
-- transportasi) sebagai variabel penentu nominal tunjangan.
CREATE TABLE IF NOT EXISTS "komponenTunjangan" (
    "id"           uuid PRIMARY KEY,
    "userId"       uuid NOT NULL REFERENCES "user" ("id") ON DELETE CASCADE,
    "jenisTunjangan" "JenisTunjangan" NOT NULL,
    "nilai"        double precision NOT NULL DEFAULT 0,
    "aktif"        boolean NOT NULL DEFAULT true,
    "createdAt"    timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"    timestamp(3) NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS komponen_tunjangan_user_idx ON "komponenTunjangan" ("userId");
CREATE INDEX IF NOT EXISTS komponen_tunjangan_jenis_idx ON "komponenTunjangan" ("jenisTunjangan");

-- Komponen potongan per pegawai (PPh 21, BPJS, kasbon, koperasi) sebagai
-- variabel penentu nominal potongan.
CREATE TABLE IF NOT EXISTS "komponenPotongan" (
    "id"             uuid PRIMARY KEY,
    "userId"         uuid NOT NULL REFERENCES "user" ("id") ON DELETE CASCADE,
    "jenisPotongan"  "JenisPotongan" NOT NULL,
    "persentase"     double precision,
    "nominal"        double precision NOT NULL DEFAULT 0,
    "porsiInstitusi" double precision,
    "porsiPegawai"   double precision,
    "aktif"          boolean NOT NULL DEFAULT true,
    "createdAt"      timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"      timestamp(3) NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS komponen_potongan_user_idx ON "komponenPotongan" ("userId");
CREATE INDEX IF NOT EXISTS komponen_potongan_jenis_idx ON "komponenPotongan" ("jenisPotongan");

-- Arsip dokumen pegawai digital (KTP, NPWP, KK, rekening, SK, pakta integritas,
-- dst.) dengan URL cloud storage.
CREATE TABLE IF NOT EXISTS "arsipDokumenPegawai" (
    "id"             uuid PRIMARY KEY,
    "userId"         uuid NOT NULL REFERENCES "user" ("id") ON DELETE CASCADE,
    "jenisDokumen"   "JenisDokumenArsip" NOT NULL,
    "judul"          text NOT NULL,
    "fileUrl"        text NOT NULL,
    "keterangan"     text,
    "berlakuSampai"  date,
    "createdAt"      timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"      timestamp(3) NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS arsip_dokumen_pegawai_user_idx ON "arsipDokumenPegawai" ("userId");
CREATE INDEX IF NOT EXISTS arsip_dokumen_pegawai_jenis_idx ON "arsipDokumenPegawai" ("jenisDokumen");

-- Riwayat sertifikasi pegawai (Serdos, profesi, kompetensi) dengan nomor
-- registrasi dan masa berlaku.
CREATE TABLE IF NOT EXISTS "riwayatSertifikasi" (
    "id"             uuid PRIMARY KEY,
    "userId"         uuid NOT NULL REFERENCES "user" ("id") ON DELETE CASCADE,
    "jenisSertifikasi" "JenisSertifikasi" NOT NULL,
    "namaSertifikat" text NOT NULL,
    "nomorRegistrasi" text,
    "penyelenggara"  text NOT NULL,
    "tanggalTerbit"  date NOT NULL,
    "berlakuSampai"  date,
    "status"         "StatusSertifikasi" NOT NULL DEFAULT 'AKTIF',
    "createdAt"      timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"      timestamp(3) NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS riwayat_sertifikasi_user_idx ON "riwayatSertifikasi" ("userId");
CREATE INDEX IF NOT EXISTS riwayat_sertifikasi_status_idx ON "riwayatSertifikasi" ("status");

-- Riwayat pelatihan pegawai (diklat, lokakarya, soft-skills).
CREATE TABLE IF NOT EXISTS "riwayatPelatihan" (
    "id"             uuid PRIMARY KEY,
    "userId"         uuid NOT NULL REFERENCES "user" ("id") ON DELETE CASCADE,
    "namaPelatihan"  text NOT NULL,
    "jenisPelatihan" "JenisPelatihan" NOT NULL,
    "penyelenggara"  text NOT NULL,
    "tanggalMulai"   date NOT NULL,
    "tanggalSelesai" date NOT NULL,
    "durasiJam"      integer NOT NULL DEFAULT 0,
    "memilikiSertifikat" boolean NOT NULL DEFAULT false,
    "createdAt"      timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"      timestamp(3) NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS riwayat_pelatihan_user_idx ON "riwayatPelatihan" ("userId");
CREATE INDEX IF NOT EXISTS riwayat_pelatihan_jenis_idx ON "riwayatPelatihan" ("jenisPelatihan");

-- Pemetaan ID eksternal (SISTER Kemdikbud, PDDIKTI Feeder) dengan id_pegawai
-- internal untuk mencegah duplikasi saat sinkronisasi dua arah.
CREATE TABLE IF NOT EXISTS "sisterSyncMap" (
    "id"                 uuid PRIMARY KEY,
    "userId"             uuid NOT NULL REFERENCES "user" ("id") ON DELETE CASCADE,
    "sistemEksternal"    "SistemEksternal" NOT NULL,
    "idEksternal"        text NOT NULL,
    "sinkronisasiTerakhir" timestamp(3),
    "status"             "StatusSinkronisasi" NOT NULL DEFAULT 'AKTIF',
    "createdAt"          timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"          timestamp(3) NOT NULL DEFAULT now(),
    CONSTRAINT "sister_sync_map_user_sistem_key" UNIQUE ("userId", "sistemEksternal")
);
CREATE INDEX IF NOT EXISTS sister_sync_map_user_idx ON "sisterSyncMap" ("userId");
CREATE INDEX IF NOT EXISTS sister_sync_map_id_eksternal_idx ON "sisterSyncMap" ("idEksternal");

-- Riwayat penghargaan pegawai (dosen teladan, satyalancana, dst.).
CREATE TABLE IF NOT EXISTS "riwayatPenghargaan" (
    "id"             uuid PRIMARY KEY,
    "userId"         uuid NOT NULL REFERENCES "user" ("id") ON DELETE CASCADE,
    "namaPenghargaan" text NOT NULL,
    "jenisPenghargaan" "JenisPenghargaan" NOT NULL,
    "penyelenggara"  text NOT NULL,
    "tanggalTerima"  date NOT NULL,
    "nomorSk"        text,
    "createdAt"      timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"      timestamp(3) NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS riwayat_penghargaan_user_idx ON "riwayatPenghargaan" ("userId");
CREATE INDEX IF NOT EXISTS riwayat_penghargaan_jenis_idx ON "riwayatPenghargaan" ("jenisPenghargaan");

-- Row Level Security default deny pada seluruh tabel baru, sesuai kebijakan
-- aplikasi. Backend (role pemilik tabel) tetap dapat diakses karena migrasi
-- tidak memakai FORCE.
ALTER TABLE "keluargaPegawai" ENABLE ROW LEVEL SECURITY;
ALTER TABLE "riwayatGolongan" ENABLE ROW LEVEL SECURITY;
ALTER TABLE "komponenTunjangan" ENABLE ROW LEVEL SECURITY;
ALTER TABLE "komponenPotongan" ENABLE ROW LEVEL SECURITY;
ALTER TABLE "arsipDokumenPegawai" ENABLE ROW LEVEL SECURITY;
ALTER TABLE "riwayatSertifikasi" ENABLE ROW LEVEL SECURITY;
ALTER TABLE "riwayatPelatihan" ENABLE ROW LEVEL SECURITY;
ALTER TABLE "sisterSyncMap" ENABLE ROW LEVEL SECURITY;
ALTER TABLE "riwayatPenghargaan" ENABLE ROW LEVEL SECURITY;

-- +goose Down
DROP TABLE IF EXISTS "riwayatPenghargaan" CASCADE;
DROP TABLE IF EXISTS "sisterSyncMap" CASCADE;
DROP TABLE IF EXISTS "riwayatPelatihan" CASCADE;
DROP TABLE IF EXISTS "riwayatSertifikasi" CASCADE;
DROP TABLE IF EXISTS "arsipDokumenPegawai" CASCADE;
DROP TABLE IF EXISTS "komponenPotongan" CASCADE;
DROP TABLE IF EXISTS "komponenTunjangan" CASCADE;
DROP TABLE IF EXISTS "riwayatGolongan" CASCADE;
DROP TABLE IF EXISTS "keluargaPegawai" CASCADE;
DROP TYPE IF EXISTS "JenisPenghargaan";
DROP TYPE IF EXISTS "StatusSinkronisasi";
DROP TYPE IF EXISTS "SistemEksternal";
DROP TYPE IF EXISTS "JenisPelatihan";
DROP TYPE IF EXISTS "StatusSertifikasi";
DROP TYPE IF EXISTS "JenisSertifikasi";
DROP TYPE IF EXISTS "JenisDokumenArsip";
DROP Type IF EXISTS "JenisPotongan";
DROP Type IF EXISTS "JenisTunjangan";
DROP Type IF EXISTS "StatusTanggungan";
DROP Type IF EXISTS "JenisHubunganKeluarga";
