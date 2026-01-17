package repository

import (
	"database/sql"

	_ "github.com/lib/pq"
	"github.com/pressly/goose"
	"github.com/tigranqic/go-musthave-diploma-tpl/internal/config"
	"go.uber.org/zap"
)

func InitStorage(cfg *config.Config, log *zap.Logger) (*sql.DB, Storage, error) {
	if cfg.DatabaseDSN != "" {
		db, err := connectPostgres(cfg)
		if err != nil {
			return nil, nil, err
		}
		store := NewPostgresStorage(db, log)
		log.Info("Init PostgreSQL storage")
		return db, store, nil
	}

	return nil, nil, nil
}

func connectPostgres(cfg *config.Config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.DatabaseDSN)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	if err := goose.Up(db, "migrations"); err != nil {
		return nil, err
	}

	return db, nil
}
