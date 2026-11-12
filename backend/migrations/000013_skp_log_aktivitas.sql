-- +goose Up
-- Tabel untuk modul SKP & Log Aktivitas (Tendik/non-dosen).

-- ═══════════════════════════ Enum types ═══════════════════════════

-- +goose StatementBegin
DO $$
BEGIN
    CREATE TYPE "StatusSKP" AS ENUM ('DRAFT', 'AKTIF', 'TUTUP');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
BEGIN
    CREATE TYPE "StatusSKPItem" AS ENUM ('DRAFT', 'DIAJUKAN', 'DISETUJUI', 'DITOLAK');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- ═══════════════════════════ Tabel SKP ═══════════════════════════

CREATE TABLE IF NOT EXISTS "skpPeriode" (
    "id"             uuid PRIMARY KEY,
    "tahun"          integer NOT NULL,
    "semester"       integer NOT NULL,
    "tanggalMulai"   date NOT NULL,
    "tanggalSelesai" date NOT NULL,
    "status"         "StatusSKP" NOT NULL DEFAULT 'DRAFT',
    "aktif"          boolean NOT NULL DEFAULT true,
    "createdAt"      timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"      timestamp(3) NOT NULL DEFAULT now(),
    CONSTRAINT "skp_periode_tahun_semester_key" UNIQUE ("tahun", "semester")
);
CREATE INDEX IF NOT EXISTS "skp_periode_status_idx" ON "skpPeriode" ("status");

CREATE TABLE IF NOT EXISTS "skpTargetKinerja" (
    "id"            uuid PRIMARY KEY,
    "userId"        uuid NOT NULL REFERENCES "user" ("id") ON DELETE CASCADE,
    "skpPeriodeId"  uuid NOT NULL REFERENCES "skpPeriode" ("id") ON DELETE CASCADE,
    "uraian"        text NOT NULL,
    "target"        double precision NOT NULL,
    "realisasi"     double precision NOT NULL DEFAULT 0,
    "bobot"         double precision NOT NULL DEFAULT 1,
    "status"        "StatusSKPItem" NOT NULL DEFAULT 'DRAFT',
    "dinilaiOleh"   uuid,
    "createdAt"     timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"     timestamp(3) NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS "skp_target_user_periode_idx" ON "skpTargetKinerja" ("userId", "skpPeriodeId");

-- ═══════════════════════════ Tabel Log Aktivitas ═══════════════════════════

CREATE TABLE IF NOT EXISTS "logAktivitasHarian" (
    "id"           uuid PRIMARY KEY,
    "userId"       uuid NOT NULL REFERENCES "user" ("id") ON DELETE CASCADE,
    "tanggal"      date NOT NULL,
    "aktivitas"    text NOT NULL,
    "durasiMenit"  integer NOT NULL DEFAULT 0,
    "keterangan"   text,
    "createdAt"    timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"    timestamp(3) NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS "log_aktivitas_user_tanggal_idx" ON "logAktivitasHarian" ("userId", "tanggal");

-- ═══════════════════════════ Tabel Penilaian Kinerja Akhir ═══════════════════════════

CREATE TABLE IF NOT EXISTS "penilaianKinerjaAkhir" (
    "id"                uuid PRIMARY KEY,
    "userId"            uuid NOT NULL REFERENCES "user" ("id") ON DELETE CASCADE,
    "skpPeriodeId"      uuid NOT NULL REFERENCES "skpPeriode" ("id") ON DELETE CASCADE,
    "nilaiAkhir"        double precision NOT NULL,
    "predikat"          text NOT NULL,
    "catatan"           text,
    "dinilaiOleh"       uuid,
    "tanggalPenilaian"  date NOT NULL,
    "createdAt"         timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"         timestamp(3) NOT NULL DEFAULT now(),
    CONSTRAINT "penilaian_user_periode_key" UNIQUE ("userId", "skpPeriodeId")
);

ALTER TABLE "skpPeriode" ENABLE ROW LEVEL SECURITY;
ALTER TABLE "skpTargetKinerja" ENABLE ROW LEVEL SECURITY;
ALTER TABLE "logAktivitasHarian" ENABLE ROW LEVEL SECURITY;
ALTER TABLE "penilaianKinerjaAkhir" ENABLE ROW LEVEL SECURITY;

-- +goose Down
DROP TABLE IF EXISTS "penilaianKinerjaAkhir" CASCADE;
DROP TABLE IF EXISTS "logAktivitasHarian" CASCADE;
DROP TABLE IF EXISTS "skpTargetKinerja" CASCADE;
DROP TABLE IF EXISTS "skpPeriode" CASCADE;
DROP TYPE IF EXISTS "StatusSKPItem";
DROP TYPE IF EXISTS "StatusSKP";
