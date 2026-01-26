package repository

import (
	"context"
	"errors"

	models "github.com/tigranqic/go-musthave-diploma-tpl/internal/model"
)

var (
	ErrInsufficientFunds = errors.New("insufficient funds")
)

func (p *PostgresStorage) Withdraw(
	ctx context.Context,
	userID int64,
	order string,
	sum float64,
) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			if err := tx.Rollback(); err != nil {
				p = err
			}
			panic(p)
		}
	}()
	var current float64
	err = tx.QueryRowContext(ctx,
		`SELECT current FROM balances WHERE user_id = $1 FOR UPDATE`,
		userID,
	).Scan(&current)
	if err != nil {
		return err
	}

	if current < sum {
		return ErrInsufficientFunds
	}

	_, err = tx.ExecContext(ctx,
		`UPDATE balances
		 SET current = current - $1,
		     withdrawn = withdrawn + $1
		 WHERE user_id = $2`,
		sum, userID,
	)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO withdrawals (user_id, order_number, sum)
		 VALUES ($1, $2, $3)`,
		userID, order, sum,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (p *PostgresStorage) ListWithdrawals(
	ctx context.Context,
	userID int64,
) ([]models.Withdrawal, error) {
	const q = `
		SELECT order_number, sum, processed_at
		FROM withdrawals
		WHERE user_id = $1
		ORDER BY processed_at DESC
	`
	rows, err := p.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()
	var res []models.Withdrawal
	for rows.Next() {
		var w models.Withdrawal
		if err := rows.Scan(&w.Order, &w.Sum, &w.ProcessedAt); err != nil {
			return nil, err
		}
		res = append(res, w)
	}
	return res, nil
}
