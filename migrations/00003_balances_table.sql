-- +goose Up
-- +goose StatementBegin
CREATE TABLE balances (
    user_id BIGINT PRIMARY KEY REFERENCES users(id),
    current NUMERIC(12,2) NOT NULL DEFAULT 0,
    withdrawn NUMERIC(12,2) NOT NULL DEFAULT 0
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE balances;
-- +goose StatementEnd
