package config

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Features FeatureFlags
	Database DatabaseConfig
	EventLog EventLogConfig
}

type FeatureFlags struct {
	AutoMigrate bool `envconfig:"AUTO_MIGRATE" default:"true"`
}

type DatabaseConfig struct {
	DSN               string        `envconfig:"POSTGRES_DSN" required:"true"`
	MaxConns          int32         `envconfig:"DB_MAX_CONNS" default:"25"`
	MinConns          int32         `envconfig:"DB_MIN_CONNS" default:"2"`
	MaxConnLifetime   time.Duration `envconfig:"DB_MAX_CONN_LIFETIME_MINUTES" default:"5m"`
	MaxConnIdleTime   time.Duration `envconfig:"DB_MAX_CONN_IDLE_TIME_MINUTES" default:"30m"`
	HealthCheckPeriod time.Duration `envconfig:"DB_HEALTH_CHECK_PERIOD_SECONDS" default:"60s"`
}

type EventLogConfig struct {
	RetentionPeriodDays int    `envconfig:"RETENTION_PERIOD_DAYS" default:"90"`
	MaxExportSize       int    `envconfig:"MAX_EXPORT_SIZE" default:"10000"`
	RetentionCron       string `envconfig:"RETENTION_CRON" default:"0 2 * * *"`
}

const (
	minRetentionDays = 1
	maxRetentionDays = 10 * 365
	minExportSize    = 1
	maxExportSize    = 100000
	minPoolSize      = 1
	maxPoolSize      = 100
)

func Load() (*Config, error) {
	if err := godotenv.Load("configs/.env"); err != nil {
		slog.Debug("no configs/.env file found, using environment variables or defaults")
	} else {
		slog.Info("loaded configuration from configs/.env")
	}

	if err := godotenv.Load("configs/.env.local"); err != nil {
		slog.Debug("no configs/.env.local file found")
	} else {
		slog.Info("loaded local configuration overrides from configs/.env.local")
	}

	var cfg Config

	if err := envconfig.Process("", &cfg.Features); err != nil {
		return nil, fmt.Errorf("failed to process feature flags: %w", err)
	}

	if err := envconfig.Process("", &cfg.Database); err != nil {
		return nil, fmt.Errorf("failed to process database config: %w", err)
	}

	if err := envconfig.Process("", &cfg.EventLog); err != nil {
		return nil, fmt.Errorf("failed to process event log config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	var errs []error

	if c.EventLog.RetentionPeriodDays < minRetentionDays || c.EventLog.RetentionPeriodDays > maxRetentionDays {
		errs = append(errs, fmt.Errorf("RETENTION_PERIOD_DAYS must be between %d and %d, got %d",
			minRetentionDays, maxRetentionDays, c.EventLog.RetentionPeriodDays))
	}

	if c.EventLog.MaxExportSize < minExportSize || c.EventLog.MaxExportSize > maxExportSize {
		errs = append(errs, fmt.Errorf("MAX_EXPORT_SIZE must be between %d and %d, got %d",
			minExportSize, maxExportSize, c.EventLog.MaxExportSize))
	}

	if c.Database.MaxConns < minPoolSize || c.Database.MaxConns > maxPoolSize {
		errs = append(errs, fmt.Errorf("DB_MAX_CONNS must be between %d and %d, got %d",
			minPoolSize, maxPoolSize, c.Database.MaxConns))
	}

	if c.Database.MinConns < 0 || c.Database.MinConns > maxPoolSize {
		errs = append(errs, fmt.Errorf("DB_MIN_CONNS must be between 0 and %d, got %d",
			maxPoolSize, c.Database.MinConns))
	}

	if c.Database.MinConns > c.Database.MaxConns {
		errs = append(errs, fmt.Errorf("DB_MIN_CONNS (%d) cannot exceed DB_MAX_CONNS (%d)",
			c.Database.MinConns, c.Database.MaxConns))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
