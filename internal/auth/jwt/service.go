package jwt

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/tigranqic/go-musthave-diploma-tpl/internal/auth"
)

type Service struct {
	secret []byte
	ttl    time.Duration
}

func New(secret []byte, ttl time.Duration) *Service {
	return &Service{secret: secret, ttl: ttl}
}

type claims struct {
	UserID int64 `json:"user_id"`
	jwt.RegisteredClaims
}

func (s *Service) GenerateToken(userID int64) (string, error) {
	c := claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	return token.SignedString(s.secret)
}

func (s *Service) Authenticate(
	_ context.Context,
	tokenStr string,
) (*auth.Identity, error) {
	c := &claims{}
	token, err := jwt.ParseWithClaims(
		tokenStr,
		c,
		func(t *jwt.Token) (interface{}, error) {
			return s.secret, nil
		},
	)

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, auth.ErrExpiredToken
		}
		return nil, auth.ErrInvalidToken
	}

	if !token.Valid {
		return nil, auth.ErrInvalidToken
	}

	return &auth.Identity{UserID: c.UserID}, nil
}
