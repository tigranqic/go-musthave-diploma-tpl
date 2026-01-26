package repository

import (
	"context"
	"database/sql"
	"errors"

	models "github.com/tigranqic/go-musthave-diploma-tpl/internal/model"
)

func (p *PostgresStorage) GetOrdersForAccrual(ctx context.Context) ([]models.Order, error) {
	const query = `
		SELECT number, status, user_id, accrual
		FROM orders
		WHERE status IN ('REGISTERED', 'PROCESSING', 'NEW')
		LIMIT 100
	`

	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()
	var orders []models.Order
	for rows.Next() {
		var o models.Order
		var accrual sql.NullFloat64
		if err := rows.Scan(&o.Number, &o.Status, &o.UserID, &accrual); err != nil {
			return nil, err
		}
		if accrual.Valid {
			o.Accrual = &accrual.Float64
		}
		orders = append(orders, o)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return orders, nil
}

func (p *PostgresStorage) UpdateOrderAccrual(ctx context.Context, orderNumber string, status string, accrual *float64) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			if err := tx.Rollback(); err != nil {
				// Log the rollback error if needed
				p = err
			}
			panic(p)
		}
	}()

	const selectOrderQuery = `
		SELECT user_id, accrual
		FROM orders
		WHERE number = $1
		FOR UPDATE
	`
	var userID int64
	var oldAccrual sql.NullFloat64
	err = tx.QueryRowContext(ctx, selectOrderQuery, orderNumber).Scan(&userID, &oldAccrual)
	if err != nil {
		defer func() {
			if p := recover(); p != nil {
				if err := tx.Rollback(); err != nil {
					p = err
				}
				panic(p)
			}
		}()
		if errors.Is(err, sql.ErrNoRows) {
			return ErrOrderNotFound
		}
		return err
	}

	const updateOrderQuery = `
		UPDATE orders
		SET status = $1, accrual = $2
		WHERE number = $3
	`
	_, err = tx.ExecContext(ctx, updateOrderQuery, status, accrual, orderNumber)
	if err != nil {
		defer func() {
			if p := recover(); p != nil {
				if err := tx.Rollback(); err != nil {
					p = err
				}
				panic(p)
			}
		}()
		return err
	}

	if accrual != nil && (oldAccrual.Float64 != *accrual) {
		const updateBalanceQuery = `
			INSERT INTO balances (user_id, current, withdrawn)
			VALUES ($1, $2, 0)
			ON CONFLICT (user_id)
			DO UPDATE SET current = balances.current + EXCLUDED.current
		`
		_, err = tx.ExecContext(ctx, updateBalanceQuery, userID, *accrual-oldAccrual.Float64)
		if err != nil {
			defer func() {
				if p := recover(); p != nil {
					if err := tx.Rollback(); err != nil {
						p = err
					}
					panic(p)
				}
			}()
			return err
		}
	}

	return tx.Commit()
}
