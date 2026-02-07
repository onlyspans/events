package handler

import (
	"log/slog"
	"net/http"

	"github.com/onlyspans/events/internal/http/response"
	"github.com/onlyspans/events/internal/ports"
	"github.com/onlyspans/events/pkg/version"
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

type healthStatus struct {
	Status   string `json:"status"`
	Database string `json:"database"`
	Error    string `json:"error,omitempty"`
}

// Readiness handles GET /readyz requests.
func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request) {
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
func (h *HealthHandler) Liveness(w http.ResponseWriter, _ *http.Request) {
	response.OK(w, healthStatus{Status: "UP"})
}

func (h *HealthHandler) Version(w http.ResponseWriter, _ *http.Request) {
	response.JSON(w, http.StatusOK, version.Get())
}
