-- +goose Up
-- Enkripsi aplikasi untuk kolom identitas PII: user.nik, user.nip,
-- user.npwp, dan RiwayatPendidikan.noIjazah (DEC-053).
--
-- Nilai tersimpan: 'enc1:' + base64url(nonce (12 byte) || AES-256-GCM(
-- ciphertext || tag)); nonce acak per enkripsi. Kunci berasal dari
-- environment PII_ENCRYPTION_KEY (base64, 32 byte) dan tidak pernah
-- ditulis di repo atau migrasi.
--
-- Setiap kolom plaintext berpasangan dengan kolom hash (blind index
-- HMAC-SHA256 atas bentuk ternormalisasi: trim, huruf besar, tanpa spasi/
-- titik/strip/garis bawah). Hash menggantikan unique index polos:
-- deterministik, tidak dapat dibalik, dan cukup untuk pencarian
-- kesetaraan dan keunikan. Pencarian substring (ILIKE) dan pengurutan
-- pada kolom terenkripsi tidak lagi bermakna.
--
-- Migrasi ini TIDAK mengenkripsi data yang sudah ada: enkripsi dilakukan
-- oleh aplikasi (driver.Valuer) pada jalur tulis, dan database target
-- produksi kosong; database uji simpeg_perf diisi ulang dengan
-- 'perfbench seed' setelah migrasi (truncate + re-seed sintetis).
-- Baris yang masih menyimpan plaintext akan ditolak dengan galat
-- eksplisit saat dibaca (prefix harus 'enc1:').

-- +goose StatementBegin
ALTER TABLE "user"
    ADD COLUMN IF NOT EXISTS "nikHash" text,
    ADD COLUMN IF NOT EXISTS "nipHash" text,
    ADD COLUMN IF NOT EXISTS "npwpHash" text;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE "RiwayatPendidikan"
    ADD COLUMN IF NOT EXISTS "noIjazahHash" text;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE "user"
    DROP CONSTRAINT IF EXISTS "user_nik_key",
    DROP CONSTRAINT IF EXISTS "user_nip_key";
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX IF EXISTS "user_npwp_idx";
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX IF EXISTS "riwayat_pendidikan_no_ijazah_idx";
-- +goose StatementEnd

-- +goose StatementBegin
CREATE UNIQUE INDEX IF NOT EXISTS "user_nik_hash_idx" ON "user" ("nikHash") WHERE "nikHash" IS NOT NULL;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE UNIQUE INDEX IF NOT EXISTS "user_nip_hash_idx" ON "user" ("nipHash") WHERE "nipHash" IS NOT NULL;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE UNIQUE INDEX IF NOT EXISTS "user_npwp_hash_idx" ON "user" ("npwpHash") WHERE "npwpHash" IS NOT NULL;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE UNIQUE INDEX IF NOT EXISTS "riwayat_pendidikan_noijazah_hash_idx" ON "RiwayatPendidikan" ("noIjazahHash") WHERE "noIjazahHash" IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- Nilai kolom nik/nip/npwp/noIjazah sudah berupa ciphertext acak; indeks
-- lama yang dibuat ulang di bawah ini menegakkan keunikan ciphertext
-- (nyaris selalu unik karena nonce acak) dan BUKAN keunikan plaintext.
-- Kunci PII_ENCRYPTION_KEY harus tersedia untuk membaca data yang sudah
-- terenkripsi; migrasi turun tidak mengembalikan plaintext.
-- +goose StatementBegin
DROP INDEX IF EXISTS "riwayat_pendidikan_noijazah_hash_idx";
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX IF EXISTS "user_npwp_hash_idx";
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX IF EXISTS "user_nip_hash_idx";
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX IF EXISTS "user_nik_hash_idx";
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE "RiwayatPendidikan"
    DROP COLUMN IF EXISTS "noIjazahHash";
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE "user"
    DROP COLUMN IF EXISTS "npwpHash",
    DROP COLUMN IF EXISTS "nipHash",
    DROP COLUMN IF EXISTS "nikHash";
-- +goose StatementEnd

-- +goose StatementBegin
CREATE UNIQUE INDEX IF NOT EXISTS "user_npwp_idx" ON "user" ("npwp") WHERE "npwp" IS NOT NULL;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE UNIQUE INDEX IF NOT EXISTS "riwayat_pendidikan_no_ijazah_idx" ON "RiwayatPendidikan" ("noIjazah") WHERE "noIjazah" IS NOT NULL;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE "user"
    ADD CONSTRAINT "user_nik_key" UNIQUE ("nik"),
    ADD CONSTRAINT "user_nip_key" UNIQUE ("nip");
-- +goose StatementEnd
