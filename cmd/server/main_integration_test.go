package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/onlyspans/events/internal/migrator"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestStartupWithMigrations verifies that the application starts correctly with auto-migration enabled
func TestStartupWithMigrations(t *testing.T) {
	ctx := context.Background()

	// Start PostgreSQL container
	postgresContainer, err := postgres.Run(ctx,
		"postgres:17-alpine",
		postgres.WithDatabase("events_test"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}
	defer func() {
		if err := testcontainers.TerminateContainer(postgresContainer); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	}()

	// Get connection string
	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	// Test: Run migrations with AUTO_MIGRATE=true (simulating startup)
	t.Run("migrations run on startup", func(t *testing.T) {
		// Parse connection string to get components
		db, err := sql.Open("postgres", connStr)
		if err != nil {
			t.Fatalf("failed to connect to database: %v", err)
		}
		defer db.Close()

		// Run migrations (simulating what main.go does)
		if err := migrator.Run(connStr); err != nil {
			t.Fatalf("migration failed: %v", err)
		}

		// Verify tables exist
		tables := []string{"events", "settings", "schema_migrations"}
		for _, table := range tables {
			var exists bool
			query := `SELECT EXISTS (
				SELECT FROM information_schema.tables
				WHERE table_schema = 'public'
				AND table_name = $1
			)`
			err := db.QueryRow(query, table).Scan(&exists)
			if err != nil {
				t.Fatalf("failed to check if table %s exists: %v", table, err)
			}
			if !exists {
				t.Errorf("table %s does not exist after migrations", table)
			}
		}

		// Verify settings table has default row
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM settings").Scan(&count)
		if err != nil {
			t.Fatalf("failed to count settings rows: %v", err)
		}
		if count != 1 {
			t.Errorf("expected 1 row in settings table, got %d", count)
		}
	})

	// Test: Running migrations again should be idempotent (ErrNoChange)
	t.Run("migrations are idempotent", func(t *testing.T) {
		// Run migrations again
		err := migrator.Run(connStr)
		if err != nil {
			t.Fatalf("second migration run failed: %v", err)
		}

		// Verify no errors on idempotent run
		db, err := sql.Open("postgres", connStr)
		if err != nil {
			t.Fatalf("failed to connect to database: %v", err)
		}
		defer db.Close()

		// Check migration version is still valid
		var version int
		var dirty bool
		err = db.QueryRow("SELECT version, dirty FROM schema_migrations").Scan(&version, &dirty)
		if err != nil {
			t.Fatalf("failed to read schema_migrations: %v", err)
		}
		if dirty {
			t.Error("schema_migrations is marked as dirty after idempotent run")
		}
	})

	// Test: Concurrent migration attempts (advisory lock test)
	t.Run("concurrent migrations are serialized", func(t *testing.T) {
		// Create fresh database for this test
		freshContainer, err := postgres.Run(ctx,
			"postgres:17-alpine",
			postgres.WithDatabase("events_concurrent"),
			postgres.WithUsername("postgres"),
			postgres.WithPassword("test"),
			testcontainers.WithWaitStrategy(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2).
					WithStartupTimeout(60*time.Second),
			),
		)
		if err != nil {
			t.Fatalf("failed to start fresh postgres container: %v", err)
		}
		defer func() {
			if err := testcontainers.TerminateContainer(freshContainer); err != nil {
				t.Logf("failed to terminate container: %v", err)
			}
		}()

		freshConnStr, err := freshContainer.ConnectionString(ctx, "sslmode=disable")
		if err != nil {
			t.Fatalf("failed to get connection string: %v", err)
		}

		// Run two migrations concurrently
		errChan := make(chan error, 2)

		go func() {
			errChan <- migrator.Run(freshConnStr)
		}()

		go func() {
			// Small delay to ensure first migration starts
			time.Sleep(10 * time.Millisecond)
			errChan <- migrator.Run(freshConnStr)
		}()

		// Both should succeed (one applies migrations, one gets ErrNoChange)
		for i := 0; i < 2; i++ {
			if err := <-errChan; err != nil {
				t.Errorf("concurrent migration %d failed: %v", i+1, err)
			}
		}

		// Verify final state is correct
		db, err := sql.Open("postgres", freshConnStr)
		if err != nil {
			t.Fatalf("failed to connect to database: %v", err)
		}
		defer db.Close()

		var dirty bool
		err = db.QueryRow("SELECT dirty FROM schema_migrations").Scan(&dirty)
		if err != nil {
			t.Fatalf("failed to read schema_migrations: %v", err)
		}
		if dirty {
			t.Error("schema_migrations is marked as dirty after concurrent runs")
		}
	})
}

// TestStartupWithoutMigrations verifies that the application can start with AUTO_MIGRATE=false
func TestStartupWithoutMigrations(t *testing.T) {
	ctx := context.Background()

	// Start PostgreSQL container with pre-applied migrations
	postgresContainer, err := postgres.Run(ctx,
		"postgres:17-alpine",
		postgres.WithDatabase("events_test"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}
	defer func() {
		if err := testcontainers.TerminateContainer(postgresContainer); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	}()

	// Get connection string
	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	// Apply migrations first
	if err := migrator.Run(connStr); err != nil {
		t.Fatalf("initial migration failed: %v", err)
	}

	// Verify database is ready (simulating AUTO_MIGRATE=false startup)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Fatalf("database ping failed: %v", err)
	}

	// Verify tables exist (as they would need to for AUTO_MIGRATE=false)
	var exists bool
	err = db.QueryRow(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'public'
			AND table_name = 'events'
		)
	`).Scan(&exists)
	if err != nil {
		t.Fatalf("failed to check if events table exists: %v", err)
	}
	if !exists {
		t.Error("events table does not exist (required for AUTO_MIGRATE=false mode)")
	}

	t.Log("Service can start successfully with AUTO_MIGRATE=false when schema is pre-applied")
}

// TestHealthEndpoints verifies health endpoints work after migrations
func TestHealthEndpoints(t *testing.T) {
	// This test would require starting the full HTTP server
	// For now, we verify that the migration + DB connection is sufficient
	// Full E2E tests would be done in a separate test suite
	t.Skip("Full HTTP server testing requires E2E test environment")
}

// Helper function to get database connection from config (for testing)
func getTestDSN(host, port, user, password, dbname string) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		user, password, host, port, dbname)
}

// TestMain handles test setup and teardown
func TestMain(m *testing.M) {
	// Set test environment variables
	os.Setenv("POSTGRES_PASSWORD", "test")
	os.Setenv("AUTO_MIGRATE", "true")
	os.Setenv("KAFKA_ENABLED", "false")

	// Run tests
	code := m.Run()

	// Cleanup
	os.Exit(code)
}
