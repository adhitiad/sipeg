-- Draft only: select and approve a versioned migration runner before applying.
-- The audit trail is append-only and grows without bound, so the default
-- listing order (createdAt DESC) needs a supporting index to stay cheap.
CREATE INDEX "audit_log_created_at_idx" ON "auditLog" ("createdAt");
