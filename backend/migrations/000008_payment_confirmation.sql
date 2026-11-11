-- Draft only: select and approve a versioned migration runner before applying.
-- Tambahkan representasi pembayaran gaji pada slip gaji.
-- Status slip gaji diperluas dengan TERBAYAR (sudah dibayar dan dikonfirmasi).
-- Tabel pembayaran baru mencatat bukti konfirmasi pembayaran.

-- Tambah kolom pada tabel gaji
ALTER TABLE "gaji" ADD COLUMN "tanggalDibayar" timestamp(3);
ALTER TABLE "gaji" ADD COLUMN "konfirmasiPembayaranAt" timestamp(3);
ALTER TABLE "gaji" ADD COLUMN "konfirmasiOlehId" uuid;

-- Tambah indeks untuk query pembayaran
CREATE INDEX "gaji_status_idx" ON "gaji" ("status");
CREATE INDEX "gaji_tanggalDibayar_idx" ON "gaji" ("tanggalDibayar");
CREATE INDEX "gaji_konfirmasiOlehId_idx" ON "gaji" ("konfirmasiOlehId");

-- Tabel pembayaran baru
CREATE TABLE "pembayaran" (
    "id" uuid PRIMARY KEY,
    "gajiId" uuid NOT NULL,
    "referensiTransaksi" varchar(200),
    "waktuKonfirmasi" timestamp(3) NOT NULL DEFAULT now(),
    "pelakuPengesahan" varchar(200) NOT NULL,
    "keterangan" text,
    "createdAt" timestamp(3) NOT NULL DEFAULT now(),
    CONSTRAINT "pembayaran_gajiId_fkey" FOREIGN KEY ("gajiId") REFERENCES "gaji" ("id") ON DELETE CASCADE
);

CREATE INDEX "pembayaran_gaji_idx" ON "pembayaran" ("gajiId");

-- Row Level Security default deny, sama seperti tabel aplikasi lainnya.
ALTER TABLE "pembayaran" ENABLE ROW LEVEL SECURITY;

-- Admin melihat semua pembayaran
DROP POLICY IF EXISTS rl_admin_all_pembayaran ON "pembayaran";
CREATE POLICY rl_admin_all_pembayaran ON "pembayaran"
    FOR ALL TO PUBLIC
    USING (app.is_admin());

-- Operator/atasan melihat pembayaran di cakupan unit mereka
DROP POLICY IF EXISTS rl_pembayaran_supervisor_scope ON "pembayaran";
CREATE POLICY rl_pembayaran_supervisor_scope ON "pembayaran"
    FOR SELECT TO PUBLIC
    USING (
        app.is_supervisor()
        AND "gajiId" = ANY(
            SELECT g."id" FROM "gaji" g
            JOIN "user" u ON u."id" = g."userId"
            WHERE u."homebaseId" = ANY(SELECT homebase_id FROM app.user_homebase_ids())
        )
    );

-- Pegawai hanya melihat pembayaran slipnya sendiri
DROP POLICY IF EXISTS rl_pembayaran_self ON "pembayaran";
CREATE POLICY rl_pembayaran_self ON "pembayaran"
    FOR SELECT TO PUBLIC
    USING (
        EXISTS (
            SELECT 1 FROM "gaji" g
            WHERE g."id" = "pembayaran"."gajiId"
              AND g."userId" = current_setting('app.current_user_id', true)::uuid
        )
    );
