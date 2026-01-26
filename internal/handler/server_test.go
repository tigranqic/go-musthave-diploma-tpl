package handler

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/lib/pq"
	"github.com/tigranqic/go-musthave-diploma-tpl/internal/config"
	"go.uber.org/zap"
)

func TestPingHandler_OK(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		_ = db.Close()
	}()

	log := zap.NewNop()

	h := NewHandler(nil, db, log, nil, config.Config{})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()

	h.pingHandler(rec, req)

	res := rec.Result()
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", res.StatusCode)
	}
}

func TestPingHandler_DBIsNil(t *testing.T) {
	log := zap.NewNop()

	h := NewHandler(nil, nil, log, nil, config.Config{})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()

	h.pingHandler(rec, req)

	res := rec.Result()
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", res.StatusCode)
	}
}

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	// dsn := os.Getenv("DATABASE_URI")
	// if dsn == "" {
	// 	t.Fatal("DATABASE_URI is not set")
	// }

	db, err := sql.Open("postgres", "postgres://postgres:postgres@localhost:15451/go-musthave-diploma-tpl?sslmode=disable")
	if err != nil {
		t.Fatalf("failed to open test DB: %v", err)
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		t.Fatalf("failed to ping test DB: %v", err)
	}

	return db
}
