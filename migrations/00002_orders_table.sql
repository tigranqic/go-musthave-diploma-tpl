-- +goose Up
-- +goose StatementBegin
CREATE TABLE orders (
    number TEXT PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    status TEXT NOT NULL,
    accrual NUMERIC,
    uploaded_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_orders_user_id ON orders(user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE orders;
-- +goose StatementEnd
