-- Draft only: select and approve a versioned migration runner before applying.
-- Kalender hari libur nasional Indonesia (keputusan 11 November 2026, DEC-028):
-- tabel referensi yang diisi dari penyedia api.co.id oleh admin
-- (POST /api/v1/holidays/{year}/refresh) dan dibaca selalu dari tabel lokal.
-- Pemberitahuan slip gaji H+1 memakai tanggal kalender tetap sehingga kalender
-- ini tidak menggesernya.
-- Policy akses langsung (Supabase Data API) menunggu persetujuan DEC-019 dan
-- sengaja tidak disertakan: RLS default deny sudah aktif (000004), dan backend
-- tetap terhubung sebagai role pemilik tabel.
CREATE TABLE IF NOT EXISTS "HariLibur" (
    "id" uuid PRIMARY KEY,
    "tanggal" date NOT NULL UNIQUE,
    "nama" text NOT NULL,
    "createdAt" timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt" timestamp(3) NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS hari_libur_tanggal_idx ON "HariLibur" ("tanggal");

ALTER TABLE "HariLibur" ENABLE ROW LEVEL SECURITY;
