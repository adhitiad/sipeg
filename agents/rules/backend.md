# Backend rules

## Stack and boundaries

- Backend stack: Go and GORM. The managed PostgreSQL database is Supabase.
- Reviewed Goose migrations under `backend/migrations/` define the application schema. They do not prove that migrations have been applied to a target database. Implement persistence in Go with GORM under `backend/`.
- Define backend persistence from confirmed product requirements and actual migration/database state. Keep GORM mappings and reviewed Goose migrations consistent.
- Keep the backend independently buildable from the frontend. Expose product behavior through a documented HTTP API rather than sharing frontend implementation details.
- Organize code around cohesive business areas. Keep transport handlers, application logic, persistence, and domain rules distinguishable without creating layers that only forward calls.
- Keep source files at or below 387 lines; use the documented exception of at most 397 lines only when splitting harms cohesion. `css`/`scss`/`sass`/`json`/`toml`/`yaml`/`yml`/`md` are exempt (see `project.md`).

## Go and API practices

- Pass `context.Context` through request-scoped operations and honor cancellation and deadlines.
- Validate and authorize input at the API boundary; do not rely on UI visibility for access control.
- Apply deny-by-default authorization to every protected endpoint and object. Verify both the requested action and the caller's organization/data scope.
- Resolve roles and permissions from trusted server-side state. Never accept role or ownership claims from request bodies, editable user metadata, or unverified headers.
- Treat user IDs from routes and payloads as untrusted; scope database queries by the authenticated principal's permissions to prevent cross-user and cross-unit access.
- Define the session/token verification contract with the authentication runtime before exposing protected API routes. A frontend route guard is not proof of identity to the Go API.
- The established contract: Better Auth issues sessions (embedded in the TanStack Start server at `/api/auth/*`); the browser holds the token in memory and sends `Authorization: Bearer <token>`; the Go API verifies opaque database tokens via the GORM `session` resolver (and JWTs via `BETTER_AUTH_SECRET`). `GET /api/v1/me` returns the policy-derived permission list; `/api/v1/auth/logout` revokes the current session and is idempotent (204). Backend CORS does not allow credentials; session transport stays Bearer. Do not introduce a second session format without updating the resolver, `docs/openapi.yaml`, and `docs/DECISIONS.md`.
- Revoke or reject sessions for disabled accounts and test logout/revocation behavior.
- Use explicit request and response DTOs. Do not expose GORM models directly as public API schemas.
- Return consistent JSON errors with appropriate HTTP status codes. Avoid leaking database errors, stack traces, or sensitive values.
- Use transactions for multi-step writes that must succeed or fail together. Handle GORM errors explicitly.
- Apply pagination, filtering, and sorting consistently using `internal/directory.ListOptions`. Sort columns come from a server-side allowlist; the raw column name from the request never reaches SQL. API versioning uses the `/api/v1` path prefix and the uniform error shape in `internal/apierror`.

## Persistence and security

- Do not use GORM `AutoMigrate` as the production schema migration process. Use reviewed, versioned migrations once a migration tool is selected.
- Update the GORM mapping and corresponding reviewed Goose migration deliberately when the backend database model changes. Do not infer deployed state from source files alone.
- Connect to Supabase PostgreSQL from the backend using a server-side connection string, encrypted transport, and a least-privilege database role. Keep credentials out of the frontend and logs.
- Enable RLS and policies on tables exposed through Supabase Data API. Policies must enforce the actual user/organization scope; authentication alone is not authorization.
- Enforce authorization on every protected operation and scope personnel records to the caller's permitted organization or role.
- Restrict role, account status, and access-scope changes to explicitly authorized administrative operations and audit those changes.
- Add authorization tests for self-access, cross-user access, cross-unit access, privilege escalation attempts, and administrative endpoints.
- Minimize collection and exposure of personal data. Exclude secrets and unnecessary personal fields from logs.
- Record auditable changes to sensitive personnel data where required by the product policy; the exact retention and access policy is still open.

## Redis, jobs, and email

- Use Upstash Redis only for explicitly selected ephemeral or cache use cases. PostgreSQL remains the source of truth for employee, payroll, and payment records.
- Use RabbitMQ (self-hosted) for asynchronous or scheduled backend work: `cmd/jobsched` publishes persistent messages to `simpeg.jobs`, the server consumes them, and failures land in `simpeg.jobs.dead`. QStash is not used on any deployment (`DEC-042`, `DEC-102`).
- Legacy `internal/qstash/*` callbacks must keep signature verification against the raw request body. Make handlers safe for retries: validate the event, use an idempotency key, and record processing/delivery outcomes. Never assume a message is delivered exactly once.
- Send salary-paid email only after the backend records a confirmed payment event; publishing or generating a payslip is not proof of payment.
- Keep email provider credentials, RabbitMQ credentials, and any QStash signing keys (legacy) server-side. Do not include salary amounts or other sensitive data in email by default; link to an authenticated page instead.
- Keep cache or queue outages from silently losing authoritative business writes. Define retry, expiration, and fallback behavior for each integration.

## Verification

- Add unit tests for business rules and handler/service tests for API behavior, including authorization and validation cases.
- Keep tests deterministic and use synthetic personnel data only.
- `git commit`/`git push` happen only after the user says "ok commit"/"ok push" (see `agents/rules/project.md`). A green gate is not permission to commit.
