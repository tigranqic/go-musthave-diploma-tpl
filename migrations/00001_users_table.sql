-- +goose Up
-- +goose StatementBegin
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    login TEXT NOT NULL UNIQUE,
    password_hash BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE users;
-- +goose StatementEnd
