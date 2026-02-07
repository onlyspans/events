package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/onlyspans/events/internal/migrations"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// PostgresContainer wraps a testcontainers PostgreSQL instance
type PostgresContainer struct {
	Container *postgres.PostgresContainer
	Pool      *pgxpool.Pool // Primary interface for pgx
	DB        *sql.DB       // Keep for backward compatibility
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

	// Parse connection config
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("failed to parse DSN: %v", err)
	}

	// Configure connection pool for testing
	poolConfig.MaxConns = 5
	poolConfig.MinConns = 2
	poolConfig.MaxConnLifetime = 5 * time.Minute

	// Create connection pool
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		t.Fatalf("failed to create connection pool: %v", err)
	}

	// Cleanup pool connection
	t.Cleanup(func() {
		pool.Close()
	})

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("failed to ping database: %v", err)
	}

	// Create database/sql connection for backward compatibility
	db := stdlib.OpenDBFromPool(pool)

	// Cleanup database connection
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("failed to close database connection: %v", err)
		}
	})

	return &PostgresContainer{
		Container: pgContainer,
		Pool:      pool,
		DB:        db,
		DSN:       dsn,
	}
}

// SetupPostgresWithMigrations creates a PostgreSQL testcontainer and runs migrations.
func SetupPostgresWithMigrations(t *testing.T, migrationsPath string) *PostgresContainer {
	t.Helper()

	pc := SetupPostgres(t)

	// Run migrations
	if migrationsPath == "" {
		migrationsPath = filepath.Join("..", "..", "migrations")
	}
	if err := migrations.Run(pc.DSN, migrationsPath); err != nil {
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
		if _, err := pc.Pool.Exec(ctx, query); err != nil {
			t.Errorf("failed to truncate table %s: %v", table, err)
		}
	}
}
