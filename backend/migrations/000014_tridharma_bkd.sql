-- +goose Up
-- Tabel untuk modul Tridharma & BKD (Dosen only).

-- ═════════════════════════════ Enum types ═══════════════════════════

-- +goose StatementBegin
DO $$
BEGIN
    CREATE TYPE "StatusPeriodeAkademik" AS ENUM ('DRAFT', 'AKTIF', 'TUTUP');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
BEGIN
    CREATE TYPE "StatusTridharmaItem" AS ENUM ('DRAFT', 'DIAJUKAN', 'DISETUJUI', 'DITOLAK');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- ═══════════════════════════ Tabel Periode Akademik ═══════════════════════════

CREATE TABLE IF NOT EXISTS "periodeAkademik" (
    "id"             uuid PRIMARY KEY,
    "tahun"          integer NOT NULL,
    "semester"       integer NOT NULL,
    "tanggalMulai"   date NOT NULL,
    "tanggalSelesai" date NOT NULL,
    "status"         "StatusPeriodeAkademik" NOT NULL DEFAULT 'DRAFT',
    "aktif"          boolean NOT NULL DEFAULT true,
    "createdAt"      timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"      timestamp(3) NOT NULL DEFAULT now(),
    CONSTRAINT "periode_akademik_tahun_semester_key" UNIQUE ("tahun", "semester")
);
CREATE INDEX IF NOT EXISTS "periode_akademik_status_idx" ON "periodeAkademik" ("status");

-- ═══════════════════════════ Tabel Evaluasi BKD ══════════════════════════════

CREATE TABLE IF NOT EXISTS "evaluasiBkd" (
    "id"                uuid PRIMARY KEY,
    "userId"            uuid NOT NULL REFERENCES "user" ("id") ON DELETE CASCADE,
    "periodeAkademikId" uuid NOT NULL REFERENCES "periodeAkademik" ("id") ON DELETE CASCADE,
    "nilai"             double precision NOT NULL,
    "predikat"          text NOT NULL,
    "catatan"           text,
    "dinilaiOleh"       uuid,
    "tanggalEvaluasi"   date NOT NULL,
    "createdAt"         timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"         timestamp(3) NOT NULL DEFAULT now(),
    CONSTRAINT "evaluasi_bkd_user_periode_key" UNIQUE ("userId", "periodeAkademikId")
);

-- ═══════════════════════════ Tabel Tridharma Pendidikan ═══════════════════════

CREATE TABLE IF NOT EXISTS "tridharmaPendidikan" (
    "id"                uuid PRIMARY KEY,
    "userId"            uuid NOT NULL REFERENCES "user" ("id") ON DELETE CASCADE,
    "periodeAkademikId" uuid NOT NULL REFERENCES "periodeAkademik" ("id") ON DELETE CASCADE,
    "uraian"            text NOT NULL,
    "target"            double precision NOT NULL,
    "realisasi"         double precision NOT NULL DEFAULT 0,
    "bobot"             double precision NOT NULL DEFAULT 1,
    "status"            "StatusTridharmaItem" NOT NULL DEFAULT 'DRAFT',
    "dinilaiOleh"       uuid,
    "createdAt"         timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"         timestamp(3) NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS "tridharma_pendidikan_user_periode_idx" ON "tridharmaPendidikan" ("userId", "periodeAkademikId");

-- ═══════════════════════════ Tabel Tridharma Penelitian ════════════════════════

CREATE TABLE IF NOT EXISTS "tridharmaPenelitian" (
    "id"                uuid PRIMARY KEY,
    "userId"            uuid NOT NULL REFERENCES "user" ("id") ON DELETE CASCADE,
    "periodeAkademikId" uuid NOT NULL REFERENCES "periodeAkademik" ("id") ON DELETE CASCADE,
    "judul"             text NOT NULL,
    "jenisPenelitian"   text NOT NULL,
    "target"            double precision NOT NULL,
    "realisasi"         double precision NOT NULL DEFAULT 0,
    "bobot"             double precision NOT NULL DEFAULT 1,
    "status"            "StatusTridharmaItem" NOT NULL DEFAULT 'DRAFT',
    "dinilaiOleh"       uuid,
    "createdAt"         timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"         timestamp(3) NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS "tridharma_penelitian_user_periode_idx" ON "tridharmaPenelitian" ("userId", "periodeAkademikId");

-- ═══════════════════════════ Tabel Tridharma Pengabdian ════════════════════════

CREATE TABLE IF NOT EXISTS "tridharmaPengabdian" (
    "id"                uuid PRIMARY KEY,
    "userId"            uuid NOT NULL REFERENCES "user" ("id") ON DELETE CASCADE,
    "periodeAkademikId" uuid NOT NULL REFERENCES "periodeAkademik" ("id") ON DELETE CASCADE,
    "judul"             text NOT NULL,
    "jenisPengabdian"   text NOT NULL,
    "target"            double precision NOT NULL,
    "realisasi"         double precision NOT NULL DEFAULT 0,
    "bobot"             double precision NOT NULL DEFAULT 1,
    "status"            "StatusTridharmaItem" NOT NULL DEFAULT 'DRAFT',
    "dinilaiOleh"       uuid,
    "createdAt"         timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"         timestamp(3) NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS "tridharma_pengabdian_user_periode_idx" ON "tridharmaPengabdian" ("userId", "periodeAkademikId");

-- ═══════════════════════════ Tabel Tridharma Penunjang ══════════════════════════

CREATE TABLE IF NOT EXISTS "tridharmaPenunjang" (
    "id"                uuid PRIMARY KEY,
    "userId"            uuid NOT NULL REFERENCES "user" ("id") ON DELETE CASCADE,
    "periodeAkademikId" uuid NOT NULL REFERENCES "periodeAkademik" ("id") ON DELETE CASCADE,
    "uraian"            text NOT NULL,
    "target"            double precision NOT NULL,
    "realisasi"         double precision NOT NULL DEFAULT 0,
    "bobot"             double precision NOT NULL DEFAULT 1,
    "status"            "StatusTridharmaItem" NOT NULL DEFAULT 'DRAFT',
    "dinilaiOleh"       uuid,
    "createdAt"         timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"         timestamp(3) NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS "tridharma_penunjang_user_periode_idx" ON "tridharmaPenunjang" ("userId", "periodeAkademikId");

ALTER TABLE "periodeAkademik" ENABLE ROW LEVEL SECURITY;
ALTER TABLE "evaluasiBkd" ENABLE ROW LEVEL SECURITY;
ALTER TABLE "tridharmaPendidikan" ENABLE ROW LEVEL SECURITY;
ALTER TABLE "tridharmaPenelitian" ENABLE ROW LEVEL SECURITY;
ALTER TABLE "tridharmaPengabdian" ENABLE ROW LEVEL SECURITY;
ALTER TABLE "tridharmaPenunjang" ENABLE ROW LEVEL SECURITY;

-- +goose Down
DROP TABLE IF EXISTS "tridharmaPenunjang" CASCADE;
DROP TABLE IF EXISTS "tridharmaPengabdian" CASCADE;
DROP TABLE IF EXISTS "tridharmaPenelitian" CASCADE;
DROP TABLE IF EXISTS "tridharmaPendidikan" CASCADE;
DROP TABLE IF EXISTS "evaluasiBkd" CASCADE;
DROP TABLE IF EXISTS "periodeAkademik" CASCADE;
DROP TYPE IF EXISTS "StatusTridharmaItem";
DROP TYPE IF EXISTS "StatusPeriodeAkademik";
