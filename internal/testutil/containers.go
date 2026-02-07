package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/onlyspans/events/internal/migrator"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// PostgresContainer wraps a testcontainers PostgreSQL instance
type PostgresContainer struct {
	Container *postgres.PostgresContainer
	DB        *sql.DB
	DSN       string
}

// SetupPostgres creates a PostgreSQL testcontainer and returns a database connection.
// The container is automatically cleaned up when the test finishes.
func SetupPostgres(t *testing.T) *PostgresContainer {
	t.Helper()

	ctx := context.Background()

	// Create PostgreSQL container
	pgContainer, err := postgres.Run(ctx,
		"postgres:17-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}

	// Cleanup on test completion
	t.Cleanup(func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Errorf("failed to terminate postgres container: %v", err)
		}
	})

	// Get connection string
	dsn, err := pgContainer.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	// Connect to database
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}

	// Cleanup database connection
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("failed to close database connection: %v", err)
		}
	})

	// Configure connection pool
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("failed to ping database: %v", err)
	}

	return &PostgresContainer{
		Container: pgContainer,
		DB:        db,
		DSN:       dsn,
	}
}

// SetupPostgresWithMigrations creates a PostgreSQL testcontainer and runs migrations.
func SetupPostgresWithMigrations(t *testing.T) *PostgresContainer {
	t.Helper()

	pc := SetupPostgres(t)

	// Run migrations
	if err := migrator.Run(pc.DSN); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	return pc
}

// TruncateTables truncates all tables in the database (useful for test cleanup between subtests)
func (pc *PostgresContainer) TruncateTables(t *testing.T, tables ...string) {
	t.Helper()

	ctx := context.Background()
	for _, table := range tables {
		query := fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE", table)
		if _, err := pc.DB.ExecContext(ctx, query); err != nil {
			t.Errorf("failed to truncate table %s: %v", table, err)
		}
	}
}
