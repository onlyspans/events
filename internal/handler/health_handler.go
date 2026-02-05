package handler

import (
	"log/slog"
	"net/http"

	"github.com/onlyspans/events/internal/http/response"
	"github.com/onlyspans/events/internal/ports"
)

// HealthHandler handles health check requests.
type HealthHandler struct {
	healthChecker ports.HealthChecker
	logger        *slog.Logger
}

func NewHealthHandler(healthChecker ports.HealthChecker, logger *slog.Logger) *HealthHandler {
	return &HealthHandler{
		healthChecker: healthChecker,
		logger:        logger,
	}
}

// healthStatus represents the health check response structure.
type healthStatus struct {
	Status   string `json:"status"`
	Database string `json:"database"`
	Error    string `json:"error,omitempty"`
}

// Readiness handles GET /readyz requests.
func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.MethodNotAllowed(w)
		return
	}

	if err := h.healthChecker.Ping(r.Context()); err != nil {
		h.logger.Error("database health check failed", "error", err)
		response.JSON(w, http.StatusServiceUnavailable, healthStatus{
			Status:   "DOWN",
			Database: "disconnected",
			Error:    err.Error(),
		})
		return
	}

	response.OK(w, healthStatus{
		Status:   "UP",
		Database: "connected",
	})
}

// Liveness handles GET /healthz requests.
func (h *HealthHandler) Liveness(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.MethodNotAllowed(w)
		return
	}

	response.OK(w, healthStatus{Status: "UP"})
}
