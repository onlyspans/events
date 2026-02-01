package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Kafka    KafkaConfig
	EventLog EventLogConfig
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Port string
}

// DatabaseConfig holds PostgreSQL configuration.
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// KafkaConfig holds Kafka configuration.
type KafkaConfig struct {
	Host              string
	Port              string
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
func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8080"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("POSTGRES_HOST", "localhost"),
			Port:     getEnv("POSTGRES_PORT", "5432"),
			User:     getEnv("POSTGRES_USER", "postgres"),
			Password: getEnv("POSTGRES_PASSWORD", "postgres"),
			DBName:   getEnv("POSTGRES_DB", "eventlogs"),
			SSLMode:  getEnv("POSTGRES_SSLMODE", "disable"),
		},
		Kafka: KafkaConfig{
			Host:              getEnv("KAFKA_HOST", "localhost"),
			Port:              getEnv("KAFKA_PORT", "9092"),
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
	}

	return cfg, nil
}

// DSN returns the PostgreSQL connection string.
func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}

// BrokerAddress returns the Kafka broker address.
func (c *KafkaConfig) BrokerAddress() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
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
