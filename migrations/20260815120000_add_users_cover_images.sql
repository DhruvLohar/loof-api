-- +goose Up
-- +goose StatementBegin
ALTER TABLE users ADD COLUMN cover_images TEXT[];
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users DROP COLUMN cover_images;
-- +goose StatementEnd
