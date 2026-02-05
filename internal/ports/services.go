package ports

import (
	"context"
	"io"

	"github.com/google/uuid"
	"github.com/onlyspans/events/internal/dto"
)

// EventService defines the contract for event business logic operations.
// This interface abstracts the event service, allowing handlers to depend
// on the interface rather than the concrete implementation.
type EventService interface {
	// IngestEvents processes and stores a batch of events from Kafka.
	IngestEvents(ctx context.Context, events []*dto.EventDTO) error

	// SearchEvents searches for events matching the query criteria.
	SearchEvents(ctx context.Context, req dto.SearchEventsRequest) (*dto.QueryResult, error)

	// ExportCSV exports events matching the query to CSV format.
	ExportCSV(ctx context.Context, req dto.ExportEventsRequest, writer io.Writer) error

	// CreateEvent creates a single event from an HTTP ingestion request.
	CreateEvent(ctx context.Context, req dto.EventIngestRequest) (uuid.UUID, error)

	// CreateEventsBatch processes a batch of event ingestion requests with partial success support.
	CreateEventsBatch(ctx context.Context, requests []dto.EventIngestRequest) dto.BatchIngestResponse
}

// SettingsService defines the contract for settings business logic operations.
// This interface abstracts the settings service for handler decoupling.
type SettingsService interface {
	// GetSettings retrieves the current application settings.
	GetSettings(ctx context.Context) (*dto.SettingsDTO, error)

	// UpdateSettings updates the application settings.
	UpdateSettings(ctx context.Context, settings *dto.SettingsDTO) (*dto.SettingsDTO, error)
}
