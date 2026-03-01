package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/onlyspans/events/internal/dto"
	"github.com/onlyspans/events/internal/http/response"
	"github.com/onlyspans/events/internal/ports"
)

const (
	searchTimeout = 30 * time.Second
	exportTimeout = 60 * time.Second
	ingestTimeout = 5 * time.Second
	batchTimeout  = 30 * time.Second
	maxPageSize   = 1000
	maxBatchSize  = 100
)

// EventHandler handles HTTP requests for events.
type EventHandler struct {
	eventService ports.EventService
	logger       *slog.Logger
}

func NewEventHandler(eventService ports.EventService, logger *slog.Logger) *EventHandler {
	return &EventHandler{
		eventService: eventService,
		logger:       logger,
	}
}

// SearchEvents handles POST /events requests.
func (h *EventHandler) SearchEvents(w http.ResponseWriter, r *http.Request) {
	var req dto.SearchEventsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode search request", "error", err)
		response.BadRequest(w, "Invalid request body")
		return
	}

	if req.Size > maxPageSize {
		h.logger.Warn("page size exceeds maximum", "requested", req.Size, "max", maxPageSize)
		response.BadRequest(w, fmt.Sprintf("Page size too large (max %d)", maxPageSize))
		return
	}

	h.logger.Debug("searching events", "request", req)

	ctx, cancel := context.WithTimeout(r.Context(), searchTimeout)
	defer cancel()

	result, err := h.eventService.SearchEvents(ctx, req)
	if err != nil {
		h.logger.Error("failed to search events", "error", err)
		response.Error(w, err)
		return
	}

	response.OK(w, result)
}

// ExportEvents handles POST /events/export requests.
func (h *EventHandler) ExportEvents(w http.ResponseWriter, r *http.Request) {
	var req dto.ExportEventsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode export request", "error", err)
		response.BadRequest(w, "Invalid request body")
		return
	}

	h.logger.Info("export request received", "entity_id", req.EntityID, "action", req.Action)

	timestamp := time.Now().UTC().Format("20060102_150405")
	filename := fmt.Sprintf("events-export_%s_utc.csv", timestamp)

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	ctx, cancel := context.WithTimeout(r.Context(), exportTimeout)
	defer cancel()

	if err := h.eventService.ExportCSV(ctx, req, w); err != nil {
		h.logger.Error("failed to export events", "error", err)
		// Note: Headers already sent, can't send error response
	}
}

// IngestEvent handles POST /events/ingest requests for single event ingestion.
func (h *EventHandler) IngestEvent(w http.ResponseWriter, r *http.Request) {
	var req dto.EventIngestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode ingest request", "error", err)
		response.BadRequest(w, "Invalid request body")
		return
	}

	h.logger.Debug("ingest event request", "entity_id", req.EntityID, "action", req.Action)

	ctx, cancel := context.WithTimeout(r.Context(), ingestTimeout)
	defer cancel()

	eventID, err := h.eventService.CreateEvent(ctx, req)
	if err != nil {
		h.logger.Error("failed to create event", "error", err)
		response.Error(w, err)
		return
	}

	h.logger.Info("event created successfully", "id", eventID)
	response.Created(w, dto.SingleIngestResponse{ID: eventID})
}

// IngestEventsBatch handles POST /events/ingest/batch requests for batch event ingestion.
func (h *EventHandler) IngestEventsBatch(w http.ResponseWriter, r *http.Request) {
	var req dto.BatchIngestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode batch ingest request", "error", err)
		response.BadRequest(w, "Invalid request body")
		return
	}

	if len(req.Events) > maxBatchSize {
		h.logger.Warn("batch size exceeds maximum", "requested", len(req.Events), "max", maxBatchSize)
		response.BadRequest(w, fmt.Sprintf("Batch size too large (max %d)", maxBatchSize))
		return
	}

	if len(req.Events) == 0 {
		h.logger.Warn("empty batch request")
		response.BadRequest(w, "Batch must contain at least one event")
		return
	}

	h.logger.Debug("batch ingest request", "count", len(req.Events))

	ctx, cancel := context.WithTimeout(r.Context(), batchTimeout)
	defer cancel()

	result := h.eventService.CreateEventsBatch(ctx, req.Events)

	h.logger.Info("batch processing completed",
		"success", result.SuccessCount,
		"failed", result.FailureCount,
		"total", len(req.Events))

	response.MultiStatus(w, result)
}
