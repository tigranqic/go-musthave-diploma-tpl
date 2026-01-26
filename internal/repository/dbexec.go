package repository

import (
	"context"
	"database/sql"
)

type DBExecutor interface {
	ExecContext(
		ctx context.Context,
		query string,
		args ...any,
	) (sql.Result, error)

	QueryRowContext(
		ctx context.Context,
		query string,
		args ...any,
	) *sql.Row

	QueryContext(
		ctx context.Context,
		query string,
		args ...any,
	) (*sql.Rows, error)
}
