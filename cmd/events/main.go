package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/onlyspans/events/internal/config"
	"github.com/onlyspans/events/internal/consumer"
	"github.com/onlyspans/events/internal/handler"
	"github.com/onlyspans/events/internal/http/middleware"
	"github.com/onlyspans/events/internal/migrator"
	"github.com/onlyspans/events/internal/repository"
	"github.com/onlyspans/events/internal/service"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/sync/errgroup"
)

func main() {
	// Initialize structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("starting event logs service")

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

	// Log feature flags status
	logger.Info("feature flags",
		"kafka_enabled", cfg.Features.KafkaEnabled,
		"auto_migrate", cfg.Features.AutoMigrate,
	)

	// Connect to database
	db, err := sql.Open("postgres", cfg.Database.DSN)
	if err != nil {
		logger.Error("failed to open database connection", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Configure connection pool
	db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)

	// Test database connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		logger.Error("failed to ping database", "error", err)
		os.Exit(1)
	}
	logger.Info("database connection established")

	// Run database migrations if auto-migrate is enabled
	if cfg.Features.AutoMigrate {
		if err := migrator.Run(cfg.Database.DSN); err != nil {
			logger.Error("migration failed", "error", err)
			os.Exit(1)
		}
	} else {
		logger.Info("auto-migrate disabled, skipping database migrations")
	}

	// Initialize repositories
	eventRepo := repository.NewEventRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)

	// Initialize services
	eventService := service.NewEventService(eventRepo, cfg.EventLog.MaxExportSize)
	settingsService := service.NewSettingsService(
		settingsRepo,
		cfg.EventLog.RetentionPeriodDays,
		cfg.EventLog.MaxExportSize,
	)

	// Initialize retention service
	retentionService := service.NewRetentionService(eventRepo, settingsService, logger)
	if err := retentionService.Start(cfg.EventLog.RetentionCron); err != nil {
		logger.Error("failed to start retention service", "error", err)
		os.Exit(1)
	}
	defer retentionService.Stop()

	// Initialize handlers
	eventHandler := handler.NewEventHandler(eventService, logger)
	settingsHandler := handler.NewSettingsHandler(settingsService, logger)
	healthChecker := handler.NewDBHealthChecker(db)
	healthHandler := handler.NewHealthHandler(healthChecker, logger)

	// Setup HTTP routes
	mux := http.NewServeMux()

	// Event routes
	mux.HandleFunc("/events", eventHandler.SearchEvents)
	mux.HandleFunc("/events/export", eventHandler.ExportEvents)
	mux.HandleFunc("/events/ingest", eventHandler.IngestEvent)
	mux.HandleFunc("/events/ingest/batch", eventHandler.IngestEventsBatch)

	// Settings routes
	mux.HandleFunc("/settings", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			settingsHandler.GetSettings(w, r)
		case http.MethodPut:
			settingsHandler.UpdateSettings(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/readyz", healthHandler.Readiness)
	mux.HandleFunc("/healthz", healthHandler.Liveness)

	mux.Handle("/metrics", promhttp.Handler())

	chain := middleware.Chain(
		middleware.Recovery(logger),
		middleware.Logging(logger),
	)

	server := &http.Server{
		Addr:         ":8080",
		Handler:      chain(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Create context for graceful shutdown
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	// Create errgroup to manage goroutines
	g, gctx := errgroup.WithContext(ctx)

	// Start HTTP server in errgroup
	g.Go(func() error {
		logger.Info("starting HTTP server", "port", "8080")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("HTTP server error", "error", err)
			return err
		}
		return nil
	})

	// Start Kafka consumer if enabled
	if cfg.Features.KafkaEnabled {
		kafkaConsumer, err := consumer.NewKafkaConsumer(&cfg.Kafka, eventService, logger)
		if err != nil {
			logger.Error("failed to create kafka consumer", "error", err)
			os.Exit(1)
		}
		defer kafkaConsumer.Close()

		g.Go(func() error {
			logger.Info("starting Kafka consumer",
				"brokers", cfg.Kafka.Brokers,
				"topic", cfg.Kafka.Topic,
				"group_id", cfg.Kafka.GroupID,
			)
			if err := kafkaConsumer.Start(gctx); err != nil {
				logger.Error("kafka consumer error", "error", err)
				return err
			}
			return nil
		})
	} else {
		logger.Info("Kafka consumer disabled")
	}

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Wait for either interrupt signal or errgroup error
	select {
	case <-quit:
		logger.Info("received shutdown signal")
	case <-gctx.Done():
		logger.Info("errgroup context cancelled")
	}

	logger.Info("shutting down service")

	// Cancel context to stop all goroutines
	cancel()

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Shutdown HTTP server
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server forced to shutdown", "error", err)
	}

	// Wait for errgroup goroutines to finish with timeout
	errChan := make(chan error, 1)
	go func() {
		errChan <- g.Wait()
	}()

	select {
	case err := <-errChan:
		if err != nil {
			logger.Error("service stopped with error", "error", err)
			os.Exit(1)
		}
	case <-shutdownCtx.Done():
		logger.Error("shutdown timeout exceeded")
		os.Exit(1)
	}

	logger.Info("service stopped gracefully")
}
