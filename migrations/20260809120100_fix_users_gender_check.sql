-- +goose Up
-- +goose StatementBegin
ALTER TABLE users DROP CONSTRAINT users_gender_check;
ALTER TABLE users ADD CONSTRAINT users_gender_check CHECK (gender IN ('male', 'female', 'other', ''));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users DROP CONSTRAINT users_gender_check;
ALTER TABLE users ADD CONSTRAINT users_gender_check CHECK (gender IN ('male', 'female', 'other'));
-- +goose StatementEnd
