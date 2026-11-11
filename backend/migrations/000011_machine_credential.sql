-- +goose Up
-- MachineCredential: kredensial autentikasi perangkat mesin absensi.
-- Tabel ini menyimpan hash kredensial perangkat (bukan plaintext)
-- untuk verifikasi identitas mesin saat mengirim event presensi.
-- Migrasi ini diterapkan setelah 000010_holiday.sql.
CREATE TABLE "machineCredential" (
    "id" uuid PRIMARY KEY,
    "machineId" uuid NOT NULL,
    "credentialType" varchar(50) NOT NULL,
    "credentialHash" text NOT NULL,
    "active" boolean NOT NULL DEFAULT true,
    "lastUsedAt" timestamp(3),
    "createdAt" timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt" timestamp(3) NOT NULL DEFAULT now(),
    CONSTRAINT "machineCredential_machineId_fkey" FOREIGN KEY ("machineId") REFERENCES "MesinAbsensi" ("id") ON DELETE CASCADE
);
CREATE INDEX "machine_credential_machine_idx" ON "machineCredential" ("machineId");
ALTER TABLE "machineCredential" ENABLE ROW LEVEL SECURITY;