package ports

import (
	"context"
	"io"

	"github.com/google/uuid"
	"github.com/onlyspans/events/internal/dto"
)

type EventService interface {
	IngestEvents(ctx context.Context, events []*dto.EventDTO) error
	SearchEvents(ctx context.Context, req dto.SearchEventsRequest) (*dto.QueryResult, error)
	ExportCSV(ctx context.Context, req dto.ExportEventsRequest, writer io.Writer) error
	CreateEvent(ctx context.Context, req dto.EventIngestRequest) (uuid.UUID, error)
	CreateEventsBatch(ctx context.Context, requests []dto.EventIngestRequest) dto.BatchIngestResponse
}

type SettingsService interface {
	GetSettings(ctx context.Context) (*dto.SettingsDTO, error)
	UpdateSettings(ctx context.Context, settings *dto.SettingsDTO) (*dto.SettingsDTO, error)
}
