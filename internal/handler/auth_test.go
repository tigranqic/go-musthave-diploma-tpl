package handler_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tigranqic/go-musthave-diploma-tpl/internal/auth/jwt"
	"github.com/tigranqic/go-musthave-diploma-tpl/internal/config"
	"github.com/tigranqic/go-musthave-diploma-tpl/internal/handler"
	"github.com/tigranqic/go-musthave-diploma-tpl/internal/repository"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	_ "github.com/lib/pq"
)

func setupTestDB(t *testing.T) *sql.DB {
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
	return db
}

func cleanupDB(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec("TRUNCATE TABLE users RESTART IDENTITY CASCADE")
	if err != nil {
		t.Fatal(err)
	}
}

func setupHandler(t *testing.T) *handler.Handler {
	db := setupTestDB(t)
	cleanupDB(t, db)

	log := zap.NewNop()
	store := repository.NewPostgresStorage(db, log)
	authSvc := jwt.New([]byte("testsecret"), 3600)

	return handler.NewHandler(store, db, log, authSvc, config.Config{})
}

func TestRegisterHandler_Success(t *testing.T) {
	h := setupHandler(t)

	body := map[string]string{
		"login":    "testuser",
		"password": "password123",
	}
	data, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h.RegisterHandler(rec, req)

	res := rec.Result()
	defer func() {
		_ = res.Body.Close()
	}()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", res.StatusCode)
	}

	var resp map[string]string
	_ = json.NewDecoder(res.Body).Decode(&resp)
	if resp["token"] == "" {
		t.Fatal("expected token in response")
	}

}

func TestRegisterHandler_LoginTaken(t *testing.T) {
	h := setupHandler(t)
	store := h.Store()
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass1"), bcrypt.DefaultCost)
	_, _ = store.CreateUser(context.Background(), "dupuser", hash)

	body := map[string]string{
		"login":    "dupuser",
		"password": "newpasstest",
	}
	data, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h.RegisterHandler(rec, req)

	res := rec.Result()
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 Conflict, got %d", res.StatusCode)
	}
}

func TestLoginHandler_Success(t *testing.T) {
	h := setupHandler(t)
	store := h.Store()

	pass := "mypassword"
	hash, _ := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	_, _ = store.CreateUser(context.Background(), "userlogin", hash)

	body := map[string]string{
		"login":    "userlogin",
		"password": pass,
	}
	data, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/user/login", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h.LoginHandler(rec, req)

	res := rec.Result()
	defer func() {
		_ = res.Body.Close()
	}()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", res.StatusCode)
	}

	var resp map[string]string
	_ = json.NewDecoder(res.Body).Decode(&resp)
	if resp["token"] == "" {
		t.Fatal("expected token in response")
	}
}

func TestLoginHandler_WrongPassword(t *testing.T) {
	h := setupHandler(t)
	store := h.Store()

	pass := "correctpass"
	hash, _ := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	_, _ = store.CreateUser(context.Background(), "userlogin2", hash)

	body := map[string]string{
		"login":    "userlogin2",
		"password": "wrongpass",
	}
	data, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/user/login", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h.LoginHandler(rec, req)

	res := rec.Result()
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", res.StatusCode)
	}
}

func TestLoginHandler_UserNotFound(t *testing.T) {
	h := setupHandler(t)

	body := map[string]string{
		"login":    "nonexistent",
		"password": "pass",
	}
	data, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/user/login", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h.LoginHandler(rec, req)

	res := rec.Result()
	defer func() {
		_ = res.Body.Close()
	}()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", res.StatusCode)
	}
}
