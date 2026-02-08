package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/onlyspans/events/internal/app"
	"github.com/onlyspans/events/internal/config"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	logger.Info("starting events service")

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	if err := cfg.Validate(); err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	logger.Info("feature flags",
		"auto_migrate", cfg.Features.AutoMigrate,
	)

	application, err := app.New(cfg, logger)
	if err != nil {
		logger.Error("failed to create application", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	errChan := make(chan error, 1)
	go func() {
		errChan <- application.Run(ctx)
	}()

	select {
	case <-quit:
		logger.Info("received shutdown signal")
	case err := <-errChan:
		if err != nil {
			logger.Error("application error", "error", err)
		}
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	cancel()

	if err := application.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown error", "error", err)
		os.Exit(1)
	}

	select {
	case err := <-errChan:
		if err != nil && !errors.Is(err, context.Canceled) {
			logger.Error("service stopped with error", "error", err)
			os.Exit(1)
		}
	case <-shutdownCtx.Done():
		logger.Error("shutdown timeout exceeded")
		os.Exit(1)
	}

	logger.Info("service stopped gracefully")
}
