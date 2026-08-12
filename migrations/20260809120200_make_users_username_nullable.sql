-- +goose Up
-- +goose StatementBegin
ALTER TABLE users ALTER COLUMN username DROP NOT NULL;
UPDATE users SET username = NULL WHERE username = '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
UPDATE users SET username = '' WHERE username IS NULL;
ALTER TABLE users ALTER COLUMN username SET NOT NULL;
-- +goose StatementEnd
