package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Features FeatureFlags
	Database DatabaseConfig
	Kafka    KafkaConfig
	EventLog EventLogConfig
}

type FeatureFlags struct {
	KafkaEnabled bool
	AutoMigrate  bool
}

type DatabaseConfig struct {
	DSN               string
	MaxConns          int32         // Maximum number of connections (replaces MaxOpenConns)
	MinConns          int32         // Minimum number of idle connections (replaces MaxIdleConns)
	MaxConnLifetime   time.Duration // Maximum connection lifetime (replaces ConnMaxLifetime)
	MaxConnIdleTime   time.Duration // Maximum connection idle time
	HealthCheckPeriod time.Duration // Connection health check interval
}

type KafkaConfig struct {
	Brokers  string
	Topic    string
	GroupID  string
	Username string
	Password string
}

type EventLogConfig struct {
	RetentionPeriodDays int
	MaxExportSize       int
	RetentionCron       string
}

const (
	minRetentionDays = 1
	maxRetentionDays = 10 * 365
	minExportSize    = 1
	maxExportSize    = 100000
	minPoolSize      = 1
	maxPoolSize      = 100
)

// Load reads configuration from environment variables with defaults.
// It loads from configs/.env and configs/.env.local if present.
func Load() (*Config, error) {
	// Load base configuration from configs/.env
	if err := godotenv.Load("configs/.env"); err != nil {
		slog.Debug("no configs/.env file found, using environment variables or defaults")
	} else {
		slog.Info("loaded configuration from configs/.env")
	}

	// Load local overrides from configs/.env.local (optional)
	if err := godotenv.Load("configs/.env.local"); err != nil {
		slog.Debug("no configs/.env.local file found")
	} else {
		slog.Info("loaded local configuration overrides from configs/.env.local")
	}

	cfg := &Config{
		Features: FeatureFlags{
			KafkaEnabled: getBoolEnv("KAFKA_ENABLED", false),
			AutoMigrate:  getBoolEnv("AUTO_MIGRATE", true),
		},
		Database: DatabaseConfig{
			DSN:               getEnv("POSTGRES_DSN", ""),
			MaxConns:          int32(getIntEnv("DB_MAX_CONNS", 25)),
			MinConns:          int32(getIntEnv("DB_MIN_CONNS", 2)),
			MaxConnLifetime:   time.Duration(getIntEnv("DB_MAX_CONN_LIFETIME_MINUTES", 5)) * time.Minute,
			MaxConnIdleTime:   time.Duration(getIntEnv("DB_MAX_CONN_IDLE_TIME_MINUTES", 30)) * time.Minute,
			HealthCheckPeriod: time.Duration(getIntEnv("DB_HEALTH_CHECK_PERIOD_SECONDS", 60)) * time.Second,
		},
		Kafka: KafkaConfig{
			Brokers:  getEnv("KAFKA_BROKERS", "localhost:9092"),
			Topic:    getEnv("KAFKA_TOPIC", "events"),
			GroupID:  getEnv("KAFKA_GROUP_ID", "events-group"),
			Username: getEnv("KAFKA_USERNAME", ""),
			Password: getEnv("KAFKA_PASSWORD", ""),
		},
		EventLog: EventLogConfig{
			RetentionPeriodDays: getIntEnv("RETENTION_PERIOD_DAYS", 90),
			MaxExportSize:       getIntEnv("MAX_EXPORT_SIZE", 10000),
			RetentionCron:       getEnv("RETENTION_CRON", "0 2 * * *"),
		},
	}

	return cfg, nil
}

// Validate checks that the configuration values are valid.
// It returns an error if any required fields are missing or if values are out of range.
func (c *Config) Validate() error {
	var errs []error

	if c.Database.DSN == "" {
		errs = append(errs, errors.New("POSTGRES_DSN is required"))
	}

	if c.Features.KafkaEnabled {
		if c.Kafka.Brokers == "" {
			errs = append(errs, errors.New("KAFKA_BROKERS is required when Kafka is enabled"))
		}
		if c.Kafka.Topic == "" {
			errs = append(errs, errors.New("KAFKA_TOPIC is required when Kafka is enabled"))
		}
		if c.Kafka.GroupID == "" {
			errs = append(errs, errors.New("KAFKA_GROUP_ID is required when Kafka is enabled"))
		}
	}

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

func (c *KafkaConfig) GetBrokers() []string {
	return strings.Split(c.Brokers, ",")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		slog.Warn("failed to parse environment variable as integer, using default",
			"key", key,
			"value", valueStr,
			"default", defaultValue,
			"error", err)
		return defaultValue
	}
	return value
}

func getBoolEnv(key string, defaultValue bool) bool {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		slog.Warn("failed to parse environment variable as boolean, using default",
			"key", key,
			"value", valueStr,
			"default", defaultValue,
			"error", err)
		return defaultValue
	}
	return value
}
