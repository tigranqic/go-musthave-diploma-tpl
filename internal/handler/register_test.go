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
	"github.com/tigranqic/go-musthave-diploma-tpl/internal/auth/jwt"
	"github.com/tigranqic/go-musthave-diploma-tpl/internal/config"
	"github.com/tigranqic/go-musthave-diploma-tpl/internal/handler"
	"github.com/tigranqic/go-musthave-diploma-tpl/internal/model"
	"github.com/tigranqic/go-musthave-diploma-tpl/internal/repository"
	"go.uber.org/zap"
)

type mockRegisterStorage struct {
	createUserFn func(ctx context.Context, login string, hash []byte) (int64, error)
}

func (m *mockRegisterStorage) CreateUser(ctx context.Context, login string, hash []byte) (int64, error) {
	if m.createUserFn != nil {
		return m.createUserFn(ctx, login, hash)
	}
	return 0, nil
}

func (m *mockRegisterStorage) GetUserByLogin(ctx context.Context, login string) (*model.User, error) {
	return nil, nil
}
func (m *mockRegisterStorage) CreateOrder(ctx context.Context, order *model.Order) error {
	return nil
}
func (m *mockRegisterStorage) GetOrder(ctx context.Context, number string) (*model.Order, error) {
	return nil, nil
}
func (m *mockRegisterStorage) GetOrdersByUserID(ctx context.Context, userID int64) ([]*model.Order, error) {
	return nil, nil
}
func (m *mockRegisterStorage) GetBalance(ctx context.Context, userID int64) (*model.Balance, error) {
	return nil, nil
}
func (m *mockRegisterStorage) GetOrdersForAccrual(ctx context.Context) ([]model.Order, error) {
	return nil, nil
}
func (m *mockRegisterStorage) UpdateOrderAccrual(ctx context.Context, order string, status string, accrual *float64) error {
	return nil
}
func (m *mockRegisterStorage) UpdateBalance(ctx context.Context, userID int64, current float64, withdrawn float64) error {
	return nil
}
func (m *mockRegisterStorage) Withdraw(ctx context.Context, userID int64, order string, sum float64) error {
	return nil
}
func (m *mockRegisterStorage) ListWithdrawals(ctx context.Context, userID int64) ([]model.Withdrawal, error) {
	return nil, nil
}

type mockAuthSvc struct {
	generateFn     func(userID int64) (string, error)
	authenticateFn func(ctx context.Context, token string) (*auth.Identity, error)
}

func (m *mockAuthSvc) GenerateToken(userID int64) (string, error) {
	if m.generateFn != nil {
		return m.generateFn(userID)
	}
	return "mocktoken", nil
}

func (m *mockAuthSvc) Authenticate(ctx context.Context, token string) (*auth.Identity, error) {
	if m.authenticateFn != nil {
		return m.authenticateFn(ctx, token)
	}
	return &auth.Identity{UserID: 1}, nil
}

func newRegisterHandler(store repository.Storage, authSvc jwt.Service) *handler.Handler {
	return handler.NewHandler(store, nil, zap.NewNop(), authSvc, config.Config{})
}

func TestRegisterHandler(t *testing.T) {
	tests := []struct {
		name       string
		body       interface{}
		createFn   func(ctx context.Context, login string, hash []byte) (int64, error)
		authFn     func(userID int64) (string, error)
		wantStatus int
		wantInBody string
	}{
		{
			name: "Success",
			body: map[string]string{"login": "user1", "password": "12345678"},
			createFn: func(ctx context.Context, login string, hash []byte) (int64, error) {
				return 1, nil
			},
			authFn:     func(userID int64) (string, error) { return "token123", nil },
			wantStatus: http.StatusOK,
			wantInBody: "token123",
		},
		{
			name:       "Invalid JSON",
			body:       "{invalid",
			wantStatus: http.StatusBadRequest,
			wantInBody: "invalid JSON",
		},
		{
			name:       "Short Password",
			body:       map[string]string{"login": "user1", "password": "123"},
			wantStatus: http.StatusBadRequest,
			wantInBody: "invalid login or password",
		},
		{
			name:       "Empty Login",
			body:       map[string]string{"login": "", "password": "12345678"},
			wantStatus: http.StatusBadRequest,
			wantInBody: "invalid login or password",
		},
		{
			name: "Login Already Taken",
			body: map[string]string{"login": "user1", "password": "12345678"},
			createFn: func(ctx context.Context, login string, hash []byte) (int64, error) {
				return 0, repository.ErrLoginTaken
			},
			wantStatus: http.StatusConflict,
			wantInBody: "login already taken",
		},
		{
			name: "Internal Error in CreateUser",
			body: map[string]string{"login": "user1", "password": "12345678"},
			createFn: func(ctx context.Context, login string, hash []byte) (int64, error) {
				return 0, errors.New("db error")
			},
			wantStatus: http.StatusInternalServerError,
			wantInBody: "internal error",
		},
		{
			name:       "Internal Error in Token Generation",
			body:       map[string]string{"login": "user1", "password": "12345678"},
			createFn:   func(ctx context.Context, login string, hash []byte) (int64, error) { return 1, nil },
			authFn:     func(userID int64) (string, error) { return "", errors.New("token error") },
			wantStatus: http.StatusInternalServerError,
			wantInBody: "internal error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockRegisterStorage{createUserFn: tt.createFn}
			authSvc := &mockAuthSvc{
				generateFn: tt.authFn,
			}
			h := newRegisterHandler(store, authSvc)

			var req *http.Request
			if str, ok := tt.body.(string); ok {
				req = httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader([]byte(str)))
			} else {
				buf := new(bytes.Buffer)
				_ = json.NewEncoder(buf).Encode(tt.body)
				req = httptest.NewRequest(http.MethodPost, "/register", buf)
			}

			w := httptest.NewRecorder()
			h.RegisterHandler(w, req)

			resp := w.Result()
			defer func() {
				_ = resp.Body.Close()
			}()
			assert.Equal(t, tt.wantStatus, resp.StatusCode)

			var body map[string]string
			_ = json.NewDecoder(resp.Body).Decode(&body)
			if tt.wantInBody != "" {
				found := false
				for _, v := range body {
					if v == tt.wantInBody {
						found = true
						break
					}
				}
				assert.True(t, found, "response body should contain %q", tt.wantInBody)
			}
		})
	}
}
