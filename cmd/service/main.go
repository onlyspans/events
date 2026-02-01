package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/onlyspans/events/internal/config"
	"github.com/onlyspans/events/internal/handler"
	"github.com/onlyspans/events/internal/repository"
	"github.com/onlyspans/events/internal/service"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
	healthHandler := handler.NewHealthHandler(db, logger)

	// Setup HTTP routes
	mux := http.NewServeMux()

	// Event routes
	mux.HandleFunc("/events", eventHandler.SearchEvents)
	mux.HandleFunc("/events/export", eventHandler.ExportEvents)

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

	// Health routes
	mux.HandleFunc("/readyz", healthHandler.Readiness)
	mux.HandleFunc("/healthz", healthHandler.Liveness)

	// Metrics route
	mux.Handle("/metrics", promhttp.Handler())

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      loggingMiddleware(mux, logger),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Info("starting HTTP server", "port", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server")

	// Graceful shutdown with timeout
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", "error", err)
		os.Exit(1)
	}

	logger.Info("server stopped gracefully")
}

// loggingMiddleware logs HTTP requests.
func loggingMiddleware(next http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		logger.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.statusCode,
			"duration", time.Since(start),
			"remote_addr", r.RemoteAddr,
		)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
