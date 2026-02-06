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
	"github.com/tigranqic/go-musthave-diploma-tpl/internal/repository/mocks"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mocks.NewMockStorage(ctrl)
	mockStore.EXPECT().
		GetOrdersForAccrual(gomock.Any()).
		Return([]model.Order{{Number: "123"}}, nil).
		AnyTimes()

	mockStore.EXPECT().
		UpdateOrderAccrual(gomock.Any(), "123", "processed", gomock.Any()).
		AnyTimes()

	worker := accrual.NewWorker(mockStore, server.URL, logger)
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
}

func TestWorker_GracefulShutdown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger, _ := zap.NewDevelopment()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mocks.NewMockStorage(ctrl)
	mockStore.EXPECT().
		GetOrdersForAccrual(gomock.Any()).
		Return([]model.Order{{Number: "123"}}, nil).
		AnyTimes()

	worker := accrual.NewWorker(mockStore, server.URL, logger)
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mocks.NewMockStorage(ctrl)
	mockStore.EXPECT().
		GetOrdersForAccrual(gomock.Any()).
		Return([]model.Order{{Number: "123"}}, nil).
		AnyTimes()

	mockStore.EXPECT().
		UpdateOrderAccrual(gomock.Any(), "123", "processed", gomock.Any()).
		AnyTimes()

	worker := accrual.NewWorker(mockStore, server.URL, logger)
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
}
