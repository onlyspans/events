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
	"github.com/onlyspans/events/internal/ports"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var _ ports.EventService = (*EventService)(nil)

var (
	eventsIngestedCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "events_ingested_total",
		Help: "Total number of events ingested",
	})

	eventsSearchedCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "events_searched_total",
		Help: "Total number of search operations",
	})

	eventsExportedCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "events_exported_total",
		Help: "Total number of events exported",
	})
)

var (
	eventsIngestSingleCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "events_ingest_single_total",
		Help: "Total number of single events ingested via HTTP",
	})

	eventsIngestBatchCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "events_ingest_batch_total",
		Help: "Total number of batch events ingested via HTTP",
	})

	eventsIngestFailedCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "events_ingest_failed_total",
		Help: "Total number of events that failed to ingest",
	})
)

type EventService struct {
	repo          ports.EventRepository
	maxExportSize int
}

func NewEventService(repo ports.EventRepository, maxExportSize int) *EventService {
	return &EventService{
		repo:          repo,
		maxExportSize: maxExportSize,
	}
}

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

func (s *EventService) SearchEvents(ctx context.Context, req dto.SearchEventsRequest) (*dto.QueryResult, error) {
	eventsSearchedCounter.Inc()

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

	query := ports.EventSearchQuery{
		EntityID:   req.EntityID,
		EntityName: req.EntityName,
		Action:     req.Action,
		UserID:     req.UserID,
		Tenant:     req.Tenant,
		StartDate:  req.StartDate,
		EndDate:    req.EndDate,
		SortBy:     req.SortBy,
		SortOrder:  req.SortOrder,
		Page:       req.Page,
		Size:       req.Size,
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

func (s *EventService) ExportCSV(ctx context.Context, req dto.ExportEventsRequest, writer io.Writer) error {
	if req.SortBy == "" {
		req.SortBy = "timestamp"
	}
	if req.SortOrder == "" {
		req.SortOrder = "desc"
	}

	query := ports.EventSearchQuery{
		EntityID:   req.EntityID,
		EntityName: req.EntityName,
		Action:     req.Action,
		UserID:     req.UserID,
		Tenant:     req.Tenant,
		StartDate:  req.StartDate,
		EndDate:    req.EndDate,
		SortBy:     req.SortBy,
		SortOrder:  req.SortOrder,
		Page:       0,
		Size:       s.maxExportSize,
	}

	events, _, err := s.repo.Search(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to search events for export: %w", err)
	}

	csvWriter := csv.NewWriter(writer)
	defer csvWriter.Flush()

	header := []string{
		"ID", "Timestamp", "Entity ID", "Entity Name", "Action",
		"User ID", "IP Address", "User Agent", "Tenant",
	}
	if err := csvWriter.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	for _, event := range events {
		row := []string{
			event.ID.String(),
			event.Timestamp.Format(time.RFC3339),
			event.EntityID.String(),
			event.EntityName,
			event.Action,
			event.UserID,
			event.IPAddress,
			event.UserAgent,
			event.Tenant,
		}

		if err := csvWriter.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	eventsExportedCounter.Add(float64(len(events)))
	return nil
}

func (s *EventService) dtoToEntity(dto *dto.EventDTO) (*domain.Event, error) {
	event := &domain.Event{
		Timestamp:  dto.Timestamp,
		EntityName: dto.EntityName,
		Action:     dto.Action,
		UserID:     dto.UserID,
		IPAddress:  dto.IPAddress,
		UserAgent:  dto.UserAgent,
		Tenant:     dto.Tenant,
	}

	if dto.ID != "" {
		id, err := uuid.Parse(dto.ID)
		if err != nil {
			event.ID = uuid.New()
		} else {
			event.ID = id
		}
	} else {
		event.ID = uuid.New()
	}

	if dto.EntityID != "" {
		entityID, err := uuid.Parse(dto.EntityID)
		if err != nil {
			return nil, fmt.Errorf("invalid entity_id: %w", err)
		}
		event.EntityID = entityID
	}

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	if len(dto.Changes) > 0 {
		event.Changes = make([]domain.Change, len(dto.Changes))
		for i, changeDTO := range dto.Changes {
			event.Changes[i] = domain.Change{
				Field:    changeDTO.Field,
				OldValue: changeDTO.OldValue,
				NewValue: changeDTO.NewValue,
			}
		}
	}

	return event, nil
}

func (s *EventService) entityToDTO(entity *domain.Event) dto.EventDTO {
	eventDTO := dto.EventDTO{
		ID:         entity.ID.String(),
		Timestamp:  entity.Timestamp,
		EntityID:   entity.EntityID.String(),
		EntityName: entity.EntityName,
		Action:     entity.Action,
		UserID:     entity.UserID,
		IPAddress:  entity.IPAddress,
		UserAgent:  entity.UserAgent,
		Tenant:     entity.Tenant,
	}

	if len(entity.Changes) > 0 {
		eventDTO.Changes = make([]dto.ChangeDTO, len(entity.Changes))
		for i, change := range entity.Changes {
			eventDTO.Changes[i] = dto.ChangeDTO{
				Field:    change.Field,
				OldValue: change.OldValue,
				NewValue: change.NewValue,
			}
		}
	}

	return eventDTO
}

func (s *EventService) processIngestRequest(req dto.EventIngestRequest) (*domain.Event, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	event := req.ToEvent()

	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	return event, nil
}

func (s *EventService) CreateEvent(ctx context.Context, req dto.EventIngestRequest) (uuid.UUID, error) {
	event, err := s.processIngestRequest(req)
	if err != nil {
		eventsIngestFailedCounter.Inc()
		return uuid.Nil, err
	}

	if err := s.repo.SaveBatch(ctx, []*domain.Event{event}); err != nil {
		eventsIngestFailedCounter.Inc()
		return uuid.Nil, fmt.Errorf("failed to save event: %w", err)
	}

	eventsIngestSingleCounter.Inc()
	return event.ID, nil
}

func (s *EventService) CreateEventsBatch(ctx context.Context, requests []dto.EventIngestRequest) dto.BatchIngestResponse {
	response := dto.BatchIngestResponse{
		SuccessCount: 0,
		FailureCount: 0,
		Errors:       []dto.BatchError{},
	}

	for i, req := range requests {
		event, err := s.processIngestRequest(req)
		if err != nil {
			response.FailureCount++
			response.Errors = append(response.Errors, dto.BatchError{
				Index: i,
				Error: fmt.Sprintf("event %d: %v", i, err),
			})
			eventsIngestFailedCounter.Inc()
			continue
		}

		if err := s.repo.SaveBatch(ctx, []*domain.Event{event}); err != nil {
			response.FailureCount++
			response.Errors = append(response.Errors, dto.BatchError{
				Index: i,
				Error: fmt.Sprintf("event %d: failed to save: %v", i, err),
			})
			eventsIngestFailedCounter.Inc()
			continue
		}

		response.SuccessCount++
		eventsIngestBatchCounter.Inc()
	}

	return response
}
