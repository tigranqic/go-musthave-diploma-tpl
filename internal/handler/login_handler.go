package handler

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

func (h *Handler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	user, err := h.store.GetUserByLogin(r.Context(), req.Login)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	if bcrypt.CompareHashAndPassword(
		user.PasswordHash,
		[]byte(req.Password),
	) != nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	token, err := h.authSvc.GenerateToken(user.ID)
	if err != nil {
		h.log.Error("generate token failed", zap.Error(err), zap.Int64("user_id", user.ID))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "Authorization",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   24 * 60 * 60,
	})

	if err := json.NewEncoder(w).Encode(map[string]string{
		"token": token,
	}); err != nil {
		h.log.Error("encode token failed", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}
