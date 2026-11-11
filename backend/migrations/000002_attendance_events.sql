-- +goose Up
-- Draft only: select and approve a versioned migration runner before applying.
-- No vendor protocol, device credential, or attendance status is implied here.
CREATE TABLE "AttendanceEvent" (
    "id" uuid PRIMARY KEY,
    "machineId" uuid NOT NULL,
    "machinePin" varchar(100) NOT NULL,
    "occurredAt" timestamptz(3) NOT NULL,
    "idempotencyKey" varchar(200) NOT NULL,
    "createdAt" timestamptz(3) NOT NULL DEFAULT now(),
    CONSTRAINT "AttendanceEvent_machineId_fkey"
        FOREIGN KEY ("machineId") REFERENCES "MesinAbsensi" ("id") ON DELETE RESTRICT,
    CONSTRAINT "attendance_event_machine_idempotency_key"
        UNIQUE ("machineId", "idempotencyKey")
);

CREATE INDEX "attendance_event_machine_time_idx"
    ON "AttendanceEvent" ("machineId", "occurredAt");
CREATE INDEX "attendance_event_pin_time_idx"
    ON "AttendanceEvent" ("machinePin", "occurredAt");
