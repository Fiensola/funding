package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fiensola/funding/internal/api"
	"github.com/fiensola/funding/internal/config"
	"github.com/fiensola/funding/internal/exchange"
	"github.com/fiensola/funding/internal/exchange/extended"
	"github.com/fiensola/funding/internal/exchange/hibachi"
	"github.com/fiensola/funding/internal/exchange/lighter"
	"github.com/fiensola/funding/internal/exchange/pacifica"
	"github.com/fiensola/funding/internal/logger"
	"github.com/fiensola/funding/internal/repository/postgres"
	"github.com/fiensola/funding/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	//config
	cfg, err := config.Load(".")
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	//logger
	logger, err := logger.NewLogger(cfg.Logger.Level, cfg.Logger.Encoding)
	if err != nil {
		return fmt.Errorf("init logger: %w", err)
	}
	defer logger.Sync()

	logger.Info("starting funding service tracker")

	//db
	ctx := context.Background()
	dbPool, err := pgxpool.New(ctx, cfg.Database.DSN())
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer dbPool.Close()

	//ping db
	if err := dbPool.Ping(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	logger.Info("database connected")

	//repos
	fundingRepo := postgres.NewFundingRepository(dbPool, logger)

	//exchanges
	exchanges := []exchange.Exchange{
		pacifica.NewClient(exchange.Config{
			BaseURL: cfg.Exchages.Pacifica.BaseURL,
			Proxy:   cfg.Proxy,
		}, logger),
		lighter.NewClient(exchange.Config{
			BaseURL: cfg.Exchages.Lighter.BaseURL,
			Proxy:   cfg.Proxy,
		}, logger),
		extended.NewClient(exchange.Config{
			BaseURL: cfg.Exchages.Extended.BaseURL,
			Proxy:   cfg.Proxy,
		}, logger),
		hibachi.NewClient(exchange.Config{
			BaseURL: cfg.Exchages.Hibachi.BaseURL,
			Proxy:   cfg.Proxy,
		}, logger),
	}

	//tracker service
	tracker := service.NewTrackerService(
		exchanges,
		fundingRepo,
		logger,
		cfg.Tracker.UpdateInterval,
	)

	go tracker.Start(ctx)

	//router
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	handler := api.NewHandler(tracker, logger)
	handler.RegisterRoutes(router)

	//http server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	//start server
	go func() {
		logger.Info("starting HTTP server")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server error", zap.Error(err))
		}
	}()

	//interrupt
	quitCh := make(chan os.Signal, 2)
	signal.Notify(quitCh, syscall.SIGINT, syscall.SIGTERM)
	<-quitCh

	logger.Info("shutting down server...")

	//graceful
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tracker.Stop()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}

	logger.Info("server stopped gracefullly")

	return nil
}
