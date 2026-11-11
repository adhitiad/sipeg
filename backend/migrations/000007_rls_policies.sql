-- +goose Up
-- Draft only: select and approve a versioned migration runner before applying.
-- Row Level Security policies per tabel aplikasi, berdasarkan matriks peran yang disetujui:
--   admin   : semua baris
--   operator: baris di homebase miliknya + turunan
--   atasan  : baris di homebase miliknya + turunan
--   pegawai : hanya baris miliknya sendiri
--
-- Cakupan dievaluasi melalui session variables yang di-set oleh aplikasi:
--   app.current_user_id  : UUID pengguna yang login
--   app.current_homebase_id : UUID homebase pengguna yang login
--   app.current_role     : nama role pengguna (admin, operator, atasan, pegawai)
--
-- Fungsi helper is_admin() dan is_scope_member(homebaseId text) didefinisikan
-- di sini dan dipakai oleh semua policy.

-- Fungsi helper: apakah role saat ini adalah admin.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.is_admin() RETURNS boolean
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN current_setting('app.current_role', true) = 'admin';
END;
$$;
-- +goose StatementEnd

-- Fungsi helper: apakah homebaseId berada dalam cakupan pengguna (sendiri atau turunan).
-- Menggunakan recursive CTE untuk mencari semua turunan dari homebase pengguna.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.user_homebase_ids() RETURNS TABLE(homebase_id uuid)
LANGUAGE plpgsql
AS $$
DECLARE
    my_homebase uuid;
BEGIN
    SELECT current_setting('app.current_homebase_id', true)::uuid INTO my_homebase;
    IF my_homebase IS NULL THEN
        RETURN;
    END IF;

    RETURN QUERY
    WITH RECURSIVE subtree AS (
        SELECT id FROM "Homebase" WHERE id = my_homebase
        UNION
        SELECT h.id FROM "Homebase" h JOIN subtree s ON h."parentId" = s.id
    )
    SELECT id FROM subtree;
END;
$$;
-- +goose StatementEnd

-- Fungsi helper: apakah role saat ini adalah atasan (atasan dapat melihat bawahan).
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.is_supervisor() RETURNS boolean
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN current_setting('app.current_role', true) IN ('operator', 'atasan');
END;
$$;
-- +goose StatementEnd

-- ============================================================
-- Tabel: user
-- Kebijakan: admin melihat semua; operator/atasan melihat pegawai di homebase cakupan;
-- pegawai hanya melihat dirinya sendiri.
-- ============================================================
ALTER TABLE "user" ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS rl_admin_all ON "user";
CREATE POLICY rl_admin_all ON "user"
    FOR ALL TO PUBLIC
    USING (app.is_admin());

DROP POLICY IF EXISTS rl_user_supervisor_scope ON "user";
CREATE POLICY rl_user_supervisor_scope ON "user"
    FOR SELECT TO PUBLIC
    USING (
        app.is_supervisor()
        AND "homebaseId" = ANY(SELECT homebase_id FROM app.user_homebase_ids())
    );

DROP POLICY IF EXISTS rl_user_self ON "user";
CREATE POLICY rl_user_self ON "user"
    FOR SELECT TO PUBLIC
    USING ("id" = current_setting('app.current_user_id', true)::uuid);

-- ============================================================
-- Tabel: Homebase (organisasi)
-- Kebijakan: admin melihat semua; operator/atasan melihat homebase cakupan;
-- pegawai hanya melihat homebase miliknya.
-- ============================================================
ALTER TABLE "Homebase" ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS rl_admin_all_homebase ON "Homebase";
CREATE POLICY rl_admin_all_homebase ON "Homebase"
    FOR ALL TO PUBLIC
    USING (app.is_admin());

DROP POLICY IF EXISTS rl_homebase_supervisor_scope ON "Homebase";
CREATE POLICY rl_homebase_supervisor_scope ON "Homebase"
    FOR SELECT TO PUBLIC
    USING (
        app.is_supervisor()
        AND id = ANY(SELECT homebase_id FROM app.user_homebase_ids())
    );

DROP POLICY IF EXISTS rl_homebase_self ON "Homebase";
CREATE POLICY rl_homebase_self ON "Homebase"
    FOR SELECT TO PUBLIC
    USING (id = current_setting('app.current_homebase_id', true)::uuid);

-- ============================================================
-- Tabel: Gaji (slip gaji)
-- Kebijakan: admin melihat semua; operator/atasan melihat gaji pegawai di cakupan;
-- pegawai hanya melihat slipnya sendiri.
-- ============================================================
ALTER TABLE "gaji" ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS rl_admin_all_gaji ON "gaji";
CREATE POLICY rl_admin_all_gaji ON "gaji"
    FOR ALL TO PUBLIC
    USING (app.is_admin());

DROP POLICY IF EXISTS rl_gaji_supervisor_scope ON "gaji";
CREATE POLICY rl_gaji_supervisor_scope ON "gaji"
    FOR SELECT TO PUBLIC
    USING (
        app.is_supervisor()
        AND "userId" = ANY(
            SELECT u."id" FROM "user" u
            WHERE u."homebaseId" = ANY(SELECT homebase_id FROM app.user_homebase_ids())
        )
    );

DROP POLICY IF EXISTS rl_gaji_self ON "gaji";
CREATE POLICY rl_gaji_self ON "gaji"
    FOR SELECT TO PUBLIC
    USING ("userId" = current_setting('app.current_user_id', true)::uuid);

-- ============================================================
-- Tabel: AuditLog
-- Kebijakan: admin melihat semua; operator melihat jejak di cakupannya;
-- pegawai tidak melihat audit (kecuali melalui endpoint khusus).
-- ============================================================
ALTER TABLE "auditLog" ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS rl_admin_all_audit ON "auditLog";
CREATE POLICY rl_admin_all_audit ON "auditLog"
    FOR ALL TO PUBLIC
    USING (app.is_admin());

DROP POLICY IF EXISTS rl_audit_supervisor_scope ON "auditLog";
CREATE POLICY rl_audit_supervisor_scope ON "auditLog"
    FOR SELECT TO PUBLIC
    USING (
        app.is_supervisor()
        AND "unitId" = ANY(SELECT homebase_id FROM app.user_homebase_ids())
    );

-- ============================================================
-- Tabel: outboxMessage
-- Kebijakan: admin melihat semua; operator/atasan melihat pesan di cakupannya.
-- ============================================================
ALTER TABLE "outboxMessage" ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS rl_admin_all_outbox ON "outboxMessage";
CREATE POLICY rl_admin_all_outbox ON "outboxMessage"
    FOR ALL TO PUBLIC
    USING (app.is_admin());

DROP POLICY IF EXISTS rl_outbox_supervisor_scope ON "outboxMessage";
CREATE POLICY rl_outbox_supervisor_scope ON "outboxMessage"
    FOR SELECT TO PUBLIC
    USING (
        app.is_supervisor()
        AND "id" = ANY(
            SELECT o."id" FROM "outboxMessage" o
            JOIN "user" u ON u."id"::text = o."payload"::jsonb->>'userId'
            WHERE u."homebaseId" = ANY(SELECT homebase_id FROM app.user_homebase_ids())
        )
    );

-- ============================================================
-- Tabel: Cuti
-- Kebijakan: admin melihat semua; operator/atasan melihat cuti di cakupan;
-- pegawai hanya melihat cuti sendiri.
-- ============================================================
ALTER TABLE "Cuti" ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS rl_admin_all_cuti ON "Cuti";
CREATE POLICY rl_admin_all_cuti ON "Cuti"
    FOR ALL TO PUBLIC
    USING (app.is_admin());

DROP POLICY IF EXISTS rl_cuti_supervisor_scope ON "Cuti";
CREATE POLICY rl_cuti_supervisor_scope ON "Cuti"
    FOR SELECT TO PUBLIC
    USING (
        app.is_supervisor()
        AND "userId" = ANY(
            SELECT u."id" FROM "user" u
            WHERE u."homebaseId" = ANY(SELECT homebase_id FROM app.user_homebase_ids())
        )
    );

DROP POLICY IF EXISTS rl_cuti_self ON "Cuti";
CREATE POLICY rl_cuti_self ON "Cuti"
    FOR SELECT TO PUBLIC
    USING ("userId" = current_setting('app.current_user_id', true)::uuid);

-- ============================================================
-- Tabel: Pengumuman
-- Kebijakan: admin melihat semua; operator/atasan melihat semua (umum);
-- pegawai melihat semua (umum).
-- ============================================================
ALTER TABLE "Pengumuman" ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS rl_admin_all_pengumuman ON "Pengumuman";
CREATE POLICY rl_admin_all_pengumuman ON "Pengumuman"
    FOR ALL TO PUBLIC
    USING (app.is_admin());

DROP POLICY IF EXISTS rl_pengumuman_all ON "Pengumuman";
CREATE POLICY rl_pengumuman_all ON "Pengumuman"
    FOR SELECT TO PUBLIC
    USING (true);
