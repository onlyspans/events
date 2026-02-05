package dto

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/onlyspans/events/internal/domain"
)

// EventDTO represents the data transfer object for events.
type EventDTO struct {
	ID            string           `json:"id,omitempty"`
	Timestamp     time.Time        `json:"timestamp"`
	User          string           `json:"user"`
	Category      string           `json:"category"`
	Action        string           `json:"action"`
	DocumentName  string           `json:"documentName,omitempty"`
	Project       string           `json:"project,omitempty"`
	Environment   string           `json:"environment,omitempty"`
	Tenant        string           `json:"tenant,omitempty"`
	CorrelationID string           `json:"correlationId,omitempty"`
	TraceID       string           `json:"traceId,omitempty"`
	Details       *EventDetailsDTO `json:"details,omitempty"`
}

// EventDetailsDTO contains additional information about the event.
type EventDetailsDTO struct {
	Changes        []ChangeDTO `json:"changes,omitempty"`
	IPAddress      string      `json:"ipAddress,omitempty"`
	UserAgent      string      `json:"userAgent,omitempty"`
	AdditionalInfo string      `json:"additionalInfo,omitempty"`
}

// ChangeDTO represents a field change.
type ChangeDTO struct {
	Field    string `json:"field,omitempty"`
	OldValue string `json:"oldValue,omitempty"`
	NewValue string `json:"newValue,omitempty"`
}

// EventFilterRequest contains common filter fields for event queries.
// It is embedded by SearchEventsRequest and aliased by ExportEventsRequest.
type EventFilterRequest struct {
	User          string     `json:"user,omitempty"`
	Category      string     `json:"category,omitempty"`
	Action        string     `json:"action,omitempty"`
	Document      string     `json:"document,omitempty"`
	Project       string     `json:"project,omitempty"`
	Environment   string     `json:"environment,omitempty"`
	Tenant        string     `json:"tenant,omitempty"`
	CorrelationID string     `json:"correlationId,omitempty"`
	TraceID       string     `json:"traceId,omitempty"`
	StartDate     *time.Time `json:"startDate,omitempty"`
	EndDate       *time.Time `json:"endDate,omitempty"`
	SortBy        string     `json:"sortBy,omitempty"`
	SortOrder     string     `json:"sortOrder,omitempty"`
}

// SearchEventsRequest represents the search request parameters.
// It embeds EventFilterRequest and adds pagination fields.
type SearchEventsRequest struct {
	EventFilterRequest
	Page int `json:"page,omitempty"`
	Size int `json:"size,omitempty"`
}

// ExportEventsRequest represents the export request parameters.
// It is an alias for EventFilterRequest since exports use the same filters
// without pagination (the export size is controlled by service configuration).
type ExportEventsRequest = EventFilterRequest

// QueryResult represents the search results with pagination.
type QueryResult struct {
	Events     []EventDTO `json:"events"`
	Total      int64      `json:"total"`
	Page       int        `json:"page"`
	Size       int        `json:"size"`
	TotalPages int        `json:"totalPages"`
}

// EventIngestRequest represents a request to ingest a single event via HTTP.
type EventIngestRequest struct {
	Timestamp     time.Time              `json:"timestamp"`
	User          string                 `json:"user"`
	Category      string                 `json:"category"`
	Action        string                 `json:"action"`
	DocumentName  string                 `json:"document_name,omitempty"`
	Project       string                 `json:"project,omitempty"`
	Environment   string                 `json:"environment,omitempty"`
	Tenant        string                 `json:"tenant,omitempty"`
	Details       map[string]interface{} `json:"details,omitempty"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	TraceID       string                 `json:"trace_id,omitempty"`
}

// Validate checks that all required fields are present and non-empty.
func (e *EventIngestRequest) Validate() error {
	if e.User == "" {
		return fmt.Errorf("missing required field: user")
	}
	if e.Category == "" {
		return fmt.Errorf("missing required field: category")
	}
	if e.Action == "" {
		return fmt.Errorf("missing required field: action")
	}
	return nil
}

// ToEvent converts the EventIngestRequest to a domain.Event.
// The ID and CreatedAt fields will be set by the service layer.
// If Timestamp is zero, it will be set to time.Now() by the service layer.
func (e *EventIngestRequest) ToEvent() *domain.Event {
	var details *domain.EventDetails
	if len(e.Details) > 0 {
		// Convert map[string]interface{} to EventDetails
		// For now, we store the raw details as AdditionalInfo
		// The service layer can further process if needed
		details = &domain.EventDetails{}

		// Try to extract known fields
		if changes, ok := e.Details["changes"].([]interface{}); ok {
			for _, change := range changes {
				if changeMap, ok := change.(map[string]interface{}); ok {
					c := domain.Change{}
					if field, ok := changeMap["field"].(string); ok {
						c.Field = field
					}
					if oldValue, ok := changeMap["oldValue"].(string); ok {
						c.OldValue = oldValue
					}
					if newValue, ok := changeMap["newValue"].(string); ok {
						c.NewValue = newValue
					}
					details.Changes = append(details.Changes, c)
				}
			}
		}
		if ipAddress, ok := e.Details["ipAddress"].(string); ok {
			details.IPAddress = ipAddress
		}
		if userAgent, ok := e.Details["userAgent"].(string); ok {
			details.UserAgent = userAgent
		}
		if additionalInfo, ok := e.Details["additionalInfo"].(string); ok {
			details.AdditionalInfo = additionalInfo
		}
	}

	return &domain.Event{
		Timestamp:     e.Timestamp,
		User:          e.User,
		Category:      e.Category,
		Action:        e.Action,
		DocumentName:  e.DocumentName,
		Project:       e.Project,
		Environment:   e.Environment,
		Tenant:        e.Tenant,
		Details:       details,
		CorrelationID: e.CorrelationID,
		TraceID:       e.TraceID,
	}
}

// BatchIngestRequest represents a request to ingest multiple events via HTTP.
type BatchIngestRequest struct {
	Events []EventIngestRequest `json:"events"`
}

// BatchIngestResponse represents the response for a batch ingestion request.
type BatchIngestResponse struct {
	SuccessCount int          `json:"success_count"`
	FailureCount int          `json:"failure_count"`
	Errors       []BatchError `json:"errors,omitempty"`
}

// BatchError represents an error that occurred during batch ingestion.
type BatchError struct {
	Index int    `json:"index"`
	Error string `json:"error"`
}

// SingleIngestResponse represents the response for a single event ingestion.
type SingleIngestResponse struct {
	ID uuid.UUID `json:"id"`
}
