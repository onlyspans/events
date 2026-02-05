package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/onlyspans/events/internal/domain"
	"github.com/onlyspans/events/internal/dto"
	"github.com/onlyspans/events/internal/ports"
)

// mockEventRepository is a mock implementation for testing.
type mockEventRepository struct {
	events       []*domain.Event
	saveError    error
	searchError  error
	searchResult []*domain.Event
	searchTotal  int64
}

func (m *mockEventRepository) Create(ctx context.Context, event *domain.Event) (uuid.UUID, error) {
	if m.saveError != nil {
		return uuid.Nil, m.saveError
	}
	m.events = append(m.events, event)
	return event.ID, nil
}

func (m *mockEventRepository) SaveBatch(ctx context.Context, events []*domain.Event) error {
	if m.saveError != nil {
		return m.saveError
	}
	m.events = append(m.events, events...)
	return nil
}

func (m *mockEventRepository) Search(ctx context.Context, query ports.EventSearchQuery) ([]*domain.Event, int64, error) {
	if m.searchError != nil {
		return nil, 0, m.searchError
	}
	return m.searchResult, m.searchTotal, nil
}

func (m *mockEventRepository) DeleteOlderThan(ctx context.Context, cutoffDate time.Time) (int64, error) {
	return 0, nil
}

func TestIngestEvents(t *testing.T) {
	tests := []struct {
		name    string
		events  []*dto.EventDTO
		wantErr bool
	}{
		{
			name: "successful ingestion",
			events: []*dto.EventDTO{
				{
					ID:        uuid.New().String(),
					Timestamp: time.Now(),
					User:      "test-user",
					Category:  "test-category",
					Action:    "test-action",
				},
			},
			wantErr: false,
		},
		{
			name:    "empty batch",
			events:  []*dto.EventDTO{},
			wantErr: false,
		},
		{
			name: "event with details",
			events: []*dto.EventDTO{
				{
					Timestamp: time.Now(),
					User:      "test-user",
					Category:  "test-category",
					Action:    "test-action",
					Details: &dto.EventDetailsDTO{
						IPAddress: "192.168.1.1",
						UserAgent: "Test Agent",
						Changes: []dto.ChangeDTO{
							{Field: "status", OldValue: "active", NewValue: "inactive"},
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockEventRepository{}
			svc := NewEventService(repo, 10000)

			err := svc.IngestEvents(context.Background(), tt.events)
			if (err != nil) != tt.wantErr {
				t.Errorf("IngestEvents() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && len(tt.events) > 0 {
				if len(repo.events) != len(tt.events) {
					t.Errorf("Expected %d events, got %d", len(tt.events), len(repo.events))
				}
			}
		})
	}
}

func TestDTOConversion(t *testing.T) {
	repo := &mockEventRepository{}
	svc := NewEventService(repo, 10000)

	originalDTO := &dto.EventDTO{
		ID:            uuid.New().String(),
		Timestamp:     time.Now(),
		User:          "test-user",
		Category:      "test-category",
		Action:        "test-action",
		DocumentName:  "test-doc",
		Project:       "test-project",
		Environment:   "test-env",
		Tenant:        "test-tenant",
		CorrelationID: "corr-123",
		TraceID:       "trace-456",
		Details: &dto.EventDetailsDTO{
			IPAddress:      "192.168.1.1",
			UserAgent:      "Test Agent",
			AdditionalInfo: "Additional info",
			Changes: []dto.ChangeDTO{
				{Field: "status", OldValue: "active", NewValue: "inactive"},
			},
		},
	}

	// Convert DTO to Entity
	entity, err := svc.dtoToEntity(originalDTO)
	if err != nil {
		t.Fatalf("dtoToEntity failed: %v", err)
	}

	// Verify entity fields
	if entity.User != originalDTO.User {
		t.Errorf("User mismatch: got %s, want %s", entity.User, originalDTO.User)
	}
	if entity.Category != originalDTO.Category {
		t.Errorf("Category mismatch: got %s, want %s", entity.Category, originalDTO.Category)
	}

	// Convert Entity back to DTO
	convertedDTO := svc.entityToDTO(entity)

	// Verify round-trip conversion
	if convertedDTO.User != originalDTO.User {
		t.Errorf("User mismatch after round-trip: got %s, want %s", convertedDTO.User, originalDTO.User)
	}
	if convertedDTO.Details == nil {
		t.Error("Details lost after round-trip conversion")
	} else {
		if convertedDTO.Details.IPAddress != originalDTO.Details.IPAddress {
			t.Errorf("IPAddress mismatch: got %s, want %s",
				convertedDTO.Details.IPAddress, originalDTO.Details.IPAddress)
		}
		if len(convertedDTO.Details.Changes) != len(originalDTO.Details.Changes) {
			t.Errorf("Changes count mismatch: got %d, want %d",
				len(convertedDTO.Details.Changes), len(originalDTO.Details.Changes))
		}
	}
}

func TestSearchEvents(t *testing.T) {
	now := time.Now()
	testEvent := &domain.Event{
		ID:        uuid.New(),
		Timestamp: now,
		User:      "test-user",
		Category:  "test-category",
		Action:    "test-action",
	}

	repo := &mockEventRepository{
		searchResult: []*domain.Event{testEvent},
		searchTotal:  1,
	}
	svc := NewEventService(repo, 10000)

	req := dto.SearchEventsRequest{
		EventFilterRequest: dto.EventFilterRequest{
			User:      "test-user",
			SortBy:    "timestamp",
			SortOrder: "desc",
		},
		Page: 0,
		Size: 20,
	}

	result, err := svc.SearchEvents(context.Background(), req)
	if err != nil {
		t.Fatalf("SearchEvents failed: %v", err)
	}

	if result.Total != 1 {
		t.Errorf("Expected total 1, got %d", result.Total)
	}
	if len(result.Events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(result.Events))
	}
	if result.Events[0].User != "test-user" {
		t.Errorf("Expected user 'test-user', got %s", result.Events[0].User)
	}
}

func TestCreateEvent(t *testing.T) {
	tests := []struct {
		name      string
		request   dto.EventIngestRequest
		saveError error
		wantErr   bool
		errMsg    string
	}{
		{
			name: "successful creation with all fields",
			request: dto.EventIngestRequest{
				User:          "test-user",
				Category:      "test-category",
				Action:        "test-action",
				DocumentName:  "test-doc.txt",
				Project:       "test-project",
				Environment:   "production",
				Tenant:        "tenant-1",
				CorrelationID: "corr-123",
				TraceID:       "trace-456",
				Details: map[string]interface{}{
					"ipAddress": "192.168.1.1",
					"userAgent": "Mozilla/5.0",
				},
			},
			wantErr: false,
		},
		{
			name: "successful creation with only required fields",
			request: dto.EventIngestRequest{
				User:     "test-user",
				Category: "test-category",
				Action:   "test-action",
			},
			wantErr: false,
		},
		{
			name: "successful creation with timestamp defaulting",
			request: dto.EventIngestRequest{
				User:      "test-user",
				Category:  "test-category",
				Action:    "test-action",
				Timestamp: time.Time{}, // Zero time, should be set to now
			},
			wantErr: false,
		},
		{
			name: "missing required field: user",
			request: dto.EventIngestRequest{
				Category: "test-category",
				Action:   "test-action",
			},
			wantErr: true,
			errMsg:  "validation failed",
		},
		{
			name: "missing required field: category",
			request: dto.EventIngestRequest{
				User:   "test-user",
				Action: "test-action",
			},
			wantErr: true,
			errMsg:  "validation failed",
		},
		{
			name: "missing required field: action",
			request: dto.EventIngestRequest{
				User:     "test-user",
				Category: "test-category",
			},
			wantErr: true,
			errMsg:  "validation failed",
		},
		{
			name: "repository save error",
			request: dto.EventIngestRequest{
				User:     "test-user",
				Category: "test-category",
				Action:   "test-action",
			},
			saveError: context.DeadlineExceeded,
			wantErr:   true,
			errMsg:    "failed to save event",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockEventRepository{
				saveError: tt.saveError,
			}
			svc := NewEventService(repo, 10000)

			eventID, err := svc.CreateEvent(context.Background(), tt.request)

			if tt.wantErr {
				if err == nil {
					t.Errorf("CreateEvent() expected error, got nil")
				} else if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("CreateEvent() error = %v, should contain %q", err, tt.errMsg)
				}
				if eventID != uuid.Nil {
					t.Errorf("CreateEvent() expected nil UUID on error, got %v", eventID)
				}
			} else {
				if err != nil {
					t.Errorf("CreateEvent() unexpected error: %v", err)
				}
				if eventID == uuid.Nil {
					t.Errorf("CreateEvent() returned nil UUID on success")
				}
				if len(repo.events) != 1 {
					t.Errorf("CreateEvent() expected 1 event saved, got %d", len(repo.events))
				} else {
					event := repo.events[0]
					if event.User != tt.request.User {
						t.Errorf("CreateEvent() user = %v, want %v", event.User, tt.request.User)
					}
					if event.Category != tt.request.Category {
						t.Errorf("CreateEvent() category = %v, want %v", event.Category, tt.request.Category)
					}
					if event.Action != tt.request.Action {
						t.Errorf("CreateEvent() action = %v, want %v", event.Action, tt.request.Action)
					}
					if event.Timestamp.IsZero() {
						t.Errorf("CreateEvent() timestamp should not be zero")
					}
					if event.ID == uuid.Nil {
						t.Errorf("CreateEvent() ID should not be nil")
					}
				}
			}
		})
	}
}

func TestCreateEventsBatch(t *testing.T) {
	tests := []struct {
		name             string
		requests         []dto.EventIngestRequest
		saveError        error
		expectedSuccess  int
		expectedFailure  int
		expectedErrors   int
		checkErrorIndices []int
	}{
		{
			name: "all events successful",
			requests: []dto.EventIngestRequest{
				{User: "user1", Category: "cat1", Action: "action1"},
				{User: "user2", Category: "cat2", Action: "action2"},
				{User: "user3", Category: "cat3", Action: "action3"},
			},
			expectedSuccess: 3,
			expectedFailure: 0,
			expectedErrors:  0,
		},
		{
			name: "partial success - validation errors",
			requests: []dto.EventIngestRequest{
				{User: "user1", Category: "cat1", Action: "action1"}, // Valid
				{User: "", Category: "cat2", Action: "action2"},      // Missing user
				{User: "user3", Category: "", Action: "action3"},     // Missing category
				{User: "user4", Category: "cat4", Action: ""},        // Missing action
				{User: "user5", Category: "cat5", Action: "action5"}, // Valid
			},
			expectedSuccess:   2,
			expectedFailure:   3,
			expectedErrors:    3,
			checkErrorIndices: []int{1, 2, 3},
		},
		{
			name: "all events with validation errors",
			requests: []dto.EventIngestRequest{
				{User: "", Category: "cat1", Action: "action1"},
				{User: "user2", Category: "", Action: "action2"},
				{User: "user3", Category: "cat3", Action: ""},
			},
			expectedSuccess:   0,
			expectedFailure:   3,
			expectedErrors:    3,
			checkErrorIndices: []int{0, 1, 2},
		},
		{
			name:            "empty batch",
			requests:        []dto.EventIngestRequest{},
			expectedSuccess: 0,
			expectedFailure: 0,
			expectedErrors:  0,
		},
		{
			name: "single valid event",
			requests: []dto.EventIngestRequest{
				{User: "user1", Category: "cat1", Action: "action1"},
			},
			expectedSuccess: 1,
			expectedFailure: 0,
			expectedErrors:  0,
		},
		{
			name: "events with optional fields",
			requests: []dto.EventIngestRequest{
				{
					User:          "user1",
					Category:      "cat1",
					Action:        "action1",
					DocumentName:  "doc1.txt",
					Project:       "proj1",
					Environment:   "prod",
					Tenant:        "tenant1",
					CorrelationID: "corr1",
					TraceID:       "trace1",
					Details: map[string]interface{}{
						"ipAddress": "192.168.1.1",
					},
				},
			},
			expectedSuccess: 1,
			expectedFailure: 0,
			expectedErrors:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockEventRepository{
				saveError: tt.saveError,
			}
			svc := NewEventService(repo, 10000)

			response := svc.CreateEventsBatch(context.Background(), tt.requests)

			if response.SuccessCount != tt.expectedSuccess {
				t.Errorf("CreateEventsBatch() success count = %v, want %v",
					response.SuccessCount, tt.expectedSuccess)
			}
			if response.FailureCount != tt.expectedFailure {
				t.Errorf("CreateEventsBatch() failure count = %v, want %v",
					response.FailureCount, tt.expectedFailure)
			}
			if len(response.Errors) != tt.expectedErrors {
				t.Errorf("CreateEventsBatch() error count = %v, want %v",
					len(response.Errors), tt.expectedErrors)
			}

			// Check that error indices are correct
			if len(tt.checkErrorIndices) > 0 {
				errorIndices := make(map[int]bool)
				for _, err := range response.Errors {
					errorIndices[err.Index] = true
				}
				for _, expectedIndex := range tt.checkErrorIndices {
					if !errorIndices[expectedIndex] {
						t.Errorf("CreateEventsBatch() expected error at index %d, not found",
							expectedIndex)
					}
				}
			}

			// Verify repository received successful events
			if len(repo.events) != tt.expectedSuccess {
				t.Errorf("CreateEventsBatch() saved %d events, want %d",
					len(repo.events), tt.expectedSuccess)
			}
		})
	}
}

func TestCreateEventsBatch_RepositoryError(t *testing.T) {
	// Test case where repository always fails to save events
	failingRepo := &mockEventRepository{
		saveError: context.DeadlineExceeded,
	}

	svc := NewEventService(failingRepo, 10000)

	requests := []dto.EventIngestRequest{
		{User: "user1", Category: "cat1", Action: "action1"},
		{User: "user2", Category: "cat2", Action: "action2"},
		{User: "user3", Category: "cat3", Action: "action3"},
	}

	response := svc.CreateEventsBatch(context.Background(), requests)

	// All should fail because repository always fails
	if response.SuccessCount != 0 {
		t.Errorf("Expected 0 successful events, got %d", response.SuccessCount)
	}
	if response.FailureCount != 3 {
		t.Errorf("Expected 3 failed events, got %d", response.FailureCount)
	}
	if len(response.Errors) != 3 {
		t.Errorf("Expected 3 errors, got %d", len(response.Errors))
	}

	// Verify all error messages contain "failed to save event"
	for i, err := range response.Errors {
		if !contains(err.Error, "failed to save event") {
			t.Errorf("Error at index %d should contain 'failed to save event', got: %s",
				i, err.Error)
		}
	}
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
