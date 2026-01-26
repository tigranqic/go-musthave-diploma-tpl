package middleware

import (
	"net/http"
	"strings"

	"go.uber.org/zap"

	"github.com/tigranqic/go-musthave-diploma-tpl/internal/auth"
)

func Auth(authSvc auth.Service, log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")

			if h == "" {
				if c, err := r.Cookie("Authorization"); err == nil {
					h = "Bearer " + c.Value
				}
			}

			log.Debug("auth header", zap.String("header", h))
			if h == "" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			parts := strings.SplitN(h, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			identity, err := authSvc.Authenticate(r.Context(), parts[1])
			if err != nil {
				log.Warn("auth failed", zap.Error(err))
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			ctx := auth.With(r.Context(), *identity)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
