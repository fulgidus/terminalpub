package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/fulgidus/terminalpub/internal/db"
)

// HealthHandler handles health check requests
type HealthHandler struct {
	db *db.DB
}

// NewHealthHandler creates a new health check handler
func NewHealthHandler(database *db.DB) *HealthHandler {
	return &HealthHandler{db: database}
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status   string            `json:"status"`
	Services map[string]string `json:"services"`
	Time     string            `json:"time"`
}

// ServeHTTP implements http.Handler
func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	response := HealthResponse{
		Status:   "healthy",
		Services: make(map[string]string),
		Time:     time.Now().UTC().Format(time.RFC3339),
	}

	// Check database health
	if h.db != nil {
		if err := h.db.Health(ctx); err != nil {
			response.Status = "unhealthy"
			response.Services["database"] = "down: " + err.Error()
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			response.Services["database"] = "up"
		}
	} else {
		response.Services["database"] = "not configured"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
