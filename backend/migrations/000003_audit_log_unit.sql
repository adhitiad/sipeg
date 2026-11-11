-- +goose Up
-- Draft only: select and approve a versioned migration runner before applying.
-- Adds the unit scope of an audit entry so audit access can be restricted
-- per unit instead of exposing the whole trail to every operator.
ALTER TABLE "auditLog" ADD COLUMN "unitId" uuid;

CREATE INDEX "audit_log_unit_idx" ON "auditLog" ("unitId");
