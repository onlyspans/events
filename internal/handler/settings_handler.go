package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/onlyspans/events/internal/dto"
	"github.com/onlyspans/events/internal/service"
)

// SettingsHandler handles HTTP requests for settings.
type SettingsHandler struct {
	settingsService *service.SettingsService
	logger          *slog.Logger
}

// NewSettingsHandler creates a new SettingsHandler.
func NewSettingsHandler(settingsService *service.SettingsService, logger *slog.Logger) *SettingsHandler {
	return &SettingsHandler{
		settingsService: settingsService,
		logger:          logger,
	}
}

// GetSettings handles GET /settings requests.
func (h *SettingsHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.logger.Debug("getting settings")

	settings, err := h.settingsService.GetSettings(r.Context())
	if err != nil {
		h.logger.Error("failed to get settings", "error", err)
		http.Error(w, "Failed to get settings", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(settings); err != nil {
		h.logger.Error("failed to encode response", "error", err)
	}
}

// UpdateSettings handles PUT /settings requests.
func (h *SettingsHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req dto.SettingsDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode settings request", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	h.logger.Info("updating settings", "request", req)

	settings, err := h.settingsService.UpdateSettings(r.Context(), &req)
	if err != nil {
		h.logger.Error("failed to update settings", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(settings); err != nil {
		h.logger.Error("failed to encode response", "error", err)
	}
}
