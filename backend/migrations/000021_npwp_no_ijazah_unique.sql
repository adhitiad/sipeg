-- +goose Up
-- Menjadikan NPWP pegawai unik dan menambahkan nomor ijazah unik pada
-- RiwayatPendidikan.
--
-- Sebelumnya `user.npwp` tidak unik (boleh sama atau kosong berulang).
-- Migrasi ini lebih dulu menormalkan string kosong/spasi menjadi NULL
-- (string kosong tidak membawa informasi), lalu membuat indeks unik parsial.
-- NPWP non-kosong yang sudah terduplikasi pada data target akan menggagalkan
-- penerapan ini — duplikat harus diselesaikan oleh pemilik data sebelum
-- migrasi dijalankan ulang; nilai mana yang benar tidak boleh ditebak.

-- +goose StatementBegin
UPDATE "user"
SET "npwp" = NULL
WHERE "npwp" IS NOT NULL AND btrim("npwp") = '';
-- +goose StatementEnd

-- +goose StatementBegin
CREATE UNIQUE INDEX IF NOT EXISTS "user_npwp_idx" ON "user" ("npwp") WHERE "npwp" IS NOT NULL;
-- +goose StatementEnd

-- Satu baris RiwayatPendidikan mewakili satu ijazah; satu orang boleh punya
-- beberapa ijazah (S1/S2/S3) sehingga uniknya berada di level baris, bukan
-- per pegawai. Kolom baru, sehingga seluruh baris awal bernilai NULL dan
-- indeks unik parsial selalu dapat dibuat.
-- +goose StatementBegin
ALTER TABLE "RiwayatPendidikan"
    ADD COLUMN IF NOT EXISTS "noIjazah" text;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE UNIQUE INDEX IF NOT EXISTS "riwayat_pendidikan_no_ijazah_idx" ON "RiwayatPendidikan" ("noIjazah") WHERE "noIjazah" IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- Catatan: Down tidak mengembalikan NPWP string kosong yang dinormalkan
-- menjadi NULL — nilai tersebut tidak membawa informasi dan tidak dapat
-- dibedakan dari NPWP yang memang kosong sejak awal.
-- +goose StatementBegin
DROP INDEX IF EXISTS "riwayat_pendidikan_no_ijazah_idx";
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE "RiwayatPendidikan"
    DROP COLUMN IF EXISTS "noIjazah";
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX IF EXISTS "user_npwp_idx";
-- +goose StatementEnd
