//go:build unix

package daemon

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gurisko/takl/internal/registry"
)

// Request/Response types

type RegisterProjectRequest struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type RegisterProjectResponse struct {
	Project *registry.Project `json:"project"`
}

type ListProjectsResponse struct {
	Projects []*registry.Project `json:"projects"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// Handler methods

func (d *Daemon) handleRegisterProject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()

	var req RegisterProjectRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)) // 1MB cap
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Name == "" {
		writeError(w, "name is required", http.StatusBadRequest)
		return
	}
	if req.Path == "" {
		writeError(w, "path is required", http.StatusBadRequest)
		return
	}

	// Create project
	project := &registry.Project{
		Name:         req.Name,
		Path:         req.Path,
		RegisteredAt: time.Now().UTC(),
	}

	// Register and save atomically
	if err := d.registry.RegisterAndSave(project); err != nil {
		if errors.Is(err, registry.ErrProjectAlreadyExists) {
			writeError(w, "project already exists at this path", http.StatusConflict)
			return
		}
		if errors.Is(err, registry.ErrInvalidPath) {
			writeError(w, fmt.Sprintf("invalid path: %v", err), http.StatusBadRequest)
			return
		}
		writeError(w, fmt.Sprintf("failed to persist project: %v", err), http.StatusInternalServerError)
		return
	}

	// Return response with Location header
	resp := RegisterProjectResponse{
		Project: project,
	}
	w.Header().Set("Location", "/api/projects/"+project.ID)
	writeJSON(w, resp, http.StatusCreated)
}

func (d *Daemon) handleListProjects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	projects := d.registry.List()

	resp := ListProjectsResponse{
		Projects: projects,
	}
	writeJSON(w, resp, http.StatusOK)
}

// handleProjectByID routes requests to /api/projects/{id} to the appropriate handler
func (d *Daemon) handleProjectByID(w http.ResponseWriter, r *http.Request) {
	// Extract project ID from path: /api/projects/{id}
	projectID := strings.TrimPrefix(r.URL.Path, "/api/projects/")
	if projectID == "" || projectID == r.URL.Path {
		writeError(w, "project ID is required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodDelete:
		d.handleRemoveProject(w, r, projectID)
	default:
		w.Header().Set("Allow", "DELETE")
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (d *Daemon) handleRemoveProject(w http.ResponseWriter, r *http.Request, projectID string) {
	// Remove and save atomically
	_, err := d.registry.UnregisterAndSave(projectID)
	if err != nil {
		if errors.Is(err, registry.ErrProjectNotFound) {
			writeError(w, "project not found", http.StatusNotFound)
			return
		}
		writeError(w, fmt.Sprintf("failed to remove project: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Helper functions

func writeJSON(w http.ResponseWriter, data interface{}, status int) {
	buf, err := json.Marshal(data)
	if err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(buf)
}

func writeError(w http.ResponseWriter, message string, status int) {
	resp := ErrorResponse{
		Error: message,
	}
	writeJSON(w, resp, status)
}
