package repository

import (
	"context"
	"database/sql"
	"errors"

	models "github.com/tigranqic/go-musthave-diploma-tpl/internal/model"
	"github.com/tigranqic/go-musthave-diploma-tpl/internal/repository/pgerrors"
)

var (
	ErrLoginTaken   = errors.New("login already taken")
	ErrUserNotFound = errors.New("user not found")
)

func (p *PostgresStorage) CreateUser(ctx context.Context, login string, passwordHash []byte) (int64, error) {
	const q = `
		INSERT INTO users (login, password_hash)
		VALUES ($1, $2)
		RETURNING id
	`
	var id int64
	err := p.dbExec.QueryRowContext(ctx, q, login, passwordHash).Scan(&id)
	if err != nil {
		if pgerrors.IsUniqueViolation(err) {
			return 0, ErrLoginTaken
		}
		return 0, err
	}
	return id, nil
}

func (p *PostgresStorage) GetUserByLogin(
	ctx context.Context,
	login string,
) (*models.User, error) {
	const q = `
		SELECT id, login, password_hash, created_at
		FROM users
		WHERE login = $1
	`

	u := &models.User{}
	err := p.dbExec.QueryRowContext(ctx, q, login).
		Scan(&u.ID, &u.Login, &u.PasswordHash, &u.CreatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}
