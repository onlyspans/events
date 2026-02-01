package domain

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Event represents the domain event entity stored in the database.
type Event struct {
	ID            uuid.UUID     `json:"id" db:"id"`
	Timestamp     time.Time     `json:"timestamp" db:"timestamp"`
	User          string        `json:"user" db:"user_name"`
	Category      string        `json:"category" db:"category"`
	Action        string        `json:"action" db:"action"`
	DocumentName  string        `json:"documentName" db:"document_name"`
	Project       string        `json:"project" db:"project"`
	Environment   string        `json:"environment" db:"environment"`
	Tenant        string        `json:"tenant" db:"tenant"`
	CorrelationID string        `json:"correlationId" db:"correlation_id"`
	TraceID       string        `json:"traceId" db:"trace_id"`
	Details       *EventDetails `json:"details" db:"details"`
	CreatedAt     time.Time     `json:"createdAt" db:"created_at"`
}

// EventDetails contains additional information about the event stored as JSONB.
type EventDetails struct {
	Changes        []Change `json:"changes,omitempty"`
	IPAddress      string   `json:"ipAddress,omitempty"`
	UserAgent      string   `json:"userAgent,omitempty"`
	AdditionalInfo string   `json:"additionalInfo,omitempty"`
}

// Change represents a field change in the event.
type Change struct {
	Field    string `json:"field,omitempty"`
	OldValue string `json:"oldValue,omitempty"`
	NewValue string `json:"newValue,omitempty"`
}

// Value implements the driver.Valuer interface for JSONB storage.
func (ed EventDetails) Value() (driver.Value, error) {
	return json.Marshal(ed)
}

// Scan implements the sql.Scanner interface for JSONB retrieval.
func (ed *EventDetails) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, ed)
}
