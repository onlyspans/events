package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/onlyspans/events/internal/config"
	"github.com/onlyspans/events/internal/consumer"
	"github.com/onlyspans/events/internal/handler"
	"github.com/onlyspans/events/internal/http/middleware"
	"github.com/onlyspans/events/internal/http/response"
	"github.com/onlyspans/events/internal/migrator"
	"github.com/onlyspans/events/internal/repository"
	"github.com/onlyspans/events/internal/service"
	"github.com/onlyspans/events/pkg/version"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/sync/errgroup"
)

var (
	dbPoolConnsGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "db_pool_connections",
			Help: "Current number of database connections",
		},
		[]string{"state"}, // idle, active, total
	)
	dbPoolMaxConnsGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_pool_max_connections",
			Help: "Maximum number of database connections",
		},
	)
)

// Application manages the lifecycle of the events service
type Application struct {
	config           *config.Config
	pool             *pgxpool.Pool
	httpServer       *http.Server
	kafkaConsumer    *consumer.KafkaConsumer
	retentionService *service.RetentionService
	logger           *slog.Logger
}

// New creates and initializes a new Application instance
func New(cfg *config.Config, logger *slog.Logger) (*Application, error) {
	app := &Application{
		config: cfg,
		logger: logger,
	}

	// Parse database connection config
	poolConfig, err := pgxpool.ParseConfig(cfg.Database.DSN)
	if err != nil {
		return nil, err
	}

	// Configure connection pool
	poolConfig.MaxConns = cfg.Database.MaxConns
	poolConfig.MinConns = cfg.Database.MinConns
	poolConfig.MaxConnLifetime = cfg.Database.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.Database.MaxConnIdleTime
	poolConfig.HealthCheckPeriod = cfg.Database.HealthCheckPeriod

	// Optional: Add connection lifecycle hooks for logging
	poolConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		logger.Debug("new database connection established")
		return nil
	}

	// Create connection pool
	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, err
	}
	app.pool = pool

	// Test database connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	logger.Info("database connection pool established",
		"max_conns", cfg.Database.MaxConns,
		"min_conns", cfg.Database.MinConns,
	)

	// Set max connections metric (constant value)
	dbPoolMaxConnsGauge.Set(float64(cfg.Database.MaxConns))

	// Start pool metrics collection goroutine
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			stat := pool.Stat()
			dbPoolConnsGauge.WithLabelValues("total").Set(float64(stat.TotalConns()))
			dbPoolConnsGauge.WithLabelValues("idle").Set(float64(stat.IdleConns()))
			dbPoolConnsGauge.WithLabelValues("active").Set(float64(stat.AcquiredConns()))
		}
	}()

	// Run database migrations if auto-migrate is enabled
	if cfg.Features.AutoMigrate {
		if err := migrator.Run(cfg.Database.DSN); err != nil {
			pool.Close()
			return nil, err
		}
	} else {
		logger.Info("auto-migrate disabled, skipping database migrations")
	}

	// Initialize repositories
	eventRepo := repository.NewEventRepository(pool)
	settingsRepo := repository.NewSettingsRepository(pool)

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
		pool.Close()
		return nil, err
	}
	app.retentionService = retentionService

	// Initialize Kafka consumer if enabled
	if cfg.Features.KafkaEnabled {
		kafkaConsumer, err := consumer.NewKafkaConsumer(&cfg.Kafka, eventService, logger)
		if err != nil {
			retentionService.Stop()
			pool.Close()
			return nil, err
		}
		app.kafkaConsumer = kafkaConsumer
	}

	// Initialize handlers
	eventHandler := handler.NewEventHandler(eventService, logger)
	settingsHandler := handler.NewSettingsHandler(settingsService, logger)
	healthChecker := handler.NewDBHealthChecker(pool)
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
			response.MethodNotAllowed(w)
		}
	})

	// Health and version routes
	mux.HandleFunc("/readyz", healthHandler.Readiness)
	mux.HandleFunc("/healthz", healthHandler.Liveness)
	mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			response.MethodNotAllowed(w)
			return
		}
		response.JSON(w, http.StatusOK, version.Get())
	})

	// Metrics
	mux.Handle("/metrics", promhttp.Handler())

	// Apply middleware
	chain := middleware.Chain(
		middleware.Recovery(logger),
		middleware.Logging(logger),
	)

	// Create HTTP server
	app.httpServer = &http.Server{
		Addr:         ":8080",
		Handler:      chain(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return app, nil
}

// Run starts the application and blocks until shutdown
func (app *Application) Run(ctx context.Context) error {
	// Create errgroup to manage goroutines
	g, gctx := errgroup.WithContext(ctx)

	// Start HTTP server
	g.Go(func() error {
		app.logger.Info("starting HTTP server", "port", "8080")
		if err := app.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			app.logger.Error("HTTP server error", "error", err)
			return err
		}
		return nil
	})

	// Start Kafka consumer if enabled
	if app.kafkaConsumer != nil {
		g.Go(func() error {
			app.logger.Info("starting Kafka consumer",
				"brokers", app.config.Kafka.Brokers,
				"topic", app.config.Kafka.Topic,
				"group_id", app.config.Kafka.GroupID,
			)
			if err := app.kafkaConsumer.Start(gctx); err != nil {
				app.logger.Error("kafka consumer error", "error", err)
				return err
			}
			return nil
		})
	} else {
		app.logger.Info("Kafka consumer disabled")
	}

	// Wait for context cancellation or error
	<-gctx.Done()

	return g.Wait()
}

// Shutdown gracefully shuts down the application
func (app *Application) Shutdown(ctx context.Context) error {
	app.logger.Info("shutting down application")

	// Shutdown HTTP server
	if err := app.httpServer.Shutdown(ctx); err != nil {
		app.logger.Error("HTTP server forced to shutdown", "error", err)
		return err
	}

	// Close Kafka consumer
	if app.kafkaConsumer != nil {
		app.kafkaConsumer.Close()
	}

	// Stop retention service
	if app.retentionService != nil {
		app.retentionService.Stop()
	}

	// Close connection pool (synchronous and graceful)
	if app.pool != nil {
		app.pool.Close()
		app.logger.Info("database connection pool closed")
	}

	app.logger.Info("application stopped gracefully")
	return nil
}
