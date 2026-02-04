package migrator

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed *.sql
var migrationsFS embed.FS

const (
	// advisoryLockID is used to prevent concurrent migrations
	// This is a PostgreSQL advisory lock ID that should be unique to this application
	advisoryLockID = 123456789
	// lockTimeout defines how long to wait for the advisory lock
	lockTimeout = 30 * time.Second
)

// Run executes database migrations using embedded migration files.
// It acquires a PostgreSQL advisory lock to prevent concurrent migrations,
// runs all pending migrations, and releases the lock.
//
// The function handles the ErrNoChange error gracefully (when migrations are already applied)
// and returns it as nil to avoid treating it as an error condition.
//
// Parameters:
//   - dbURL: PostgreSQL connection string (e.g., "postgres://user:pass@host:port/dbname?sslmode=disable")
//
// Returns:
//   - error: nil on success or if no changes needed, error otherwise
func Run(dbURL string) error {
	slog.Info("starting database migrations")

	// Connect to database for advisory lock
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Acquire advisory lock with timeout
	ctx, cancel := context.WithTimeout(context.Background(), lockTimeout)
	defer cancel()

	if err := acquireAdvisoryLock(ctx, db); err != nil {
		db.Close()
		return fmt.Errorf("failed to acquire advisory lock: %w", err)
	}

	slog.Info("acquired migration lock")

	// Create migration source from embedded filesystem
	sourceDriver, err := iofs.New(migrationsFS, ".")
	if err != nil {
		releaseAdvisoryLock(db)
		db.Close()
		return fmt.Errorf("failed to create migration source: %w", err)
	}

	// Create database driver for migrate
	// Note: postgres.WithInstance takes ownership of the db connection
	databaseDriver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		releaseAdvisoryLock(db)
		db.Close()
		return fmt.Errorf("failed to create database driver: %w", err)
	}

	// Create migrator instance
	m, err := migrate.NewWithInstance("iofs", sourceDriver, "postgres", databaseDriver)
	if err != nil {
		releaseAdvisoryLock(db)
		db.Close()
		return fmt.Errorf("failed to create migrator: %w", err)
	}

	// Run migrations
	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			slog.Info("no migrations to apply, database is up to date")
		} else {
			// Release lock and close before returning error
			releaseAdvisoryLock(db)
			m.Close()
			return fmt.Errorf("failed to run migrations: %w", err)
		}
	} else {
		slog.Info("migrations completed successfully")
	}

	// Release advisory lock before closing
	releaseAdvisoryLock(db)

	// Close migrator (this also closes the database connection)
	sourceErr, dbErr := m.Close()
	if sourceErr != nil {
		return fmt.Errorf("failed to close migration source: %w", sourceErr)
	}
	if dbErr != nil {
		return fmt.Errorf("failed to close database: %w", dbErr)
	}

	return nil
}

// acquireAdvisoryLock attempts to acquire a PostgreSQL advisory lock with the given context timeout.
// It uses pg_try_advisory_lock which is non-blocking and returns immediately.
// If the lock is not available, it retries with a small delay until the context times out.
func acquireAdvisoryLock(ctx context.Context, db *sql.DB) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for migration lock: %w", ctx.Err())
		case <-ticker.C:
			var locked bool
			err := db.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", advisoryLockID).Scan(&locked)
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
func releaseAdvisoryLock(db *sql.DB) {
	var released bool
	err := db.QueryRow("SELECT pg_advisory_unlock($1)", advisoryLockID).Scan(&released)
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
