-- +goose Up
-- Menambahkan nilai BATAL pada enum PostgreSQL StatusPersetujuan untuk
-- pembatalan pengajuan cuti (DEC-080). ADD VALUE IF NOT EXISTS aman diulang.
-- Down sengaja tidak melepas nilai (PostgreSQL tidak mendukung DROP VALUE).
ALTER TYPE "StatusPersetujuan" ADD VALUE IF NOT EXISTS 'BATAL';

-- +goose Down
-- Tidak ada down: nilai enum yang sudah dipakai tidak dapat dihapus dengan
-- aman. Down biarkan no-op.
SELECT 1;
