-- Menambahkan kerangka RBAC berbasis scoped assignment (`DEC-087`):
-- role master berbasis tier, permission master per modul/aksi, matriks
-- role-permission, dan assignment user-role dengan scope organisasi serta
-- daftar permission yang dilarang (blacklist).

-- +goose Up
CREATE TABLE IF NOT EXISTS "rbac_permissions" (
    id BIGSERIAL PRIMARY KEY,
    module VARCHAR(50) NOT NULL,
    action VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_rbac_permissions_module_action UNIQUE (module, action)
);

CREATE TABLE IF NOT EXISTS "rbac_roles" (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    tier_level INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS "rbac_role_permissions" (
    role_id BIGINT NOT NULL REFERENCES "rbac_roles" (id) ON DELETE CASCADE,
    permission_id BIGINT NOT NULL REFERENCES "rbac_permissions" (id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE IF NOT EXISTS "rbac_user_role_scopes" (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL,
    role_id BIGINT NOT NULL REFERENCES "rbac_roles" (id) ON DELETE CASCADE,
    biro_code VARCHAR(50) NULL,
    fakultas_id UUID NULL,
    jurusan_id UUID NULL,
    unit_id UUID NULL,
    denied_permissions JSONB DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE "rbac_permissions" ENABLE ROW LEVEL SECURITY;
ALTER TABLE "rbac_roles" ENABLE ROW LEVEL SECURITY;
ALTER TABLE "rbac_role_permissions" ENABLE ROW LEVEL SECURITY;
ALTER TABLE "rbac_user_role_scopes" ENABLE ROW LEVEL SECURITY;

-- +goose Down
DROP TABLE IF EXISTS "rbac_user_role_scopes";
DROP TABLE IF EXISTS "rbac_role_permissions";
DROP TABLE IF EXISTS "rbac_roles";
DROP TABLE IF EXISTS "rbac_permissions";
