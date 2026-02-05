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
	Database DatabaseConfig
	Kafka    KafkaConfig
	EventLog EventLogConfig
	Features FeatureFlags
}

// FeatureFlags holds feature toggles for optional functionality.
type FeatureFlags struct {
	KafkaEnabled bool
	AutoMigrate  bool
}

// DatabaseConfig holds PostgreSQL configuration.
type DatabaseConfig struct {
	// DSN is the PostgreSQL connection string (e.g., "postgres://user:pass@localhost:5432/dbname?sslmode=disable")
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// KafkaConfig holds Kafka configuration.
type KafkaConfig struct {
	// Brokers is a comma-separated list of broker addresses (e.g., "localhost:9092,localhost:9093")
	Brokers           string
	Topic             string
	GroupID           string
	Username          string
	Password          string
	MaxPollRecords    int
	FetchMinBytes     int
	FetchMaxWaitMs    int
	SessionTimeoutMs  int
	HeartbeatInterval int
}

// EventLogConfig holds event log specific configuration.
type EventLogConfig struct {
	RetentionPeriodDays int
	MaxExportSize       int
	RetentionCron       string
}

// Validation constants
const (
	minRetentionDays = 1
	maxRetentionDays = 3650 // 10 years
	minExportSize    = 1
	maxExportSize    = 100000
	minPoolSize      = 1
	maxPoolSize      = 100
)

// Load reads configuration from environment variables with defaults.
// It automatically loads from .env file if present.
func Load() (*Config, error) {
	// Load .env file if it exists (ignore error if file doesn't exist)
	if err := godotenv.Load(); err != nil {
		slog.Debug("no .env file found, using environment variables or defaults")
	} else {
		slog.Info("loaded configuration from .env file")
	}

	cfg := &Config{
		Database: DatabaseConfig{
			DSN:             getEnv("POSTGRES_DSN", ""),
			MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: time.Duration(getEnvAsInt("DB_CONN_MAX_LIFETIME_MINUTES", 5)) * time.Minute,
		},
		Kafka: KafkaConfig{
			Brokers:           getEnv("KAFKA_BROKERS", "localhost:9092"),
			Topic:             getEnv("KAFKA_TOPIC", "events"),
			GroupID:           getEnv("KAFKA_GROUP_ID", "events-group"),
			Username:          getEnv("KAFKA_USERNAME", ""),
			Password:          getEnv("KAFKA_PASSWORD", ""),
			MaxPollRecords:    getEnvAsInt("KAFKA_MAX_POLL_RECORDS", 100),
			FetchMinBytes:     getEnvAsInt("KAFKA_FETCH_MIN_BYTES", 1),
			FetchMaxWaitMs:    getEnvAsInt("KAFKA_FETCH_MAX_WAIT_MS", 500),
			SessionTimeoutMs:  getEnvAsInt("KAFKA_SESSION_TIMEOUT_MS", 30000),
			HeartbeatInterval: getEnvAsInt("KAFKA_HEARTBEAT_INTERVAL_MS", 10000),
		},
		EventLog: EventLogConfig{
			RetentionPeriodDays: getEnvAsInt("RETENTION_PERIOD_DAYS", 90),
			MaxExportSize:       getEnvAsInt("MAX_EXPORT_SIZE", 10000),
			RetentionCron:       getEnv("RETENTION_CRON", "0 2 * * *"), // Daily at 2 AM
		},
		Features: FeatureFlags{
			KafkaEnabled: getEnvAsBool("KAFKA_ENABLED", false),
			AutoMigrate:  getEnvAsBool("AUTO_MIGRATE", true),
		},
	}

	return cfg, nil
}

// Validate checks that the configuration values are valid.
// It returns an error if any required fields are missing or if values are out of range.
func (c *Config) Validate() error {
	var errs []error

	// Required fields
	if c.Database.DSN == "" {
		errs = append(errs, errors.New("POSTGRES_DSN is required"))
	}

	// Kafka validation (only when enabled)
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

	// Numeric range validations
	if c.EventLog.RetentionPeriodDays < minRetentionDays || c.EventLog.RetentionPeriodDays > maxRetentionDays {
		errs = append(errs, fmt.Errorf("RETENTION_PERIOD_DAYS must be between %d and %d, got %d",
			minRetentionDays, maxRetentionDays, c.EventLog.RetentionPeriodDays))
	}

	if c.EventLog.MaxExportSize < minExportSize || c.EventLog.MaxExportSize > maxExportSize {
		errs = append(errs, fmt.Errorf("MAX_EXPORT_SIZE must be between %d and %d, got %d",
			minExportSize, maxExportSize, c.EventLog.MaxExportSize))
	}

	if c.Database.MaxOpenConns < minPoolSize || c.Database.MaxOpenConns > maxPoolSize {
		errs = append(errs, fmt.Errorf("DB_MAX_OPEN_CONNS must be between %d and %d, got %d",
			minPoolSize, maxPoolSize, c.Database.MaxOpenConns))
	}

	if c.Database.MaxIdleConns < minPoolSize || c.Database.MaxIdleConns > maxPoolSize {
		errs = append(errs, fmt.Errorf("DB_MAX_IDLE_CONNS must be between %d and %d, got %d",
			minPoolSize, maxPoolSize, c.Database.MaxIdleConns))
	}

	if c.Database.MaxIdleConns > c.Database.MaxOpenConns {
		errs = append(errs, fmt.Errorf("DB_MAX_IDLE_CONNS (%d) cannot exceed DB_MAX_OPEN_CONNS (%d)",
			c.Database.MaxIdleConns, c.Database.MaxOpenConns))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// GetBrokers returns a list of Kafka broker addresses.
func (c *KafkaConfig) GetBrokers() []string {
	return strings.Split(c.Brokers, ",")
}

// FetchMaxWait returns the fetch max wait duration.
func (c *KafkaConfig) FetchMaxWait() time.Duration {
	return time.Duration(c.FetchMaxWaitMs) * time.Millisecond
}

// SessionTimeout returns the session timeout duration.
func (c *KafkaConfig) SessionTimeout() time.Duration {
	return time.Duration(c.SessionTimeoutMs) * time.Millisecond
}

// HeartbeatIntervalDuration returns the heartbeat interval duration.
func (c *KafkaConfig) HeartbeatIntervalDuration() time.Duration {
	return time.Duration(c.HeartbeatInterval) * time.Millisecond
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
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

func getEnvAsBool(key string, defaultValue bool) bool {
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
