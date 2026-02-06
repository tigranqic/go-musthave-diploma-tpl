package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/tigranqic/go-musthave-diploma-tpl/internal/auth"
	models "github.com/tigranqic/go-musthave-diploma-tpl/internal/model"
	"github.com/tigranqic/go-musthave-diploma-tpl/internal/repository"
	"github.com/tigranqic/go-musthave-diploma-tpl/pkg/utils"
	"go.uber.org/zap"
)

const (
	msgOrderExists        = "order number has already been submitted by this user"
	msgOrderAccepted      = "new order number accepted for processing"
	msgBadRequest         = "invalid request format"
	msgOrderOwnedByOther  = "order number has already been submitted by another user"
	msgInvalidOrderNumber = "invalid order number format"
)

func (h *Handler) CreateOrderHandler(w http.ResponseWriter, r *http.Request) {
	defer func() {
		_ = r.Body.Close()
	}()
	identity, ok := auth.From(r.Context())
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil || len(body) == 0 {
		http.Error(w, msgBadRequest, http.StatusBadRequest)
		return
	}

	number := strings.TrimSpace(string(body))

	if !isDigits(number) {
		http.Error(w, msgBadRequest, http.StatusBadRequest)
		return
	}

	if !utils.ValidLuhn(number) {
		http.Error(w, msgInvalidOrderNumber, http.StatusUnprocessableEntity)
		return
	}

	order := &models.Order{
		Number: number,
		UserID: identity.UserID,
		Status: string(models.OrderStatusNew),
	}

	err = h.store.CreateOrder(r.Context(), order)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrOrderExists):
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(msgOrderExists))
			if err != nil {
				h.log.Error("failed to write response", zap.Error(err))
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
		case errors.Is(err, repository.ErrOrderOwnedByOther):
			http.Error(w, msgOrderOwnedByOther, http.StatusConflict)

		default:
			h.log.Error("create order failed", zap.Error(err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": msgOrderAccepted})
}

func (h *Handler) ListOrdersHandler(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.From(r.Context())
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	orders, err := h.store.GetOrdersByUserID(r.Context(), identity.UserID)
	if err != nil {
		h.log.Error("get orders failed", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if len(orders) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	type responseItem struct {
		Number     string   `json:"number"`
		Status     string   `json:"status"`
		Accrual    *float64 `json:"accrual,omitempty"`
		UploadedAt string   `json:"uploaded_at"`
	}

	resp := make([]responseItem, 0, len(orders))

	for _, order := range orders {
		item := responseItem{
			Number:     order.Number,
			Status:     order.Status,
			UploadedAt: order.UploadedAt.Format(time.RFC3339),
		}

		if order.Accrual != nil {
			item.Accrual = order.Accrual
		}

		resp = append(resp, item)
	}

	sort.Slice(resp, func(i, j int) bool {
		return orders[i].UploadedAt.After(orders[j].UploadedAt)
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
