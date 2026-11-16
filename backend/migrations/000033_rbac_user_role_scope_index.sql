-- +goose Up
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS "rbac_user_role_scope_user_idx"
    ON "rbac_user_role_scopes" ("user_id");
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS "rbac_user_role_scope_user_idx";
-- +goose StatementEnd