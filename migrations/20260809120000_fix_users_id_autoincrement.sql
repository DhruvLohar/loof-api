-- +goose Up
-- +goose StatementBegin
ALTER TABLE users ALTER COLUMN id DROP DEFAULT;
ALTER TABLE users ALTER COLUMN id TYPE BIGINT USING (CASE WHEN id ~ '^\d+$' THEN id::BIGINT ELSE NULL END);
CREATE SEQUENCE IF NOT EXISTS users_id_seq OWNED BY users.id;
ALTER TABLE users ALTER COLUMN id SET DEFAULT nextval('users_id_seq');
SELECT setval('users_id_seq', COALESCE((SELECT MAX(id) FROM users), 0) + 1, false);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users ALTER COLUMN id DROP DEFAULT;
ALTER TABLE users ALTER COLUMN id TYPE VARCHAR(255) USING id::VARCHAR;
DROP SEQUENCE IF EXISTS users_id_seq;
-- +goose StatementEnd
