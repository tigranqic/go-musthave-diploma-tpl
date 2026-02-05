package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tigranqic/go-musthave-diploma-tpl/internal/auth"
	"github.com/tigranqic/go-musthave-diploma-tpl/internal/config"
	"github.com/tigranqic/go-musthave-diploma-tpl/internal/handler"
	"github.com/tigranqic/go-musthave-diploma-tpl/internal/model"
	"github.com/tigranqic/go-musthave-diploma-tpl/internal/repository"
	"go.uber.org/zap"
)

type mockStorage struct {
	withdrawFn        func(ctx context.Context, userID int64, order string, sum float64) error
	listWithdrawalsFn func(ctx context.Context, userID int64) ([]model.Withdrawal, error)
}

func (m *mockStorage) Withdraw(ctx context.Context, userID int64, order string, sum float64) error {
	if m.withdrawFn != nil {
		return m.withdrawFn(ctx, userID, order, sum)
	}
	return nil
}

func (m *mockStorage) ListWithdrawals(ctx context.Context, userID int64) ([]model.Withdrawal, error) {
	if m.listWithdrawalsFn != nil {
		return m.listWithdrawalsFn(ctx, userID)
	}
	return nil, nil
}

func (m *mockStorage) CreateUser(ctx context.Context, login string, passwordHash []byte) (int64, error) {
	return 0, nil
}
func (m *mockStorage) GetUserByLogin(ctx context.Context, login string) (*model.User, error) {
	return nil, nil
}
func (m *mockStorage) CreateOrder(ctx context.Context, order *model.Order) error { return nil }
func (m *mockStorage) GetOrder(ctx context.Context, number string) (*model.Order, error) {
	return nil, nil
}
func (m *mockStorage) GetOrdersByUserID(ctx context.Context, userID int64) ([]*model.Order, error) {
	return nil, nil
}
func (m *mockStorage) GetBalance(ctx context.Context, userID int64) (*model.Balance, error) {
	return nil, nil
}
func (m *mockStorage) GetOrdersForAccrual(ctx context.Context) ([]model.Order, error) {
	return nil, nil
}
func (m *mockStorage) UpdateOrderAccrual(ctx context.Context, order string, status string, accrual *float64) error {
	return nil
}
func (m *mockStorage) UpdateBalance(ctx context.Context, userID int64, current float64, withdrawn float64) error {
	return nil
}

func newHandler(store repository.Storage) *handler.Handler {
	return handler.NewHandler(store, nil, zap.NewNop(), nil, config.Config{})
}

func newAuthRequest(t *testing.T, method, url string, body interface{}, userID int64) *http.Request {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, url, &buf)
	if userID != 0 {
		req = req.WithContext(auth.With(req.Context(), auth.Identity{UserID: userID}))
	}
	return req
}

func TestWithdrawHandlerTable(t *testing.T) {
	validOrder := "79927398713"
	tests := []struct {
		name       string
		body       interface{}
		userID     int64
		withdrawFn func(ctx context.Context, userID int64, order string, sum float64) error
		wantStatus int
	}{
		{"Success", map[string]interface{}{"order": validOrder, "sum": 100.0}, 1,
			func(ctx context.Context, userID int64, order string, sum float64) error { return nil }, http.StatusOK},
		{"Unauthorized", map[string]interface{}{"order": validOrder, "sum": 100.0}, 0, nil, http.StatusUnauthorized},
		{"Invalid JSON", "{invalid", 1, nil, http.StatusBadRequest},
		{"Invalid Luhn", map[string]interface{}{"order": "1234567890", "sum": 100.0}, 1, nil, http.StatusUnprocessableEntity},
		{"InsufficientFunds", map[string]interface{}{"order": validOrder, "sum": 100.0}, 1,
			func(ctx context.Context, userID int64, order string, sum float64) error {
				return repository.ErrInsufficientFunds
			}, http.StatusPaymentRequired},
		{"InternalError", map[string]interface{}{"order": validOrder, "sum": 100.0}, 1,
			func(ctx context.Context, userID int64, order string, sum float64) error {
				return errors.New("db error")
			}, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockStorage{withdrawFn: tt.withdrawFn}
			h := newHandler(store)

			var req *http.Request
			if bodyStr, ok := tt.body.(string); ok {
				req = httptest.NewRequest(http.MethodPost, "/withdraw", bytes.NewReader([]byte(bodyStr)))
				if tt.userID != 0 {
					req = req.WithContext(auth.With(req.Context(), auth.Identity{UserID: tt.userID}))
				}
			} else {
				req = newAuthRequest(t, http.MethodPost, "/withdraw", tt.body, tt.userID)
			}

			w := httptest.NewRecorder()
			h.WithdrawHandler(w, req)

			resp := w.Result()
			defer func() {
				_ = resp.Body.Close()
			}()
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestWithdrawalsHandlerTable(t *testing.T) {
	tests := []struct {
		name       string
		userID     int64
		listFn     func(ctx context.Context, userID int64) ([]model.Withdrawal, error)
		wantStatus int
		wantLen    int
	}{
		{"Success", 1,
			func(ctx context.Context, userID int64) ([]model.Withdrawal, error) {
				return []model.Withdrawal{
					{Order: "123", Sum: 50},
					{Order: "456", Sum: 100},
				}, nil
			}, http.StatusOK, 2},
		{"NoWithdrawals", 1,
			func(ctx context.Context, userID int64) ([]model.Withdrawal, error) { return nil, nil },
			http.StatusNoContent, 0},
		{"Unauthorized", 0, nil, http.StatusUnauthorized, 0},
		{"InternalError", 1,
			func(ctx context.Context, userID int64) ([]model.Withdrawal, error) {
				return nil, errors.New("db error")
			}, http.StatusInternalServerError, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockStorage{listWithdrawalsFn: tt.listFn}
			h := newHandler(store)

			req := newAuthRequest(t, http.MethodGet, "/withdrawals", nil, tt.userID)
			if tt.userID == 0 {
				req = httptest.NewRequest(http.MethodGet, "/withdrawals", nil)
			}

			w := httptest.NewRecorder()
			h.WithdrawalsHandler(w, req)

			resp := w.Result()
			defer func() {
				_ = resp.Body.Close()
			}()
			assert.Equal(t, tt.wantStatus, resp.StatusCode)

			if tt.wantStatus == http.StatusOK {
				var ws []model.Withdrawal
				err := json.NewDecoder(resp.Body).Decode(&ws)
				assert.NoError(t, err)
				assert.Len(t, ws, tt.wantLen)
			}
		})
	}
}
