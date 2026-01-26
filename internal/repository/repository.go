package repository

import (
	"context"
	"database/sql"

	models "github.com/tigranqic/go-musthave-diploma-tpl/internal/model"

	"go.uber.org/zap"
)

type PostgresStorage struct {
	dbExec DBExecutor
	db     *sql.DB
	log    *zap.Logger
}

type Storage interface {
	CreateUser(ctx context.Context, login string, passwordHash []byte) (int64, error)
	GetUserByLogin(ctx context.Context, login string) (*models.User, error)
	CreateOrder(ctx context.Context, order *models.Order) error
	GetOrder(ctx context.Context, number string) (*models.Order, error)
	GetOrdersByUserID(ctx context.Context, userID int64) ([]*models.Order, error)

	GetBalance(ctx context.Context, userID int64) (*models.Balance, error)
	Withdraw(ctx context.Context, userID int64, order string, sum float64) error
	ListWithdrawals(ctx context.Context, userID int64) ([]models.Withdrawal, error)

	GetOrdersForAccrual(ctx context.Context) ([]models.Order, error)
	UpdateOrderAccrual(
		ctx context.Context,
		order string,
		status string,
		accrual *float64,
	) error
	UpdateBalance(ctx context.Context, userID int64, current float64, withdrawn float64) error
}

func NewPostgresStorage(db *sql.DB, log *zap.Logger) *PostgresStorage {
	return &PostgresStorage{dbExec: db, db: db, log: log}
}
