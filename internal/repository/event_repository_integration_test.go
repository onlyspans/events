package repository

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/onlyspans/events/internal/domain"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()

	ctx := context.Background()

	// Get absolute path to migrations directory
	migrationsPath, err := filepath.Abs(filepath.Join("..", "..", "migrations"))
	if err != nil {
		t.Fatalf("failed to get migrations path: %v", err)
	}

	// Discover all .up.sql migration files
	migrations, err := filepath.Glob(filepath.Join(migrationsPath, "*.up.sql"))
	if err != nil {
		t.Fatalf("failed to find migration files: %v", err)
	}
	if len(migrations) == 0 {
		t.Fatalf("no migration files found in %s", migrationsPath)
	}

	// Start PostgreSQL container with migration scripts
	// WithInitScripts executes scripts in alphabetical order
	pgContainer, err := postgres.Run(ctx,
		"postgres:17-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		postgres.WithInitScripts(migrations...),
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

	// Connect to database
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}

	// Cleanup function
	cleanup := func() {
		db.Close()
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Errorf("failed to terminate container: %v", err)
		}
	}

	return db, cleanup
}

func TestEventRepository_SaveBatch(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewEventRepository(db)
	ctx := context.Background()

	tests := []struct {
		name    string
		events  []*domain.Event
		wantErr bool
	}{
		{
			name: "save single event",
			events: []*domain.Event{
				{
					ID:        uuid.New(),
					Timestamp: time.Now(),
					User:      "test-user",
					Category:  "test-category",
					Action:    "test-action",
					Project:   "test-project",
				},
			},
			wantErr: false,
		},
		{
			name: "save multiple events",
			events: []*domain.Event{
				{
					ID:        uuid.New(),
					Timestamp: time.Now(),
					User:      "user1",
					Category:  "category1",
					Action:    "action1",
				},
				{
					ID:        uuid.New(),
					Timestamp: time.Now(),
					User:      "user2",
					Category:  "category2",
					Action:    "action2",
				},
			},
			wantErr: false,
		},
		{
			name: "save event with details",
			events: []*domain.Event{
				{
					ID:        uuid.New(),
					Timestamp: time.Now(),
					User:      "test-user",
					Category:  "test-category",
					Action:    "test-action",
					Details: &domain.EventDetails{
						IPAddress: "192.168.1.1",
						UserAgent: "Test Agent",
						Changes: []domain.Change{
							{Field: "status", OldValue: "active", NewValue: "inactive"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "save empty batch",
			events:  []*domain.Event{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.SaveBatch(ctx, tt.events)
			if (err != nil) != tt.wantErr {
				t.Errorf("SaveBatch() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Verify events were saved
			if !tt.wantErr && len(tt.events) > 0 {
				var count int
				err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM events WHERE user_name = $1", tt.events[0].User).Scan(&count)
				if err != nil {
					t.Errorf("failed to verify saved events: %v", err)
				}
				if count == 0 {
					t.Error("events were not saved to database")
				}
			}
		})
	}
}

func TestEventRepository_Search(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewEventRepository(db)
	ctx := context.Background()

	// Insert test events
	testEvents := []*domain.Event{
		{
			ID:          uuid.New(),
			Timestamp:   time.Now().Add(-1 * time.Hour),
			User:        "user1",
			Category:    "category1",
			Action:      "action1",
			Project:     "project1",
			Environment: "production",
			Tenant:      "tenant1",
		},
		{
			ID:          uuid.New(),
			Timestamp:   time.Now().Add(-2 * time.Hour),
			User:        "user2",
			Category:    "category2",
			Action:      "action2",
			Project:     "project2",
			Environment: "staging",
			Tenant:      "tenant1",
		},
		{
			ID:          uuid.New(),
			Timestamp:   time.Now().Add(-3 * time.Hour),
			User:        "user1",
			Category:    "category1",
			Action:      "action3",
			Project:     "project1",
			Environment: "production",
			Tenant:      "tenant2",
		},
	}

	if err := repo.SaveBatch(ctx, testEvents); err != nil {
		t.Fatalf("failed to insert test events: %v", err)
	}

	tests := []struct {
		name        string
		query       SearchQuery
		wantCount   int
		wantTotal   int64
		checkResult func(t *testing.T, events []*domain.Event)
	}{
		{
			name: "search by user",
			query: SearchQuery{
				User: "user1",
				Page: 0,
				Size: 10,
			},
			wantCount: 2,
			wantTotal: 2,
			checkResult: func(t *testing.T, events []*domain.Event) {
				for _, e := range events {
					if e.User != "user1" {
						t.Errorf("expected user 'user1', got '%s'", e.User)
					}
				}
			},
		},
		{
			name: "search by category",
			query: SearchQuery{
				Category: "category1",
				Page:     0,
				Size:     10,
			},
			wantCount: 2,
			wantTotal: 2,
		},
		{
			name: "search by project and environment",
			query: SearchQuery{
				Project:     "project1",
				Environment: "production",
				Page:        0,
				Size:        10,
			},
			wantCount: 2,
			wantTotal: 2,
		},
		{
			name: "search with pagination",
			query: SearchQuery{
				Page: 0,
				Size: 1,
			},
			wantCount: 1,
			wantTotal: 3,
		},
		{
			name: "search all",
			query: SearchQuery{
				Page: 0,
				Size: 10,
			},
			wantCount: 3,
			wantTotal: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events, total, err := repo.Search(ctx, tt.query)
			if err != nil {
				t.Errorf("Search() error = %v", err)
				return
			}

			if len(events) != tt.wantCount {
				t.Errorf("Search() got %d events, want %d", len(events), tt.wantCount)
			}

			if total != tt.wantTotal {
				t.Errorf("Search() got total %d, want %d", total, tt.wantTotal)
			}

			if tt.checkResult != nil {
				tt.checkResult(t, events)
			}
		})
	}
}

func TestEventRepository_DeleteOlderThan(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewEventRepository(db)
	ctx := context.Background()

	now := time.Now()

	// Insert test events with different timestamps
	testEvents := []*domain.Event{
		{
			ID:        uuid.New(),
			Timestamp: now.Add(-100 * 24 * time.Hour), // 100 days old
			User:      "old-user",
			Category:  "category1",
			Action:    "action1",
		},
		{
			ID:        uuid.New(),
			Timestamp: now.Add(-50 * 24 * time.Hour), // 50 days old
			User:      "mid-user",
			Category:  "category1",
			Action:    "action1",
		},
		{
			ID:        uuid.New(),
			Timestamp: now.Add(-1 * time.Hour), // Recent
			User:      "new-user",
			Category:  "category1",
			Action:    "action1",
		},
	}

	if err := repo.SaveBatch(ctx, testEvents); err != nil {
		t.Fatalf("failed to insert test events: %v", err)
	}

	// Delete events older than 90 days
	cutoffDate := now.Add(-90 * 24 * time.Hour)
	deleted, err := repo.DeleteOlderThan(ctx, cutoffDate)
	if err != nil {
		t.Fatalf("DeleteOlderThan() error = %v", err)
	}

	if deleted != 1 {
		t.Errorf("DeleteOlderThan() deleted %d events, want 1", deleted)
	}

	// Verify remaining events
	events, total, err := repo.Search(ctx, SearchQuery{Page: 0, Size: 10})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if total != 2 {
		t.Errorf("Expected 2 remaining events, got %d", total)
	}

	// Verify the old event was deleted
	for _, e := range events {
		if e.User == "old-user" {
			t.Error("Old event was not deleted")
		}
	}
}
