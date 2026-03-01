package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/onlyspans/events/internal/config"
	grpchandler "github.com/onlyspans/events/internal/grpc"
	"github.com/onlyspans/events/internal/handler"
	"github.com/onlyspans/events/internal/http/middleware"
	"github.com/onlyspans/events/internal/repository"
	"github.com/onlyspans/events/internal/service"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

type Application struct {
	config           *config.Config
	pool             *pgxpool.Pool
	httpServer       *http.Server
	grpcServer       *grpc.Server
	retentionService *service.RetentionService
	logger           *slog.Logger
}

func New(cfg *config.Config, logger *slog.Logger) (*Application, error) {
	app := &Application{
		config: cfg,
		logger: logger,
	}

	pool, err := setupPostgres(cfg, logger)
	if err != nil {
		return nil, err
	}
	app.pool = pool

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

	app.grpcServer = grpchandler.NewServer(eventService, settingsService, logger)

	eventHandler := handler.NewEventHandler(eventService, logger)
	settingsHandler := handler.NewSettingsHandler(settingsService, logger)
	healthChecker := handler.NewDBHealthChecker(pool)
	healthHandler := handler.NewHealthHandler(healthChecker, logger)

	apiMux := http.NewServeMux()

	apiMux.HandleFunc("POST /events", eventHandler.SearchEvents)
	apiMux.HandleFunc("POST /events/export", eventHandler.ExportEvents)
	apiMux.HandleFunc("POST /events/ingest", eventHandler.IngestEvent)
	apiMux.HandleFunc("POST /events/ingest/batch", eventHandler.IngestEventsBatch)

	apiMux.HandleFunc("GET /settings", settingsHandler.GetSettings)
	apiMux.HandleFunc("PUT /settings", settingsHandler.UpdateSettings)

	apiMux.HandleFunc("GET /version", healthHandler.Version)

	pipeline := middleware.Pipeline(
		middleware.Recovery(logger),
		middleware.Logging(logger),
	)

	mux := http.NewServeMux()
	mux.Handle("/", pipeline(apiMux))

	mux.HandleFunc("GET /readyz", healthHandler.Readiness)
	mux.HandleFunc("GET /healthz", healthHandler.Liveness)
	mux.Handle("GET /metrics", promhttp.Handler())

	app.httpServer = &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return app, nil
}

func (app *Application) Handler() http.Handler {
	return app.httpServer.Handler
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

	g.Go(func() error {
		addr := fmt.Sprintf(":%d", app.config.GRPC.Port)
		lis, err := net.Listen("tcp", addr)
		if err != nil {
			return fmt.Errorf("failed to listen on %s: %w", addr, err)
		}
		app.logger.Info("starting gRPC server", "port", app.config.GRPC.Port)
		if err := app.grpcServer.Serve(lis); err != nil {
			return fmt.Errorf("gRPC server error: %w", err)
		}
		return nil
	})

	<-globalCtx.Done()

	return g.Wait()
}

func (app *Application) Shutdown(ctx context.Context) error {
	app.logger.Info("shutting down application")

	if err := app.httpServer.Shutdown(ctx); err != nil {
		app.logger.Error("HTTP server forced to shutdown", "error", err)
		return err
	}

	if app.grpcServer != nil {
		stopped := make(chan struct{})
		go func() { app.grpcServer.GracefulStop(); close(stopped) }()
		select {
		case <-stopped:
		case <-ctx.Done():
			app.grpcServer.Stop()
		}
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
