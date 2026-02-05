package handler

import (
	"net/http"

	"github.com/tigranqic/go-musthave-diploma-tpl/internal/auth"
	"go.uber.org/zap"
)

func (h *Handler) GetBalanceHandler(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.From(r.Context())
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	b, err := h.store.GetBalance(r.Context(), identity.UserID)
	if err != nil {
		h.log.Error("get balance failed", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), 500)
		return
	}

	writeJSON(w, http.StatusOK, b)
}
