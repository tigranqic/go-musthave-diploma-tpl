-- +goose Up
-- +goose StatementBegin
CREATE TABLE withdrawals (
    id SERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    order_number TEXT NOT NULL,
    sum NUMERIC(12,2) NOT NULL,
    processed_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX ux_withdrawals_order ON withdrawals(order_number);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE withdrawals;
-- +goose StatementEnd
