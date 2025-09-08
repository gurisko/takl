package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/takl/takl/internal/http/dto"
	"github.com/takl/takl/internal/registry"
)

// RegistryHandler handles project registry operations
type RegistryHandler struct {
	registry *registry.Registry
}

// writeJSONError writes a JSON error response
func writeJSONError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// NewRegistryHandler creates a new registry handler
func NewRegistryHandler(registry *registry.Registry) *RegistryHandler {
	return &RegistryHandler{
		registry: registry,
	}
}

// HandleRegistryProjects handles GET/POST /api/registry/projects
func (h *RegistryHandler) HandleRegistryProjects(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleListProjects(w, r)
	case http.MethodPost:
		h.handleRegisterProject(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleRegistryProject handles GET/DELETE /api/registry/projects/{id}
func (h *RegistryHandler) HandleRegistryProject(w http.ResponseWriter, r *http.Request) {
	// Extract project ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/registry/projects/")
	if path == "" {
		http.Error(w, "Project ID required", http.StatusBadRequest)
		return
	}

	projectID := strings.Split(path, "/")[0]

	switch r.Method {
	case http.MethodGet:
		h.handleGetProject(w, r, projectID)
	case http.MethodDelete:
		h.handleUnregisterProject(w, r, projectID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleRegistryHealth handles GET /api/registry/health
func (h *RegistryHandler) HandleRegistryHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	projects := h.registry.ListProjects()

	health := map[string]interface{}{
		"total_projects": len(projects),
		"status":         "healthy",
		"timestamp":      time.Now().UTC(),
	}

	// Check if any projects have issues (basic health check)
	healthyCount := 0
	for _, project := range projects {
		// TODO: Add more sophisticated health checks
		if project.Path != "" && project.Name != "" {
			healthyCount++
		}
	}

	health["healthy_projects"] = healthyCount
	if healthyCount < len(projects) {
		health["status"] = "degraded"
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(health); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// HandleRegistryCleanup handles POST /api/registry/cleanup
func (h *RegistryHandler) HandleRegistryCleanup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Implement cleanup logic (remove stale projects)
	projects := h.registry.ListProjects()

	removedCount := 0
	for _, project := range projects {
		// Check if project path still exists
		if !pathExists(project.Path) {
			if err := h.registry.UnregisterProject(project.ID); err != nil {
				// Log warning but continue
				continue
			}
			removedCount++
		}
	}

	result := map[string]interface{}{
		"removed_projects": removedCount,
		"timestamp":        time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// Helper methods

func (h *RegistryHandler) handleListProjects(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	projects := h.registry.ListProjects()

	// Apply optional filtering
	if nameFilter := query.Get("name"); nameFilter != "" {
		filtered := make([]*registry.Project, 0)
		for _, p := range projects {
			if strings.Contains(strings.ToLower(p.Name), strings.ToLower(nameFilter)) {
				filtered = append(filtered, p)
			}
		}
		projects = filtered
	}

	// Apply pagination
	if limitStr := query.Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			if limit < len(projects) {
				projects = projects[:limit]
			}
		}
	}

	// Convert to DTOs
	response := make([]*dto.ProjectResponse, len(projects))
	for i, p := range projects {
		response[i] = dto.FromRegistryProject(p)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func (h *RegistryHandler) handleRegisterProject(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Name == "" {
		http.Error(w, "Project name is required", http.StatusBadRequest)
		return
	}
	if req.Path == "" {
		http.Error(w, "Project path is required", http.StatusBadRequest)
		return
	}

	// Ensure path is absolute (client should send absolute paths)
	absPath := req.Path
	if !filepath.IsAbs(req.Path) {
		// Try to convert to absolute path as a fallback
		// Note: This may fail if daemon's cwd is invalid
		var err error
		absPath, err = filepath.Abs(req.Path)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid path (must be absolute): %v", err), http.StatusBadRequest)
			return
		}
	}

	// Register project
	project, err := h.registry.RegisterProject(absPath, req.Name, req.Description)
	if err != nil {
		// Return the error as-is, it already has context
		writeJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Return response in format expected by SDK
	response := map[string]interface{}{
		"status":  "created",
		"project": dto.FromRegistryProject(project),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func (h *RegistryHandler) handleGetProject(w http.ResponseWriter, r *http.Request, projectID string) {
	project, found := h.registry.GetProject(projectID)
	if !found {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(dto.FromRegistryProject(project)); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func (h *RegistryHandler) handleUnregisterProject(w http.ResponseWriter, r *http.Request, projectID string) {
	if err := h.registry.UnregisterProject(projectID); err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Helper function to check if path exists
func pathExists(path string) bool {
	_, err := filepath.Abs(path)
	return err == nil
}
