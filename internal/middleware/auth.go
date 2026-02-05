package middleware

import (
	"net/http"
	"strings"

	"go.uber.org/zap"

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
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			parts := strings.SplitN(h, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			identity, err := authSvc.Authenticate(r.Context(), parts[1])
			if err != nil {
				log.Warn("auth failed", zap.Error(err))
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			if identity == nil {
				log.Error("auth returned nil identity without error")
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			ctx := auth.With(r.Context(), *identity)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
