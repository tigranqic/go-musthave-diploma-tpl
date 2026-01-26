package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/lib/pq"
	"go.uber.org/zap"

	"github.com/tigranqic/go-musthave-diploma-tpl/internal/auth/jwt"
	"github.com/tigranqic/go-musthave-diploma-tpl/internal/config"
	"github.com/tigranqic/go-musthave-diploma-tpl/internal/handler"
	"github.com/tigranqic/go-musthave-diploma-tpl/internal/middleware"
	"github.com/tigranqic/go-musthave-diploma-tpl/internal/repository"
	"github.com/tigranqic/go-musthave-diploma-tpl/pkg/logger"
)

func main() {
	cfg, err := config.Load(false)
	if err != nil {
		println("failed to load server config:", err.Error())
		os.Exit(1)
	}

	logger.Init(cfg.LogLevel, cfg.LogFormat)
	log := logger.Get()

	db, store, err := repository.InitStorage(cfg, log)
	if err != nil {
		log.Fatal("failed to initialize storage", zap.Error(err))
	}
	defer func() {
		_ = db.Close()
	}()

	authSvc := jwt.New(cfg.JWTSecret, cfg.JWTExpiration)

	h := handler.NewHandler(store, db, log, authSvc, *cfg)
	rootHandler := middleware.LoggingMiddleware(log)(h.Router())

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	go func() {
		if err := h.RunAccrualWorker(ctx); err != nil {
			log.Error("accrual worker stopped", zap.Error(err))
		}
	}()

	srv := &http.Server{
		Addr:    cfg.ServerAddr,
		Handler: rootHandler,
	}

	log.Info("starting HTTP server", zap.String("address", cfg.ServerAddr))

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("http server error", zap.Error(err))
		}
	}()

	<-ctx.Done()
	log.Info("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 1)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("graceful shutdown failed", zap.Error(err))
	}

	log.Info("server stopped gracefully")
}
