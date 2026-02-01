package dto

import (
	"time"
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

// SearchEventsRequest represents the search request parameters.
type SearchEventsRequest struct {
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
	Page          int        `json:"page,omitempty"`
	Size          int        `json:"size,omitempty"`
}

// ExportEventsRequest represents the export request parameters.
type ExportEventsRequest struct {
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

// QueryResult represents the search results with pagination.
type QueryResult struct {
	Events     []EventDTO `json:"events"`
	Total      int64      `json:"total"`
	Page       int        `json:"page"`
	Size       int        `json:"size"`
	TotalPages int        `json:"totalPages"`
}
