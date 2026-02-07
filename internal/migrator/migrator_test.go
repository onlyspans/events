package migrator

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// TestAdvisoryLock tests the advisory lock acquisition and release
func TestAdvisoryLock(t *testing.T) {
	// This test requires a real PostgreSQL database
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	dbURL := getTestDatabaseURL(t)
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Test acquiring lock
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = acquireAdvisoryLock(ctx, pool)
	if err != nil {
		t.Fatalf("failed to acquire advisory lock: %v", err)
	}

	// Test that we can check if lock is held
	var locked bool
	err = pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_locks WHERE locktype = 'advisory' AND objid = $1)", advisoryLockID).Scan(&locked)
	if err != nil {
		t.Fatalf("failed to check lock status: %v", err)
	}
	if !locked {
		t.Error("lock should be held but it's not")
	}

	// Release lock
	releaseAdvisoryLock(pool)

	// Verify lock is released
	err = pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_locks WHERE locktype = 'advisory' AND objid = $1)", advisoryLockID).Scan(&locked)
	if err != nil {
		t.Fatalf("failed to check lock status after release: %v", err)
	}
	if locked {
		t.Error("lock should be released but it's still held")
	}
}

// TestAdvisoryLockConcurrency tests that concurrent lock attempts are blocked
func TestAdvisoryLockConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	dbURL := getTestDatabaseURL(t)

	// First connection acquires lock
	pool1, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	defer pool1.Close()

	ctx1, cancel1 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel1()

	err = acquireAdvisoryLock(ctx1, pool1)
	if err != nil {
		t.Fatalf("failed to acquire advisory lock: %v", err)
	}
	defer releaseAdvisoryLock(pool1)

	// Second connection should timeout trying to acquire
	pool2, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	defer pool2.Close()

	ctx2, cancel2 := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel2()

	err = acquireAdvisoryLock(ctx2, pool2)
	if err == nil {
		t.Error("second lock acquisition should have failed but succeeded")
		releaseAdvisoryLock(pool2)
	}
	if err != nil && ctx2.Err() == nil {
		t.Errorf("expected timeout error, got: %v", err)
	}
}

// getTestDatabaseURL returns a database URL for testing
// This should match your test environment setup
func getTestDatabaseURL(t *testing.T) string {
	t.Helper()

	// This is a placeholder - in real tests, this would come from environment
	// or use testcontainers
	dbURL := "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"

	// Try to connect to verify database is available
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		t.Skipf("skipping test: cannot connect to test database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		t.Skipf("skipping test: test database not available: %v", err)
	}

	return dbURL
}
