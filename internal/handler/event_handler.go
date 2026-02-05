package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/onlyspans/events/internal/dto"
	"github.com/onlyspans/events/internal/ports"
)

// EventHandler handles HTTP requests for events.
type EventHandler struct {
	eventService ports.EventService
	logger       *slog.Logger
}

// NewEventHandler creates a new EventHandler.
func NewEventHandler(eventService ports.EventService, logger *slog.Logger) *EventHandler {
	return &EventHandler{
		eventService: eventService,
		logger:       logger,
	}
}

// SearchEvents handles POST /events requests.
func (h *EventHandler) SearchEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req dto.SearchEventsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode search request", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate page size to prevent DoS attacks
	const maxPageSize = 1000
	if req.Size > maxPageSize {
		h.logger.Warn("page size exceeds maximum", "requested", req.Size, "max", maxPageSize)
		http.Error(w, fmt.Sprintf("Page size too large (max %d)", maxPageSize), http.StatusBadRequest)
		return
	}

	h.logger.Debug("searching events", "request", req)

	// Add context timeout to prevent hanging requests
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	result, err := h.eventService.SearchEvents(ctx, req)
	if err != nil {
		h.logger.Error("failed to search events", "error", err)
		http.Error(w, "Failed to search events", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		h.logger.Error("failed to encode response", "error", err)
	}
}

// ExportEvents handles POST /events/export requests.
func (h *EventHandler) ExportEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req dto.ExportEventsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode export request", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	h.logger.Info("export request received", "user", req.User, "category", req.Category)

	// Generate filename with UTC timestamp
	timestamp := time.Now().UTC().Format("20060102_150405")
	filename := fmt.Sprintf("events-export_%s_utc.csv", timestamp)

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	// Add longer timeout for export operations
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	if err := h.eventService.ExportCSV(ctx, req, w); err != nil {
		h.logger.Error("failed to export events", "error", err)
		// Can't send error response after headers are written
		return
	}
}

// IngestEvent handles POST /events/ingest requests for single event ingestion.
func (h *EventHandler) IngestEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req dto.EventIngestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode ingest request", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	h.logger.Debug("ingest event request", "user", req.UserName, "category", req.Category, "action", req.Action)

	// Add context timeout for single event ingestion (5 seconds)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	eventID, err := h.eventService.CreateEvent(ctx, req)
	if err != nil {
		h.logger.Error("failed to create event", "error", err)
		http.Error(w, fmt.Sprintf("Failed to create event: %v", err), http.StatusInternalServerError)
		return
	}

	h.logger.Info("event created successfully", "id", eventID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(dto.SingleIngestResponse{ID: eventID}); err != nil {
		h.logger.Error("failed to encode response", "error", err)
	}
}

// IngestEventsBatch handles POST /events/ingest/batch requests for batch event ingestion.
func (h *EventHandler) IngestEventsBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req dto.BatchIngestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode batch ingest request", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Enforce batch size limit (max 100 events)
	const maxBatchSize = 100
	if len(req.Events) > maxBatchSize {
		h.logger.Warn("batch size exceeds maximum", "requested", len(req.Events), "max", maxBatchSize)
		http.Error(w, fmt.Sprintf("Batch size too large (max %d)", maxBatchSize), http.StatusBadRequest)
		return
	}

	if len(req.Events) == 0 {
		h.logger.Warn("empty batch request")
		http.Error(w, "Batch must contain at least one event", http.StatusBadRequest)
		return
	}

	h.logger.Debug("batch ingest request", "count", len(req.Events))

	// Add context timeout for batch ingestion (30 seconds)
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	response := h.eventService.CreateEventsBatch(ctx, req.Events)

	h.logger.Info("batch processing completed",
		"success", response.SuccessCount,
		"failed", response.FailureCount,
		"total", len(req.Events))

	// Return 207 Multi-Status for partial success scenarios
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusMultiStatus)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("failed to encode batch response", "error", err)
	}
}
