package migrator

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// setupTestDatabase creates a PostgreSQL container for testing
func setupTestDatabase(t *testing.T) (string, func()) {
	t.Helper()

	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:17-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}

	// Get connection string
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	cleanup := func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	}

	return connStr, cleanup
}

// TestRunMigrations tests that migrations run successfully on a fresh database
func TestRunMigrations(t *testing.T) {
	dbURL, cleanup := setupTestDatabase(t)
	defer cleanup()

	// Run migrations
	err := Run(dbURL)
	if err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	// Verify tables were created
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	// Check events table exists
	var eventsExists bool
	err = db.QueryRow("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'events')").Scan(&eventsExists)
	if err != nil {
		t.Fatalf("failed to check events table: %v", err)
	}
	if !eventsExists {
		t.Error("events table should exist after migration")
	}

	// Check settings table exists
	var settingsExists bool
	err = db.QueryRow("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'settings')").Scan(&settingsExists)
	if err != nil {
		t.Fatalf("failed to check settings table: %v", err)
	}
	if !settingsExists {
		t.Error("settings table should exist after migration")
	}

	// Verify settings table has default data
	var retentionDays int
	err = db.QueryRow("SELECT retention_period_days FROM settings WHERE id = 'global'").Scan(&retentionDays)
	if err != nil {
		t.Fatalf("failed to query settings: %v", err)
	}
	if retentionDays != 90 {
		t.Errorf("expected retention_period_days = 90, got %d", retentionDays)
	}
}

// TestRunMigrationsIdempotent tests that running migrations multiple times is safe
func TestRunMigrationsIdempotent(t *testing.T) {
	dbURL, cleanup := setupTestDatabase(t)
	defer cleanup()

	// Run migrations first time
	err := Run(dbURL)
	if err != nil {
		t.Fatalf("first migration run failed: %v", err)
	}

	// Run migrations second time - should succeed with no changes
	err = Run(dbURL)
	if err != nil {
		t.Fatalf("second migration run failed: %v", err)
	}

	// Run migrations third time - still should succeed
	err = Run(dbURL)
	if err != nil {
		t.Fatalf("third migration run failed: %v", err)
	}
}

// TestMigrationIndices tests that all expected indices are created
func TestMigrationIndices(t *testing.T) {
	dbURL, cleanup := setupTestDatabase(t)
	defer cleanup()

	// Run migrations
	err := Run(dbURL)
	if err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	// Check for expected indices on events table
	expectedIndices := []string{
		"idx_events_timestamp",
		"idx_events_user",
		"idx_events_category",
		"idx_events_action",
		"idx_events_document",
		"idx_events_project",
		"idx_events_environment",
		"idx_events_tenant",
		"idx_events_correlation_id",
		"idx_events_trace_id",
		"idx_events_details",
	}

	for _, indexName := range expectedIndices {
		var exists bool
		query := `SELECT EXISTS (
			SELECT FROM pg_indexes
			WHERE tablename = 'events'
			AND indexname = $1
		)`
		err := db.QueryRow(query, indexName).Scan(&exists)
		if err != nil {
			t.Fatalf("failed to check index %s: %v", indexName, err)
		}
		if !exists {
			t.Errorf("index %s should exist but doesn't", indexName)
		}
	}
}

// TestConcurrentMigrations tests that concurrent migration attempts are handled safely
func TestConcurrentMigrations(t *testing.T) {
	dbURL, cleanup := setupTestDatabase(t)
	defer cleanup()

	// Start multiple migration goroutines concurrently
	const numGoroutines = 5
	errChan := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			err := Run(dbURL)
			errChan <- err
		}(i)
	}

	// Collect results
	var successCount int
	for i := 0; i < numGoroutines; i++ {
		err := <-errChan
		if err == nil {
			successCount++
		} else {
			t.Logf("goroutine %d returned error: %v", i, err)
		}
	}

	// At least one should succeed, others should either succeed or handle gracefully
	if successCount == 0 {
		t.Error("expected at least one migration to succeed")
	}

	// Verify database is in correct state
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	var eventsExists bool
	err = db.QueryRow("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'events')").Scan(&eventsExists)
	if err != nil {
		t.Fatalf("failed to check events table: %v", err)
	}
	if !eventsExists {
		t.Error("events table should exist after concurrent migrations")
	}
}

// TestMigrationSchemaVersion tests that migration version is tracked correctly
func TestMigrationSchemaVersion(t *testing.T) {
	dbURL, cleanup := setupTestDatabase(t)
	defer cleanup()

	// Run migrations
	err := Run(dbURL)
	if err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	// Check schema_migrations table exists and has correct version
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	var version int
	var dirty bool
	err = db.QueryRow("SELECT version, dirty FROM schema_migrations").Scan(&version, &dirty)
	if err != nil {
		t.Fatalf("failed to query schema_migrations: %v", err)
	}

	// We have 2 migration files (000001 and 000002)
	expectedVersion := 2
	if version != expectedVersion {
		t.Errorf("expected version %d, got %d", expectedVersion, version)
	}

	if dirty {
		t.Error("migration should not be in dirty state")
	}
}

// TestMigrationColumnTypes tests that columns have correct types
func TestMigrationColumnTypes(t *testing.T) {
	dbURL, cleanup := setupTestDatabase(t)
	defer cleanup()

	// Run migrations
	err := Run(dbURL)
	if err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test events table structure
	type columnInfo struct {
		columnName string
		dataType   string
	}

	expectedColumns := []columnInfo{
		{"id", "uuid"},
		{"timestamp", "timestamp with time zone"},
		{"user_name", "character varying"},
		{"category", "character varying"},
		{"action", "character varying"},
		{"details", "jsonb"},
		{"created_at", "timestamp with time zone"},
	}

	for _, expected := range expectedColumns {
		var dataType string
		query := `SELECT data_type
			FROM information_schema.columns
			WHERE table_name = 'events'
			AND column_name = $1`
		err := db.QueryRow(query, expected.columnName).Scan(&dataType)
		if err != nil {
			t.Errorf("failed to check column %s: %v", expected.columnName, err)
			continue
		}
		if dataType != expected.dataType {
			t.Errorf("column %s: expected type %s, got %s", expected.columnName, expected.dataType, dataType)
		}
	}
}

// TestDatabaseURLValidation tests error handling for invalid database URLs
func TestDatabaseURLValidation(t *testing.T) {
	// Test with invalid URL
	err := Run("postgres://invalid:invalid@nonexistent:5432/testdb")
	if err == nil {
		t.Error("expected error with invalid database URL")
	}
}

// BenchmarkRunMigrations benchmarks migration execution time
func BenchmarkRunMigrations(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark in short mode")
	}

	ctx := context.Background()

	// Create a single container for all iterations
	pgContainer, err := postgres.Run(ctx,
		"postgres:17-alpine",
		postgres.WithDatabase("benchdb"),
		postgres.WithUsername("benchuser"),
		postgres.WithPassword("benchpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		b.Fatalf("failed to start postgres container: %v", err)
	}
	defer pgContainer.Terminate(ctx)

	baseConnStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		b.Fatalf("failed to get connection string: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Create a fresh database for each iteration
		dbName := fmt.Sprintf("bench_%d", i)
		db, _ := sql.Open("postgres", baseConnStr)
		db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
		db.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
		db.Close()

		// Modify connection string to use new database
		testURL := baseConnStr[:len(baseConnStr)-len("benchdb?sslmode=disable")] + dbName + "?sslmode=disable"
		b.StartTimer()

		err := Run(testURL)
		if err != nil {
			b.Fatalf("migration failed: %v", err)
		}
	}
}
