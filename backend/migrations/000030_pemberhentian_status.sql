-- Status persetujuan untuk pengajuan pemberhentian: MENUNGGU | DISETUJUI |
-- DITOLAK. Kolom alasan (sub-alasan) dan kompensasiBulan dipakai pada
-- alur pengajuan + persetujuan (`DEC-083`).

-- +goose Up
ALTER TABLE "Pemberhentian" ADD COLUMN IF NOT EXISTS "status" "StatusPersetujuan" NOT NULL DEFAULT 'MENUNGGU';
ALTER TABLE "Pemberhentian" ADD COLUMN IF NOT EXISTS "alasan" text;
ALTER TABLE "Pemberhentian" ADD COLUMN IF NOT EXISTS "kompensasiBulan" double precision NOT NULL DEFAULT 0;
CREATE INDEX IF NOT EXISTS pemberhentian_status_idx ON "Pemberhentian" ("status");

-- +goose Down
DROP INDEX IF EXISTS pemberhentian_status_idx;
ALTER TABLE "Pemberhentian" DROP COLUMN IF EXISTS "kompensasiBulan";
ALTER TABLE "Pemberhentian" DROP COLUMN IF EXISTS "alasan";
ALTER TABLE "Pemberhentian" DROP COLUMN IF EXISTS "status";
