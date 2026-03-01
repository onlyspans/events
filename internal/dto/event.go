package dto

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/onlyspans/events/internal/domain"
)

type EventDTO struct {
	ID         string      `json:"id,omitempty"`
	Timestamp  time.Time   `json:"timestamp"`
	EntityID   string      `json:"entityId"`
	EntityName string      `json:"entityName,omitempty"`
	Action     string      `json:"action"`
	UserID     string      `json:"userId,omitempty"`
	IPAddress  string      `json:"ipAddress,omitempty"`
	UserAgent  string      `json:"userAgent,omitempty"`
	Tenant     string      `json:"tenant,omitempty"`
	Changes    []ChangeDTO `json:"changes,omitempty"`
}

type ChangeDTO struct {
	Field    string `json:"field,omitempty"`
	OldValue string `json:"oldValue,omitempty"`
	NewValue string `json:"newValue,omitempty"`
}

type EventFilterRequest struct {
	EntityID   string     `json:"entityId,omitempty"`
	EntityName string     `json:"entityName,omitempty"`
	Action     string     `json:"action,omitempty"`
	UserID     string     `json:"userId,omitempty"`
	Tenant     string     `json:"tenant,omitempty"`
	StartDate  *time.Time `json:"startDate,omitempty"`
	EndDate    *time.Time `json:"endDate,omitempty"`
	SortBy     string     `json:"sortBy,omitempty"`
	SortOrder  string     `json:"sortOrder,omitempty"`
}

type SearchEventsRequest struct {
	EventFilterRequest
	Page int `json:"page,omitempty"`
	Size int `json:"size,omitempty"`
}

type ExportEventsRequest = EventFilterRequest

type QueryResult struct {
	Events     []EventDTO `json:"events"`
	Total      int64      `json:"total"`
	Page       int        `json:"page"`
	Size       int        `json:"size"`
	TotalPages int        `json:"totalPages"`
}

type EventIngestRequest struct {
	Timestamp  time.Time   `json:"timestamp"`
	EntityID   string      `json:"entity_id"`
	EntityName string      `json:"entity_name,omitempty"`
	Action     string      `json:"action"`
	UserID     string      `json:"user_id,omitempty"`
	IPAddress  string      `json:"ip_address,omitempty"`
	UserAgent  string      `json:"user_agent,omitempty"`
	Tenant     string      `json:"tenant,omitempty"`
	Changes    []ChangeDTO `json:"changes,omitempty"`
}

func (e *EventIngestRequest) Validate() error {
	if e.EntityID == "" {
		return fmt.Errorf("missing required field: entity_id")
	}
	if e.Action == "" {
		return fmt.Errorf("missing required field: action")
	}
	return nil
}

func (e *EventIngestRequest) ToEvent() *domain.Event {
	entityID, err := uuid.Parse(e.EntityID)
	if err != nil {
		entityID = uuid.Nil
	}

	var changes []domain.Change
	if len(e.Changes) > 0 {
		changes = make([]domain.Change, len(e.Changes))
		for i, c := range e.Changes {
			changes[i] = domain.Change{
				Field:    c.Field,
				OldValue: c.OldValue,
				NewValue: c.NewValue,
			}
		}
	}

	return &domain.Event{
		Timestamp:  e.Timestamp,
		EntityID:   entityID,
		EntityName: e.EntityName,
		Action:     e.Action,
		UserID:     e.UserID,
		IPAddress:  e.IPAddress,
		UserAgent:  e.UserAgent,
		Tenant:     e.Tenant,
		Changes:    changes,
	}
}

type BatchIngestRequest struct {
	Events []EventIngestRequest `json:"events"`
}

type BatchIngestResponse struct {
	SuccessCount int          `json:"success_count"`
	FailureCount int          `json:"failure_count"`
	Errors       []BatchError `json:"errors,omitempty"`
}

type BatchError struct {
	Index int    `json:"index"`
	Error string `json:"error"`
}

type SingleIngestResponse struct {
	ID uuid.UUID `json:"id"`
}
