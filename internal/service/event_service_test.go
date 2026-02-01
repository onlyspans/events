package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/onlyspans/events/internal/domain"
	"github.com/onlyspans/events/internal/dto"
	"github.com/onlyspans/events/internal/repository"
)

// mockEventRepository is a mock implementation for testing.
type mockEventRepository struct {
	events       []*domain.Event
	saveError    error
	searchError  error
	searchResult []*domain.Event
	searchTotal  int64
}

func (m *mockEventRepository) SaveBatch(ctx context.Context, events []*domain.Event) error {
	if m.saveError != nil {
		return m.saveError
	}
	m.events = append(m.events, events...)
	return nil
}

func (m *mockEventRepository) Search(ctx context.Context, query repository.SearchQuery) ([]*domain.Event, int64, error) {
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
		User:      "test-user",
		Page:      0,
		Size:      20,
		SortBy:    "timestamp",
		SortOrder: "desc",
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
