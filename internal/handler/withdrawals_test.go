package handler_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/tigranqic/go-musthave-diploma-tpl/internal/auth"
	"github.com/tigranqic/go-musthave-diploma-tpl/internal/config"
	"github.com/tigranqic/go-musthave-diploma-tpl/internal/handler"
	"github.com/tigranqic/go-musthave-diploma-tpl/internal/model"
	"github.com/tigranqic/go-musthave-diploma-tpl/internal/repository"
	repoMocks "github.com/tigranqic/go-musthave-diploma-tpl/internal/repository/mocks"
	"go.uber.org/zap"
)

func newHandler(store *repoMocks.MockStorage) *handler.Handler {
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	validOrder := "79927398713"

	tests := []struct {
		name       string
		body       interface{}
		userID     int64
		setupMocks func(store *repoMocks.MockStorage)
		wantStatus int
	}{
		{
			name:   "Success",
			body:   map[string]interface{}{"order": validOrder, "sum": 100.0},
			userID: 1,
			setupMocks: func(store *repoMocks.MockStorage) {
				store.EXPECT().
					Withdraw(gomock.Any(), int64(1), validOrder, 100.0).
					Return(nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "Unauthorized",
			body:       map[string]interface{}{"order": validOrder, "sum": 100.0},
			userID:     0,
			setupMocks: func(store *repoMocks.MockStorage) {},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "Invalid JSON",
			body:       "{invalid",
			userID:     1,
			setupMocks: func(store *repoMocks.MockStorage) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "InsufficientFunds",
			body:   map[string]interface{}{"order": validOrder, "sum": 100.0},
			userID: 1,
			setupMocks: func(store *repoMocks.MockStorage) {
				store.EXPECT().
					Withdraw(gomock.Any(), int64(1), validOrder, 100.0).
					Return(repository.ErrInsufficientFunds)
			},
			wantStatus: http.StatusPaymentRequired,
		},
		{
			name:   "InternalError",
			body:   map[string]interface{}{"order": validOrder, "sum": 100.0},
			userID: 1,
			setupMocks: func(store *repoMocks.MockStorage) {
				store.EXPECT().
					Withdraw(gomock.Any(), int64(1), validOrder, 100.0).
					Return(errors.New("db error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := repoMocks.NewMockStorage(ctrl)
			tt.setupMocks(store)

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
			defer resp.Body.Close()
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestWithdrawalsHandlerTable(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name       string
		userID     int64
		setupMocks func(store *repoMocks.MockStorage)
		wantStatus int
		wantLen    int
	}{
		{
			name:   "Success",
			userID: 1,
			setupMocks: func(store *repoMocks.MockStorage) {
				store.EXPECT().
					ListWithdrawals(gomock.Any(), int64(1)).
					Return([]model.Withdrawal{
						{Order: "123", Sum: 50},
						{Order: "456", Sum: 100},
					}, nil)
			},
			wantStatus: http.StatusOK,
			wantLen:    2,
		},
		{
			name:   "NoWithdrawals",
			userID: 1,
			setupMocks: func(store *repoMocks.MockStorage) {
				store.EXPECT().
					ListWithdrawals(gomock.Any(), int64(1)).
					Return(nil, nil)
			},
			wantStatus: http.StatusNoContent,
			wantLen:    0,
		},
		{
			name:       "Unauthorized",
			userID:     0,
			setupMocks: func(store *repoMocks.MockStorage) {},
			wantStatus: http.StatusUnauthorized,
			wantLen:    0,
		},
		{
			name:   "InternalError",
			userID: 1,
			setupMocks: func(store *repoMocks.MockStorage) {
				store.EXPECT().
					ListWithdrawals(gomock.Any(), int64(1)).
					Return(nil, errors.New("db error"))
			},
			wantStatus: http.StatusInternalServerError,
			wantLen:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := repoMocks.NewMockStorage(ctrl)
			tt.setupMocks(store)

			h := newHandler(store)

			req := newAuthRequest(t, http.MethodGet, "/withdrawals", nil, tt.userID)
			if tt.userID == 0 {
				req = httptest.NewRequest(http.MethodGet, "/withdrawals", nil)
			}

			w := httptest.NewRecorder()
			h.WithdrawalsHandler(w, req)

			resp := w.Result()
			defer resp.Body.Close()
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
