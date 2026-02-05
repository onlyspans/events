package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/onlyspans/events/internal/dto"
	"github.com/onlyspans/events/internal/http/response"
	"github.com/onlyspans/events/internal/ports"
)

// SettingsHandler handles HTTP requests for settings.
type SettingsHandler struct {
	settingsService ports.SettingsService
	logger          *slog.Logger
}

func NewSettingsHandler(settingsService ports.SettingsService, logger *slog.Logger) *SettingsHandler {
	return &SettingsHandler{
		settingsService: settingsService,
		logger:          logger,
	}
}

// GetSettings handles GET /settings requests.
// Note: Method routing is handled by the caller in main.go.
func (h *SettingsHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("getting settings")

	settings, err := h.settingsService.GetSettings(r.Context())
	if err != nil {
		h.logger.Error("failed to get settings", "error", err)
		response.Error(w, err)
		return
	}

	response.OK(w, settings)
}

// UpdateSettings handles PUT /settings requests.
// Note: Method routing is handled by the caller in main.go.
func (h *SettingsHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var req dto.SettingsDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode settings request", "error", err)
		response.BadRequest(w, "Invalid request body")
		return
	}

	h.logger.Info("updating settings", "request", req)

	settings, err := h.settingsService.UpdateSettings(r.Context(), &req)
	if err != nil {
		h.logger.Error("failed to update settings", "error", err)
		response.Error(w, err)
		return
	}

	response.OK(w, settings)
}
