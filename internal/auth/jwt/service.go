package jwt

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"

	"github.com/tigranqic/go-musthave-diploma-tpl/internal/auth"
)

type JWTService struct {
	log    *zap.Logger
	secret []byte
	ttl    time.Duration
}

type Service interface {
	GenerateToken(userID int64) (string, error)
	Authenticate(ctx context.Context, token string) (*auth.Identity, error)
}

func New(log *zap.Logger, secret []byte, ttl time.Duration) Service {
	return &JWTService{
		log:    log,
		secret: secret,
		ttl:    ttl,
	}
}

type claims struct {
	UserID int64 `json:"user_id"`
	jwt.RegisteredClaims
}

func (s *JWTService) GenerateToken(userID int64) (string, error) {
	now := time.Now()
	c := claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	signed, err := token.SignedString(s.secret)
	if err != nil {
		s.log.Error("failed to sign token", zap.Error(err), zap.Int64("user_id", userID))
		return "", err
	}

	return signed, nil
}

func (s *JWTService) Authenticate(ctx context.Context, tokenStr string) (*auth.Identity, error) {
	c := &claims{}

	token, err := jwt.ParseWithClaims(tokenStr, c, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.secret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			s.log.Warn("token expired", zap.Error(err))
			return nil, auth.ErrExpiredToken
		}
		s.log.Warn("invalid token", zap.Error(err))
		return nil, auth.ErrInvalidToken
	}

	if !token.Valid {
		s.log.Warn("invalid token: token is not valid")
		return nil, auth.ErrInvalidToken
	}

	return &auth.Identity{UserID: c.UserID}, nil
}
