package config

import (
	"os"
	"testing"
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

			result := getEnvAsBool(testKey, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getEnvAsBool(%q, %v) = %v; want %v", tt.envValue, tt.defaultValue, result, tt.expected)
			}
		})
	}
}

func TestLoad_MinimalConfiguration(t *testing.T) {
	// Clear all environment variables to test minimal config
	envVars := []string{
		"SERVER_PORT",
		"POSTGRES_HOST", "POSTGRES_PORT", "POSTGRES_USER", "POSTGRES_PASSWORD", "POSTGRES_DB", "POSTGRES_SSLMODE",
		"KAFKA_ENABLED", "KAFKA_HOST", "KAFKA_PORT", "KAFKA_TOPIC", "KAFKA_GROUP_ID",
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

	// Verify defaults
	if cfg.Server.Port != "8080" {
		t.Errorf("expected default SERVER_PORT=8080, got %s", cfg.Server.Port)
	}
	if cfg.Database.Host != "localhost" {
		t.Errorf("expected default POSTGRES_HOST=localhost, got %s", cfg.Database.Host)
	}
	if cfg.Database.Port != "5432" {
		t.Errorf("expected default POSTGRES_PORT=5432, got %s", cfg.Database.Port)
	}
	if cfg.Database.User != "postgres" {
		t.Errorf("expected default POSTGRES_USER=postgres, got %s", cfg.Database.User)
	}
	if cfg.Database.Password != "" {
		t.Errorf("expected empty default POSTGRES_PASSWORD (required), got %s", cfg.Database.Password)
	}
	if cfg.Database.DBName != "events" {
		t.Errorf("expected default POSTGRES_DB=events, got %s", cfg.Database.DBName)
	}
	if cfg.Database.SSLMode != "disable" {
		t.Errorf("expected default POSTGRES_SSLMODE=disable, got %s", cfg.Database.SSLMode)
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

func TestLoad_RequiredVsOptionalVariables(t *testing.T) {
	// Test that service can start with only POSTGRES_PASSWORD set
	envVars := []string{
		"SERVER_PORT",
		"POSTGRES_HOST", "POSTGRES_PORT", "POSTGRES_USER", "POSTGRES_PASSWORD", "POSTGRES_DB", "POSTGRES_SSLMODE",
		"KAFKA_ENABLED", "KAFKA_HOST", "KAFKA_PORT", "KAFKA_TOPIC", "KAFKA_GROUP_ID",
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

	// Set only the required variable
	os.Setenv("POSTGRES_PASSWORD", "test-password")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() with only POSTGRES_PASSWORD set should succeed, got error: %v", err)
	}

	// Verify required field is set
	if cfg.Database.Password != "test-password" {
		t.Errorf("expected POSTGRES_PASSWORD=test-password, got %s", cfg.Database.Password)
	}

	// Verify all defaults are applied
	if cfg.Server.Port != "8080" {
		t.Errorf("expected default SERVER_PORT=8080, got %s", cfg.Server.Port)
	}
	if cfg.Database.Host != "localhost" {
		t.Errorf("expected default POSTGRES_HOST=localhost, got %s", cfg.Database.Host)
	}
	if cfg.Database.User != "postgres" {
		t.Errorf("expected default POSTGRES_USER=postgres, got %s", cfg.Database.User)
	}
}
