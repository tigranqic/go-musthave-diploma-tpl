package repository

import (
	"context"
	"database/sql"

	models "github.com/tigranqic/go-musthave-diploma-tpl/internal/model"
)

func (p *PostgresStorage) GetBalance(ctx context.Context, userID int64) (*models.Balance, error) {
	const q = `
		SELECT current, withdrawn
		FROM balances
		WHERE user_id = $1
	`
	var b models.Balance
	err := p.db.QueryRowContext(ctx, q, userID).Scan(&b.Current, &b.Withdrawn)
	if err == sql.ErrNoRows {
		return &models.Balance{}, nil
	}
	return &b, err
}

func (p *PostgresStorage) UpdateBalance(ctx context.Context, userID int64, current, withdrawn float64) error {
	_, err := p.db.ExecContext(ctx, `
		INSERT INTO balances (user_id, current, withdrawn)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id) DO UPDATE
		SET current = $2, withdrawn = $3
	`, userID, current, withdrawn)
	return err
}
