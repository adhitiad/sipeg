-- +goose Up
-- Enkripsi aplikasi untuk kolom keluargaPegawai.nik (DEC-054, perluasan
-- DEC-053).
--
-- Nilai tersimpan: 'enc1:' + base64url(nonce (12 byte) || AES-256-GCM(
-- ciphertext || tag)); nonce acak per enkripsi. Kunci berasal dari
-- environment PII_ENCRYPTION_KEY dan tidak pernah ditulis di repo atau
-- migrasi.
--
-- Kolom nik tidak pernah unik (hanya indeks biasa), sehingga cukup
-- berpasangan dengan kolom hash nikHash (blind index HMAC-SHA256 atas bentuk
-- ternormalisasi) yang dipakai untuk pencarian kesetaraan. Pencarian
-- substring (ILIKE) dan pengurutan pada nik tidak lagi bermakna.
--
-- Migrasi ini TIDAK mengenkripsi data yang sudah ada: database produksi
-- 'postgres' dan database uji 'simpeg_perf' tidak memiliki baris keluarga
-- saat migrasi diterapkan (diverifikasi 13 November 2026). Baris baru
-- disimpan terenkripsi oleh aplikasi. Baris yang masih menyimpan plaintext
-- akan ditolak dengan galat eksplisit saat dibaca (prefix harus 'enc1:').

-- +goose StatementBegin
ALTER TABLE "keluargaPegawai"
    ADD COLUMN IF NOT EXISTS "nikHash" text;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX IF EXISTS "keluarga_pegawai_nik_idx";
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS "keluarga_pegawai_nik_hash_idx" ON "keluargaPegawai" ("nikHash");
-- +goose StatementEnd

-- +goose Down
-- Nilai kolom nik sudah berupa ciphertext acak; indeks plaintext yang dibuat
-- ulang di bawah ini hanya menegakkan keberadaan nilai, bukan keunikan atau
-- kesetaraan plaintext. Kunci PII_ENCRYPTION_KEY harus tersedia untuk
-- membaca data yang sudah terenkripsi; migrasi turun tidak mengembalikan
-- plaintext.
-- +goose StatementBegin
DROP INDEX IF EXISTS "keluarga_pegawai_nik_hash_idx";
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE "keluargaPegawai"
    DROP COLUMN IF EXISTS "nikHash";
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS "keluarga_pegawai_nik_idx" ON "keluargaPegawai" ("nik");
-- +goose StatementEnd
