package migrator

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

//go:embed *.sql
var migrationsFS embed.FS

const (
	// advisoryLockID is used to prevent concurrent migrations
	// This is a PostgreSQL advisory lock ID that should be unique to this application
	advisoryLockID = 123456789
	// lockTimeout defines how long to wait for the advisory lock
	lockTimeout = 30 * time.Second
	// migrationTimeout defines the maximum time for the entire migration process
	migrationTimeout = 5 * time.Minute
)

// Run executes database migrations using embedded migration files.
// It acquires a PostgreSQL advisory lock to prevent concurrent migrations,
// runs all pending migrations, and releases the lock.
//
// The function handles the ErrNoChange error gracefully (when migrations are already applied)
// and returns it as nil to avoid treating it as an error condition.
//
// The entire migration process has a 5-minute timeout to prevent indefinite hangs.
//
// Parameters:
//   - dbURL: PostgreSQL connection string (e.g., "postgres://user:pass@host:port/dbname?sslmode=disable")
//
// Returns:
//   - error: nil on success or if no changes needed, error otherwise
func Run(dbURL string) error {
	slog.Info("starting database migrations")

	// Create context with timeout for entire migration process
	ctx, cancel := context.WithTimeout(context.Background(), migrationTimeout)
	defer cancel()

	// Parse connection config
	poolConfig, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Create connection pool for migrations
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return fmt.Errorf("failed to create connection pool: %w", err)
	}
	defer pool.Close()

	// Acquire advisory lock with timeout
	lockCtx, lockCancel := context.WithTimeout(ctx, lockTimeout)
	defer lockCancel()

	if err := acquireAdvisoryLock(lockCtx, pool); err != nil {
		return fmt.Errorf("failed to acquire advisory lock: %w", err)
	}

	slog.Info("acquired migration lock")

	// Ensure lock is released on exit
	defer releaseAdvisoryLock(pool)

	// Create migration source from embedded filesystem
	sourceDriver, err := iofs.New(migrationsFS, ".")
	if err != nil {
		return fmt.Errorf("failed to create migration source: %w", err)
	}

	// Get stdlib database connection from pool for migrate
	db := stdlib.OpenDBFromPool(pool)

	// Create database driver for migrate using pgx v5
	databaseDriver, err := pgx.WithInstance(db, &pgx.Config{})
	if err != nil {
		return fmt.Errorf("failed to create database driver: %w", err)
	}

	// Create migrator instance
	m, err := migrate.NewWithInstance("iofs", sourceDriver, "postgres", databaseDriver)
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}
	defer m.Close()

	// Run migrations
	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			slog.Info("no migrations to apply, database is up to date")
		} else {
			return fmt.Errorf("failed to run migrations: %w", err)
		}
	} else {
		slog.Info("migrations completed successfully")
	}

	return nil
}

// acquireAdvisoryLock attempts to acquire a PostgreSQL advisory lock with the given context timeout.
// It uses pg_try_advisory_lock which is non-blocking and returns immediately.
// If the lock is not available, it retries with a small delay until the context times out.
func acquireAdvisoryLock(ctx context.Context, pool *pgxpool.Pool) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for migration lock: %w", ctx.Err())
		case <-ticker.C:
			var locked bool
			err := pool.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", advisoryLockID).Scan(&locked)
			if err != nil {
				return fmt.Errorf("failed to acquire advisory lock: %w", err)
			}
			if locked {
				return nil
			}
			slog.Debug("waiting for migration lock to become available")
		}
	}
}

// releaseAdvisoryLock releases the PostgreSQL advisory lock.
// This should be called with defer after successfully acquiring the lock.
func releaseAdvisoryLock(pool *pgxpool.Pool) {
	var released bool
	err := pool.QueryRow(context.Background(), "SELECT pg_advisory_unlock($1)", advisoryLockID).Scan(&released)
	if err != nil {
		slog.Error("failed to release advisory lock", "error", err)
		return
	}
	if !released {
		slog.Warn("advisory lock was not held when trying to release")
	} else {
		slog.Info("released migration lock")
	}
}
