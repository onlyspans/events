package service

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/onlyspans/events/internal/domain"
	"github.com/onlyspans/events/internal/dto"
	"github.com/onlyspans/events/internal/repository"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	eventsIngestedCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "event_logs_ingested",
		Help: "Total number of events ingested",
	})

	eventsSearchedCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "event_logs_searched",
		Help: "Total number of search operations",
	})

	eventsExportedCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "event_logs_exported",
		Help: "Total number of events exported",
	})
)

// EventRepository defines the interface for event data access.
type EventRepository interface {
	SaveBatch(ctx context.Context, events []*domain.Event) error
	Search(ctx context.Context, query repository.SearchQuery) ([]*domain.Event, int64, error)
	DeleteOlderThan(ctx context.Context, cutoffDate time.Time) (int64, error)
}

// EventService handles event business logic.
type EventService struct {
	repo          EventRepository
	maxExportSize int
}

// NewEventService creates a new EventService.
func NewEventService(repo EventRepository, maxExportSize int) *EventService {
	return &EventService{
		repo:          repo,
		maxExportSize: maxExportSize,
	}
}

// IngestEvents processes and stores a batch of events.
func (s *EventService) IngestEvents(ctx context.Context, eventDTOs []*dto.EventDTO) error {
	if len(eventDTOs) == 0 {
		return nil
	}

	events := make([]*domain.Event, 0, len(eventDTOs))
	for _, dto := range eventDTOs {
		event, err := s.dtoToEntity(dto)
		if err != nil {
			return fmt.Errorf("failed to convert DTO to entity: %w", err)
		}
		events = append(events, event)
	}

	if err := s.repo.SaveBatch(ctx, events); err != nil {
		return fmt.Errorf("failed to save events: %w", err)
	}

	eventsIngestedCounter.Add(float64(len(events)))
	return nil
}

// SearchEvents searches for events matching the query criteria.
func (s *EventService) SearchEvents(ctx context.Context, req dto.SearchEventsRequest) (*dto.QueryResult, error) {
	eventsSearchedCounter.Inc()

	// Set defaults
	if req.SortBy == "" {
		req.SortBy = "timestamp"
	}
	if req.SortOrder == "" {
		req.SortOrder = "desc"
	}
	if req.Page < 0 {
		req.Page = 0
	}
	if req.Size <= 0 {
		req.Size = 20
	}

	query := repository.SearchQuery{
		User:          req.User,
		Category:      req.Category,
		Action:        req.Action,
		Document:      req.Document,
		Project:       req.Project,
		Environment:   req.Environment,
		Tenant:        req.Tenant,
		CorrelationID: req.CorrelationID,
		TraceID:       req.TraceID,
		StartDate:     req.StartDate,
		EndDate:       req.EndDate,
		SortBy:        req.SortBy,
		SortOrder:     req.SortOrder,
		Page:          req.Page,
		Size:          req.Size,
	}

	events, total, err := s.repo.Search(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to search events: %w", err)
	}

	eventDTOs := make([]dto.EventDTO, 0, len(events))
	for _, event := range events {
		eventDTOs = append(eventDTOs, s.entityToDTO(event))
	}

	totalPages := 0
	if req.Size > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(req.Size)))
	}

	return &dto.QueryResult{
		Events:     eventDTOs,
		Total:      total,
		Page:       req.Page,
		Size:       req.Size,
		TotalPages: totalPages,
	}, nil
}

// ExportCSV exports events matching the query to CSV format.
func (s *EventService) ExportCSV(ctx context.Context, req dto.ExportEventsRequest, writer io.Writer) error {
	// Set defaults
	if req.SortBy == "" {
		req.SortBy = "timestamp"
	}
	if req.SortOrder == "" {
		req.SortOrder = "desc"
	}

	query := repository.SearchQuery{
		User:          req.User,
		Category:      req.Category,
		Action:        req.Action,
		Document:      req.Document,
		Project:       req.Project,
		Environment:   req.Environment,
		Tenant:        req.Tenant,
		CorrelationID: req.CorrelationID,
		TraceID:       req.TraceID,
		StartDate:     req.StartDate,
		EndDate:       req.EndDate,
		SortBy:        req.SortBy,
		SortOrder:     req.SortOrder,
		Page:          0,
		Size:          s.maxExportSize,
	}

	events, _, err := s.repo.Search(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to search events for export: %w", err)
	}

	csvWriter := csv.NewWriter(writer)
	defer csvWriter.Flush()

	// Write header
	header := []string{
		"ID", "Timestamp", "User", "Category", "Action", "Document",
		"Project", "Environment", "Tenant", "Correlation ID", "Trace ID",
		"IP Address", "User Agent", "Additional Info",
	}
	if err := csvWriter.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write rows
	for _, event := range events {
		ipAddress := ""
		userAgent := ""
		additionalInfo := ""

		if event.Details != nil {
			ipAddress = event.Details.IPAddress
			userAgent = event.Details.UserAgent
			additionalInfo = event.Details.AdditionalInfo
		}

		row := []string{
			event.ID.String(),
			event.Timestamp.Format(time.RFC3339),
			event.User,
			event.Category,
			event.Action,
			event.DocumentName,
			event.Project,
			event.Environment,
			event.Tenant,
			event.CorrelationID,
			event.TraceID,
			ipAddress,
			userAgent,
			additionalInfo,
		}

		if err := csvWriter.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	eventsExportedCounter.Add(float64(len(events)))
	return nil
}

// dtoToEntity converts a DTO to a domain entity.
func (s *EventService) dtoToEntity(dto *dto.EventDTO) (*domain.Event, error) {
	event := &domain.Event{
		Timestamp:     dto.Timestamp,
		User:          dto.User,
		Category:      dto.Category,
		Action:        dto.Action,
		DocumentName:  dto.DocumentName,
		Project:       dto.Project,
		Environment:   dto.Environment,
		Tenant:        dto.Tenant,
		CorrelationID: dto.CorrelationID,
		TraceID:       dto.TraceID,
		CreatedAt:     time.Now(),
	}

	// Parse ID if provided, otherwise generate new one
	if dto.ID != "" {
		id, err := uuid.Parse(dto.ID)
		if err != nil {
			// Log warning but continue with new UUID
			event.ID = uuid.New()
		} else {
			event.ID = id
		}
	} else {
		event.ID = uuid.New()
	}

	// Set default timestamp if not provided
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Convert details
	if dto.Details != nil {
		details := &domain.EventDetails{
			IPAddress:      dto.Details.IPAddress,
			UserAgent:      dto.Details.UserAgent,
			AdditionalInfo: dto.Details.AdditionalInfo,
		}

		if len(dto.Details.Changes) > 0 {
			details.Changes = make([]domain.Change, len(dto.Details.Changes))
			for i, changeDTO := range dto.Details.Changes {
				details.Changes[i] = domain.Change{
					Field:    changeDTO.Field,
					OldValue: changeDTO.OldValue,
					NewValue: changeDTO.NewValue,
				}
			}
		}

		event.Details = details
	}

	return event, nil
}

// entityToDTO converts a domain entity to a DTO.
func (s *EventService) entityToDTO(entity *domain.Event) dto.EventDTO {
	eventDTO := dto.EventDTO{
		ID:            entity.ID.String(),
		Timestamp:     entity.Timestamp,
		User:          entity.User,
		Category:      entity.Category,
		Action:        entity.Action,
		DocumentName:  entity.DocumentName,
		Project:       entity.Project,
		Environment:   entity.Environment,
		Tenant:        entity.Tenant,
		CorrelationID: entity.CorrelationID,
		TraceID:       entity.TraceID,
	}

	if entity.Details != nil {
		eventDetailsDTO := &dto.EventDetailsDTO{
			IPAddress:      entity.Details.IPAddress,
			UserAgent:      entity.Details.UserAgent,
			AdditionalInfo: entity.Details.AdditionalInfo,
		}

		if len(entity.Details.Changes) > 0 {
			eventDetailsDTO.Changes = make([]dto.ChangeDTO, len(entity.Details.Changes))
			for i, change := range entity.Details.Changes {
				eventDetailsDTO.Changes[i] = dto.ChangeDTO{
					Field:    change.Field,
					OldValue: change.OldValue,
					NewValue: change.NewValue,
				}
			}
		}

		eventDTO.Details = eventDetailsDTO
	}

	return eventDTO
}
