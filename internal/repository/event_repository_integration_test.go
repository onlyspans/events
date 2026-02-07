package repository

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/onlyspans/events/internal/domain"
	"github.com/onlyspans/events/internal/migrations"
	"github.com/onlyspans/events/internal/ports"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupTestDB(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()

	ctx := context.Background()

	// Start PostgreSQL container
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

	// Run migrations from migrations directory
	migrationsPath := filepath.Join("..", "..", "migrations")
	if err := migrations.Run(connStr, migrationsPath); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	// Create connection pool
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("failed to create connection pool: %v", err)
	}

	// Cleanup function
	cleanup := func() {
		pool.Close()
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Errorf("failed to terminate container: %v", err)
		}
	}

	return pool, cleanup
}

func TestEventRepository_Create(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewEventRepository(pool)
	ctx := context.Background()

	tests := []struct {
		name    string
		event   *domain.Event
		wantErr bool
		verify  func(t *testing.T, pool *pgxpool.Pool, id uuid.UUID)
	}{
		{
			name: "create single event with required fields",
			event: &domain.Event{
				ID:        uuid.New(),
				Timestamp: time.Now(),
				User:      "test-user",
				Category:  "test-category",
				Action:    "test-action",
			},
			wantErr: false,
			verify: func(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) {
				var user, category, action string
				err := pool.QueryRow(ctx,
					"SELECT user_name, category, action FROM events WHERE id = $1",
					id).Scan(&user, &category, &action)
				if err != nil {
					t.Fatalf("failed to verify event: %v", err)
				}
				if user != "test-user" || category != "test-category" || action != "test-action" {
					t.Errorf("event fields mismatch: got (%s, %s, %s)", user, category, action)
				}
			},
		},
		{
			name: "create event with all fields",
			event: &domain.Event{
				ID:            uuid.New(),
				Timestamp:     time.Now(),
				User:          "user1",
				Category:      "category1",
				Action:        "action1",
				DocumentName:  "doc1",
				Project:       "project1",
				Environment:   "production",
				Tenant:        "tenant1",
				CorrelationID: "corr-123",
				TraceID:       "trace-456",
			},
			wantErr: false,
			verify: func(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) {
				var count int
				err := pool.QueryRow(ctx,
					"SELECT COUNT(*) FROM events WHERE id = $1 AND project = $2 AND environment = $3",
					id, "project1", "production").Scan(&count)
				if err != nil {
					t.Fatalf("failed to verify event: %v", err)
				}
				if count != 1 {
					t.Error("event not found with expected fields")
				}
			},
		},
		{
			name: "create event with JSONB details",
			event: &domain.Event{
				ID:        uuid.New(),
				Timestamp: time.Now(),
				User:      "test-user",
				Category:  "test-category",
				Action:    "test-action",
				Details: &domain.EventDetails{
					IPAddress:      "192.168.1.1",
					UserAgent:      "Test Agent/1.0",
					AdditionalInfo: "Additional test info",
					Changes: []domain.Change{
						{Field: "status", OldValue: "active", NewValue: "inactive"},
						{Field: "name", OldValue: "old", NewValue: "new"},
					},
				},
			},
			wantErr: false,
			verify: func(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) {
				var detailsJSON []byte
				err := pool.QueryRow(ctx,
					"SELECT details FROM events WHERE id = $1",
					id).Scan(&detailsJSON)
				if err != nil {
					t.Fatalf("failed to verify event details: %v", err)
				}
				if len(detailsJSON) == 0 {
					t.Error("details field is empty")
				}

				// Verify JSONB can be queried
				var ipAddress string
				err = pool.QueryRow(ctx,
					"SELECT details->>'ipAddress' FROM events WHERE id = $1",
					id).Scan(&ipAddress)
				if err != nil {
					t.Fatalf("failed to query JSONB field: %v", err)
				}
				if ipAddress != "192.168.1.1" {
					t.Errorf("expected IP '192.168.1.1', got '%s'", ipAddress)
				}
			},
		},
		{
			name: "create event with nil details",
			event: &domain.Event{
				ID:        uuid.New(),
				Timestamp: time.Now(),
				User:      "test-user",
				Category:  "test-category",
				Action:    "test-action",
				Details:   nil,
			},
			wantErr: false,
			verify: func(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) {
				var details *string
				err := pool.QueryRow(ctx,
					"SELECT details FROM events WHERE id = $1",
					id).Scan(&details)
				if err != nil {
					t.Fatalf("failed to verify event: %v", err)
				}
				if details != nil {
					t.Error("expected NULL details, got non-NULL value")
				}
			},
		},
		{
			name: "create event with optional fields empty",
			event: &domain.Event{
				ID:          uuid.New(),
				Timestamp:   time.Now(),
				User:        "test-user",
				Category:    "test-category",
				Action:      "test-action",
				Project:     "", // Empty optional field
				Environment: "", // Empty optional field
			},
			wantErr: false,
			verify: func(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) {
				var project, environment *string
				err := pool.QueryRow(ctx,
					"SELECT project, environment FROM events WHERE id = $1",
					id).Scan(&project, &environment)
				if err != nil {
					t.Fatalf("failed to verify event: %v", err)
				}
				if project != nil || environment != nil {
					t.Error("expected NULL for empty optional fields")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := repo.Create(ctx, tt.event)
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if id == uuid.Nil {
					t.Error("Create() returned nil UUID")
				}
				if id != tt.event.ID {
					t.Errorf("Create() returned ID %s, expected %s", id, tt.event.ID)
				}

				if tt.verify != nil {
					tt.verify(t, pool, id)
				}
			}
		})
	}
}

func TestEventRepository_SaveBatch(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewEventRepository(pool)
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
				err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM events WHERE user_name = $1", tt.events[0].User).Scan(&count)
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
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewEventRepository(pool)
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
		query       ports.EventSearchQuery
		wantCount   int
		wantTotal   int64
		checkResult func(t *testing.T, events []*domain.Event)
	}{
		{
			name: "search by user",
			query: ports.EventSearchQuery{
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
			query: ports.EventSearchQuery{
				Category: "category1",
				Page:     0,
				Size:     10,
			},
			wantCount: 2,
			wantTotal: 2,
		},
		{
			name: "search by project and environment",
			query: ports.EventSearchQuery{
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
			query: ports.EventSearchQuery{
				Page: 0,
				Size: 1,
			},
			wantCount: 1,
			wantTotal: 3,
		},
		{
			name: "search all",
			query: ports.EventSearchQuery{
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
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewEventRepository(pool)
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
	events, total, err := repo.Search(ctx, ports.EventSearchQuery{Page: 0, Size: 10})
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
