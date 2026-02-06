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

	authMocks "github.com/tigranqic/go-musthave-diploma-tpl/internal/auth/jwt/mocks"
	"github.com/tigranqic/go-musthave-diploma-tpl/internal/config"
	"github.com/tigranqic/go-musthave-diploma-tpl/internal/handler"
	"github.com/tigranqic/go-musthave-diploma-tpl/internal/repository"
	repoMocks "github.com/tigranqic/go-musthave-diploma-tpl/internal/repository/mocks"
	"go.uber.org/zap"
)

func newRegisterHandler(store *repoMocks.MockStorage, authSvc *authMocks.MockService) *handler.Handler {
	return handler.NewHandler(store, nil, zap.NewNop(), authSvc, config.Config{})
}

func TestRegisterHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name       string
		body       interface{}
		setupMocks func(store *repoMocks.MockStorage, authSvc *authMocks.MockService)
		wantStatus int
		wantInBody string
	}{
		{
			name: "Success",
			body: map[string]string{"login": "user1", "password": "12345678"},
			setupMocks: func(store *repoMocks.MockStorage, authSvc *authMocks.MockService) {
				store.EXPECT().
					CreateUser(gomock.Any(), "user1", gomock.Any()).
					Return(int64(1), nil)
				authSvc.EXPECT().
					GenerateToken(int64(1)).
					Return("token123", nil)
			},
			wantStatus: http.StatusOK,
			wantInBody: "token123",
		},
		{
			name:       "Invalid JSON",
			body:       "{invalid",
			setupMocks: func(store *repoMocks.MockStorage, authSvc *authMocks.MockService) {},
			wantStatus: http.StatusBadRequest,
			wantInBody: "invalid JSON",
		},
		{
			name:       "Short Password",
			body:       map[string]string{"login": "user1", "password": "123"},
			setupMocks: func(store *repoMocks.MockStorage, authSvc *authMocks.MockService) {},
			wantStatus: http.StatusBadRequest,
			wantInBody: "invalid login or password",
		},
		{
			name:       "Empty Login",
			body:       map[string]string{"login": "", "password": "12345678"},
			setupMocks: func(store *repoMocks.MockStorage, authSvc *authMocks.MockService) {},
			wantStatus: http.StatusBadRequest,
			wantInBody: "invalid login or password",
		},
		{
			name: "Login Already Taken",
			body: map[string]string{"login": "user1", "password": "12345678"},
			setupMocks: func(store *repoMocks.MockStorage, authSvc *authMocks.MockService) {
				store.EXPECT().
					CreateUser(gomock.Any(), "user1", gomock.Any()).
					Return(int64(0), repository.ErrLoginTaken)
			},
			wantStatus: http.StatusConflict,
			wantInBody: "login already taken",
		},
		{
			name: "Internal Error in CreateUser",
			body: map[string]string{"login": "user1", "password": "12345678"},
			setupMocks: func(store *repoMocks.MockStorage, authSvc *authMocks.MockService) {
				store.EXPECT().
					CreateUser(gomock.Any(), "user1", gomock.Any()).
					Return(int64(0), errors.New("db error"))
			},
			wantStatus: http.StatusInternalServerError,
			wantInBody: "internal error",
		},
		{
			name: "Internal Error in Token Generation",
			body: map[string]string{"login": "user1", "password": "12345678"},
			setupMocks: func(store *repoMocks.MockStorage, authSvc *authMocks.MockService) {
				store.EXPECT().
					CreateUser(gomock.Any(), "user1", gomock.Any()).
					Return(int64(1), nil)
				authSvc.EXPECT().
					GenerateToken(int64(1)).
					Return("", errors.New("token error"))
			},
			wantStatus: http.StatusInternalServerError,
			wantInBody: "internal error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := repoMocks.NewMockStorage(ctrl)
			authSvc := authMocks.NewMockService(ctrl)
			tt.setupMocks(store, authSvc)

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
			defer resp.Body.Close()
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
