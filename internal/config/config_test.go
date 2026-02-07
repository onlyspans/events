package config

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestLoad_DefaultFeatureFlags(t *testing.T) {
	// Clear environment variables to test defaults
	os.Unsetenv("KAFKA_ENABLED")
	os.Unsetenv("AUTO_MIGRATE")
	os.Setenv("POSTGRES_DSN", "postgres://test")
	defer os.Unsetenv("POSTGRES_DSN")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Test default values
	if cfg.Features.KafkaEnabled != false {
		t.Errorf("expected KafkaEnabled=false by default, got %v", cfg.Features.KafkaEnabled)
	}
	if cfg.Features.AutoMigrate != true {
		t.Errorf("expected AutoMigrate=true by default, got %v", cfg.Features.AutoMigrate)
	}
}

func TestLoad_FeatureFlagsFromEnv(t *testing.T) {
	tests := []struct {
		name                string
		kafkaEnabledEnv     string
		autoMigrateEnv      string
		expectedKafka       bool
		expectedAutoMigrate bool
	}{
		{
			name:                "both true",
			kafkaEnabledEnv:     "true",
			autoMigrateEnv:      "true",
			expectedKafka:       true,
			expectedAutoMigrate: true,
		},
		{
			name:                "both false",
			kafkaEnabledEnv:     "false",
			autoMigrateEnv:      "false",
			expectedKafka:       false,
			expectedAutoMigrate: false,
		},
		{
			name:                "kafka true, migrate false",
			kafkaEnabledEnv:     "true",
			autoMigrateEnv:      "false",
			expectedKafka:       true,
			expectedAutoMigrate: false,
		},
		{
			name:                "kafka false, migrate true",
			kafkaEnabledEnv:     "false",
			autoMigrateEnv:      "true",
			expectedKafka:       false,
			expectedAutoMigrate: true,
		},
		{
			name:                "numeric true (1)",
			kafkaEnabledEnv:     "1",
			autoMigrateEnv:      "1",
			expectedKafka:       true,
			expectedAutoMigrate: true,
		},
		{
			name:                "numeric false (0)",
			kafkaEnabledEnv:     "0",
			autoMigrateEnv:      "0",
			expectedKafka:       false,
			expectedAutoMigrate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			os.Setenv("POSTGRES_DSN", "postgres://test")
			os.Setenv("KAFKA_ENABLED", tt.kafkaEnabledEnv)
			os.Setenv("AUTO_MIGRATE", tt.autoMigrateEnv)
			defer func() {
				os.Unsetenv("POSTGRES_DSN")
				os.Unsetenv("KAFKA_ENABLED")
				os.Unsetenv("AUTO_MIGRATE")
			}()

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() failed: %v", err)
			}

			if cfg.Features.KafkaEnabled != tt.expectedKafka {
				t.Errorf("expected KafkaEnabled=%v, got %v", tt.expectedKafka, cfg.Features.KafkaEnabled)
			}
			if cfg.Features.AutoMigrate != tt.expectedAutoMigrate {
				t.Errorf("expected AutoMigrate=%v, got %v", tt.expectedAutoMigrate, cfg.Features.AutoMigrate)
			}
		})
	}
}

func TestLoad_InvalidBooleanValues(t *testing.T) {
	tests := []struct {
		name            string
		kafkaEnabledEnv string
		autoMigrateEnv  string
		shouldError     bool
	}{
		{
			name:            "invalid kafka value",
			kafkaEnabledEnv: "invalid",
			autoMigrateEnv:  "true",
			shouldError:     true,
		},
		{
			name:            "invalid migrate value",
			kafkaEnabledEnv: "true",
			autoMigrateEnv:  "invalid",
			shouldError:     true,
		},
		{
			name:            "both invalid",
			kafkaEnabledEnv: "invalid",
			autoMigrateEnv:  "invalid",
			shouldError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			os.Setenv("KAFKA_ENABLED", tt.kafkaEnabledEnv)
			os.Setenv("AUTO_MIGRATE", tt.autoMigrateEnv)
			os.Setenv("POSTGRES_DSN", "postgres://test") // Required field
			defer func() {
				os.Unsetenv("KAFKA_ENABLED")
				os.Unsetenv("AUTO_MIGRATE")
				os.Unsetenv("POSTGRES_DSN")
			}()

			_, err := Load()
			if tt.shouldError && err == nil {
				t.Error("Load() should return error for invalid boolean value")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("Load() returned unexpected error: %v", err)
			}
		})
	}
}

func TestLoad_MinimalConfiguration(t *testing.T) {
	// Clear all environment variables to test minimal config (except required POSTGRES_DSN)
	envVars := []string{
		"SERVER_PORT",
		"POSTGRES_DSN",
		"KAFKA_ENABLED", "KAFKA_BROKERS", "KAFKA_TOPIC", "KAFKA_GROUP_ID",
		"RETENTION_PERIOD_DAYS", "MAX_EXPORT_SIZE", "RETENTION_CRON",
		"AUTO_MIGRATE",
	}
	for _, v := range envVars {
		os.Unsetenv(v)
	}
	defer func() {
		for _, v := range envVars {
			os.Unsetenv(v)
		}
	}()

	// Set required field
	testDSN := "postgres://testuser:testpass@localhost:5432/testdb"
	os.Setenv("POSTGRES_DSN", testDSN)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.Database.DSN != testDSN {
		t.Errorf("expected POSTGRES_DSN=%s, got %s", testDSN, cfg.Database.DSN)
	}
	if cfg.Kafka.Brokers != "localhost:9092" {
		t.Errorf("expected default KAFKA_BROKERS=localhost:9092, got %s", cfg.Kafka.Brokers)
	}
	if cfg.EventLog.RetentionPeriodDays != 90 {
		t.Errorf("expected default RETENTION_PERIOD_DAYS=90, got %d", cfg.EventLog.RetentionPeriodDays)
	}
	if cfg.EventLog.MaxExportSize != 10000 {
		t.Errorf("expected default MAX_EXPORT_SIZE=10000, got %d", cfg.EventLog.MaxExportSize)
	}
	if cfg.EventLog.RetentionCron != "0 2 * * *" {
		t.Errorf("expected default RETENTION_CRON='0 2 * * *', got %s", cfg.EventLog.RetentionCron)
	}
	if cfg.Features.KafkaEnabled != false {
		t.Errorf("expected default KAFKA_ENABLED=false, got %v", cfg.Features.KafkaEnabled)
	}
	if cfg.Features.AutoMigrate != true {
		t.Errorf("expected default AUTO_MIGRATE=true, got %v", cfg.Features.AutoMigrate)
	}
}

func TestLoad_WithPostgresDSN(t *testing.T) {
	// Clear all environment variables
	envVars := []string{
		"SERVER_PORT",
		"POSTGRES_DSN",
		"KAFKA_ENABLED", "KAFKA_BROKERS", "KAFKA_TOPIC", "KAFKA_GROUP_ID",
		"RETENTION_PERIOD_DAYS", "MAX_EXPORT_SIZE", "RETENTION_CRON",
		"AUTO_MIGRATE",
	}
	for _, v := range envVars {
		os.Unsetenv(v)
	}
	defer func() {
		for _, v := range envVars {
			os.Unsetenv(v)
		}
	}()

	// Set only POSTGRES_DSN
	testDSN := "postgres://testuser:testpass@testhost:5433/testdb?sslmode=require"
	os.Setenv("POSTGRES_DSN", testDSN)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify DSN field is set
	if cfg.Database.DSN != testDSN {
		t.Errorf("expected POSTGRES_DSN=%q, got %q", testDSN, cfg.Database.DSN)
	}
}

func TestLoad_WithKafkaBrokers(t *testing.T) {
	// Clear all environment variables
	envVars := []string{
		"SERVER_PORT",
		"POSTGRES_DSN",
		"KAFKA_ENABLED", "KAFKA_BROKERS", "KAFKA_TOPIC", "KAFKA_GROUP_ID",
		"RETENTION_PERIOD_DAYS", "MAX_EXPORT_SIZE", "RETENTION_CRON",
		"AUTO_MIGRATE",
	}
	for _, v := range envVars {
		os.Unsetenv(v)
	}
	defer func() {
		for _, v := range envVars {
			os.Unsetenv(v)
		}
	}()

	// Set required field and Kafka brokers
	os.Setenv("POSTGRES_DSN", "postgres://test")
	testBrokers := "broker1:9092,broker2:9093,broker3:9094"
	os.Setenv("KAFKA_BROKERS", testBrokers)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify Brokers field is set
	if cfg.Kafka.Brokers != testBrokers {
		t.Errorf("expected KAFKA_BROKERS=%q, got %q", testBrokers, cfg.Kafka.Brokers)
	}

	// Verify GetBrokers() returns parsed brokers
	brokers := cfg.Kafka.GetBrokers()
	expectedBrokers := []string{"broker1:9092", "broker2:9093", "broker3:9094"}

	if len(brokers) != len(expectedBrokers) {
		t.Errorf("expected %d brokers, got %d", len(expectedBrokers), len(brokers))
		return
	}

	for i, broker := range brokers {
		if broker != expectedBrokers[i] {
			t.Errorf("broker[%d] = %q; want %q", i, broker, expectedBrokers[i])
		}
	}
}

func TestKafkaConfig_GetBrokers(t *testing.T) {
	tests := []struct {
		name     string
		brokers  string
		expected []string
	}{
		{
			name:     "single broker",
			brokers:  "localhost:9092",
			expected: []string{"localhost:9092"},
		},
		{
			name:     "multiple brokers",
			brokers:  "localhost:9092,localhost:9093,localhost:9094",
			expected: []string{"localhost:9092", "localhost:9093", "localhost:9094"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := KafkaConfig{
				Brokers: tt.brokers,
			}

			result := cfg.GetBrokers()

			if len(result) != len(tt.expected) {
				t.Errorf("GetBrokers() returned %d brokers; want %d", len(result), len(tt.expected))
				return
			}

			for i, broker := range result {
				if broker != tt.expected[i] {
					t.Errorf("GetBrokers()[%d] = %q; want %q", i, broker, tt.expected[i])
				}
			}
		})
	}
}

// validConfig returns a Config with valid values for testing.
func validConfig() *Config {
	return &Config{
		Database: DatabaseConfig{
			DSN:               "postgres://user:pass@localhost:5432/db",
			MaxConns:          25,
			MinConns:          2,
			MaxConnLifetime:   5 * time.Minute,
			MaxConnIdleTime:   30 * time.Minute,
			HealthCheckPeriod: 60 * time.Second,
		},
		Kafka: KafkaConfig{
			Brokers: "localhost:9092",
			Topic:   "events",
			GroupID: "events-group",
		},
		EventLog: EventLogConfig{
			RetentionPeriodDays: 90,
			MaxExportSize:       10000,
			RetentionCron:       "0 2 * * *",
		},
		Features: FeatureFlags{
			KafkaEnabled: false,
			AutoMigrate:  true,
		},
	}
}

func TestConfig_Validate_Valid(t *testing.T) {
	cfg := validConfig()
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() returned unexpected error: %v", err)
	}
}

func TestConfig_Validate_RetentionPeriodDays(t *testing.T) {
	tests := []struct {
		name    string
		days    int
		wantErr bool
	}{
		{"valid minimum", 1, false},
		{"valid middle", 90, false},
		{"valid maximum", 3650, false},
		{"too low", 0, true},
		{"negative", -1, true},
		{"too high", 3651, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			cfg.EventLog.RetentionPeriodDays = tt.days

			err := cfg.Validate()
			if tt.wantErr {
				if err == nil {
					t.Error("Validate() should return error")
				} else if !strings.Contains(err.Error(), "RETENTION_PERIOD_DAYS") {
					t.Errorf("error should mention RETENTION_PERIOD_DAYS, got: %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() returned unexpected error: %v", err)
				}
			}
		})
	}
}

func TestConfig_Validate_MaxExportSize(t *testing.T) {
	tests := []struct {
		name    string
		size    int
		wantErr bool
	}{
		{"valid minimum", 1, false},
		{"valid middle", 10000, false},
		{"valid maximum", 100000, false},
		{"too low", 0, true},
		{"negative", -1, true},
		{"too high", 100001, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			cfg.EventLog.MaxExportSize = tt.size

			err := cfg.Validate()
			if tt.wantErr {
				if err == nil {
					t.Error("Validate() should return error")
				} else if !strings.Contains(err.Error(), "MAX_EXPORT_SIZE") {
					t.Errorf("error should mention MAX_EXPORT_SIZE, got: %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() returned unexpected error: %v", err)
				}
			}
		})
	}
}

func TestConfig_Validate_DatabasePoolSize(t *testing.T) {
	tests := []struct {
		name        string
		maxConns    int32
		minConns    int32
		wantErr     bool
		errContains string
	}{
		{"valid", 25, 5, false, ""},
		{"valid equal", 10, 10, false, ""},
		{"max conns too low", 0, 5, true, "DB_MAX_CONNS"},
		{"max conns too high", 101, 5, true, "DB_MAX_CONNS"},
		{"min conns negative", 25, -1, true, "DB_MIN_CONNS"},
		{"min conns too high", 25, 101, true, "DB_MIN_CONNS"},
		{"min exceeds max", 10, 15, true, "cannot exceed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			cfg.Database.MaxConns = tt.maxConns
			cfg.Database.MinConns = tt.minConns

			err := cfg.Validate()
			if tt.wantErr {
				if err == nil {
					t.Error("Validate() should return error")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error should contain %q, got: %v", tt.errContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() returned unexpected error: %v", err)
				}
			}
		})
	}
}

func TestConfig_Validate_MultipleErrors(t *testing.T) {
	cfg := validConfig()
	cfg.EventLog.RetentionPeriodDays = 0
	cfg.EventLog.MaxExportSize = 0
	cfg.Database.MaxConns = 0

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() should return error")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "RETENTION_PERIOD_DAYS") {
		t.Error("error should mention RETENTION_PERIOD_DAYS")
	}
	if !strings.Contains(errStr, "MAX_EXPORT_SIZE") {
		t.Error("error should mention MAX_EXPORT_SIZE")
	}
	if !strings.Contains(errStr, "DB_MAX_CONNS") {
		t.Error("error should mention DB_MAX_CONNS")
	}
}

func TestLoad_NewConfigFields(t *testing.T) {
	// Clear all environment variables
	envVars := []string{
		"POSTGRES_DSN", "DB_MAX_CONNS", "DB_MIN_CONNS", "DB_MAX_CONN_LIFETIME_MINUTES",
		"KAFKA_ENABLED", "KAFKA_BROKERS", "KAFKA_TOPIC", "KAFKA_GROUP_ID",
		"RETENTION_PERIOD_DAYS", "MAX_EXPORT_SIZE", "RETENTION_CRON",
		"AUTO_MIGRATE",
	}
	for _, v := range envVars {
		os.Unsetenv(v)
	}
	defer func() {
		for _, v := range envVars {
			os.Unsetenv(v)
		}
	}()

	// Set required field
	os.Setenv("POSTGRES_DSN", "postgres://test")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.Database.MaxConns != 25 {
		t.Errorf("expected default MaxConns=25, got %d", cfg.Database.MaxConns)
	}
	if cfg.Database.MinConns != 2 {
		t.Errorf("expected default MinConns=2, got %d", cfg.Database.MinConns)
	}
	if cfg.Database.MaxConnLifetime != 5*time.Minute {
		t.Errorf("expected default MaxConnLifetime=5m, got %v", cfg.Database.MaxConnLifetime)
	}
}

func TestLoad_CustomConfigFields(t *testing.T) {
	// Set custom values
	os.Setenv("POSTGRES_DSN", "postgres://test")
	os.Setenv("DB_MAX_CONNS", "50")
	os.Setenv("DB_MIN_CONNS", "10")
	os.Setenv("DB_MAX_CONN_LIFETIME_MINUTES", "10m")
	defer func() {
		os.Unsetenv("POSTGRES_DSN")
		os.Unsetenv("DB_MAX_CONNS")
		os.Unsetenv("DB_MIN_CONNS")
		os.Unsetenv("DB_MAX_CONN_LIFETIME_MINUTES")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.Database.MaxConns != 50 {
		t.Errorf("expected MaxConns=50, got %d", cfg.Database.MaxConns)
	}
	if cfg.Database.MinConns != 10 {
		t.Errorf("expected MinConns=10, got %d", cfg.Database.MinConns)
	}
	if cfg.Database.MaxConnLifetime != 10*time.Minute {
		t.Errorf("expected MaxConnLifetime=10m, got %v", cfg.Database.MaxConnLifetime)
	}
}
