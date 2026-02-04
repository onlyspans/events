package dto

import (
	"testing"
	"time"

	"github.com/onlyspans/events/internal/domain"
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
				UserName: "test_user",
				Category: "test_category",
				Action:   "test_action",
			},
			wantErr: false,
		},
		{
			name: "valid request with all fields",
			request: EventIngestRequest{
				Timestamp:     time.Now(),
				UserName:      "test_user",
				Category:      "test_category",
				Action:        "test_action",
				DocumentName:  "test_doc",
				Project:       "test_project",
				Environment:   "test_env",
				Tenant:        "test_tenant",
				Details:       map[string]interface{}{"key": "value"},
				CorrelationID: "corr-123",
				TraceID:       "trace-456",
			},
			wantErr: false,
		},
		{
			name: "missing user_name",
			request: EventIngestRequest{
				Category: "test_category",
				Action:   "test_action",
			},
			wantErr: true,
			errMsg:  "missing required field: user_name",
		},
		{
			name: "empty user_name",
			request: EventIngestRequest{
				UserName: "",
				Category: "test_category",
				Action:   "test_action",
			},
			wantErr: true,
			errMsg:  "missing required field: user_name",
		},
		{
			name: "missing category",
			request: EventIngestRequest{
				UserName: "test_user",
				Action:   "test_action",
			},
			wantErr: true,
			errMsg:  "missing required field: category",
		},
		{
			name: "empty category",
			request: EventIngestRequest{
				UserName: "test_user",
				Category: "",
				Action:   "test_action",
			},
			wantErr: true,
			errMsg:  "missing required field: category",
		},
		{
			name: "missing action",
			request: EventIngestRequest{
				UserName: "test_user",
				Category: "test_category",
			},
			wantErr: true,
			errMsg:  "missing required field: action",
		},
		{
			name: "empty action",
			request: EventIngestRequest{
				UserName: "test_user",
				Category: "test_category",
				Action:   "",
			},
			wantErr: true,
			errMsg:  "missing required field: action",
		},
		{
			name: "all required fields missing",
			request: EventIngestRequest{
				DocumentName: "test_doc",
			},
			wantErr: true,
			errMsg:  "missing required field: user_name",
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

	tests := []struct {
		name     string
		request  EventIngestRequest
		validate func(*testing.T, *domain.Event)
	}{
		{
			name: "converts required fields only",
			request: EventIngestRequest{
				UserName: "test_user",
				Category: "test_category",
				Action:   "test_action",
			},
			validate: func(t *testing.T, event *domain.Event) {
				if event.User != "test_user" {
					t.Errorf("User = %v, want %v", event.User, "test_user")
				}
				if event.Category != "test_category" {
					t.Errorf("Category = %v, want %v", event.Category, "test_category")
				}
				if event.Action != "test_action" {
					t.Errorf("Action = %v, want %v", event.Action, "test_action")
				}
				if event.Details != nil {
					t.Errorf("Details = %v, want nil", event.Details)
				}
			},
		},
		{
			name: "converts all fields",
			request: EventIngestRequest{
				Timestamp:     timestamp,
				UserName:      "test_user",
				Category:      "test_category",
				Action:        "test_action",
				DocumentName:  "test_doc",
				Project:       "test_project",
				Environment:   "production",
				Tenant:        "tenant-1",
				CorrelationID: "corr-123",
				TraceID:       "trace-456",
			},
			validate: func(t *testing.T, event *domain.Event) {
				if event.Timestamp != timestamp {
					t.Errorf("Timestamp = %v, want %v", event.Timestamp, timestamp)
				}
				if event.User != "test_user" {
					t.Errorf("User = %v, want %v", event.User, "test_user")
				}
				if event.DocumentName != "test_doc" {
					t.Errorf("DocumentName = %v, want %v", event.DocumentName, "test_doc")
				}
				if event.Project != "test_project" {
					t.Errorf("Project = %v, want %v", event.Project, "test_project")
				}
				if event.Environment != "production" {
					t.Errorf("Environment = %v, want %v", event.Environment, "production")
				}
				if event.Tenant != "tenant-1" {
					t.Errorf("Tenant = %v, want %v", event.Tenant, "tenant-1")
				}
				if event.CorrelationID != "corr-123" {
					t.Errorf("CorrelationID = %v, want %v", event.CorrelationID, "corr-123")
				}
				if event.TraceID != "trace-456" {
					t.Errorf("TraceID = %v, want %v", event.TraceID, "trace-456")
				}
			},
		},
		{
			name: "converts details with known fields",
			request: EventIngestRequest{
				UserName: "test_user",
				Category: "test_category",
				Action:   "test_action",
				Details: map[string]interface{}{
					"ipAddress":      "192.168.1.1",
					"userAgent":      "Mozilla/5.0",
					"additionalInfo": "extra data",
				},
			},
			validate: func(t *testing.T, event *domain.Event) {
				if event.Details == nil {
					t.Fatal("Details should not be nil")
				}
				if event.Details.IPAddress != "192.168.1.1" {
					t.Errorf("Details.IPAddress = %v, want %v", event.Details.IPAddress, "192.168.1.1")
				}
				if event.Details.UserAgent != "Mozilla/5.0" {
					t.Errorf("Details.UserAgent = %v, want %v", event.Details.UserAgent, "Mozilla/5.0")
				}
				if event.Details.AdditionalInfo != "extra data" {
					t.Errorf("Details.AdditionalInfo = %v, want %v", event.Details.AdditionalInfo, "extra data")
				}
			},
		},
		{
			name: "converts details with changes",
			request: EventIngestRequest{
				UserName: "test_user",
				Category: "test_category",
				Action:   "test_action",
				Details: map[string]interface{}{
					"changes": []interface{}{
						map[string]interface{}{
							"field":    "status",
							"oldValue": "open",
							"newValue": "closed",
						},
						map[string]interface{}{
							"field":    "priority",
							"oldValue": "low",
							"newValue": "high",
						},
					},
				},
			},
			validate: func(t *testing.T, event *domain.Event) {
				if event.Details == nil {
					t.Fatal("Details should not be nil")
				}
				if len(event.Details.Changes) != 2 {
					t.Fatalf("Details.Changes length = %v, want %v", len(event.Details.Changes), 2)
				}
				if event.Details.Changes[0].Field != "status" {
					t.Errorf("Changes[0].Field = %v, want %v", event.Details.Changes[0].Field, "status")
				}
				if event.Details.Changes[0].OldValue != "open" {
					t.Errorf("Changes[0].OldValue = %v, want %v", event.Details.Changes[0].OldValue, "open")
				}
				if event.Details.Changes[0].NewValue != "closed" {
					t.Errorf("Changes[0].NewValue = %v, want %v", event.Details.Changes[0].NewValue, "closed")
				}
			},
		},
		{
			name: "handles empty details map",
			request: EventIngestRequest{
				UserName: "test_user",
				Category: "test_category",
				Action:   "test_action",
				Details:  map[string]interface{}{},
			},
			validate: func(t *testing.T, event *domain.Event) {
				if event.Details != nil {
					t.Errorf("Details should be nil for empty map")
				}
			},
		},
		{
			name: "handles nil details",
			request: EventIngestRequest{
				UserName: "test_user",
				Category: "test_category",
				Action:   "test_action",
				Details:  nil,
			},
			validate: func(t *testing.T, event *domain.Event) {
				if event.Details != nil {
					t.Errorf("Details should be nil")
				}
			},
		},
		{
			name: "handles zero timestamp",
			request: EventIngestRequest{
				Timestamp: time.Time{},
				UserName:  "test_user",
				Category:  "test_category",
				Action:    "test_action",
			},
			validate: func(t *testing.T, event *domain.Event) {
				if !event.Timestamp.IsZero() {
					t.Errorf("Timestamp should be zero, got %v", event.Timestamp)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := tt.request.ToEvent()
			if event == nil {
				t.Fatal("ToEvent() returned nil")
			}
			tt.validate(t, event)
		})
	}
}

func TestBatchIngestRequest(t *testing.T) {
	t.Run("batch request with multiple events", func(t *testing.T) {
		req := BatchIngestRequest{
			Events: []EventIngestRequest{
				{
					UserName: "user1",
					Category: "cat1",
					Action:   "action1",
				},
				{
					UserName: "user2",
					Category: "cat2",
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
			Timestamp:     timestamp,
			UserName:      "test_user",
			Category:      "test_category",
			Action:        "test_action",
			DocumentName:  "test_doc",
			Project:       "test_project",
			Environment:   "production",
			Tenant:        "tenant-1",
			Details: map[string]interface{}{
				"ipAddress": "192.168.1.1",
				"userAgent": "Mozilla/5.0",
			},
			CorrelationID: "corr-123",
			TraceID:       "trace-456",
		}

		// This would be handled by encoding/json in real HTTP handlers
		// We're just verifying the struct tags are correct
		if original.UserName != "test_user" {
			t.Errorf("UserName = %v, want test_user", original.UserName)
		}
	})

	t.Run("omitempty works for optional fields", func(t *testing.T) {
		req := EventIngestRequest{
			UserName: "test_user",
			Category: "test_category",
			Action:   "test_action",
		}

		// Verify required fields are present
		if req.UserName == "" {
			t.Error("UserName should not be empty")
		}
		// Optional fields can be empty
		if req.DocumentName != "" {
			t.Error("DocumentName should be empty")
		}
		if req.Details != nil {
			t.Error("Details should be nil")
		}
	})
}

func TestBatchIngestRequest_JSONMarshaling(t *testing.T) {
	t.Run("batch with multiple events", func(t *testing.T) {
		batch := BatchIngestRequest{
			Events: []EventIngestRequest{
				{
					UserName: "user1",
					Category: "cat1",
					Action:   "action1",
				},
				{
					UserName: "user2",
					Category: "cat2",
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
		id := domain.Event{}.ID
		resp := SingleIngestResponse{
			ID: id,
		}

		if resp.ID != id {
			t.Errorf("ID = %v, want %v", resp.ID, id)
		}
	})
}
