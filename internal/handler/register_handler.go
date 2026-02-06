package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/tigranqic/go-musthave-diploma-tpl/internal/repository"
)

type registerRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (h *Handler) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	defer func() {
		_ = r.Body.Close()
	}()
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB

	var req registerRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	req.Login = strings.TrimSpace(req.Login)

	if req.Login == "" || len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "invalid login or password")
		return
	}

	hash, err := bcrypt.GenerateFromPassword(
		[]byte(req.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		h.log.Error("password hash failed", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	userID, err := h.store.CreateUser(r.Context(), req.Login, hash)
	if err != nil {
		if errors.Is(err, repository.ErrLoginTaken) {
			writeError(w, http.StatusConflict, "login already taken")
			return
		}
		h.log.Error("create user failed", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	token, err := h.authSvc.GenerateToken(userID)
	if err != nil {
		h.log.Error("token generation failed", zap.Error(err), zap.Int64("user_id", userID))
		writeError(w, http.StatusInternalServerError, "internal error")
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

	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}
