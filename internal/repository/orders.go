package repository

import (
	"context"
	"database/sql"
	"errors"

	models "github.com/tigranqic/go-musthave-diploma-tpl/internal/model"
	"github.com/tigranqic/go-musthave-diploma-tpl/internal/repository/pgerrors"
)

var (
	ErrOrderExists       = errors.New("order exists")
	ErrOrderOwnedByOther = errors.New("order owned by another user")
	ErrOrderNotFound     = errors.New("order not found")
)

func (p *PostgresStorage) CreateOrder(ctx context.Context, o *models.Order) error {
	const q = `
		INSERT INTO orders (number, user_id, status)
		VALUES ($1, $2, $3)
	`

	_, err := p.dbExec.ExecContext(ctx, q, o.Number, o.UserID, o.Status)
	if err == nil {
		return nil
	}

	if !pgerrors.IsUniqueViolation(err) {
		return err
	}

	const checkQ = `
		SELECT user_id
		FROM orders
		WHERE number = $1
	`

	var existingUserID int64
	checkErr := p.dbExec.QueryRowContext(ctx, checkQ, o.Number).Scan(&existingUserID)
	if checkErr != nil {
		return checkErr
	}

	if existingUserID == o.UserID {
		return ErrOrderExists
	}

	return ErrOrderOwnedByOther
}

func (p *PostgresStorage) GetOrder(ctx context.Context, number string) (*models.Order, error) {
	const q = `
		SELECT number, user_id, status, accrual, uploaded_at
		FROM orders
		WHERE number = $1
	`

	o := &models.Order{}
	err := p.dbExec.QueryRowContext(ctx, q, number).
		Scan(&o.Number, &o.UserID, &o.Status, &o.Accrual, &o.UploadedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrOrderNotFound
	}
	if err != nil {
		return nil, err
	}

	return o, nil
}

func (p *PostgresStorage) GetOrdersByUserID(ctx context.Context, userID int64) ([]*models.Order, error) {
	const q = `
		SELECT number, user_id, status, accrual, uploaded_at
		FROM orders
		WHERE user_id = $1
		ORDER BY uploaded_at DESC
	`

	rows, err := p.dbExec.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()
	var orders []*models.Order
	for rows.Next() {
		o := &models.Order{}
		if err := rows.Scan(
			&o.Number,
			&o.UserID,
			&o.Status,
			&o.Accrual,
			&o.UploadedAt,
		); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}

	return orders, nil
}

func (p *PostgresStorage) GetOrdersByStatus(ctx context.Context, status string) ([]*models.Order, error) {
	const q = `
		SELECT number, user_id, status, accrual, uploaded_at
		FROM orders
		WHERE status = $1
	`

	rows, err := p.dbExec.QueryContext(ctx, q, status)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()
	var orders []*models.Order
	for rows.Next() {
		o := &models.Order{}
		if err := rows.Scan(
			&o.Number,
			&o.UserID,
			&o.Status,
			&o.Accrual,
			&o.UploadedAt,
		); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}

	return orders, nil
}

func (p *PostgresStorage) UpdateOrderStatus(ctx context.Context, o *models.Order) error {
	const q = `
		UPDATE orders
		SET status = $1,
		    accrual = $2
		WHERE number = $3
	`

	res, err := p.dbExec.ExecContext(ctx, q, o.Status, o.Accrual, o.Number)
	if err != nil {
		return err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrOrderNotFound
	}

	return nil
}
