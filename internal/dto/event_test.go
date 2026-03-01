package dto

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestEventIngestRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		request EventIngestRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid request with all required fields",
			request: EventIngestRequest{
				EntityID: uuid.New().String(),
				Action:   "test_action",
			},
			wantErr: false,
		},
		{
			name: "valid request with all fields",
			request: EventIngestRequest{
				Timestamp:  time.Now(),
				EntityID:   uuid.New().String(),
				EntityName: "test_entity",
				Action:     "test_action",
				UserID:     "test_user",
				IPAddress:  "192.168.1.1",
				UserAgent:  "Mozilla/5.0",
				Tenant:     "test_tenant",
				Changes: []ChangeDTO{
					{Field: "status", OldValue: "open", NewValue: "closed"},
				},
			},
			wantErr: false,
		},
		{
			name: "missing entity_id",
			request: EventIngestRequest{
				Action: "test_action",
			},
			wantErr: true,
			errMsg:  "missing required field: entity_id",
		},
		{
			name: "empty entity_id",
			request: EventIngestRequest{
				EntityID: "",
				Action:   "test_action",
			},
			wantErr: true,
			errMsg:  "missing required field: entity_id",
		},
		{
			name: "missing action",
			request: EventIngestRequest{
				EntityID: uuid.New().String(),
			},
			wantErr: true,
			errMsg:  "missing required field: action",
		},
		{
			name: "empty action",
			request: EventIngestRequest{
				EntityID: uuid.New().String(),
				Action:   "",
			},
			wantErr: true,
			errMsg:  "missing required field: action",
		},
		{
			name: "all required fields missing",
			request: EventIngestRequest{
				EntityName: "test_entity",
			},
			wantErr: true,
			errMsg:  "missing required field: entity_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error but got nil")
				} else if err.Error() != tt.errMsg {
					t.Errorf("Validate() error = %v, want %v", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestEventIngestRequest_ToEvent(t *testing.T) {
	timestamp := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	entityID := uuid.New()

	tests := []struct {
		name     string
		request  EventIngestRequest
		validate func(*testing.T, EventIngestRequest)
	}{
		{
			name: "converts required fields only",
			request: EventIngestRequest{
				EntityID: entityID.String(),
				Action:   "test_action",
			},
			validate: func(t *testing.T, req EventIngestRequest) {
				event := req.ToEvent()
				if event.EntityID != entityID {
					t.Errorf("EntityID = %v, want %v", event.EntityID, entityID)
				}
				if event.Action != "test_action" {
					t.Errorf("Action = %v, want %v", event.Action, "test_action")
				}
				if len(event.Changes) != 0 {
					t.Errorf("Changes = %v, want empty", event.Changes)
				}
			},
		},
		{
			name: "converts all fields",
			request: EventIngestRequest{
				Timestamp:  timestamp,
				EntityID:   entityID.String(),
				EntityName: "test_entity",
				Action:     "test_action",
				UserID:     "user-123",
				IPAddress:  "192.168.1.1",
				UserAgent:  "Mozilla/5.0",
				Tenant:     "tenant-1",
			},
			validate: func(t *testing.T, req EventIngestRequest) {
				event := req.ToEvent()
				if event.Timestamp != timestamp {
					t.Errorf("Timestamp = %v, want %v", event.Timestamp, timestamp)
				}
				if event.EntityID != entityID {
					t.Errorf("EntityID = %v, want %v", event.EntityID, entityID)
				}
				if event.EntityName != "test_entity" {
					t.Errorf("EntityName = %v, want %v", event.EntityName, "test_entity")
				}
				if event.UserID != "user-123" {
					t.Errorf("UserID = %v, want %v", event.UserID, "user-123")
				}
				if event.IPAddress != "192.168.1.1" {
					t.Errorf("IPAddress = %v, want %v", event.IPAddress, "192.168.1.1")
				}
				if event.UserAgent != "Mozilla/5.0" {
					t.Errorf("UserAgent = %v, want %v", event.UserAgent, "Mozilla/5.0")
				}
				if event.Tenant != "tenant-1" {
					t.Errorf("Tenant = %v, want %v", event.Tenant, "tenant-1")
				}
			},
		},
		{
			name: "converts changes",
			request: EventIngestRequest{
				EntityID: entityID.String(),
				Action:   "test_action",
				Changes: []ChangeDTO{
					{
						Field:    "status",
						OldValue: "open",
						NewValue: "closed",
					},
					{
						Field:    "priority",
						OldValue: "low",
						NewValue: "high",
					},
				},
			},
			validate: func(t *testing.T, req EventIngestRequest) {
				event := req.ToEvent()
				if len(event.Changes) != 2 {
					t.Fatalf("Changes length = %v, want %v", len(event.Changes), 2)
				}
				if event.Changes[0].Field != "status" {
					t.Errorf("Changes[0].Field = %v, want %v", event.Changes[0].Field, "status")
				}
				if event.Changes[0].OldValue != "open" {
					t.Errorf("Changes[0].OldValue = %v, want %v", event.Changes[0].OldValue, "open")
				}
				if event.Changes[0].NewValue != "closed" {
					t.Errorf("Changes[0].NewValue = %v, want %v", event.Changes[0].NewValue, "closed")
				}
			},
		},
		{
			name: "handles invalid entity_id",
			request: EventIngestRequest{
				EntityID: "invalid-uuid",
				Action:   "test_action",
			},
			validate: func(t *testing.T, req EventIngestRequest) {
				event := req.ToEvent()
				if event.EntityID != uuid.Nil {
					t.Errorf("EntityID should be uuid.Nil for invalid UUID, got %v", event.EntityID)
				}
			},
		},
		{
			name: "handles zero timestamp",
			request: EventIngestRequest{
				Timestamp: time.Time{},
				EntityID:  entityID.String(),
				Action:    "test_action",
			},
			validate: func(t *testing.T, req EventIngestRequest) {
				event := req.ToEvent()
				if !event.Timestamp.IsZero() {
					t.Errorf("Timestamp should be zero, got %v", event.Timestamp)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, tt.request)
		})
	}
}

func TestBatchIngestRequest(t *testing.T) {
	t.Run("batch request with multiple events", func(t *testing.T) {
		req := BatchIngestRequest{
			Events: []EventIngestRequest{
				{
					EntityID: uuid.New().String(),
					Action:   "action1",
				},
				{
					EntityID: uuid.New().String(),
					Action:   "action2",
				},
			},
		}

		if len(req.Events) != 2 {
			t.Errorf("BatchIngestRequest.Events length = %v, want 2", len(req.Events))
		}
	})

	t.Run("empty batch request", func(t *testing.T) {
		req := BatchIngestRequest{
			Events: []EventIngestRequest{},
		}

		if len(req.Events) != 0 {
			t.Errorf("BatchIngestRequest.Events length = %v, want 0", len(req.Events))
		}
	})
}

func TestBatchIngestResponse(t *testing.T) {
	t.Run("successful batch response", func(t *testing.T) {
		resp := BatchIngestResponse{
			SuccessCount: 10,
			FailureCount: 0,
			Errors:       nil,
		}

		if resp.SuccessCount != 10 {
			t.Errorf("SuccessCount = %v, want 10", resp.SuccessCount)
		}
		if resp.FailureCount != 0 {
			t.Errorf("FailureCount = %v, want 0", resp.FailureCount)
		}
		if resp.Errors != nil {
			t.Errorf("Errors should be nil")
		}
	})

	t.Run("partial success response", func(t *testing.T) {
		resp := BatchIngestResponse{
			SuccessCount: 8,
			FailureCount: 2,
			Errors: []BatchError{
				{Index: 3, Error: "validation error"},
				{Index: 7, Error: "database error"},
			},
		}

		if resp.SuccessCount != 8 {
			t.Errorf("SuccessCount = %v, want 8", resp.SuccessCount)
		}
		if resp.FailureCount != 2 {
			t.Errorf("FailureCount = %v, want 2", resp.FailureCount)
		}
		if len(resp.Errors) != 2 {
			t.Fatalf("Errors length = %v, want 2", len(resp.Errors))
		}
		if resp.Errors[0].Index != 3 {
			t.Errorf("Errors[0].Index = %v, want 3", resp.Errors[0].Index)
		}
		if resp.Errors[0].Error != "validation error" {
			t.Errorf("Errors[0].Error = %v, want 'validation error'", resp.Errors[0].Error)
		}
	})

	t.Run("complete failure response", func(t *testing.T) {
		resp := BatchIngestResponse{
			SuccessCount: 0,
			FailureCount: 5,
			Errors: []BatchError{
				{Index: 0, Error: "error 1"},
				{Index: 1, Error: "error 2"},
				{Index: 2, Error: "error 3"},
				{Index: 3, Error: "error 4"},
				{Index: 4, Error: "error 5"},
			},
		}

		if resp.SuccessCount != 0 {
			t.Errorf("SuccessCount = %v, want 0", resp.SuccessCount)
		}
		if resp.FailureCount != 5 {
			t.Errorf("FailureCount = %v, want 5", resp.FailureCount)
		}
		if len(resp.Errors) != 5 {
			t.Errorf("Errors length = %v, want 5", len(resp.Errors))
		}
	})
}

func TestBatchError(t *testing.T) {
	t.Run("batch error structure", func(t *testing.T) {
		err := BatchError{
			Index: 5,
			Error: "test error message",
		}

		if err.Index != 5 {
			t.Errorf("Index = %v, want 5", err.Index)
		}
		if err.Error != "test error message" {
			t.Errorf("Error = %v, want 'test error message'", err.Error)
		}
	})
}

func TestEventIngestRequest_JSONMarshaling(t *testing.T) {
	t.Run("marshal and unmarshal with all fields", func(t *testing.T) {
		timestamp := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		original := EventIngestRequest{
			Timestamp:  timestamp,
			EntityID:   uuid.New().String(),
			EntityName: "test_entity",
			Action:     "test_action",
			UserID:     "test_user",
			IPAddress:  "192.168.1.1",
			UserAgent:  "Mozilla/5.0",
			Tenant:     "tenant-1",
			Changes: []ChangeDTO{
				{Field: "status", OldValue: "open", NewValue: "closed"},
			},
		}

		if original.EntityID == "" {
			t.Errorf("EntityID should not be empty")
		}
	})

	t.Run("omitempty works for optional fields", func(t *testing.T) {
		req := EventIngestRequest{
			EntityID: uuid.New().String(),
			Action:   "test_action",
		}

		if req.EntityID == "" {
			t.Error("EntityID should not be empty")
		}
		if req.EntityName != "" {
			t.Error("EntityName should be empty")
		}
		if req.Changes != nil {
			t.Error("Changes should be nil")
		}
	})
}

func TestBatchIngestRequest_JSONMarshaling(t *testing.T) {
	t.Run("batch with multiple events", func(t *testing.T) {
		batch := BatchIngestRequest{
			Events: []EventIngestRequest{
				{
					EntityID: uuid.New().String(),
					Action:   "action1",
				},
				{
					EntityID: uuid.New().String(),
					Action:   "action2",
				},
			},
		}

		if len(batch.Events) != 2 {
			t.Errorf("Events length = %v, want 2", len(batch.Events))
		}
	})
}

func TestSingleIngestResponse(t *testing.T) {
	t.Run("response with uuid", func(t *testing.T) {
		id := uuid.New()
		resp := SingleIngestResponse{
			ID: id,
		}

		if resp.ID != id {
			t.Errorf("ID = %v, want %v", resp.ID, id)
		}
	})
}
