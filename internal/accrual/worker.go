package accrual

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tigranqic/go-musthave-diploma-tpl/internal/httpclient"
	"github.com/tigranqic/go-musthave-diploma-tpl/internal/repository"
	"go.uber.org/zap"
)

type Worker struct {
	store   repository.Storage
	client  *httpclient.RetryClient
	baseURL string
	logger  *zap.Logger

	PollInterval time.Duration
	WorkerCount  int

	SleepUntil atomic.Int64
}

func NewWorker(store repository.Storage, baseURL string, logger *zap.Logger) *Worker {
	return &Worker{
		store:        store,
		client:       httpclient.NewRetryClient(),
		baseURL:      baseURL,
		logger:       logger,
		PollInterval: 500 * time.Millisecond,
		WorkerCount:  5,
	}
}

func (w *Worker) Run(ctx context.Context) {
	wg := &sync.WaitGroup{}
	for i := 0; i < w.WorkerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.workerLoop(ctx)
		}()
	}
	<-ctx.Done()
	wg.Wait()
	w.logger.Info("accrual worker stopped")
}

func (w *Worker) workerLoop(ctx context.Context) {
	for {
		if ctx.Err() != nil || w.waitIfSleeping(ctx) {
			return
		}

		orders, err := w.store.GetOrdersForAccrual(ctx)
		if err != nil {
			w.logger.Error("failed to get orders", zap.Error(err))
			return
		}

		if len(orders) == 0 {
			time.Sleep(w.PollInterval)
			continue
		}

		for _, order := range orders {
			if ctx.Err() != nil || w.waitIfSleeping(ctx) {
				return
			}
			if err := w.processOrder(ctx, order.Number); err != nil {
				w.logger.Error("failed to process order", zap.String("order", order.Number), zap.Error(err))
			}
		}
	}
}

func (w *Worker) processOrder(ctx context.Context, number string) error {
	w.logger.Info("processing order", zap.String("baseURL", w.baseURL), zap.String("order", number))
	url := fmt.Sprintf("%s/api/orders/%s", w.baseURL, number)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := w.client.Do(ctx, req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	switch resp.StatusCode {
	case http.StatusTooManyRequests:
		w.handle429(resp)
		return nil
	case http.StatusNoContent:
		return nil
	case http.StatusOK:
		var data struct {
			Order   string   `json:"order"`
			Status  string   `json:"status"`
			Accrual *float64 `json:"accrual,omitempty"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			return err
		}
		return w.store.UpdateOrderAccrual(ctx, data.Order, data.Status, data.Accrual)
	default:
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
}

func (w *Worker) handle429(resp *http.Response) {
	retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
	until := time.Now().Add(retryAfter).UnixNano()

	for {
		old := w.SleepUntil.Load()
		if until <= old {
			return
		}
		if w.SleepUntil.CompareAndSwap(old, until) {
			w.logger.Warn("received 429, sleeping all workers", zap.Duration("sleep", retryAfter))
			return
		}
	}
}

func (w *Worker) waitIfSleeping(ctx context.Context) bool {
	until := w.SleepUntil.Load()
	if until == 0 {
		return false
	}
	remaining := time.Until(time.Unix(0, until))
	if remaining <= 0 {
		return false
	}
	select {
	case <-time.After(remaining):
		return false
	case <-ctx.Done():
		return true
	}
}

func parseRetryAfter(header string) time.Duration {
	if header == "" {
		return 2 * time.Second
	}
	if sec, err := strconv.Atoi(header); err == nil {
		return time.Duration(sec) * time.Second
	}
	return 2 * time.Second
}
