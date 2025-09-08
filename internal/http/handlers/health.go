package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/takl/takl/internal/registry"
)

// HealthHandler handles health and monitoring endpoints
type HealthHandler struct {
	registry *registry.Registry
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(registry *registry.Registry) *HealthHandler {
	return &HealthHandler{
		registry: registry,
	}
}

// HandleHealth handles GET /health
func (h *HealthHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, fmt.Errorf("method not allowed"))
		return
	}

	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"version":   "dev", // TODO: Get from build info
	}

	// Basic registry health check
	projects := h.registry.ListProjects()
	health["projects_count"] = len(projects)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(health); err != nil {
		writeError(w, fmt.Errorf("failed to encode response: %w", err))
	}
}

// HandleStats handles GET /stats
func (h *HealthHandler) HandleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, fmt.Errorf("method not allowed"))
		return
	}

	projects := h.registry.ListProjects()

	stats := map[string]interface{}{
		"projects": map[string]interface{}{
			"total": len(projects),
		},
		"timestamp": time.Now().UTC(),
	}

	// TODO: Add more detailed stats:
	// - Total issues across all projects
	// - Issues by status/type/priority
	// - Indexer stats
	// - Watcher stats

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// HandleReload handles POST /api/reload
func (h *HealthHandler) HandleReload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, fmt.Errorf("method not allowed"))
		return
	}

	// TODO: Implement registry reload logic
	result := map[string]interface{}{
		"status":    "reloaded",
		"timestamp": time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
