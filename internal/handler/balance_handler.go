package handler

import (
	"net/http"

	"github.com/tigranqic/go-musthave-diploma-tpl/internal/auth"
)

func (h *Handler) GetBalanceHandler(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.From(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	b, err := h.store.GetBalance(r.Context(), identity.UserID)
	if err != nil {
		http.Error(w, "internal error", 500)
		return
	}

	writeJSON(w, http.StatusOK, b)
}
