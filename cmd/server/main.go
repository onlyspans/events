package main

import (
	"context"
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
	// Initialize structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("starting events service")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	// Log feature flags
	logger.Info("feature flags",
		"kafka_enabled", cfg.Features.KafkaEnabled,
		"auto_migrate", cfg.Features.AutoMigrate,
	)

	// Create application
	application, err := app.New(cfg, logger)
	if err != nil {
		logger.Error("failed to create application", "error", err)
		os.Exit(1)
	}

	// Setup signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Run application in background
	errChan := make(chan error, 1)
	go func() {
		errChan <- application.Run(ctx)
	}()

	// Wait for interrupt signal or application error
	select {
	case <-quit:
		logger.Info("received shutdown signal")
	case err := <-errChan:
		if err != nil {
			logger.Error("application error", "error", err)
		}
	}

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Cancel application context
	cancel()

	// Shutdown application
	if err := application.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown error", "error", err)
		os.Exit(1)
	}

	// Wait for Run() to finish
	select {
	case err := <-errChan:
		if err != nil && err != context.Canceled {
			logger.Error("service stopped with error", "error", err)
			os.Exit(1)
		}
	case <-shutdownCtx.Done():
		logger.Error("shutdown timeout exceeded")
		os.Exit(1)
	}

	logger.Info("service stopped gracefully")
}
