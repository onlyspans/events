package config

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all application configuration.
//
// Environment Variables:
//
// Required:
//   - POSTGRES_DSN: PostgreSQL connection string
//     Example: "postgres://user:pass@localhost:5432/dbname?sslmode=disable"
//
// Optional:
//   - SERVER_PORT (default: 8080)
//   - KAFKA_ENABLED (default: false)
//   - AUTO_MIGRATE (default: true)
//   - RETENTION_PERIOD_DAYS (default: 90)
//   - MAX_EXPORT_SIZE (default: 10000)
//   - RETENTION_CRON (default: "0 2 * * *")
//
// Kafka Configuration (when KAFKA_ENABLED=true):
//   - KAFKA_BROKERS: Comma-separated broker addresses (e.g., "localhost:9092,broker2:9092")
//   - KAFKA_TOPIC (default: event-logs)
//   - KAFKA_GROUP_ID (default: event-logs-group)
//   - KAFKA_USERNAME (optional, enables SASL/SCRAM)
//   - KAFKA_PASSWORD (optional, enables SASL/SCRAM)
//   - KAFKA_MAX_POLL_RECORDS (default: 100)
//   - KAFKA_FETCH_MIN_BYTES (default: 1)
//   - KAFKA_FETCH_MAX_WAIT_MS (default: 500)
//   - KAFKA_SESSION_TIMEOUT_MS (default: 30000)
//   - KAFKA_HEARTBEAT_INTERVAL_MS (default: 10000)
type Config struct {
	Server   ServerConfig
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

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Port string
}

// DatabaseConfig holds PostgreSQL configuration.
type DatabaseConfig struct {
	// DSN is the PostgreSQL connection string (e.g., "postgres://user:pass@localhost:5432/dbname?sslmode=disable")
	DSN string
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
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8080"),
		},
		Database: DatabaseConfig{
			DSN: getEnv("POSTGRES_DSN", ""),
		},
		Kafka: KafkaConfig{
			Brokers:           getEnv("KAFKA_BROKERS", "localhost:9092"),
			Topic:             getEnv("KAFKA_TOPIC", "event-logs"),
			GroupID:           getEnv("KAFKA_GROUP_ID", "event-logs-group"),
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

