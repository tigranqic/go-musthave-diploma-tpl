package accrual_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tigranqic/go-musthave-diploma-tpl/internal/accrual"
	"github.com/tigranqic/go-musthave-diploma-tpl/internal/model"
	"go.uber.org/zap"
)

type mockStore struct {
	calls int32
}

func (m *mockStore) GetOrdersForAccrual(ctx context.Context) ([]model.Order, error) {
	return []model.Order{{Number: "123"}}, nil
}

func (m *mockStore) UpdateOrderAccrual(ctx context.Context, number, status string, accrual *float64) error {
	atomic.AddInt32(&m.calls, 1)
	return nil
}

func (m *mockStore) CreateUser(ctx context.Context, login string, passwordHash []byte) (int64, error) {
	return 1, nil
}
func (m *mockStore) GetUserByLogin(ctx context.Context, login string) (*model.User, error) {
	return &model.User{ID: 1, Login: login}, nil
}
func (m *mockStore) CreateOrder(ctx context.Context, order *model.Order) error {
	return nil
}
func (m *mockStore) GetOrder(ctx context.Context, number string) (*model.Order, error) {
	return &model.Order{Number: number}, nil
}
func (m *mockStore) GetOrdersByUserID(ctx context.Context, userID int64) ([]*model.Order, error) {
	return []*model.Order{{Number: "123"}}, nil
}
func (m *mockStore) GetBalance(ctx context.Context, userID int64) (*model.Balance, error) {
	return &model.Balance{Current: 100, Withdrawn: 0}, nil
}
func (m *mockStore) Withdraw(ctx context.Context, userID int64, order string, sum float64) error {
	return nil
}
func (m *mockStore) ListWithdrawals(ctx context.Context, userID int64) ([]model.Withdrawal, error) {
	return []model.Withdrawal{}, nil
}
func (m *mockStore) UpdateBalance(ctx context.Context, userID int64, current float64, withdrawn float64) error {
	return nil
}

func TestWorker_Handles429AndSleep(t *testing.T) {
	var counter int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.LoadInt32(&counter) == 0 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			atomic.AddInt32(&counter, 1)
			return
		}
		resp := map[string]interface{}{"order": "123", "status": "processed", "accrual": 10.0}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	logger, _ := zap.NewDevelopment()
	store := &mockStore{}

	worker := accrual.NewWorker(store, server.URL, logger)
	worker.PollInterval = 50 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	start := time.Now()
	go worker.Run(ctx)

	<-ctx.Done()
	elapsed := time.Since(start)

	if elapsed < 1*time.Second {
		t.Fatalf("expected sleep for at least 1 second after 429, got %v", elapsed)
	}
	if atomic.LoadInt32(&store.calls) == 0 {
		t.Fatalf("expected at least 1 successful update after 429")
	}
}

func TestWorker_GracefulShutdown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger, _ := zap.NewDevelopment()
	store := &mockStore{}

	worker := accrual.NewWorker(store, server.URL, logger)
	worker.PollInterval = 50 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	go worker.Run(ctx)
	time.Sleep(100 * time.Millisecond)

	cancel()

	time.Sleep(100 * time.Millisecond)
}

func TestWorker_MultipleWorkers_Handle429(t *testing.T) {
	var callCount int32
	first429 := make(chan struct{}, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.LoadInt32(&callCount) == 0 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			atomic.AddInt32(&callCount, 1)
			first429 <- struct{}{}
			return
		}
		resp := map[string]interface{}{"order": "123", "status": "processed", "accrual": 10.0}
		_ = json.NewEncoder(w).Encode(resp)
		atomic.AddInt32(&callCount, 1)
	}))
	defer server.Close()

	logger, _ := zap.NewDevelopment()
	store := &mockStore{}

	worker := accrual.NewWorker(store, server.URL, logger)
	worker.PollInterval = 20 * time.Millisecond
	worker.WorkerCount = 3

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go worker.Run(ctx)

	select {
	case <-first429:
		time.Sleep(20 * time.Millisecond)

		sleepUntil := worker.SleepUntil.Load()
		if sleepUntil == 0 {
			t.Fatal("SleepUntil not set after 429")
		}

		remaining := time.Until(time.Unix(0, sleepUntil))
		if remaining <= 0 {
			t.Fatal("SleepUntil already expired, expected workers to sleep")
		}
		t.Logf("workers will sleep for %v after 429", remaining)

	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected first 429 request")
	}

	time.Sleep(50 * time.Millisecond)

	if atomic.LoadInt32(&callCount) > 5 {
		t.Fatalf("expected workers to stop quickly after 429, got %d calls", callCount)
	}

	<-ctx.Done()

	if atomic.LoadInt32(&store.calls) == 0 {
		t.Fatalf("expected at least 1 successful update after 429")
	}
}
