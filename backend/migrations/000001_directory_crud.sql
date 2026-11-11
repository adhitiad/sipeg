-- Draft only: select and approve a versioned migration runner before applying.
-- Adds the typed organization tree and the active flag used by the directory
-- endpoints. No permission or access scope is implied by this migration.
DO $$
BEGIN
    CREATE TYPE "JenisOrganisasi" AS ENUM ('UNIT', 'FAKULTAS', 'JURUSAN', 'BIRO', 'BAGIAN');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END
$$;

ALTER TABLE "Homebase"
    ADD COLUMN IF NOT EXISTS "jenis" "JenisOrganisasi" NOT NULL DEFAULT 'UNIT',
    ADD COLUMN IF NOT EXISTS "parentId" uuid,
    ADD COLUMN IF NOT EXISTS "aktif" boolean NOT NULL DEFAULT true;

ALTER TABLE "user"
    ADD COLUMN IF NOT EXISTS "aktif" boolean NOT NULL DEFAULT true;

CREATE INDEX IF NOT EXISTS homebase_parent_idx ON "Homebase" ("parentId");

DO $$
BEGIN
    ALTER TABLE "Homebase"
        ADD CONSTRAINT "Homebase_parentId_fkey"
        FOREIGN KEY ("parentId") REFERENCES "Homebase" ("id");
EXCEPTION
    WHEN duplicate_object THEN NULL;
END
$$;

DO $$
BEGIN
    ALTER TABLE "Homebase"
        ADD CONSTRAINT "Homebase_parent_not_self_check"
        CHECK ("parentId" IS NULL OR "parentId" <> "id");
EXCEPTION
    WHEN duplicate_object THEN NULL;
END
$$;
