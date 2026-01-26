package accrual

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/tigranqic/go-musthave-diploma-tpl/internal/repository"
	"go.uber.org/zap"
)

type Worker struct {
	store        repository.Storage
	client       *http.Client
	baseURL      string
	logger       *zap.Logger
	pollInterval time.Duration
}

func NewWorker(store repository.Storage, baseURL string, logger *zap.Logger) *Worker {
	return &Worker{
		store:        store,
		client:       &http.Client{Timeout: 10 * time.Second},
		baseURL:      baseURL,
		logger:       logger,
		pollInterval: 2 * time.Second,
	}
}

type accrualResponse struct {
	Order   string   `json:"order"`
	Status  string   `json:"status"`
	Accrual *float64 `json:"accrual,omitempty"`
}

func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.processOrders(ctx)
		case <-ctx.Done():
			w.logger.Info("accrual worker stopped")
			return
		}
	}
}

func (w *Worker) processOrders(ctx context.Context) {
	orders, err := w.store.GetOrdersForAccrual(ctx)
	if err != nil {
		w.logger.Error("failed to fetch orders for accrual", zap.Error(err))
		return
	}

	for _, o := range orders {
		w.processOrder(ctx, o.Number)
	}
}

func (w *Worker) processOrder(ctx context.Context, number string) {
	url := fmt.Sprintf("%s/api/orders/%s", w.baseURL, number)

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := w.client.Do(req)
	if err != nil {
		w.logger.Error("failed request to accrual system", zap.String("order", number), zap.Error(err))
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	switch resp.StatusCode {
	case 200:
		var data accrualResponse
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			w.logger.Error("failed to decode accrual response", zap.Error(err))
			return
		}
		if err := w.store.UpdateOrderAccrual(ctx, data.Order, data.Status, data.Accrual); err != nil {
			w.logger.Error("failed to update order accrual", zap.String("order", data.Order), zap.Error(err))
		}
		w.logger.Warn("accrual system update", zap.String("order", number))

	case 204:

	case 429:
		w.logger.Warn("too many requests to accrual system", zap.String("order", number))
	default:
		w.logger.Error("unexpected status from accrual system", zap.String("order", number), zap.Int("status", resp.StatusCode))
	}
}
