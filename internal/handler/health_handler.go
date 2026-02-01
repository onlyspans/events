package handler

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
)

// HealthHandler handles health check requests.
type HealthHandler struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler(db *sql.DB, logger *slog.Logger) *HealthHandler {
	return &HealthHandler{
		db:     db,
		logger: logger,
	}
}

// Readiness handles GET /readyz requests.
func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	health := make(map[string]interface{})

	// Check database connection
	if err := h.db.Ping(); err != nil {
		h.logger.Error("database health check failed", "error", err)
		health["status"] = "DOWN"
		health["database"] = "disconnected"
		health["error"] = err.Error()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(health)
		return
	}

	health["status"] = "UP"
	health["database"] = "connected"

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(health)
}

// Liveness handles GET /healthz requests.
func (h *HealthHandler) Liveness(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	health := map[string]string{"status": "UP"}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(health)
}
