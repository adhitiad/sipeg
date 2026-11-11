-- +goose Up
-- Menambahkan kolom NIDN dan NIDK pada tabel user untuk identifikasi
-- dosen (NIDN) dan tenaga kependidikan (NIDK), masing-masing unik.

-- +goose StatementBegin
ALTER TABLE "user"
    ADD COLUMN IF NOT EXISTS "nidn" text,
    ADD COLUMN IF NOT EXISTS "nidk" text;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE UNIQUE INDEX IF NOT EXISTS "user_nidn_idx" ON "user" ("nidn") WHERE "nidn" IS NOT NULL;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE UNIQUE INDEX IF NOT EXISTS "user_nidk_idx" ON "user" ("nidk") WHERE "nidk" IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS "user_nidk_idx";
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX IF EXISTS "user_nidn_idx";
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE "user"
    DROP COLUMN IF EXISTS "nidk",
    DROP COLUMN IF EXISTS "nidn";
-- +goose StatementEnd
