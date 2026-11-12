-- +goose Up
-- Perbaikan keselarasan skema tabel "pembayaran" dengan model GORM
-- models.Pembayaran.
--
-- Model Pembayaran memakai BaseModel sehingga GORM menulis kolom "updatedAt"
-- pada setiap INSERT. Tabel "pembayaran" yang dibuat pada migrasi 000008
-- hanya memiliki "createdAt". Akibatnya setiap konfirmasi pembayaran gaji
-- pada POST /api/v1/payroll/slips/{payrollId}/payment gagal dengan
-- SQLSTATE 42703 (column "updatedAt" of relation "pembayaran" does not exist).
--
-- Migrasi ini menambahkan kolom yang sudah dinyatakan model, sehingga tidak
-- ada perubahan perilaku aplikasi.
ALTER TABLE "pembayaran"
    ADD COLUMN IF NOT EXISTS "updatedAt" timestamp(3) NOT NULL DEFAULT now();

-- +goose Down
ALTER TABLE "pembayaran" DROP COLUMN IF EXISTS "updatedAt";
