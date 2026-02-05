package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"go.uber.org/zap"

	"github.com/tigranqic/go-musthave-diploma-tpl/internal/accrual"
	"github.com/tigranqic/go-musthave-diploma-tpl/internal/auth/jwt"
	"github.com/tigranqic/go-musthave-diploma-tpl/internal/config"
	"github.com/tigranqic/go-musthave-diploma-tpl/internal/handler"
	"github.com/tigranqic/go-musthave-diploma-tpl/internal/middleware"
	"github.com/tigranqic/go-musthave-diploma-tpl/internal/repository"
	"github.com/tigranqic/go-musthave-diploma-tpl/pkg/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		println("failed to load server config:", err.Error())
		os.Exit(1)
	}

	baseLog, err := logger.New(cfg.LogLevel, cfg.LogFormat)
	if err != nil {
		fmt.Println("failed to initialize logger:", err)
		os.Exit(1)
	}
	baseLog.Info("logger initialized",
		zap.String("level", cfg.LogLevel),
		zap.String("format", cfg.LogFormat),
	)

	authLog := baseLog.With(zap.String("component", "auth"))
	serverLog := baseLog.With(zap.String("component", "server"))
	repoLog := baseLog.With(zap.String("component", "repository"))
	handlerLog := baseLog.With(zap.String("component", "handler"))
	accrualLog := baseLog.With(zap.String("component", "accrual worker"))

	db, store, err := repository.InitStorage(cfg, repoLog)
	if err != nil {
		baseLog.Error("failed to initialize storage", zap.Error(err))
		return
	}
	defer func() {
		if err := db.Close(); err != nil {
			repoLog.Warn("failed to close db", zap.Error(err))
		}
	}()

	authSvc := jwt.New(authLog, cfg.JWTSecret, cfg.JWTExpiration)

	h := handler.NewHandler(store, db, handlerLog, authSvc, *cfg)
	rootHandler := middleware.LoggingMiddleware(baseLog)(h.Router())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	worker := accrual.NewWorker(store, cfg.AccrualAddr, accrualLog)
	go worker.Run(ctx)

	srv := &http.Server{
		Addr:    cfg.ServerAddr,
		Handler: rootHandler,
	}

	serverLog.Info("starting HTTP server", zap.String("address", cfg.ServerAddr))

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverLog.Error("http server error", zap.Error(err))
			stop()
		}
	}()

	<-ctx.Done()
	serverLog.Info("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		serverLog.Error("graceful shutdown failed", zap.Error(err))
		serverLog.Info("server stopped with errors")
	} else {
		serverLog.Info("server stopped gracefully")
	}
}
