-- +goose Up
-- Draft only: select and approve a versioned migration runner before applying.
-- Jadwal tanggal gajian per unit organisasi (keputusan 11 November 2026):
-- admin menginput unit mana pun, operator hanya unit dalam cakupannya.
-- Tanggal gajian adalah tanggal tetap tiap bulan (1-28); pemberitahuan slip
-- gaji dikirim H+1 pada hari kalender berikutnya tanpa penyesuaian hari
-- libur. Pengaturan penggajian global (pengaturanGaji) digantikan tabel ini.
CREATE TABLE IF NOT EXISTS "jadwalGaji" (
    "id" uuid PRIMARY KEY,
    "unitId" uuid NOT NULL UNIQUE,
    "tanggalGajian" integer NOT NULL CHECK ("tanggalGajian" BETWEEN 1 AND 28),
    "createdAt" timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt" timestamp(3) NOT NULL DEFAULT now()
);

ALTER TABLE "jadwalGaji"
    ADD CONSTRAINT "jadwalGaji_unitId_fkey"
    FOREIGN KEY ("unitId") REFERENCES "Homebase" ("id");

-- Pengaturan penggajian global tidak lagi dipakai.
DROP TABLE IF EXISTS "pengaturanGaji";

-- Row Level Security default deny, sama seperti tabel aplikasi lainnya.
ALTER TABLE "jadwalGaji" ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS rl_admin_all_jadwal_gaji ON "jadwalGaji";
CREATE POLICY rl_admin_all_jadwal_gaji ON "jadwalGaji"
    FOR ALL TO PUBLIC
    USING (app.is_admin());

DROP POLICY IF EXISTS rl_jadwal_gaji_supervisor_scope ON "jadwalGaji";
CREATE POLICY rl_jadwal_gaji_supervisor_scope ON "jadwalGaji"
    FOR SELECT TO PUBLIC
    USING (
        app.is_supervisor()
        AND "unitId" = ANY(SELECT homebase_id FROM app.user_homebase_ids())
    );
