package domain

import (
	"time"

	"github.com/google/uuid"
)

// Event represents the domain event entity stored in the database.
type Event struct {
	ID         uuid.UUID `db:"id"`
	Timestamp  time.Time `db:"timestamp"`
	EntityID   uuid.UUID `db:"entity_id"`
	EntityName string    `db:"entity_name"`
	Action     string    `db:"action"`
	UserID     string    `db:"user_id"`
	IPAddress  string    `db:"ip_address"`
	UserAgent  string    `db:"user_agent"`
	Tenant     string    `db:"tenant"`
	Changes    []Change  `db:"changes" json:"changes,omitempty"`
}

// Change represents a field change in the event.
type Change struct {
	Field    string `json:"field,omitempty"`
	OldValue string `json:"oldValue,omitempty"`
	NewValue string `json:"newValue,omitempty"`
}
