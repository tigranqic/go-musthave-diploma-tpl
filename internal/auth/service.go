package auth

import "context"

type Identity struct {
	UserID int64
}

type Service interface {
	GenerateToken(userID int64) (string, error)
	Authenticate(ctx context.Context, token string) (*Identity, error)
}
