-- +goose Up
-- Draft only: select and approve a versioned migration runner before applying.
-- Persistent outbox queue. Business transactions write a row here in the same
-- transaction, so work is never lost when an external queue fails.
CREATE TABLE "outboxMessage" (
    "id" uuid PRIMARY KEY,
    "kind" varchar(100) NOT NULL,
    "dedupeKey" varchar(200) NOT NULL,
    "payload" text NOT NULL,
    "status" varchar(20) NOT NULL DEFAULT 'PENDING',
    "attempts" integer NOT NULL DEFAULT 0,
    "availableAt" timestamp(3) NOT NULL DEFAULT now(),
    "lockedAt" timestamp(3),
    "lastError" text,
    "createdAt" timestamp(3) NOT NULL DEFAULT now(),
    "processedAt" timestamp(3),
    CONSTRAINT "outbox_dedupe_key" UNIQUE ("dedupeKey")
);

CREATE INDEX "outbox_dispatch_idx"
    ON "outboxMessage" ("status", "availableAt");

-- Row Level Security default deny, sama seperti tabel aplikasi lainnya.
ALTER TABLE "outboxMessage" ENABLE ROW LEVEL SECURITY;
