package grpc

import (
	"context"
	"log/slog"
	"time"

	eventsv1 "github.com/onlyspans/events/gen/go/events/v1"
	"github.com/onlyspans/events/internal/dto"
	"github.com/onlyspans/events/internal/ports"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type EventServer struct {
	eventsv1.UnimplementedEventServiceServer
	eventService ports.EventService
	logger       *slog.Logger
}

func NewEventServer(eventService ports.EventService, logger *slog.Logger) *EventServer {
	return &EventServer{eventService: eventService, logger: logger}
}

func (s *EventServer) IngestEvent(ctx context.Context, req *eventsv1.IngestEventRequest) (*eventsv1.IngestEventResponse, error) {
	ingestReq := protoToIngestRequest(req)
	id, err := s.eventService.CreateEvent(ctx, ingestReq)
	if err != nil {
		return nil, toGRPCError(s.logger, err)
	}
	return &eventsv1.IngestEventResponse{Id: id.String()}, nil
}

func (s *EventServer) IngestEventBatch(ctx context.Context, req *eventsv1.IngestEventBatchRequest) (*eventsv1.IngestEventBatchResponse, error) {
	reqs := make([]dto.EventIngestRequest, len(req.Events))
	for i, e := range req.Events {
		reqs[i] = protoToIngestRequest(e)
	}

	result := s.eventService.CreateEventsBatch(ctx, reqs)

	batchErrors := make([]*eventsv1.BatchError, len(result.Errors))
	for i, e := range result.Errors {
		batchErrors[i] = &eventsv1.BatchError{
			Index:   int32(e.Index),
			Message: e.Error,
		}
	}

	return &eventsv1.IngestEventBatchResponse{
		SuccessCount: int32(result.SuccessCount),
		FailureCount: int32(result.FailureCount),
		Errors:       batchErrors,
	}, nil
}

func (s *EventServer) SearchEvents(ctx context.Context, req *eventsv1.SearchEventsRequest) (*eventsv1.SearchEventsResponse, error) {
	searchReq := dto.SearchEventsRequest{
		EventFilterRequest: dto.EventFilterRequest{
			EntityID:   req.EntityId,
			EntityName: req.EntityName,
			Action:     req.Action,
			UserID:     req.UserId,
			Tenant:     req.Tenant,
			SortBy:     req.SortBy,
			SortOrder:  req.SortOrder,
		},
		Page: int(req.Page),
		Size: int(req.Size),
	}
	if req.StartDate != nil {
		t := req.StartDate.AsTime()
		searchReq.StartDate = &t
	}
	if req.EndDate != nil {
		t := req.EndDate.AsTime()
		searchReq.EndDate = &t
	}

	result, err := s.eventService.SearchEvents(ctx, searchReq)
	if err != nil {
		return nil, toGRPCError(s.logger, err)
	}

	events := make([]*eventsv1.Event, len(result.Events))
	for i, e := range result.Events {
		events[i] = dtoToProtoEvent(e)
	}

	return &eventsv1.SearchEventsResponse{
		Events:     events,
		Total:      result.Total,
		Page:       int32(result.Page),
		Size:       int32(result.Size),
		TotalPages: int32(result.TotalPages),
	}, nil
}

func protoToIngestRequest(req *eventsv1.IngestEventRequest) dto.EventIngestRequest {
	var ts time.Time
	if req.Timestamp != nil {
		ts = req.Timestamp.AsTime()
	}

	changes := make([]dto.ChangeDTO, len(req.Changes))
	for i, c := range req.Changes {
		changes[i] = dto.ChangeDTO{
			Field:    c.Field,
			OldValue: c.OldValue,
			NewValue: c.NewValue,
		}
	}

	return dto.EventIngestRequest{
		Timestamp:  ts,
		EntityID:   req.EntityId,
		EntityName: req.EntityName,
		Action:     req.Action,
		UserID:     req.UserId,
		IPAddress:  req.IpAddress,
		UserAgent:  req.UserAgent,
		Tenant:     req.Tenant,
		Changes:    changes,
	}
}

func dtoToProtoEvent(e dto.EventDTO) *eventsv1.Event {
	changes := make([]*eventsv1.Change, len(e.Changes))
	for i, c := range e.Changes {
		changes[i] = &eventsv1.Change{
			Field:    c.Field,
			OldValue: c.OldValue,
			NewValue: c.NewValue,
		}
	}
	return &eventsv1.Event{
		Id:         e.ID,
		Timestamp:  timestamppb.New(e.Timestamp),
		EntityId:   e.EntityID,
		EntityName: e.EntityName,
		Action:     e.Action,
		UserId:     e.UserID,
		IpAddress:  e.IPAddress,
		UserAgent:  e.UserAgent,
		Tenant:     e.Tenant,
		Changes:    changes,
	}
}
