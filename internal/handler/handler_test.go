package handler_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tigranqic/go-musthave-diploma-tpl/internal/auth/jwt"
	"github.com/tigranqic/go-musthave-diploma-tpl/internal/config"
	"github.com/tigranqic/go-musthave-diploma-tpl/internal/handler"
	models "github.com/tigranqic/go-musthave-diploma-tpl/internal/model"
	"github.com/tigranqic/go-musthave-diploma-tpl/internal/repository"
	"go.uber.org/zap"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func setupDB(t *testing.T) *sql.DB {
	t.Helper()

	// dsn := os.Getenv("DATABASE_URI")
	// if dsn == "" {
	// 	t.Fatal("DATABASE_URI is not set")
	// }

	db, err := sql.Open("postgres", "postgres://postgres:postgres@localhost:15451/go-musthave-diploma-tpl?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}

	if err := db.Ping(); err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec(`TRUNCATE users, orders RESTART IDENTITY CASCADE`)
	if err != nil {
		t.Fatal(err)
	}

	return db
}

func createUser(t *testing.T, store repository.Storage, login, password string) int64 {
	t.Helper()

	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	id, err := store.CreateUser(context.Background(), login, hash)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestCreateOrder_Accepted(t *testing.T) {
	db := setupDB(t)
	defer func() {
		_ = db.Close()
	}()
	log := zap.NewNop()
	store := repository.NewPostgresStorage(db, log)
	authSvc := jwt.New([]byte("testsecret"), 24*time.Hour)

	userID := createUser(t, store, "user1", "passwordtest")

	token, _ := authSvc.GenerateToken(userID)
	t.Logf("token: %s", token)

	h := handler.NewHandler(store, db, log, authSvc, config.Config{})
	ts := httptest.NewServer(h.Router())
	defer ts.Close()

	req, _ := http.NewRequest(
		http.MethodPost,
		ts.URL+"/api/user/orders",
		bytes.NewBufferString("79927398713"),
	)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "text/plain")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", res.StatusCode)
	}
}

func TestCreateOrder_AlreadyExistsSameUser(t *testing.T) {
	db := setupDB(t)
	defer func() {
		_ = db.Close()
	}()
	log := zap.NewNop()
	store := repository.NewPostgresStorage(db, log)
	authSvc := jwt.New([]byte("testsecret"), 24*time.Hour)

	userID := createUser(t, store, "user2", "passwordtest")

	_ = store.CreateOrder(context.Background(), &models.Order{
		Number: "79927398713",
		UserID: userID,
		Status: string(models.OrderStatusNew),
	})

	token, _ := authSvc.GenerateToken(userID)
	t.Logf("token: %s", token)

	h := handler.NewHandler(store, db, log, authSvc, config.Config{})
	ts := httptest.NewServer(h.Router())
	defer ts.Close()

	req, _ := http.NewRequest(
		http.MethodPost,
		ts.URL+"/api/user/orders",
		bytes.NewBufferString("79927398713"),
	)
	req.Header.Set("Authorization", "Bearer "+token)

	res, _ := http.DefaultClient.Do(req)
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
}

func TestCreateOrder_OwnedByAnotherUser(t *testing.T) {
	db := setupDB(t)
	defer func() {
		_ = db.Close()
	}()
	log := zap.NewNop()
	store := repository.NewPostgresStorage(db, log)
	authSvc := jwt.New([]byte("testsecret"), 24*time.Hour)

	user1 := createUser(t, store, "user3", "passwordtest")
	user2 := createUser(t, store, "user4", "passwordtest")

	_ = store.CreateOrder(context.Background(), &models.Order{
		Number: "79927398713",
		UserID: user1,
		Status: string(models.OrderStatusNew),
	})

	token, _ := authSvc.GenerateToken(user2)
	t.Logf("token: %s", token)

	h := handler.NewHandler(store, db, log, authSvc, config.Config{})
	ts := httptest.NewServer(h.Router())
	defer ts.Close()

	req, _ := http.NewRequest(
		http.MethodPost,
		ts.URL+"/api/user/orders",
		bytes.NewBufferString("79927398713"),
	)
	req.Header.Set("Authorization", "Bearer "+token)

	res, _ := http.DefaultClient.Do(req)
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", res.StatusCode)
	}
}

func TestListOrders_OK(t *testing.T) {
	db := setupDB(t)
	defer func() {
		_ = db.Close()
	}()
	log := zap.NewNop()
	store := repository.NewPostgresStorage(db, log)
	authSvc := jwt.New([]byte("testsecret"), 24*time.Hour)

	userID := createUser(t, store, "user5", "passwordtest")

	_ = store.CreateOrder(context.Background(), &models.Order{
		Number: "1111111111",
		UserID: userID,
		Status: string(models.OrderStatusProcessed),
	})

	token, _ := authSvc.GenerateToken(userID)
	t.Logf("token: %s", token)

	h := handler.NewHandler(store, db, log, authSvc, config.Config{})
	ts := httptest.NewServer(h.Router())
	defer ts.Close()

	req, _ := http.NewRequest(
		http.MethodGet,
		ts.URL+"/api/user/orders",
		nil,
	)
	req.Header.Set("Authorization", "Bearer "+token)

	res, _ := http.DefaultClient.Do(req)
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
}

func TestListOrders_Empty(t *testing.T) {
	db := setupDB(t)
	defer func() {
		_ = db.Close()
	}()
	log := zap.NewNop()
	store := repository.NewPostgresStorage(db, log)
	authSvc := jwt.New([]byte("testsecret"), 24*time.Hour)

	userID := createUser(t, store, "user6", "passwordtest")
	token, _ := authSvc.GenerateToken(userID)
	t.Logf("token: %s", token)

	h := handler.NewHandler(store, db, log, authSvc, config.Config{})
	ts := httptest.NewServer(h.Router())
	defer ts.Close()

	req, _ := http.NewRequest(
		http.MethodGet,
		ts.URL+"/api/user/orders",
		nil,
	)
	req.Header.Set("Authorization", "Bearer "+token)

	res, _ := http.DefaultClient.Do(req)
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
}

func createUserWithBalance(t *testing.T, store repository.Storage, login, password string, balance float64) int64 {
	t.Helper()
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	userID, err := store.CreateUser(context.Background(), login, hash)
	if err != nil {
		t.Fatal(err)
	}

	err = store.UpdateBalance(context.Background(), userID, balance, 0)
	if err != nil {
		t.Fatal(err)
	}

	return userID
}

func TestGetBalanceHandler(t *testing.T) {
	db := setupDB(t)
	defer func() {
		_ = db.Close()
	}()

	log := zap.NewNop()
	store := repository.NewPostgresStorage(db, log)
	authSvc := jwt.New([]byte("testsecret"), 24*time.Hour)

	userID := createUserWithBalance(t, store, "user1", "password", 100.5)

	h := handler.NewHandler(store, db, log, authSvc, config.Config{})
	ts := httptest.NewServer(h.Router())
	defer ts.Close()

	token, _ := authSvc.GenerateToken(userID)

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/user/balance", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	res, _ := http.DefaultClient.Do(req)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	defer func() {
		_ = res.Body.Close()
	}()
	var b struct {
		Current   float64 `json:"current"`
		Withdrawn float64 `json:"withdrawn"`
	}

	if err := json.NewDecoder(res.Body).Decode(&b); err != nil {
		t.Fatal(err)
	}

	if b.Current != 100.5 {
		t.Errorf("expected current balance 100.5, got %v", b.Current)
	}
	if b.Withdrawn != 0 {
		t.Errorf("expected withdrawn 0, got %v", b.Withdrawn)
	}
}

func TestWithdrawHandler(t *testing.T) {
	db := setupDB(t)
	defer func() {
		_ = db.Close()
	}()
	log := zap.NewNop()
	store := repository.NewPostgresStorage(db, log)
	authSvc := jwt.New([]byte("testsecret"), 24*time.Hour)

	userID := createUserWithBalance(t, store, "user1", "password", 200)

	h := handler.NewHandler(store, db, log, authSvc, config.Config{})
	ts := httptest.NewServer(h.Router())
	defer ts.Close()

	token, _ := authSvc.GenerateToken(userID)

	reqBody, _ := json.Marshal(map[string]interface{}{
		"order": "79927398713",
		"sum":   150,
	})

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/user/balance/withdraw", bytes.NewBuffer(reqBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	res, _ := http.DefaultClient.Do(req)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	defer func() {
		_ = res.Body.Close()
	}()
	balance, _ := store.GetBalance(context.Background(), userID)
	if balance.Current != 50 {
		t.Errorf("expected current balance 50, got %v", balance.Current)
	}
	if balance.Withdrawn != 150 {
		t.Errorf("expected withdrawn 150, got %v", balance.Withdrawn)
	}
}

func TestWithdrawalsHandler(t *testing.T) {
	db := setupDB(t)
	defer func() {
		_ = db.Close()
	}()
	log := zap.NewNop()
	store := repository.NewPostgresStorage(db, log)
	authSvc := jwt.New([]byte("testsecret"), 24*time.Hour)

	userID := createUserWithBalance(t, store, "user1", "password", 300)

	_ = store.Withdraw(context.Background(), userID, "79927398713", 50)
	_ = store.Withdraw(context.Background(), userID, "79927398714", 75)

	h := handler.NewHandler(store, db, log, authSvc, config.Config{})
	ts := httptest.NewServer(h.Router())
	defer ts.Close()

	token, _ := authSvc.GenerateToken(userID)

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/user/withdrawals", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	res, _ := http.DefaultClient.Do(req)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	defer func() {
		_ = res.Body.Close()
	}()
	var ws []struct {
		Order       string  `json:"order"`
		Sum         float64 `json:"sum"`
		ProcessedAt string  `json:"processed_at"`
	}

	if err := json.NewDecoder(res.Body).Decode(&ws); err != nil {
		t.Fatal(err)
	}

	if len(ws) != 2 {
		t.Errorf("expected 2 withdrawals, got %d", len(ws))
	}

	if ws[0].Order != "79927398714" {
		t.Errorf("expected most recent order first, got %v", ws[0].Order)
	}
	if ws[1].Order != "79927398713" {
		t.Errorf("expected oldest order second, got %v", ws[1].Order)
	}
}
