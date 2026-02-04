package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/onlyspans/events/internal/dto"
	"github.com/onlyspans/events/internal/migrator"
	"github.com/onlyspans/events/internal/repository"
	"github.com/onlyspans/events/internal/service"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupTestDB(t *testing.T) (*sql.DB, func()) {
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

	// Run embedded migrations using the migrator package
	if err := migrator.Run(connStr); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
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

func TestEventHandler_IngestEvent(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Setup dependencies
	eventRepo := repository.NewEventRepository(db)
	eventService := service.NewEventService(eventRepo, 10000) // maxExportSize = 10000
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	handler := NewEventHandler(eventService, logger)

	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name: "valid event",
			requestBody: dto.EventIngestRequest{
				UserName: "test-user",
				Category: "test-category",
				Action:   "test-action",
				Project:  "test-project",
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, body []byte) {
				var resp dto.SingleIngestResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.ID.String() == "" {
					t.Error("expected non-empty event ID")
				}
			},
		},
		{
			name: "missing required field",
			requestBody: dto.EventIngestRequest{
				UserName: "test-user",
				Category: "test-category",
				// Missing Action
			},
			expectedStatus: http.StatusInternalServerError, // Validation error returns 500 from service
			checkResponse:  nil,
		},
		{
			name: "invalid json",
			requestBody: "invalid json",
			expectedStatus: http.StatusBadRequest,
			checkResponse:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare request body
			var body []byte
			var err error
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				if err != nil {
					t.Fatalf("failed to marshal request: %v", err)
				}
			}

			// Create HTTP request
			req := httptest.NewRequest(http.MethodPost, "/events/ingest", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Call handler
			handler.IngestEvent(w, req)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check response if needed
			if tt.checkResponse != nil {
				tt.checkResponse(t, w.Body.Bytes())
			}
		})
	}
}

func TestEventHandler_IngestEventsBatch(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Setup dependencies
	eventRepo := repository.NewEventRepository(db)
	eventService := service.NewEventService(eventRepo, 10000) // maxExportSize = 10000
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	handler := NewEventHandler(eventService, logger)

	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name: "valid batch",
			requestBody: dto.BatchIngestRequest{
				Events: []dto.EventIngestRequest{
					{
						UserName: "user1",
						Category: "category1",
						Action:   "action1",
					},
					{
						UserName: "user2",
						Category: "category2",
						Action:   "action2",
					},
				},
			},
			expectedStatus: http.StatusMultiStatus,
			checkResponse: func(t *testing.T, body []byte) {
				var resp dto.BatchIngestResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.SuccessCount != 2 {
					t.Errorf("expected 2 successes, got %d", resp.SuccessCount)
				}
				if resp.FailureCount != 0 {
					t.Errorf("expected 0 failures, got %d", resp.FailureCount)
				}
			},
		},
		{
			name: "partial success",
			requestBody: dto.BatchIngestRequest{
				Events: []dto.EventIngestRequest{
					{
						UserName: "user1",
						Category: "category1",
						Action:   "action1",
					},
					{
						UserName: "user2",
						Category: "category2",
						// Missing Action - validation error
					},
				},
			},
			expectedStatus: http.StatusMultiStatus,
			checkResponse: func(t *testing.T, body []byte) {
				var resp dto.BatchIngestResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.SuccessCount != 1 {
					t.Errorf("expected 1 success, got %d", resp.SuccessCount)
				}
				if resp.FailureCount != 1 {
					t.Errorf("expected 1 failure, got %d", resp.FailureCount)
				}
				if len(resp.Errors) != 1 {
					t.Errorf("expected 1 error detail, got %d", len(resp.Errors))
				}
			},
		},
		{
			name: "batch size exceeds limit",
			requestBody: func() dto.BatchIngestRequest {
				events := make([]dto.EventIngestRequest, 101)
				for i := range events {
					events[i] = dto.EventIngestRequest{
						UserName: "user",
						Category: "category",
						Action:   "action",
					}
				}
				return dto.BatchIngestRequest{Events: events}
			}(),
			expectedStatus: http.StatusBadRequest,
			checkResponse:  nil,
		},
		{
			name: "empty batch",
			requestBody: dto.BatchIngestRequest{
				Events: []dto.EventIngestRequest{},
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse:  nil,
		},
		{
			name:           "invalid json",
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
			checkResponse:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare request body
			var body []byte
			var err error
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				if err != nil {
					t.Fatalf("failed to marshal request: %v", err)
				}
			}

			// Create HTTP request
			req := httptest.NewRequest(http.MethodPost, "/events/ingest/batch", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Call handler
			handler.IngestEventsBatch(w, req)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check response if needed
			if tt.checkResponse != nil {
				tt.checkResponse(t, w.Body.Bytes())
			}
		})
	}
}

func TestEventHandler_IngestEvent_MethodNotAllowed(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Setup dependencies
	eventRepo := repository.NewEventRepository(db)
	eventService := service.NewEventService(eventRepo, 10000) // maxExportSize = 10000
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	handler := NewEventHandler(eventService, logger)

	// Test GET method (should be POST only)
	req := httptest.NewRequest(http.MethodGet, "/events/ingest", nil)
	w := httptest.NewRecorder()

	handler.IngestEvent(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestEventHandler_IngestEventsBatch_MethodNotAllowed(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Setup dependencies
	eventRepo := repository.NewEventRepository(db)
	eventService := service.NewEventService(eventRepo, 10000) // maxExportSize = 10000
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	handler := NewEventHandler(eventService, logger)

	// Test GET method (should be POST only)
	req := httptest.NewRequest(http.MethodGet, "/events/ingest/batch", nil)
	w := httptest.NewRecorder()

	handler.IngestEventsBatch(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}
