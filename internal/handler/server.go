package handler

import (
	"database/sql"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/tigranqic/go-musthave-diploma-tpl/internal/accrual"
	"github.com/tigranqic/go-musthave-diploma-tpl/internal/auth"
	"github.com/tigranqic/go-musthave-diploma-tpl/internal/config"
	"github.com/tigranqic/go-musthave-diploma-tpl/internal/middleware"
	"github.com/tigranqic/go-musthave-diploma-tpl/internal/repository"
)

type Handler struct {
	store   repository.Storage
	db      *sql.DB
	log     *zap.Logger
	authSvc auth.Service
	worker  *accrual.Worker
}

func NewHandler(
	store repository.Storage,
	db *sql.DB,
	log *zap.Logger,
	authSvc auth.Service,
	cfg config.Config,
) *Handler {

	worker := accrual.NewWorker(
		store,
		cfg.AccrualAddr,
		log,
	)

	return &Handler{
		store:   store,
		db:      db,
		log:     log,
		authSvc: authSvc,
		worker:  worker,
	}
}

func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.GzipDecompress)
	r.Use(middleware.GzipCompress)

	r.Get("/ping", h.pingHandler)
	r.Post("/api/user/register", h.RegisterHandler)
	r.Post("/api/user/login", h.LoginHandler)

	r.Route("/api/user", func(r chi.Router) {
		r.Use(middleware.Auth(h.authSvc, h.log))

		r.Get("/orders", h.ListOrdersHandler)
		r.Post("/orders", h.CreateOrderHandler)
		r.Get("/balance", h.GetBalanceHandler)
		r.Post("/balance/withdraw", h.WithdrawHandler)
		r.Get("/withdrawals", h.WithdrawalsHandler)
	})

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

func (h *Handler) Store() repository.Storage {
	return h.store
}
