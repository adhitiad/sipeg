-- MesinAbsensiPin: pemetaan machinePin mesin absensi ke pegawai.
--
-- Pegawai dapat tap di mesin mana pun (mesin sejenis, terdaftar pada
-- `MesinAbsensi`), sehingga `machinePin` unik global — bukan per mesin.
-- Pemetaan ini dipakai penurunan status harian `Presensi` dari event
-- `AttendanceEvent` (DEC-076/DEC-077). Tanpa baris pemetaan, tap mesin
-- tidak dapat dikaitkan ke pegawai dan tidak menulis `Presensi`.
--
-- Hak pengelolaan: admin (semua unit) dan operator (hanya unit penugasannya
-- beserta turunannya) — ditegakkan di lapisan aplikasi, bukan RLS.

-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS "mesinAbsensiPin" (
    "id"         uuid PRIMARY KEY,
    "machinePin" text NOT NULL UNIQUE,
    "userId"     uuid NOT NULL UNIQUE REFERENCES "user"("id") ON DELETE CASCADE,
    "aktif"      boolean NOT NULL DEFAULT true,
    "catatan"    text,
    "createdAt"  timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"  timestamp(3) NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS mesin_absensi_pin_user_idx ON "mesinAbsensiPin" ("userId");
ALTER TABLE "mesinAbsensiPin" ENABLE ROW LEVEL SECURITY;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "mesinAbsensiPin";
-- +goose StatementEnd
