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
			os.Setenv("KAFKA_ENABLED", tt.kafkaEnabledEnv)
			os.Setenv("AUTO_MIGRATE", tt.autoMigrateEnv)
			defer func() {
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
		name                string
		kafkaEnabledEnv     string
		autoMigrateEnv      string
		expectedKafka       bool // should fall back to defaults
		expectedAutoMigrate bool // should fall back to defaults
	}{
		{
			name:                "invalid kafka value",
			kafkaEnabledEnv:     "invalid",
			autoMigrateEnv:      "true",
			expectedKafka:       false, // default
			expectedAutoMigrate: true,
		},
		{
			name:                "invalid migrate value",
			kafkaEnabledEnv:     "true",
			autoMigrateEnv:      "invalid",
			expectedKafka:       true,
			expectedAutoMigrate: true, // default
		},
		{
			name:                "both invalid",
			kafkaEnabledEnv:     "invalid",
			autoMigrateEnv:      "invalid",
			expectedKafka:       false, // default
			expectedAutoMigrate: true,  // default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			os.Setenv("KAFKA_ENABLED", tt.kafkaEnabledEnv)
			os.Setenv("AUTO_MIGRATE", tt.autoMigrateEnv)
			defer func() {
				os.Unsetenv("KAFKA_ENABLED")
				os.Unsetenv("AUTO_MIGRATE")
			}()

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() failed: %v", err)
			}

			// Invalid values should fall back to defaults
			if cfg.Features.KafkaEnabled != tt.expectedKafka {
				t.Errorf("expected KafkaEnabled=%v (default on invalid input), got %v", tt.expectedKafka, cfg.Features.KafkaEnabled)
			}
			if cfg.Features.AutoMigrate != tt.expectedAutoMigrate {
				t.Errorf("expected AutoMigrate=%v (default on invalid input), got %v", tt.expectedAutoMigrate, cfg.Features.AutoMigrate)
			}
		})
	}
}

func TestGetEnvAsBool(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue bool
		expected     bool
	}{
		{"empty uses default true", "", true, true},
		{"empty uses default false", "", false, false},
		{"true string", "true", false, true},
		{"false string", "false", true, false},
		{"1 is true", "1", false, true},
		{"0 is false", "0", true, false},
		{"TRUE uppercase", "TRUE", false, true},
		{"FALSE uppercase", "FALSE", true, false},
		{"invalid uses default", "invalid", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testKey := "TEST_BOOL_VAR"
			if tt.envValue != "" {
				os.Setenv(testKey, tt.envValue)
				defer os.Unsetenv(testKey)
			} else {
				os.Unsetenv(testKey)
			}

			result := getBoolEnv(testKey, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getBoolEnv(%q, %v) = %v; want %v", tt.envValue, tt.defaultValue, result, tt.expected)
			}
		})
	}
}

func TestLoad_MinimalConfiguration(t *testing.T) {
	// Clear all environment variables to test minimal config
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

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.Database.DSN != "" {
		t.Errorf("expected empty default POSTGRES_DSN, got %s", cfg.Database.DSN)
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

	// Set Kafka brokers
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
			DSN:             "postgres://user:pass@localhost:5432/db",
			MaxOpenConns:    25,
			MaxIdleConns:    5,
			ConnMaxLifetime: 5 * time.Minute,
		},
		Kafka: KafkaConfig{
			Brokers:      "localhost:9092",
			Topic:        "events",
			GroupID:      "events-group",
			BatchSize:    100,
			BatchTimeout: 500 * time.Millisecond,
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

func TestConfig_Validate_MissingPostgresDSN(t *testing.T) {
	cfg := validConfig()
	cfg.Database.DSN = ""

	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() should return error for missing POSTGRES_DSN")
	}
	if !strings.Contains(err.Error(), "POSTGRES_DSN") {
		t.Errorf("error should mention POSTGRES_DSN, got: %v", err)
	}
}

func TestConfig_Validate_KafkaEnabled(t *testing.T) {
	tests := []struct {
		name        string
		brokers     string
		topic       string
		groupID     string
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid kafka config",
			brokers: "localhost:9092",
			topic:   "events",
			groupID: "events-group",
			wantErr: false,
		},
		{
			name:        "missing brokers",
			brokers:     "",
			topic:       "events",
			groupID:     "events-group",
			wantErr:     true,
			errContains: "KAFKA_BROKERS",
		},
		{
			name:        "missing topic",
			brokers:     "localhost:9092",
			topic:       "",
			groupID:     "events-group",
			wantErr:     true,
			errContains: "KAFKA_TOPIC",
		},
		{
			name:        "missing group id",
			brokers:     "localhost:9092",
			topic:       "events",
			groupID:     "",
			wantErr:     true,
			errContains: "KAFKA_GROUP_ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			cfg.Features.KafkaEnabled = true
			cfg.Kafka.Brokers = tt.brokers
			cfg.Kafka.Topic = tt.topic
			cfg.Kafka.GroupID = tt.groupID

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

func TestConfig_Validate_KafkaBatchSize(t *testing.T) {
	tests := []struct {
		name    string
		size    int
		wantErr bool
	}{
		{"valid minimum", 1, false},
		{"valid middle", 100, false},
		{"valid maximum", 1000, false},
		{"too low", 0, true},
		{"negative", -1, true},
		{"too high", 1001, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			cfg.Features.KafkaEnabled = true
			cfg.Kafka.BatchSize = tt.size

			err := cfg.Validate()
			if tt.wantErr {
				if err == nil {
					t.Error("Validate() should return error")
				} else if !strings.Contains(err.Error(), "KAFKA_BATCH_SIZE") {
					t.Errorf("error should mention KAFKA_BATCH_SIZE, got: %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() returned unexpected error: %v", err)
				}
			}
		})
	}
}

func TestConfig_Validate_KafkaBatchTimeout(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
		wantErr bool
	}{
		{"valid minimum", 100 * time.Millisecond, false},
		{"valid middle", 500 * time.Millisecond, false},
		{"valid maximum", 30 * time.Second, false},
		{"too low", 50 * time.Millisecond, true},
		{"zero", 0, true},
		{"too high", 31 * time.Second, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			cfg.Features.KafkaEnabled = true
			cfg.Kafka.BatchTimeout = tt.timeout

			err := cfg.Validate()
			if tt.wantErr {
				if err == nil {
					t.Error("Validate() should return error")
				} else if !strings.Contains(err.Error(), "KAFKA_BATCH_TIMEOUT_MS") {
					t.Errorf("error should mention KAFKA_BATCH_TIMEOUT_MS, got: %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() returned unexpected error: %v", err)
				}
			}
		})
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
		maxOpen     int
		maxIdle     int
		wantErr     bool
		errContains string
	}{
		{"valid", 25, 5, false, ""},
		{"valid equal", 10, 10, false, ""},
		{"max open too low", 0, 5, true, "DB_MAX_OPEN_CONNS"},
		{"max open too high", 101, 5, true, "DB_MAX_OPEN_CONNS"},
		{"max idle too low", 25, 0, true, "DB_MAX_IDLE_CONNS"},
		{"max idle too high", 25, 101, true, "DB_MAX_IDLE_CONNS"},
		{"idle exceeds open", 10, 15, true, "cannot exceed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			cfg.Database.MaxOpenConns = tt.maxOpen
			cfg.Database.MaxIdleConns = tt.maxIdle

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
	cfg.Database.DSN = ""
	cfg.EventLog.RetentionPeriodDays = 0
	cfg.EventLog.MaxExportSize = 0

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() should return error")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "POSTGRES_DSN") {
		t.Error("error should mention POSTGRES_DSN")
	}
	if !strings.Contains(errStr, "RETENTION_PERIOD_DAYS") {
		t.Error("error should mention RETENTION_PERIOD_DAYS")
	}
	if !strings.Contains(errStr, "MAX_EXPORT_SIZE") {
		t.Error("error should mention MAX_EXPORT_SIZE")
	}
}

func TestLoad_NewConfigFields(t *testing.T) {
	// Clear all environment variables
	envVars := []string{
		"SERVER_PORT", "SERVER_READ_TIMEOUT_SECONDS", "SERVER_WRITE_TIMEOUT_SECONDS", "SERVER_IDLE_TIMEOUT_SECONDS",
		"POSTGRES_DSN", "DB_MAX_OPEN_CONNS", "DB_MAX_IDLE_CONNS", "DB_CONN_MAX_LIFETIME_MINUTES",
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

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.Database.MaxOpenConns != 25 {
		t.Errorf("expected default MaxOpenConns=25, got %d", cfg.Database.MaxOpenConns)
	}
	if cfg.Database.MaxIdleConns != 5 {
		t.Errorf("expected default MaxIdleConns=5, got %d", cfg.Database.MaxIdleConns)
	}
	if cfg.Database.ConnMaxLifetime != 5*time.Minute {
		t.Errorf("expected default ConnMaxLifetime=5m, got %v", cfg.Database.ConnMaxLifetime)
	}
}

func TestLoad_CustomConfigFields(t *testing.T) {
	// Set custom values
	os.Setenv("SERVER_READ_TIMEOUT_SECONDS", "10")
	os.Setenv("SERVER_WRITE_TIMEOUT_SECONDS", "20")
	os.Setenv("SERVER_IDLE_TIMEOUT_SECONDS", "45")
	os.Setenv("DB_MAX_OPEN_CONNS", "50")
	os.Setenv("DB_MAX_IDLE_CONNS", "10")
	os.Setenv("DB_CONN_MAX_LIFETIME_MINUTES", "10")
	defer func() {
		os.Unsetenv("SERVER_READ_TIMEOUT_SECONDS")
		os.Unsetenv("SERVER_WRITE_TIMEOUT_SECONDS")
		os.Unsetenv("SERVER_IDLE_TIMEOUT_SECONDS")
		os.Unsetenv("DB_MAX_OPEN_CONNS")
		os.Unsetenv("DB_MAX_IDLE_CONNS")
		os.Unsetenv("DB_CONN_MAX_LIFETIME_MINUTES")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.Database.MaxOpenConns != 50 {
		t.Errorf("expected MaxOpenConns=50, got %d", cfg.Database.MaxOpenConns)
	}
	if cfg.Database.MaxIdleConns != 10 {
		t.Errorf("expected MaxIdleConns=10, got %d", cfg.Database.MaxIdleConns)
	}
	if cfg.Database.ConnMaxLifetime != 10*time.Minute {
		t.Errorf("expected ConnMaxLifetime=10m, got %v", cfg.Database.ConnMaxLifetime)
	}
}
