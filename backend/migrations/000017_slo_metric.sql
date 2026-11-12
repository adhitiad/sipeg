-- +goose Up
-- Metrik Service Level Objective (SLO).
--
-- Tabel ini menyimpan agregasi latensi dan jumlah kesalahan per rute per
-- jendela waktu. Data ditulis oleh pengoleksi metrik di dalam proses backend
-- dan dibaca oleh pekerjaan terjadwal yang mengevaluasi target SLO lalu
-- mengirim alert bila target terlampaui.
--
-- Penyimpanan memakai PostgreSQL, bukan Redis, karena Redis pada repo ini
-- diperuntukkan hanya untuk cache data referensi dan ditolak untuk data
-- historis maupun status.
CREATE TABLE IF NOT EXISTS "sloMetric" (
    "id" uuid PRIMARY KEY,
    "windowStart" timestamp(3) NOT NULL,
    "windowEnd" timestamp(3) NOT NULL,
    "method" varchar(10) NOT NULL,
    "route" varchar(200) NOT NULL,
    "requestCount" bigint NOT NULL DEFAULT 0,
    "errorCount" bigint NOT NULL DEFAULT 0,
    "latencySumMs" bigint NOT NULL DEFAULT 0,
    "latencyMaxMs" bigint NOT NULL DEFAULT 0,
    "bucketLe5" bigint NOT NULL DEFAULT 0,
    "bucketLe10" bigint NOT NULL DEFAULT 0,
    "bucketLe25" bigint NOT NULL DEFAULT 0,
    "bucketLe50" bigint NOT NULL DEFAULT 0,
    "bucketLe100" bigint NOT NULL DEFAULT 0,
    "bucketLe200" bigint NOT NULL DEFAULT 0,
    "bucketLe400" bigint NOT NULL DEFAULT 0,
    "bucketLe800" bigint NOT NULL DEFAULT 0,
    "bucketLe1200" bigint NOT NULL DEFAULT 0,
    "bucketLe1600" bigint NOT NULL DEFAULT 0,
    "bucketLe2400" bigint NOT NULL DEFAULT 0,
    "bucketLe3200" bigint NOT NULL DEFAULT 0,
    "bucketLe4800" bigint NOT NULL DEFAULT 0,
    "bucketLe6400" bigint NOT NULL DEFAULT 0,
    "createdAt" timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt" timestamp(3) NOT NULL DEFAULT now(),
    CONSTRAINT "slo_metric_window_route_key" UNIQUE ("windowStart", "method", "route")
);

CREATE INDEX IF NOT EXISTS "slo_metric_window_idx" ON "sloMetric" ("windowStart");

-- Row Level Security default deny, sama seperti tabel aplikasi lainnya.
ALTER TABLE "sloMetric" ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS rl_admin_all_slo_metric ON "sloMetric";
CREATE POLICY rl_admin_all_slo_metric ON "sloMetric"
    FOR ALL TO PUBLIC
    USING (app.is_admin());

-- +goose Down
DROP TABLE IF EXISTS "sloMetric";
