package handler

import (
	"encoding/json"
	"net/http"

	"github.com/tigranqic/go-musthave-diploma-tpl/internal/auth"
	"github.com/tigranqic/go-musthave-diploma-tpl/internal/repository"
	"github.com/tigranqic/go-musthave-diploma-tpl/pkg/utils"
	"go.uber.org/zap"
)

const (
	msgInvalidOrder      = "invalid order"
	msgInsufficientFunds = "not enough funds"
)

type withdrawRequest struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}

func (h *Handler) WithdrawHandler(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.From(r.Context())
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	var req withdrawRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusBadRequest)
		return
	}

	if !utils.ValidLuhn(req.Order) {
		http.Error(w, msgInvalidOrder, http.StatusUnprocessableEntity)
		return
	}

	err := h.store.Withdraw(r.Context(), identity.UserID, req.Order, req.Sum)
	if err == repository.ErrInsufficientFunds {
		http.Error(w, msgInsufficientFunds, http.StatusPaymentRequired)
		return
	}
	if err != nil {
		h.log.Error("withdraw failed", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) WithdrawalsHandler(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.From(r.Context())
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	ws, err := h.store.ListWithdrawals(r.Context(), identity.UserID)
	if err != nil {
		h.log.Error("withdraw failed", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if len(ws) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	writeJSON(w, http.StatusOK, ws)
}
