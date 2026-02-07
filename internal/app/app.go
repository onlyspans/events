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
	"github.com/onlyspans/events/internal/migrations"
	"github.com/onlyspans/events/internal/repository"
	"github.com/onlyspans/events/internal/service"
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
		[]string{"state"},
	)
	dbPoolMaxConnsGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_pool_max_connections",
			Help: "Maximum number of database connections",
		},
	)
)

type Application struct {
	config           *config.Config
	pool             *pgxpool.Pool
	httpServer       *http.Server
	kafkaConsumer    *consumer.KafkaConsumer
	retentionService *service.RetentionService
	logger           *slog.Logger
}

func New(cfg *config.Config, logger *slog.Logger) (*Application, error) {
	app := &Application{
		config: cfg,
		logger: logger,
	}

	poolConfig, err := pgxpool.ParseConfig(cfg.Database.DSN)
	if err != nil {
		return nil, err
	}

	poolConfig.MaxConns = cfg.Database.MaxConns
	poolConfig.MinConns = cfg.Database.MinConns
	poolConfig.MaxConnLifetime = cfg.Database.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.Database.MaxConnIdleTime
	poolConfig.HealthCheckPeriod = cfg.Database.HealthCheckPeriod

	poolConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		logger.Debug("new database connection established")
		return nil
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, err
	}
	app.pool = pool

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

	dbPoolMaxConnsGauge.Set(float64(cfg.Database.MaxConns))

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

	if cfg.Features.AutoMigrate {
		if err := migrations.Run(cfg.Database.DSN); err != nil {
			pool.Close()
			return nil, err
		}
	} else {
		logger.Info("auto-migrate disabled, skipping database migrations")
	}

	eventRepo := repository.NewEventRepository(pool)
	settingsRepo := repository.NewSettingsRepository(pool)

	eventService := service.NewEventService(eventRepo, cfg.EventLog.MaxExportSize)
	settingsService := service.NewSettingsService(
		settingsRepo,
		cfg.EventLog.RetentionPeriodDays,
		cfg.EventLog.MaxExportSize,
	)

	retentionService := service.NewRetentionService(eventRepo, settingsService, logger)
	if err := retentionService.Start(cfg.EventLog.RetentionCron); err != nil {
		pool.Close()
		return nil, err
	}
	app.retentionService = retentionService

	if cfg.Features.KafkaEnabled {
		kafkaConsumer, err := consumer.NewKafkaConsumer(&cfg.Kafka, eventService, logger)
		if err != nil {
			retentionService.Stop()
			pool.Close()
			return nil, err
		}
		app.kafkaConsumer = kafkaConsumer
	}

	eventHandler := handler.NewEventHandler(eventService, logger)
	settingsHandler := handler.NewSettingsHandler(settingsService, logger)
	healthChecker := handler.NewDBHealthChecker(pool)
	healthHandler := handler.NewHealthHandler(healthChecker, logger)

	mux := http.NewServeMux()

	mux.HandleFunc("POST /events", eventHandler.SearchEvents)
	mux.HandleFunc("POST /events/export", eventHandler.ExportEvents)
	mux.HandleFunc("POST /events/ingest", eventHandler.IngestEvent)
	mux.HandleFunc("POST /events/ingest/batch", eventHandler.IngestEventsBatch)

	mux.HandleFunc("GET /settings", settingsHandler.GetSettings)
	mux.HandleFunc("PUT /settings", settingsHandler.UpdateSettings)

	mux.HandleFunc("GET /readyz", healthHandler.Readiness)
	mux.HandleFunc("GET /healthz", healthHandler.Liveness)
	mux.HandleFunc("GET /version", healthHandler.Version)

	mux.Handle("GET /metrics", promhttp.Handler())

	pipeline := middleware.Pipeline(
		middleware.Recovery(logger),
		middleware.Logging(logger),
	)

	app.httpServer = &http.Server{
		Addr:         ":8080",
		Handler:      pipeline(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return app, nil
}

func (app *Application) Run(ctx context.Context) error {
	g, globalCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		app.logger.Info("starting HTTP server", "port", "8080")
		if err := app.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			app.logger.Error("HTTP server error", "error", err)
			return err
		}
		return nil
	})

	if app.kafkaConsumer != nil {
		g.Go(func() error {
			app.logger.Info("starting Kafka consumer",
				"brokers", app.config.Kafka.Brokers,
				"topic", app.config.Kafka.Topic,
				"group_id", app.config.Kafka.GroupID,
			)
			if err := app.kafkaConsumer.Start(globalCtx); err != nil {
				app.logger.Error("kafka consumer error", "error", err)
				return err
			}
			return nil
		})
	} else {
		app.logger.Info("Kafka consumer disabled")
	}

	<-globalCtx.Done()

	return g.Wait()
}

func (app *Application) Shutdown(ctx context.Context) error {
	app.logger.Info("shutting down application")

	if err := app.httpServer.Shutdown(ctx); err != nil {
		app.logger.Error("HTTP server forced to shutdown", "error", err)
		return err
	}

	if app.kafkaConsumer != nil {
		_ = app.kafkaConsumer.Close()
	}

	if app.retentionService != nil {
		app.retentionService.Stop()
	}

	if app.pool != nil {
		app.pool.Close()
		app.logger.Info("database connection pool closed")
	}

	app.logger.Info("application stopped gracefully")
	return nil
}
