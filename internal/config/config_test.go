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
