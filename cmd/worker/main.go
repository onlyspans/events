package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/onlyspans/events/internal/config"
	"github.com/onlyspans/events/internal/consumer"
	"github.com/onlyspans/events/internal/repository"
	"github.com/onlyspans/events/internal/service"
)

func main() {
	// Initialize structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("starting event logs worker")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Connect to database
	db, err := sql.Open("postgres", cfg.Database.DSN())
	if err != nil {
		logger.Error("failed to open database connection", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test database connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		logger.Error("failed to ping database", "error", err)
		os.Exit(1)
	}
	logger.Info("database connection established")

	// Initialize repositories
	eventRepo := repository.NewEventRepository(db)

	// Initialize services
	eventService := service.NewEventService(eventRepo, cfg.EventLog.MaxExportSize)

	// Initialize Kafka consumer
	kafkaConsumer, err := consumer.NewKafkaConsumer(&cfg.Kafka, eventService, logger)
	if err != nil {
		logger.Error("failed to create kafka consumer", "error", err)
		os.Exit(1)
	}
	defer kafkaConsumer.Close()

	// Create context for graceful shutdown
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	// Start consuming in goroutine
	go func() {
		if err := kafkaConsumer.Start(ctx); err != nil {
			logger.Error("kafka consumer error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down worker")

	// Cancel context to stop consumer
	cancel()

	// Give some time for graceful shutdown
	time.Sleep(5 * time.Second)

	logger.Info("worker stopped gracefully")
}
