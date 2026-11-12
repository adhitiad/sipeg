-- +goose Up
-- Menghapus indeks duplikat pada tabel "twoFactor".
--
-- Kolom "userId" sudah memiliki indeks unik "twoFactor_userId_key" dari
-- constraint UNIQUE pada migrasi 000001, sehingga indeks eksplisit
-- "twoFactor_user_idx" pada kolom yang sama tidak memberi manfaat apa pun
-- dan hanya menambah biaya tulis. Audit indeks pada pengukuran performa
-- menemukan duplikasi ini.
--
-- Model GORM models.TwoFactor juga tidak lagi menuntut indeks tambahan:
-- kolom "secret" tidak diindeks karena belum ada jalur query yang membaca
-- berdasarkan secret (runtime MFA Better Auth belum dibuat).
DROP INDEX IF EXISTS "twoFactor_user_idx";

-- +goose Down
CREATE INDEX IF NOT EXISTS "twoFactor_user_idx" ON "twoFactor" ("userId");
