package middleware

import (
	"errors"
	"net/http"
	"strings"

	"go.uber.org/zap"

	jwtV5 "github.com/golang-jwt/jwt/v5"
	"github.com/tigranqic/go-musthave-diploma-tpl/internal/auth"
	"github.com/tigranqic/go-musthave-diploma-tpl/internal/auth/jwt"
)

func Auth(authSvc jwt.Service, log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var h string

			h = r.Header.Get("Authorization")
			if h == "" {
				if c, err := r.Cookie("Authorization"); err == nil {
					h = "Bearer " + c.Value
				}
			}

			if h == "" {
				log.Error(
					"required header not provided",
					zap.String("path", r.URL.Path),
					zap.String("method", r.Method),
					zap.String("remote_addr", r.RemoteAddr),
				)
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			parts := strings.SplitN(h, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				log.Error(
					"auth failed token malformed",
					zap.String("path", r.URL.Path),
					zap.String("method", r.Method),
					zap.String("remote_addr", r.RemoteAddr),
				)
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			identity, err := authSvc.Authenticate(r.Context(), parts[1])
			if err != nil {

				switch {
				case errors.Is(err, jwtV5.ErrTokenExpired):
					log.Info(
						"token expired",
						zap.Error(err),
						zap.String("path", r.URL.Path),
						zap.String("method", r.Method),
						zap.String("remote_addr", r.RemoteAddr),
					)
					http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)

				case errors.Is(err, jwtV5.ErrTokenSignatureInvalid),
					errors.Is(err, jwtV5.ErrTokenMalformed),
					errors.Is(err, jwtV5.ErrTokenInvalidClaims):
					log.Warn(
						"invalid token",
						zap.Error(err),
						zap.String("path", r.URL.Path),
						zap.String("method", r.Method),
						zap.String("remote_addr", r.RemoteAddr),
					)
					http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)

				default:
					log.Error(
						"auth internal error",
						zap.Error(err),
						zap.String("path", r.URL.Path),
						zap.String("method", r.Method),
						zap.String("remote_addr", r.RemoteAddr),
					)
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}

				return
			}

			if identity == nil {
				log.Error(
					"auth returned nil identity",
					zap.String("path", r.URL.Path),
					zap.String("method", r.Method),
					zap.String("remote_addr", r.RemoteAddr),
				)
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			ctx := auth.With(r.Context(), *identity)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
