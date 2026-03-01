package app

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/onlyspans/events/internal/config"
	"github.com/onlyspans/events/internal/migrations"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
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

func setupPostgres(cfg *config.Config, logger *slog.Logger) (*pgxpool.Pool, error) {
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

	return pool, nil
}
