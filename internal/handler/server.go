package handler

import (
	"net/http"

	"github.com/tigranqic/go-musthave-diploma-tpl/internal/middleware"
	"github.com/tigranqic/go-musthave-diploma-tpl/internal/repository"
	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"

	"database/sql"
)

type Handler struct {
	store repository.Storage
	db    *sql.DB
	log   *zap.Logger
}

func NewHandler(store repository.Storage, db *sql.DB, log *zap.Logger) *Handler {
	return &Handler{
		store: store,
		db:    db,
		log:   log,
	}
}

func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.GzipDecompress)
	r.Use(middleware.GzipCompress)

	r.Get("/ping", h.pingHandler)

	return r
}

func (h *Handler) pingHandler(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		h.log.Error("db is nil")
		http.Error(w, "database not initialized", http.StatusInternalServerError)
		return
	}

	if err := h.db.Ping(); err != nil {
		h.log.Error("DB ping failed", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
