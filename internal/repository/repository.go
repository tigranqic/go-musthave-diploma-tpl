package repository

import (
	"database/sql"

	"go.uber.org/zap"
)

type PostgresStorage struct {
	dbExec DBExecutor
	db     *sql.DB
	log    *zap.Logger
}

type Storage interface {
}

func NewPostgresStorage(db *sql.DB, log *zap.Logger) *PostgresStorage {
	return &PostgresStorage{dbExec: db, db: db, log: log}
}
